package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestLocatePerFileStrands keeps strand detection local in every search mode.
func TestLocatePerFileStrands(t *testing.T) {
	dir := t.TempDir()
	dna := auditInput(t, dir, "dna.fa", ">dna\nACGTTGCA\n")
	protein := auditInput(t, dir, "protein.fa", ">protein\nZZTTGCAACGZZ\n")
	for _, mode := range [][]string{nil, {"-F"}, {"-m", "1"}, {"-r"}, {"-d"}} {
		for _, files := range [][]string{{dna, protein}, {protein, dna}} {
			for _, positive := range []bool{false, true} {
				args := append([]string{"locate", "-c", "-p", "TTGCAACG"}, mode...)
				if positive {
					args = append(args, "-P")
				}
				out, stderr, err := runCLI(t, "", append(args, files...)...)
				minus := strings.Contains(out, "dna\tTTGCAACG\tTTGCAACG\t-\t2\t9\t")
				if err != nil || minus == positive || strings.Contains(out, "protein\tTTGCAACG\tTTGCAACG\t-\t") {
					t.Fatalf("mode=%v positive=%v: strand mismatch: %q %v %s", mode, positive, out, err, stderr)
				}
			}
		}
	}
}

// TestLocateDuplicatePatternIDs checks warnings without losing repeated matches.
func TestLocateDuplicatePatternIDs(t *testing.T) {
	for _, mode := range []struct {
		name string
		args []string
	}{
		{"exact", nil},
		{"fmi", []string{"-F"}},
		{"mismatch", []string{"-m", "1"}},
		{"regexp", []string{"-r"}},
		{"degenerate", []string{"-d"}},
	} {
		t.Run(mode.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "patterns.fa")
			if err := os.WriteFile(file, []byte(">dup first\nAC\n>dup second\nGT\n>dup third\nAC\n"), 0600); err != nil {
				t.Fatal(err)
			}
			for _, quiet := range []bool{false, true} {
				args := append([]string{"locate", "-P", "-f", file}, mode.args...)
				if quiet {
					args = append(args, "--quiet")
				}
				out, stderr, err := runCLI(t, ">s\nACGT\n", args...)
				if err != nil || strings.Count(out, "\tdup\tAC\t") != 2 || strings.Count(out, "\tdup\tGT\t") != 1 {
					t.Fatalf("quiet=%v: patterns lost: out=%q, err=%v, stderr=%s", quiet, out, err, stderr)
				}
				warnings := strings.Count(stderr, "duplicate pattern ID")
				if quiet {
					if stderr != "" {
						t.Fatalf("--quiet emitted diagnostics: %s", stderr)
					}
				} else if warnings != 2 || !strings.Contains(stderr, `duplicate pattern ID "dup" in `+file) || !strings.Contains(stderr, "all patterns with this ID will be searched") {
					t.Fatalf("missing duplicate-ID diagnostics: %s", stderr)
				}
			}
		})
	}
}

// TestLocateUniquePatternIDs distinguishes repeated sequences from repeated IDs.
func TestLocateUniquePatternIDs(t *testing.T) {
	file := filepath.Join(t.TempDir(), "patterns.fa")
	if err := os.WriteFile(file, []byte(">one\nAC\n>two\nAC\n"), 0600); err != nil {
		t.Fatal(err)
	}
	out, stderr, err := runCLI(t, ">s\nAC\n", "locate", "-P", "-f", file)
	if err != nil || stderr != "" || strings.Count(out, "\tAC\t+\t1\t2\tAC\n") != 2 {
		t.Fatalf("unique IDs: out=%q, err=%v, stderr=%s", out, err, stderr)
	}
}

// TestLocateRegexpBoundaries preserves assertions while retaining overlapping matches.
func TestLocateRegexpBoundaries(t *testing.T) {
	for _, test := range []struct {
		pattern string
		starts  []string
	}{
		{`^A`, []string{"1"}}, {`\AA`, []string{"1"}}, {`\bA`, []string{"1"}},
		{`\BA`, []string{"2", "3", "4", "5"}}, {`AA`, []string{"1", "2", "3", "4"}},
		{`^AA|AA$`, []string{"1", "4"}}, {`A$`, []string{"5"}},
	} {
		for _, greedy := range []bool{false, true} {
			args := []string{"locate", "-P", "-r", "-p", test.pattern}
			if greedy {
				args = append(args, "-G")
			}
			out, stderr, err := runCLI(t, ">r\nAAAAA\n", args...)
			if err != nil {
				t.Fatalf("pattern=%s: %v %s", test.pattern, err, stderr)
			}
			var starts []string
			for _, line := range strings.Split(strings.TrimSpace(out), "\n")[1:] {
				starts = append(starts, strings.Split(line, "\t")[4])
			}
			want := test.starts
			if greedy && test.pattern == "AA" {
				want = []string{"1", "3"}
			}
			if !reflect.DeepEqual(starts, want) {
				t.Fatalf("pattern=%s -G=%v: %v want %v", test.pattern, greedy, starts, want)
			}
		}
	}
	out, stderr, err := runCLI(t, ">r\nTTTTT\n", "locate", "-r", "-p", "^A")
	if err != nil || strings.Count(out, "\n") != 2 || !strings.Contains(out, "\t-\t5\t5\tA\n") {
		t.Fatalf("reverse anchor: %v %s", err, stderr)
	}
}

// TestLocateRegexpContext covers multiline, alternatives, captures and Unicode context.
func TestLocateRegexpContext(t *testing.T) {
	for _, test := range []struct {
		pattern, text string
		offset        int
		want          []int
	}{
		{`(?m)^A`, "AA\nAA", 1, []int{2, 3}},
		{`^A|A`, "AAA", 1, []int{0, 1}},
		{`(\bA)|(AA)`, "AAAA", 1, []int{0, 2}},
		{`\BA`, "AAA", 1, []int{0, 1}},
		{`\bA`, "éAA", 2, []int{0, 1}},
		{`\bA`, "AAA", 1, nil},
		{`\AA`, "AAA", 1, nil},
	} {
		re, err := newLocateRegexp(test.pattern)
		if err != nil {
			t.Fatal(err)
		}
		if got := re.findFrom([]byte(test.text), test.offset); !reflect.DeepEqual(got, test.want) {
			t.Fatalf("%s at %d: %v want %v", test.pattern, test.offset, got, test.want)
		}
	}
}
