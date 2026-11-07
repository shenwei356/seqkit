package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConvertDryRunOutput leaves both existing and new output paths untouched.
func TestConvertDryRunOutput(t *testing.T) {
	for _, from := range []string{"", "Sanger"} {
		for _, name := range []string{"out.fq", "out.fq.gz", "missing/out.fq"} {
			t.Run(from+"/"+name, func(t *testing.T) {
				dir := t.TempDir()
				output := filepath.Join(dir, name)
				args := []string{"convert", "--dry-run", "-o", output}
				if from != "" {
					args = append(args, "--from", from)
				}
				for _, existing := range []bool{false, true} {
					if existing {
						if err := os.MkdirAll(filepath.Dir(output), 0700); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(output, []byte("keep\n"), 0600); err != nil {
							t.Fatal(err)
						}
					}
					_, stderr, err := runCLI(t, "@s\nAC\n+\n!!\n", args...)
					if err != nil || !strings.Contains(stderr, "converting") {
						t.Fatalf("dry run failed: %v %s", err, stderr)
					}
					data, err := os.ReadFile(output)
					if existing && (err != nil || string(data) != "keep\n") || !existing && !os.IsNotExist(err) {
						t.Fatalf("dry run changed output (existing=%v): %v", existing, err)
					}
				}
			})
		}
	}
}

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
