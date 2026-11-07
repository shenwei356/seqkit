package cmd

import (
	"os"
	"strings"
	"testing"
)

// TestHeadMixedFileFormats preserves each input file's format.
func TestHeadMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "head", "-n", "100")
}

// TestHeadInvalidLengths preserves existing output when a length is invalid.
func TestHeadInvalidLengths(t *testing.T) {
	for _, length := range []string{"", "0", "-1", "0.5", "0.0001K"} {
		t.Run(length, func(t *testing.T) {
			output := auditInput(t, t.TempDir(), "output.fa", "sentinel\n")
			_, stderr, err := runCLI(t, ">s\nACGT\n", "head", "-l", length, "-o", output)
			if err == nil || !strings.Contains(stderr, "invalid sequence length") {
				t.Fatalf("invalid length accepted: %v %s", err, stderr)
			}
			data, err := os.ReadFile(output)
			if err != nil || string(data) != "sentinel\n" {
				t.Fatalf("output changed: %q %v", data, err)
			}
		})
	}
}

// TestHeadLengthOverridesNumber keeps the documented length-mode precedence.
func TestHeadLengthOverridesNumber(t *testing.T) {
	out, stderr, err := runCLI(t, ">a\nAC\n>b\nGT\n>c\nAA\n", "head", "-n", "1", "-l", "0.004K")
	if err != nil || out != ">a\nAC\n>b\nGT\n" {
		t.Fatalf("length selection: %q %v %s", out, err, stderr)
	}
}
