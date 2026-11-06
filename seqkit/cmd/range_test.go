package cmd

import "testing"

// TestRangeMixedFileFormats preserves each input file's format.
func TestRangeMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "range", "-r", "1:-1")
}
