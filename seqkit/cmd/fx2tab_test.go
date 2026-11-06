package cmd

import (
	"strings"
	"testing"
)

// TestFx2tabFileHeader keeps the last column named in every tabular output mode.
func TestFx2tabFileHeader(t *testing.T) {
	for _, mode := range [][]string{nil, {"-n"}, {"-Q"}, {"-l", "-g", "-s"}} {
		args := append([]string{"fx2tab", "-H", "-f", "-i", "--stdin-label", "input"}, mode...)
		out, stderr, err := runCLI(t, ">r1\nACGT\n", args...)
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		if err != nil || len(lines) != 2 {
			t.Fatalf("unexpected output: %q %v %s", out, err, stderr)
		}
		header, row := strings.Split(lines[0], "\t"), strings.Split(lines[1], "\t")
		if len(header) != len(row) || header[len(header)-1] != "file" || row[len(row)-1] != "input" {
			t.Fatalf("column mismatch: %q", out)
		}
	}
}
