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

// fastaIndexName matches the FAI writer's replacement of tab runs with one space.
func fastaIndexName(name string) string {
	if strings.IndexByte(name, '\t') < 0 {
		return name
	}
	return indexHeaderTabs.ReplaceAllString(name, " ")
}

// closeOutput reports buffered write and compression-finalization failures.
func closeOutput(w io.Closer) { checkError(w.Close()) }

// checkFASTAIndexFile rejects directories and special files before index updates.
// Follow symlinks so a directory alias is rejected without removing the link.
func checkFASTAIndexFile(file string) error {
	info, err := os.Stat(file)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("FASTA index must be a regular file: %s", file)
	}
	return nil
}

// removeFASTAIndex never recursively removes an index path or its contents.
func removeFASTAIndex(file string) error {
	if err := checkFASTAIndexFile(file); err != nil {
		return err
	}
	return os.Remove(file)
}

// resolvedPath resolves parent and leaf links without cleaning .. before symlinks.
func resolvedPath(path string) (string, error) {
	paths, err := filePathVariants(path)
	if err != nil {
		return "", err
	}
	return paths[len(paths)-1], nil
}

// Adapted from LexicMap's file path checks.
// filePathVariants preserves every symlink entry traversed on the way to a file.
// Directory cleanup can remove an intermediate entry even if the target is outside.
func filePathVariants(path string) ([]string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	paths := []string{absolute}
	// Keep .. components until preceding links have been followed.
	if filepath.IsAbs(path) {
		absolute = path
	} else if filepath.VolumeName(path) == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		absolute = cwd + string(filepath.Separator) + path
	}
	volume := filepath.VolumeName(absolute)
	parent := volume + string(filepath.Separator)
	components := func(p string) []string {
		return strings.FieldsFunc(p, func(c rune) bool { return c < 128 && os.IsPathSeparator(uint8(c)) })
	}
	pending := components(absolute[len(volume):])
	links := 0
	for len(pending) > 0 {
		part := pending[0]
		pending = pending[1:]
		if part == "." {
			continue
		}
		if part == ".." {
			parent = filepath.Dir(parent)
			continue
		}
		named := filepath.Join(parent, part)
		info, err := os.Lstat(named)
		if os.IsNotExist(err) {
			// Auxiliary outputs may not exist yet; their existing parents matter.
			// Continue through .. and later symlinks after a missing component.
			parent = named
			continue
		}
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			if len(pending) > 0 && !info.IsDir() {
				return nil, fmt.Errorf("path component %s is not a directory", named)
			}
			parent = named
			continue
		}
		paths = append(paths, named)
		links++
		if links > 255 {
			return nil, fmt.Errorf("too many symbolic links in %s", path)
		}
		target, err := os.Readlink(named)
		if err != nil {
			return nil, err
		}
		// Resolve one link at a time; EvalSymlinks on the parent would hide
		// entries in chains such as alias/db/inputs -> outside -> elsewhere.
		v := filepath.VolumeName(target)
		if filepath.IsAbs(target) {
			parent = v + string(filepath.Separator)
		} else if len(target) > len(v) && os.IsPathSeparator(target[len(v)]) {
			parent = filepath.VolumeName(parent) + string(filepath.Separator)
		}
		pending = append(components(target[len(v):]), pending...)
	}
	paths = append(paths, parent)
	return paths, nil
}

// isWorkingDir also recognizes absolute, relative, and symlink aliases of cwd.
func isWorkingDir(dir string) bool {
	resolved, err := resolvedPath(dir)
	checkError(err)
	cwd, err := resolvedPath(".")
	checkError(err)
	return resolved == cwd
}

// lexicalPath normalizes paths without probing their components. Cache cwd once
// for large input lists; filepath.Abs would look it up for every relative path.
func lexicalPath(path, cwd string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	if filepath.VolumeName(path) != "" || (len(path) > 0 && os.IsPathSeparator(path[0])) {
		// Windows drive-relative and rooted paths use the platform's cwd rules.
		return filepath.Abs(path)
	}
	return filepath.Join(cwd, path), nil
}

// checkOutputFiles rejects input/output and output/output aliases before writing.
// With checkInputFiles disabled, named inputs use only lexical path comparisons.
// Outputs and redirected stdin retain their filesystem identity checks.
func checkOutputFiles(inputs, outputs []string, checkInputFiles bool) {
	checkOutputFilesWithStdin(inputs, outputs, checkInputFiles, nil)
}

// inputStdinInfo captures stdin identity before sequence readers may close it.
func inputStdinInfo(inputs []string) (os.FileInfo, error) {
	for _, input := range inputs {
		if isStdin(input) {
			return os.Stdin.Stat()
		}
	}
	return nil, nil
}

// checkOutputFilesWithStdin accepts an earlier stdin snapshot for outputs whose
// names cannot be determined until the first input record has been read.
func checkOutputFilesWithStdin(inputs, outputs []string, checkInputFiles bool, stdinInfo os.FileInfo) {
	// outputFile retains only the few output identities, not every input's metadata.
	type outputFile struct {
		path     string
		absolute string // Lexical identity for inputs whose metadata checks are disabled.
		resolved string
		info     os.FileInfo // Nil for a new output, which cannot have hard-link aliases.
	}
	files := make([]outputFile, 0, len(outputs))
	for _, out := range outputs {
		if isStdin(out) {
			continue
		}
		path, err := resolvedPath(out)
		checkError(err)
		info, err := os.Stat(out)
		if err != nil && !os.IsNotExist(err) {
			checkError(err)
		}
		for _, previous := range files {
			if path == previous.resolved || (info != nil && previous.info != nil && os.SameFile(info, previous.info)) {
				checkError(fmt.Errorf("output and output files cannot be the same: %s and %s", previous.path, out))
			}
		}
		absolute, err := filepath.Abs(out)
		checkError(err)
		files = append(files, outputFile{path: out, absolute: absolute, resolved: path, info: info})
	}
	// Streaming to stdout needs no filesystem checks, even for large input lists.
	if len(files) == 0 {
		return
	}
	var cwd string
	if !checkInputFiles {
		var err error
		cwd, err = os.Getwd()
		checkError(err)
	}
	for _, input := range inputs {
		if !checkInputFiles && !isStdin(input) {
			path, err := lexicalPath(input, cwd)
			checkError(err)
			for _, output := range files {
				if path == output.absolute {
					checkError(fmt.Errorf("input and output files cannot be the same: %s and %s", input, output.path))
				}
			}
			continue
		}
		var path string
		var info os.FileInfo
		var err error
		statDone := false
		if isStdin(input) {
			// Redirected stdin can be the same file that -o is about to truncate.
			info = stdinInfo
			if info == nil {
				info, err = os.Stdin.Stat()
				checkError(err)
			}
			statDone = true
		} else {
			path, err = resolvedPath(input)
			checkError(err)
		}
		for _, output := range files {
			same := path != "" && path == output.resolved
			if !same && output.info != nil {
				if !statDone {
					info, err = os.Stat(input)
					if err != nil && !os.IsNotExist(err) {
						checkError(err)
					}
					statDone = true
				}
				same = info != nil && os.SameFile(info, output.info)
			}
			if same {
				checkError(fmt.Errorf("input and output files cannot be the same: %s and %s", input, output.path))
			}
		}
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
		// The FASTA reader and index builder allow comments before the first record.
		if records == 0 && line[0] == ';' {
			continue
		}
		if line[0] != '>' {
			return fmt.Errorf("FASTA index does not match %s; use --update-faidx", file)
		}
		name := fastaIndexName(string(line[1:]))
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
