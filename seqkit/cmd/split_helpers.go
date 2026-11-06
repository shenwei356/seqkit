package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
)

// prepareOutputDir preserves unrelated files even when force is set.
func prepareOutputDir(dir string, force bool) {
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		checkError(err)
	}
	if len(entries) > 0 && !force {
		log.Warningf("outdir not empty: %s; matching output files will be overwritten", dir)
	}
	checkError(os.MkdirAll(dir, 0755))
}

// writeFastxRecord keeps each input's format, including empty FASTQ records.
func writeFastxRecord(w *xopen.Writer, record *fastx.Record, fastq bool, width int) {
	if fastq && len(record.Seq.Qual) == 0 {
		w.WriteString("@" + string(record.Name) + "\n" + string(record.Seq.Seq) + "\n+\n\n")
		return
	}
	record.FormatToWriter(w, width)
}

// validateOutputPrefix restricts prefix flags to filenames on every platform.
func validateOutputPrefix(prefix string) {
	drive := len(prefix) >= 2 && prefix[1] == ':' && (prefix[0] >= 'a' && prefix[0] <= 'z' || prefix[0] >= 'A' && prefix[0] <= 'Z')
	if strings.ContainsAny(prefix, "/\\") || drive || filepath.IsAbs(prefix) || filepath.VolumeName(prefix) != "" {
		checkError(fmt.Errorf("output prefix must be a file name, not a path: %q", prefix))
	}
}

// splitOutputRegistry rejects aliases of inputs and previously opened outputs.
type splitOutputRegistry struct {
	inputs             []string
	auxiliaryInputs    []string        // Indexes and input lists retain full identity checks.
	paths              map[string]bool // Resolved paths make the common case constant time.
	identities         map[string]bool // Physical identities also catch hard-link aliases.
	skipInputFileCheck bool            // Compare only lexical paths for sequence inputs.
	stdinInfo          os.FileInfo     // Captured before readers may close stdin.
}

// open registers each logical output before truncating any existing file.
func (r *splitOutputRegistry) open(path string) (*xopen.Writer, error) {
	resolved, err := resolvedPath(path)
	if err != nil {
		return nil, err
	}
	if r.paths == nil {
		r.paths = make(map[string]bool)
	}
	if r.paths[resolved] {
		return nil, fmt.Errorf("output files cannot be the same: %s", path)
	}
	info, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	lexicalTarget := resolved
	var cwd string
	if r.skipInputFileCheck {
		cwd, err = os.Getwd()
		if err != nil {
			return nil, err
		}
		lexicalTarget, err = lexicalPath(path, cwd)
		if err != nil {
			return nil, err
		}
	}
	for group, inputs := range [][]string{r.inputs, r.auxiliaryInputs} {
		skipCheck := group == 0 && r.skipInputFileCheck
		inputTarget := resolved
		if skipCheck {
			inputTarget = lexicalTarget
		}
		for _, input := range inputs {
			if isStdin(input) {
				if info != nil && r.stdinInfo != nil && os.SameFile(info, r.stdinInfo) {
					return nil, fmt.Errorf("input and output files cannot be the same: %s", path)
				}
				continue
			}
			var inPath string
			if skipCheck {
				inPath, err = lexicalPath(input, cwd)
			} else {
				inPath, err = resolvedPath(input)
			}
			if err != nil {
				return nil, err
			}
			if inPath == inputTarget {
				return nil, fmt.Errorf("input and output files cannot be the same: %s", path)
			}
			if !skipCheck && info != nil {
				inInfo, err := os.Stat(input)
				if err != nil {
					return nil, err
				}
				if os.SameFile(info, inInfo) {
					return nil, fmt.Errorf("input and output files cannot be the same: %s", path)
				}
			}
		}
	}
	if info != nil {
		identity, err := fileIdentity(path, info)
		if err != nil {
			return nil, err
		}
		if r.identities[identity] {
			return nil, fmt.Errorf("output files cannot be the same: %s", path)
		}
	}
	w, err := xopen.Wopen(path)
	if err != nil {
		return nil, err
	}
	r.paths[resolved] = true
	info, err = os.Stat(path)
	if err != nil {
		w.Close()
		return nil, err
	}
	identity, err := fileIdentity(path, info)
	if err != nil {
		w.Close()
		return nil, err
	}
	if r.identities == nil {
		r.identities = make(map[string]bool)
	}
	r.identities[identity] = true
	return w, nil
}

// balancedPartSize assigns the remainder to the earliest nonempty parts.
func balancedPartSize(total, parts, index int) int {
	if total == 0 {
		return 0
	}
	parts = min(parts, total)
	if index >= parts {
		return 0
	}
	n := total / parts
	if index < total%parts {
		n++
	}
	return n
}
