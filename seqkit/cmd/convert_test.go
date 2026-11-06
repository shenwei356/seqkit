package cmd

import (
	"os"
	"strings"
	"testing"
)

// TestConvertGuessFraction rejects invalid fractions before opening output.
func TestConvertGuessFraction(t *testing.T) {
	for _, value := range []string{"NaN", "+Inf", "-Inf", "-0.1", "1.1"} {
		output := auditInput(t, t.TempDir(), "out.fq", "keep\n")
		_, stderr, err := runCLI(t, "@s\nAC\n+\nBB\n", "convert", "-F", value, "-o", output)
		after, _ := os.ReadFile(output)
		if err == nil || !strings.Contains(stderr, "[0, 1]") || string(after) != "keep\n" {
			t.Fatalf("invalid fraction changed output: %v %s", err, stderr)
		}
	}
}
