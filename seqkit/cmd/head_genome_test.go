package cmd

import "testing"

// TestHeadGenomeMixedFileFormats preserves each input file's format.
func TestHeadGenomeMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "head-genome")
}
