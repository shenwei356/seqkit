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
	inputs []string
	paths  map[string]bool // Resolved paths make the common case constant time.
	infos  []os.FileInfo   // Physical identities also catch hard-link aliases.
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
	for _, input := range r.inputs {
		if isStdin(input) {
			continue
		}
		inPath, err := resolvedPath(input)
		if err != nil {
			return nil, err
		}
		if inPath == resolved {
			return nil, fmt.Errorf("input and output files cannot be the same: %s", path)
		}
		if info != nil {
			inInfo, err := os.Stat(input)
			if err != nil {
				return nil, err
			}
			if os.SameFile(info, inInfo) {
				return nil, fmt.Errorf("input and output files cannot be the same: %s", path)
			}
		}
	}
	if info != nil {
		for _, previous := range r.infos {
			if os.SameFile(info, previous) {
				return nil, fmt.Errorf("output files cannot be the same: %s", path)
			}
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
	r.infos = append(r.infos, info)
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
