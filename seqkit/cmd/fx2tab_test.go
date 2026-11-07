package cmd

import (
	"strings"
	"testing"
)

// TestFx2tabRepeatedBases counts each requested base once without changing column labels.
func TestFx2tabRepeatedBases(t *testing.T) {
	for _, tt := range []struct {
		query, count, content string
		caseSensitive         bool
	}{
		{"Aa", "2", "100.00", false},
		{"AAA", "2", "100.00", false},
		{"Aa", "2", "100.00", true},
		{"AA", "1", "50.00", true},
		{"aa", "1", "50.00", true},
	} {
		args := []string{"fx2tab", "-n", "-i", "-H", "-B", tt.query, "-C", tt.query}
		if tt.caseSensitive {
			args = append(args, "-I")
		}
		out, stderr, err := runCLI(t, ">s\nAa\n", args...)
		want := "#id\t" + tt.query + "\t" + tt.query + "\ns\t" + tt.count + "\t" + tt.content + "\n"
		if err != nil || out != want {
			t.Fatalf("query=%q case-sensitive=%v: got %q, want %q; error: %v; stderr: %s", tt.query, tt.caseSensitive, out, want, err, stderr)
		}
	}
}

// TestFx2tabAlphabetBytes reports all distinct bytes, even around non-ASCII input.
func TestFx2tabAlphabetBytes(t *testing.T) {
	for _, input := range []string{"\x80ACGT", "AC\x80GT", "ACGT\x80", "TGCAACGT"} {
		want := "s\tACGT"
		if strings.Contains(input, "\x80") {
			want += "\x80"
		}
		out, stderr, err := runCLI(t, ">s\n"+input+"\n", "fx2tab", "-t", "unlimit", "-n", "-i", "-a")
		if err != nil || out != want+"\n" {
			t.Fatalf("alphabet of %q = %q, want %q; error: %v; stderr: %s", input, out, want, err, stderr)
		}
	}
}

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

// TestFx2tabGCStatistics preserves mixed-case DNA/RNA counts and zero denominators.
func TestFx2tabGCStatistics(t *testing.T) {
	input := ">r\nGggCAaTtUuN\n>empty\n>unknown\nNN\n"
	for _, tc := range []struct {
		flags []string
		want  string
	}{
		{[]string{"-G"}, "r\t50.00\nempty\tNaN\nunknown\tNaN\n"},
		{[]string{"-g"}, "r\t40.00\nempty\tNaN\nunknown\tNaN\n"},
		{[]string{"-g", "-G"}, "r\t40.00\t50.00\nempty\tNaN\tNaN\nunknown\tNaN\tNaN\n"},
	} {
		out, stderr, err := runCLI(t, input, append([]string{"fx2tab", "-i", "-n"}, tc.flags...)...)
		if err != nil || out != tc.want {
			t.Fatalf("GC statistics changed: %q %v %s", out, err, stderr)
		}
	}
}
