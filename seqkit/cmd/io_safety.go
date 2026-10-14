package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shenwei356/bio/seqio/fai"
)

// indexHeaderTabs matches FAI's normalization of consecutive header tabs.
var indexHeaderTabs = regexp.MustCompile(`\t+`)

// closeOutput reports buffered write and compression-finalization failures.
func closeOutput(w io.Closer) { checkError(w.Close()) }

// resolvedPath resolves existing parents as well as not-yet-created output names.
func resolvedPath(path string) (string, error) {
	if path == "" {
		path = "."
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return filepath.Abs(resolved)
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	if target, linkErr := os.Readlink(path); linkErr == nil {
		if !filepath.IsAbs(target) {
			parent, _ := filepath.Split(path)
			target = parent + target // Do not clean away symlink/.. before resolving it.
		}
		return resolvedPath(target)
	}
	separators := string(filepath.Separator)
	if filepath.Separator == '\\' {
		separators += "/"
	}
	parent, base := filepath.Split(strings.TrimRight(path, separators))
	if base == "" {
		return "", err
	}
	parent, err = resolvedPath(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, base), nil
}

// isWorkingDir also recognizes absolute, relative, and symlink aliases of cwd.
func isWorkingDir(dir string) bool {
	resolved, err := resolvedPath(dir)
	checkError(err)
	cwd, err := resolvedPath(".")
	checkError(err)
	return resolved == cwd
}

// checkOutputFiles rejects input/output and output/output aliases before writing.
func checkOutputFiles(inputs, outputs []string) {
	paths := make(map[string]string, len(inputs)+len(outputs))
	for _, in := range inputs {
		if isStdin(in) {
			continue
		}
		path, err := resolvedPath(in)
		checkError(err)
		paths[path] = "input"
	}
	var infos []os.FileInfo // Cache metadata; new outputs need no pairwise file checks.
	var kinds []string
	loadedInputs := false
	for _, out := range outputs {
		if isStdin(out) {
			continue
		}
		path, err := resolvedPath(out)
		checkError(err)
		if kind, exists := paths[path]; exists {
			checkError(fmt.Errorf("%s and output files cannot be the same", kind))
		}
		paths[path] = "output"
		info, err := os.Stat(out)
		if os.IsNotExist(err) {
			continue
		}
		checkError(err)
		if !loadedInputs {
			for _, in := range inputs {
				if isStdin(in) {
					continue
				}
				inputInfo, err := os.Stat(in)
				if os.IsNotExist(err) {
					continue
				}
				checkError(err)
				infos, kinds = append(infos, inputInfo), append(kinds, "input")
			}
			loadedInputs = true
		}
		for i, previous := range infos {
			if os.SameFile(info, previous) {
				checkError(fmt.Errorf("%s and output files cannot be the same", kinds[i]))
			}
		}
		infos, kinds = append(infos, info), append(kinds, "output")
	}
}

// checkOutputDir prevents force cleanup from deleting any of the input files.
func checkOutputDir(inputs []string, dir string) {
	dir, err := resolvedPath(dir)
	checkError(err)
	if filepath.Dir(dir) == dir {
		checkError(fmt.Errorf("refusing to clear filesystem root: %s", dir))
	}
	cwd, err := os.Getwd()
	checkError(err)
	cwd, err = resolvedPath(cwd)
	checkError(err)
	rel, err := filepath.Rel(dir, cwd)
	checkError(err)
	if rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		checkError(fmt.Errorf("output directory contains the working directory: %s", dir))
	}
	for _, in := range inputs {
		if isStdin(in) {
			continue
		}
		path, err := resolvedPath(in)
		checkError(err)
		original, err := filepath.Abs(in)
		checkError(err)
		for _, candidate := range []string{path, original} {
			rel, err := filepath.Rel(dir, candidate)
			checkError(err)
			if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				checkError(fmt.Errorf("output directory contains input file: %s", in))
			}
		}
	}
}

// copySeqsToTemp exclusively reserves a temporary name and removes failed copies.
func copySeqsToTemp(file, dir string) (string, int, error) {
	fh, err := os.CreateTemp(dir, "seqkit-*.fastx")
	if err != nil {
		return "", 0, err
	}
	name := fh.Name()
	if err = fh.Close(); err != nil {
		_ = os.Remove(name)
		return "", 0, err
	}
	n, err := copySeqs(file, name)
	if err != nil {
		_ = os.Remove(name)
		return "", 0, err
	}
	return name, n, nil
}

// tempSeqDir keeps decompressed inputs on their original filesystem.
func tempSeqDir(file string) string {
	if isStdin(file) {
		return "."
	}
	return filepath.Dir(file)
}

// cleanupTempSeqs removes only this invocation's reserved file and derived index.
func cleanupTempSeqs(file string) {
	for _, path := range []string{file, file + ".seqkit.fai"} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			checkError(err)
		}
	}
}

// validateFASTAIndex checks full headers and offsets without scanning indexed bodies.
// Duplicate headers omitted by FAI are detected in both new and reused indexes.
func validateFASTAIndex(file string, index fai.Index) error {
	fh, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fh.Close()
	info, err := fh.Stat()
	if err != nil {
		return err
	}
	reader := bufio.NewReaderSize(fh, 64<<10)
	var offset int64 // Logical position, excluding buffered read-ahead.
	var records int  // Every indexed record must have exactly one physical header.
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return err
		}
		if len(line) == 0 {
			if records != len(index) {
				return fmt.Errorf("FASTA index does not match %s; use --update-faidx", file)
			}
			return nil
		}
		first := offset == 0
		offset += int64(len(line))
		line = bytes.TrimRight(line, "\r\n")
		if first {
			line = bytes.TrimPrefix(line, []byte{0xef, 0xbb, 0xbf})
		}
		if len(line) == 0 {
			continue
		}
		if line[0] != '>' {
			return fmt.Errorf("FASTA index does not match %s; use --update-faidx", file)
		}
		name := string(line[1:])
		if strings.ContainsRune(name, '\t') {
			name = indexHeaderTabs.ReplaceAllString(name, " ")
		}
		r, ok := index[name]
		if !ok {
			return fmt.Errorf("FASTA index does not match %s; use --update-faidx", file)
		}
		if r.Start != offset {
			return fmt.Errorf("duplicated sequence or outdated FASTA index: %s; rename duplicated headers or use --update-faidx", name)
		}
		records++
		end := r.Start
		if r.Length > 0 {
			if r.BasesPerLine <= 0 || r.BytesPerLine < r.BasesPerLine {
				return fmt.Errorf("invalid FASTA index for %s", name)
			}
			last := r.Length - 1
			end += int64(last/r.BasesPerLine)*int64(r.BytesPerLine) + int64(last%r.BasesPerLine) + 1
		}
		skip := end - offset
		if skip < 0 || end > info.Size() {
			return fmt.Errorf("invalid FASTA index for %s", name)
		}
		if skip <= int64(reader.Buffered()) {
			_, err = reader.Discard(int(skip))
		} else {
			_, err = fh.Seek(end, io.SeekStart)
			reader.Reset(fh)
		}
		if err != nil {
			return err
		}
		offset = end
	}
}
