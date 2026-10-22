//go:build !windows

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestScatScanDescriptorLimit reads all files under a small descriptor budget.
func TestScatScanDescriptorLimit(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 128; i++ {
		auditInput(t, dir, fmt.Sprintf("%d.fa", i), ">a\nAC\n>b\nGG\n>c\nTT\n>d\nCC\n")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", `ulimit -n 64; exec "$@"`, "scat-test", os.Args[0], "-test.run=^TestCLIProcess$", "--", "scat", "-f", "-i", "fasta", "-j", "2", dir)
	cmd.Env = append(os.Environ(), "SEQKIT_CLI_TEST_HELPER=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if err != nil || strings.Count(stdout.String(), ">") != 512 {
		t.Fatalf("descriptor budget lost records: %d %v %s", strings.Count(stdout.String(), ">"), err, stderr.String())
	}
}
