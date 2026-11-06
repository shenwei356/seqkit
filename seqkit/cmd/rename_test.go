package cmd

import "testing"

// TestRenameMixedFileFormats preserves each input file's format.
func TestRenameMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "rename")
}
