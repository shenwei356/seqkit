package cmd

import (
	"regexp"
	"slices"
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

// TestReverseInt preserves caller data and returns an independent reversed slice.
func TestReverseInt(t *testing.T) {
	for _, input := range [][]int{nil, {}, {4}, {1, -2, 3, -4}} {
		before := slices.Clone(input)
		got := ReverseInt(input)
		want := slices.Clone(before)
		slices.Reverse(want)
		if !slices.Equal(got, want) || !slices.Equal(input, before) {
			t.Errorf("reverse %v: got %v, want %v; input became %v", before, got, want, input)
		}
		if len(got) > 0 {
			after := slices.Clone(input)
			got[0]++
			if !slices.Equal(input, after) {
				t.Error("result aliases input")
			}
		}
	}
}

// TestMaxInts covers negative-only inputs and retains zero for an empty slice.
func TestMaxInts(t *testing.T) {
	for _, tc := range []struct {
		input []int
		want  int
	}{
		{nil, 0}, {[]int{-5}, -5}, {[]int{-7, -2, -9}, -2},
		{[]int{-3, 0, -1}, 0}, {[]int{1, 4, 2}, 4},
	} {
		if got := MaxInts(tc.input); got != tc.want {
			t.Errorf("max %v = %d, want %d", tc.input, got, tc.want)
		}
	}
}
