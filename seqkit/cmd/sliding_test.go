package cmd

import (
	"strconv"
	"strings"
	"testing"
)

// TestSlidingMixedFileFormats preserves each input file's format.
func TestSlidingMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "sliding", "-s", "4", "-W", "4")
}

// TestSlidingHugeWindow clips greedy windows before integer addition can overflow.
func TestSlidingHugeWindow(t *testing.T) {
	out, stderr, err := runCLI(t, ">s\nACGT\n", "sliding", "-s", "1", "-W", strconv.Itoa(int(^uint(0)>>1)), "-g")
	if err != nil || strings.Count(out, ">") != 4 || !strings.HasSuffix(out, "\nT\n") {
		t.Fatalf("greedy oversized window failed: %v %s", err, stderr)
	}
}
