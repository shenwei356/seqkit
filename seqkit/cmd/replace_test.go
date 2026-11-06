package cmd

import "testing"

// TestReplaceMixedFileFormats preserves each input file's format.
func TestReplaceMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "replace", "-p", "^", "-r", "x")
}
