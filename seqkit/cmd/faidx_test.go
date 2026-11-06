package cmd

import "testing"

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
