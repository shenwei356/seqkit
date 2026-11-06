package cmd

import "testing"

// TestDuplicateMixedFileFormats preserves each input file's format.
func TestDuplicateMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "duplicate", "-n", "1")
}
