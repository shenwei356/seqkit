package cmd

import "testing"

// TestHeadMixedFileFormats preserves each input file's format.
func TestHeadMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "head", "-n", "100")
}
