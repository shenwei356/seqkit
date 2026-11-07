package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenAutocompleteInvalidShellLeavesNoDirectory rejects invalid requests before writes.
func TestGenAutocompleteInvalidShellLeavesNoDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "completion")
	_, stderr, err := runCLI(t, "", "genautocomplete", "--shell", "invalid", "--file", filepath.Join(dir, "seqkit"))
	if err == nil || !strings.Contains(stderr, "unsupported shell") {
		t.Fatalf("invalid shell was not rejected: %v %s", err, stderr)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("invalid request created an output directory: %v", err)
	}
}
