package cmd

import (
	"os"
	"strings"
	"testing"
)

// TestSubseqQuietIndexUpdate ensures quiet mode does not suppress requested work.
func TestSubseqQuietIndexUpdate(t *testing.T) {
	dir := t.TempDir()
	file := auditInput(t, dir, "input.fa", ">r1\nAAAACCCC\n")
	_, stderr, err := runCLI(t, "", "subseq", "--chr", "r1", "-r", "1:-1", file)
	if err != nil {
		t.Fatalf("create index: %v %s", err, stderr)
	}
	if err := os.WriteFile(file, []byte(">r1\nTTTTGGGGAAAA\n"), 0600); err != nil {
		t.Fatal(err)
	}
	out, stderr, err := runCLI(t, "", "subseq", "--chr", "r1", "-r", "1:-1", "-U", "--quiet", file)
	index, readErr := os.ReadFile(file + ".seqkit.fai")
	if err != nil || stderr != "" || !strings.HasSuffix(out, "\nTTTTGGGGAAAA\n") || readErr != nil || !strings.HasPrefix(string(index), "r1\t12\t") {
		t.Fatalf("index not updated: %v %s read=%v", err, stderr, readErr)
	}
}
