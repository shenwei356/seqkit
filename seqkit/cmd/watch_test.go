package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWatchEmptyStatistics skips images but retains pass-through records.
func TestWatchEmptyStatistics(t *testing.T) {
	for _, input := range []string{"", "@a\nAAAA\n+\nIIII\n"} {
		dir := t.TempDir()
		image := filepath.Join(dir, "hist.png")
		out, stderr, err := runCLI(t, input, "watch", "-f", "GCSkew", "-x", "-O", image)
		if err != nil || out != input || !strings.Contains(stderr, "no valid values") || strings.Contains(stderr, "\x1b[2J") {
			t.Fatalf("empty statistics: %q %v %s", out, err, stderr)
		}
		if _, err := os.Stat(image); !os.IsNotExist(err) {
			t.Fatal("image created without values")
		}
	}
}

// TestWatchImageFailures protects old images and reports errors without panic.
func TestWatchImageFailures(t *testing.T) {
	dir := t.TempDir()
	old := auditInput(t, dir, "hist.xyz", "keep")
	for _, image := range []string{old, filepath.Join(dir, "no-extension"), filepath.Join(dir, "missing", "hist.png")} {
		_, stderr, err := runCLI(t, ">s\nACGT\n", "watch", "-O", image)
		if err == nil || !strings.Contains(stderr, "[ERRO]") || strings.Contains(stderr, "panic:") {
			t.Fatalf("image failure: %v %s", err, stderr)
		}
	}
	data, _ := os.ReadFile(old)
	if string(data) != "keep" {
		t.Fatal("existing image truncated")
	}
	paths, _ := filepath.Glob(filepath.Join(dir, ".seqkit-image-*"))
	if len(paths) > 0 {
		t.Fatalf("partial image remains: %v", paths)
	}
	image := filepath.Join(dir, "valid.PNG")
	_, stderr, err := runCLI(t, ">s\nACGT\n", "watch", "-O", image, "-Q")
	info, statErr := os.Stat(image)
	if err != nil || statErr != nil || info.Size() == 0 {
		t.Fatalf("valid image failed: %v %v %s", err, statErr, stderr)
	}
}

// TestWatchQualityAndFieldErrors retains the common CLI error convention.
func TestWatchQualityAndFieldErrors(t *testing.T) {
	_, stderr, err := runCLI(t, "@a\nACGT\n+\n#III\n", "watch", "-f", "MeanQual", "-b", "40", "-y")
	if err == nil || !strings.Contains(stderr, "a:") || !strings.Contains(stderr, "40") || !strings.Contains(stderr, "[ERRO]") {
		t.Fatalf("invalid quality accepted: %v %s", err, stderr)
	}
	_, stderr, err = runCLI(t, "", "watch", "-f", "foo")
	if err == nil || !strings.Contains(stderr, "[ERRO] Invalid field: foo") {
		t.Fatalf("invalid field: %v %s", err, stderr)
	}
}
