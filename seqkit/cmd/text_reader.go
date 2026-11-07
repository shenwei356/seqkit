package cmd

import (
	"bufio"
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
		scanner := bufio.NewScanner(reader)
		// Grow for long lines without imposing Scanner's default 64 KiB limit.
		scanner.Buffer(make([]byte, 64<<10), int(^uint(0)>>1))
		for scanner.Scan() {
			// Check errors before parsing a trailing token from a failed read.
			if err := scanner.Err(); err != nil {
				yield("", err)
				return
			}
			if !yield(strings.TrimRight(scanner.Text(), "\r\n"), nil) {
				return
			}
		}
		if err := scanner.Err(); err != nil {
			yield("", err)
		}
	}
}
