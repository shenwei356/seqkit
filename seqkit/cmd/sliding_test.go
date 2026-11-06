package cmd

import "testing"

// TestSlidingMixedFileFormats preserves each input file's format.
func TestSlidingMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "sliding", "-s", "4", "-W", "4")
}
