package cmd

import "testing"

// TestAmpliconMixedFileFormats preserves each input file's format.
func TestAmpliconMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "amplicon", "-P", "-F", "ACGT", "-u")
}

// TestAmpliconParallelMixedFileFormats preserves each input file's format.
func TestAmpliconParallelMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "amplicon", "-P", "-F", "ACGT", "-u", "-m", "1", "-j", "4")
}
