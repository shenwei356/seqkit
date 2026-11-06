package cmd

import "testing"

// TestReplaceMixedFileFormats preserves each input file's format.
func TestReplaceMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "replace", "-p", "^", "-r", "x")
}

// TestReplaceEditFilters shares the sequence-editing filter contract.
func TestReplaceEditFilters(t *testing.T) {
	assertEditFilterContracts(t, []string{"replace", "-s", "-p", ".", "-r", "X"})
}
