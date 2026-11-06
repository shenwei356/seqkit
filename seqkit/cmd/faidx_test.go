package cmd

import (
	"os"
	"testing"
)

// TestFaidxUpdateProtectsWorkingDirectory rejects an index path equal to cwd.
func TestFaidxUpdateProtectsWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	input := auditInput(t, dir, "input.fa", ">a\nACGT\n")
	index := input + ".fai"
	if err := os.Mkdir(index, 0700); err != nil {
		t.Fatal(err)
	}
	marker := auditInput(t, index, "keep", "keep\n")
	_, stderr, err := auditCLI(t, index, "", "faidx", "-U", "../input.fa")
	data, readErr := os.ReadFile(marker)
	if err == nil || readErr != nil || string(data) != "keep\n" {
		t.Fatalf("working directory was not protected: %v %s; marker=%v", err, stderr, readErr)
	}
}

// TestFaidxClippedStrands clips both orientations without turning empty ranges into hits.
func TestFaidxClippedStrands(t *testing.T) {
	file := auditInput(t, t.TempDir(), "input.fa", ">r\nACGTAACCGA\n>rna\nACGUAACCGA\n")
	for _, test := range []struct{ query, sequence string }{
		{"r:-12--1", "ACGTAACCGA"},
		{"r:-1--12", "TCGGTTACGT"},
		{"r:-12--3", "ACGTAACC"},
		{"r:-3--12", "GGTTACGT"},
		{"r:1-12", "ACGTAACCGA"},
		{"r:12-1", "TCGGTTACGT"},
		{"r:-20--12", ""},
		{"r:-12--20", ""},
		{"r:12-20", ""},
		{"r:20-12", ""},
		{"r:2--2", "CGTAACCG"},
		{"rna:-1--12", "UCGGUUACGU"},
	} {
		out, stderr, err := runCLI(t, "", "faidx", "--quiet", file, test.query)
		if err != nil || out != ">"+test.query+"\n"+test.sequence+"\n" {
			t.Fatalf("query %s: %q %v %s", test.query, out, err, stderr)
		}
	}
}
