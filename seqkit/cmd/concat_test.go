package cmd

import (
	"strings"
	"testing"
)

// TestConcatFullSingleNonemptyInput preserves duplicate records and qualities.
func TestConcatFullSingleNonemptyInput(t *testing.T) {
	for _, input := range []string{">r\nAC\n>r\nGT\n", "@r\nAC\n+\nIJ\n@r\nGT\n+\nKL\n"} {
		dir := t.TempDir()
		file := auditInput(t, dir, "records.fx", input)
		empty := auditInput(t, dir, "empty.fx", "")
		for _, files := range [][]string{{empty, file}, {file, empty}, {empty, file, empty}} {
			out, stderr, err := runCLI(t, "", append([]string{"concat", "-f", "-F", "N"}, files...)...)
			if err != nil || out != input {
				t.Fatalf("single nonempty input changed: %q %v %s", out, err, stderr)
			}
		}
	}
}

// TestConcatCustomIDDescription excludes text already captured as part of the ID.
func TestConcatCustomIDDescription(t *testing.T) {
	description := strings.Repeat("x", 3) + " " + strings.Repeat("y", 2)
	header := "r " + description
	file := auditInput(t, t.TempDir(), "input.fa", ">"+header+"\nAC\n")
	for _, test := range []struct{ pattern, header string }{
		{`^(.+)$`, header},
		{`^(\S+)`, header + "|" + description},
		{`^(\S+ \S+)`, header + "|" + strings.Repeat("y", 2)},
		{`^(\S+) \S+`, header + "|" + description},
		{`^(missing)`, header},
	} {
		out, stderr, err := runCLI(t, "", "concat", "--id-regexp", test.pattern, file, file)
		if err != nil || out != ">"+test.header+"\nACAC\n" {
			t.Fatalf("custom ID/description mismatch for %q: %v %s", test.pattern, err, stderr)
		}
	}
}
