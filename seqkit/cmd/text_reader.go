package cmd

import (
	"io"
	"iter"
	"strings"

	"github.com/shenwei356/xopen"
)

// readTextLines streams arbitrarily long auxiliary-file lines and reports read
// failures. A partial line from a failed read is never processed.
func readTextLines(file string) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		reader, err := xopen.Ropen(file)
		if err == xopen.ErrNoContent {
			return
		}
		if err != nil {
			yield("", err)
			return
		}
		defer reader.Close()
		for {
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				yield("", err)
				return
			}
			if len(line) > 0 && !yield(strings.TrimRight(line, "\r\n"), nil) {
				return
			}
			if err == io.EOF {
				return
			}
		}
	}
}
