package cmd

import (
	"strconv"
	"strings"
	"testing"
)

// TestTranslateReverseComplement checks negative frames against explicit
// reverse complements, including RNA and ambiguous nucleotide pairs.
func TestTranslateReverseComplement(t *testing.T) {
	for _, tt := range []struct {
		name, forward, reverse, peptide string
	}{
		{"DNA", "CAT", "ATG", "M"},
		{"RNA", "CAU", "AUG", "M"},
		{"ambiguous DNA", "YTT", "AAR", "K"},
		{"ambiguous RNA", "YUU", "AAR", "K"},
		{"lowercase RNA", "yuu", "aar", "K"},
		{"ambiguous pairs", "ACGTRYSWKMBDHVN", "NBDHVKMWSRYACGT", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for frame := 1; frame <= 3; frame++ {
				want, stderr, err := runCLI(t, ">s\n"+tt.reverse+"\n", "translate", "-f", strconv.Itoa(frame))
				if err != nil {
					t.Fatalf("forward translation: %v %s", err, stderr)
				}
				if frame == 1 && tt.peptide != "" && want != ">s\n"+tt.peptide+"\n" {
					t.Fatalf("forward translation = %q, want %q", want, tt.peptide)
				}
				got, stderr, err := runCLI(t, ">s\n"+tt.forward+"\n", "translate", "-f", strconv.Itoa(-frame))
				if err != nil || got != want {
					t.Errorf("frame %d = %q, want %q: %v %s", -frame, got, want, err, stderr)
				}
			}
		})
	}
}

// TestTranslateSixFrames keeps the reverse-complement cache local to each record
// and preserves nucleotide coordinates when emitting translated subsequences.
func TestTranslateSixFrames(t *testing.T) {
	input := ">a\nCAU\n>b\nYUU\n"
	out, stderr, err := runCLI(t, input, "translate", "-f", "6", "-F")
	if err != nil || !strings.Contains(out, ">a_frame=-1 \nM\n") || !strings.Contains(out, ">b_frame=-1 \nK\n") {
		t.Fatalf("six-frame translation: %q %v %s", out, err, stderr)
	}
	out, stderr, err = runCLI(t, ">s\nUUAYUUCAU\n", "translate", "-f", "-1", "-s")
	if err != nil || out != ">s_frame=-1_begin=4_end=9 \nMK\n" {
		t.Fatalf("reverse subsequence coordinates: %q %v %s", out, err, stderr)
	}
}
