package cmd

import (
	"os"
	"strings"
	"testing"
)

// TestSeqQualityThresholds applies zero limits and rejects nonfinite bounds safely.
func TestSeqQualityThresholds(t *testing.T) {
	input := "@s\nAC\n+\n55\n"
	out, stderr, err := runCLI(t, input, "seq", "-R", "0")
	if err != nil || out != "" {
		t.Fatalf("zero maximum was ignored: %v %s", err, stderr)
	}
	for _, flag := range []string{"-Q", "-R"} {
		for _, value := range []string{"NaN", "+Inf", "-Inf"} {
			output := auditInput(t, t.TempDir(), "out.fq", "keep\n")
			_, stderr, err = runCLI(t, input, "seq", flag, value, "-o", output)
			after, _ := os.ReadFile(output)
			if err == nil || !strings.Contains(stderr, "finite") || string(after) != "keep\n" {
				t.Fatalf("invalid quality bound changed output: %v %s", err, stderr)
			}
		}
	}
}

// assertEditFilterContracts checks literal/regexp strand policy and empty patterns.
func assertEditFilterContracts(t *testing.T, command []string) {
	t.Helper()
	for _, flags := range [][]string{{"--f-pattern", ","}, {"--f-pattern", ",s"}, {"--f-pattern", ",s", "--quiet"}} {
		_, stderr, err := runCLI(t, ">s\nACGT\n", append(append([]string(nil), command...), flags...)...)
		if err != nil {
			t.Fatalf("empty filter pattern crashed: %v %s", err, stderr)
		}
	}
	for _, tc := range []struct {
		input, pattern string
		edit           bool
	}{
		{">s\nTTTC\n", "GAAA", true},
		{">s\nGUUU\n", "AAAC", true},
		{">s\nEFILPQ\n", "QPLIFE", false},
	} {
		for _, regex := range []bool{false, true} {
			for _, positive := range []bool{false, true} {
				args := append([]string(nil), command...)
				args = append(args, "--f-by-seq", "--f-pattern", tc.pattern)
				if regex {
					args = append(args, "--f-use-regexp")
				}
				if positive {
					args = append(args, "--f-only-positive-strand")
				}
				out, stderr, err := runCLI(t, tc.input, args...)
				wantEdit := tc.edit && !positive
				if err != nil || (out != tc.input) != wantEdit {
					t.Fatalf("filter strand/regex mismatch: command=%s pattern=%s regex=%v positive=%v: %v %s", command[0], tc.pattern, regex, positive, err, stderr)
				}
			}
		}
	}
}

// TestSeqEditFilters covers filter behavior through the sequence-editing command.
func TestSeqEditFilters(t *testing.T) { assertEditFilterContracts(t, []string{"seq", "-l"}) }

// TestSeqDamagedAuxiliaryInput rejects incomplete auxiliary data.
func TestSeqDamagedAuxiliaryInput(t *testing.T) {
	assertDamagedTextInput(t, "r\n", "seq", "--f-pattern-file")
}

// TestSeqNucleotideConversion checks every selected record, not the first alphabet guess.
func TestSeqNucleotideConversion(t *testing.T) {
	for _, tc := range []struct{ flag, input, want string }{
		{"--rna2dna", "ACG", "ACG"},
		{"--rna2dna", "AUgu", "ATgt"},
		{"--dna2rna", "AUgu", "AUgu"},
		{"--dna2rna", "ATgt", "AUgu"},
	} {
		t.Run(tc.flag+tc.input, func(t *testing.T) {
			// The first record already has the target alphabet, but later records may not.
			first := "ACG"
			if tc.flag == "--dna2rna" {
				first = "AUG"
			}
			input := ">first\n" + first + "\n>second\n" + tc.input + "\n"
			want := ">first\n" + first + "\n>second\n" + tc.want + "\n"
			out, stderr, err := runCLI(t, input, "seq", tc.flag)
			if err != nil || out != want {
				t.Fatalf("conversion: %q %v %s", out, err, stderr)
			}
			out, stderr, err = runCLI(t, input, "seq", tc.flag, "--f-pattern", "first")
			if err != nil || out != input {
				t.Fatalf("unselected record changed: %q %v %s", out, err, stderr)
			}
			fq := "@first\n" + first + "\n+\nABC\n@second\n" + tc.input + "\n+\n" + strings.Repeat("I", len(tc.input)) + "\n"
			fqWant := strings.Replace(fq, "\n"+tc.input+"\n+", "\n"+tc.want+"\n+", 1)
			out, stderr, err = runCLI(t, fq, "seq", tc.flag)
			if err != nil || out != fqWant {
				t.Fatalf("FASTQ conversion: %q %v %s", out, err, stderr)
			}
		})
	}
}
