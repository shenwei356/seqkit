package cmd

import (
	"strings"
	"testing"
)

// TestSumEmptyFiles retains every digest metadata field without any input records.
func TestSumEmptyFiles(t *testing.T) {
	empty := auditInput(t, t.TempDir(), "empty.fa", "")
	for _, tt := range []struct {
		name, tag string
		args      []string
	}{
		{"auto", "NLS_k0", nil},
		{"dna", "DLS_k0", []string{"-t", "dna"}},
		{"rna", "RLS_k0", []string{"-t", "rna"}},
		{"protein", "PLS_k0", []string{"-t", "protein"}},
		{"unlimit", "NLS_k0", []string{"-t", "unlimit"}},
		{"rna2dna", "DLS_k0", []string{"-t", "rna", "--rna2dna"}},
		{"circular", "NCD_k31", []string{"-c"}},
		{"circular dna", "DCD_k31", []string{"-c", "-t", "dna"}},
		{"circular rna2dna", "DCS_k31", []string{"-c", "-s", "-t", "rna", "--rna2dna"}},
	} {
		for _, input := range []string{"-", empty} {
			t.Run(tt.name+"/"+input, func(t *testing.T) {
				args := append([]string{"sum", "--quiet", "-k", "31"}, tt.args...)
				out, stderr, err := runCLI(t, "", append(args, input)...)
				// The metadata changes; the digest of zero records remains stable.
				want := "seqkit.v0.1_" + tt.tag + "_56fc8fa222734a7d4b9ddedcca604bbf"
				fields := strings.Fields(out)
				if err != nil || len(fields) == 0 || fields[0] != want {
					t.Fatalf("empty digest = %q, want %q; error: %v; stderr: %s", out, want, err, stderr)
				}
			})
		}
	}
}

// TestSumErrorFileNames keeps percent signs literal in circular-input diagnostics.
func TestSumErrorFileNames(t *testing.T) {
	for _, tc := range []struct{ bases, k, message string }{
		{"ACGT", "31", "is too big"},
		{"MELK", "1", "does not support protein"},
	} {
		input := auditInput(t, t.TempDir(), "input%name.fa", ">r\n"+tc.bases+"\n")
		out, stderr, err := runCLI(t, "", "sum", "--quiet", "-c", "-k", tc.k, input)
		if err == nil || out != "" || !strings.Contains(stderr, tc.message) || !strings.Contains(stderr, input) || strings.Contains(stderr, "%!") {
			t.Fatalf("circular-input diagnostic lost the file name: %v %s", err, stderr)
		}
	}
}

// TestSumEmptyRecordOrder keeps the digest and type independent of empty records'
// positions, including RNA normalization and optional ID hashing.
func TestSumEmptyRecordOrder(t *testing.T) {
	for _, tc := range []struct {
		name, bases, tag string
		flags            []string
	}{
		{"DNA", "ACGT", "DLS", nil},
		{"RNA", "ACGU", "RLS", nil},
		{"protein", "MELK", "PLS", nil},
		{"RNA to DNA", "ACGU", "DLS", []string{"--rna2dna"}},
		{"empty", "", "NLS", nil},
		{"explicit empty DNA", "", "DLS", []string{"-t", "dna"}},
		{"unrestricted", "ACGT", "NLS", []string{"-t", "unlimit"}},
	} {
		for _, fastq := range []bool{false, true} {
			for _, includeID := range []bool{false, true} {
				args := append([]string{"sum", "--quiet"}, tc.flags...)
				if includeID {
					args = append(args, "--include-id")
				}
				empty := ">empty\n\n"
				record := ">r\n" + tc.bases + "\n"
				if fastq {
					empty = "@empty\n\n+\n\n"
					record = "@r\n" + tc.bases + "\n+\n" + strings.Repeat("I", len(tc.bases)) + "\n"
				}
				want, stderr, err := runCLI(t, record+empty+empty, args...)
				if err != nil || !strings.Contains(want, "_"+tc.tag) {
					t.Fatalf("%s: reference digest: %v %s", tc.name, err, stderr)
				}
				for _, input := range []string{empty + record + empty, empty + empty + record} {
					got, stderr, err := runCLI(t, input, args...)
					if err != nil || got != want {
						t.Errorf("%s includeID=%v: reordered empty records changed digest: %q != %q; %v %s", tc.name, includeID, got, want, err, stderr)
					}
				}
			}
		}
	}
}
