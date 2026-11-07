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

// TestConvertGuessCompatibleScores keeps heuristic guesses compatible with every
// sampled quality, regardless of record order or empty records.
func TestConvertGuessCompatibleScores(t *testing.T) {
	for _, quality := range []string{"II", "BB", ""} {
		for _, reverse := range []bool{false, true} {
			first := "@a\n" + strings.Repeat("A", len(quality)) + "\n+\n" + quality + "\n"
			second := "@b\nAC\n+\n!!\n"
			if reverse {
				first, second = second, first
			}
			input := first + second
			for _, fraction := range []string{"0", "0.1", "1"} {
				out, stderr, err := runCLI(t, input, "convert", "-F", fraction)
				if err != nil || out != input || !strings.Contains(stderr, "guessed quality encoding: Sanger") {
					t.Fatalf("incompatible guess for %q (reverse=%v, fraction=%s): %v %s", quality, reverse, fraction, err, stderr)
				}
			}
		}
	}
	// Retain the Illumina 1.5 heuristic when every record is compatible.
	for _, fraction := range []string{"0.1", "1"} {
		out, stderr, err := runCLI(t, "@a\nAA\n+\nBB\n@b\nCC\n+\nCC\n", "convert", "-F", fraction)
		if err != nil || out != "@a\nAA\n+\n!!\n@b\nCC\n+\n$$\n" {
			t.Fatalf("compatible Illumina guess changed: %v %s", err, stderr)
		}
	}
}
