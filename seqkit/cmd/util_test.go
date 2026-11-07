package cmd

import (
	"regexp"
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

// TestPrettyPrintTsvLiteralPercent keeps text unchanged when ANSI colors are enabled.
func TestPrettyPrintTsvLiteralPercent(t *testing.T) {
	cols := []string{"File%", "Value"}
	fields := [][]string{{"sample%s.bam", "100%%.bam", "%[1]s.bam"}, {"1", "2", "3"}}
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	for _, width := range []int{0, -1} {
		plain, _ := PrettyPrintTsv(cols, fields, width, false)
		colored, _ := PrettyPrintTsv(cols, fields, width, true)
		if got := ansi.ReplaceAllString(colored, ""); got != plain {
			t.Errorf("width=%d: colored text=%q, want %q", width, got, plain)
		}
	}
}
