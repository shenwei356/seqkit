package cmd

import (
	"strings"
	"testing"
)

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
