package cmd

import (
	"strings"
	"testing"
)

// TestRevCompDNA preserves all IUPAC DNA codes on both strands.
func TestRevCompDNA(t *testing.T) {
	const input = "ACGTRYSWKMBDHVN-"
	const want = "-NBDHVKMWSRYACGT"
	for _, s := range []string{input, strings.ToLower(input)} {
		if got := RevCompDNA(s); got != want {
			t.Errorf("reverse complement = %q, want %q", got, want)
		}
	}
	if got := RevCompDNA(RevCompDNA(input)); got != input {
		t.Errorf("double reverse complement = %q, want %q", got, input)
	}
}
