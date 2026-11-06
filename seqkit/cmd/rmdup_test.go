package cmd

import "testing"

// TestRmdupMixedFileFormats preserves each input file's format.
func TestRmdupMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "rmdup")
}
