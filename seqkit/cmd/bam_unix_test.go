//go:build !windows

package cmd

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestBamIndexDescriptorLimit closes each BAI before processing the next input.
func TestBamIndexDescriptorLimit(t *testing.T) {
	dir := t.TempDir()
	file := bamFixture(t, dir, "in.bam", true)
	bamIndexFixture(t, file)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	args := []string{"-c", `ulimit -n 64; exec "$@"`, "bam-test", os.Args[0], "-test.run=^TestCLIProcess$", "--", "bam", "--idx-stat"}
	for i := 0; i < 80; i++ {
		args = append(args, file)
	}
	command := exec.CommandContext(ctx, "sh", args...)
	// A reader must release its descriptor without depending on GC finalizers.
	command.Env = append(os.Environ(), "SEQKIT_CLI_TEST_HELPER=1", "GOGC=off", "GOMEMLIMIT=off")
	out, err := command.CombinedOutput()
	if err != nil || strings.Count(string(out), file) != 80 {
		t.Fatalf("BAI descriptor budget: %v %s", err, out)
	}
}
