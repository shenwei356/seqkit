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

// TestSubseqDescriptorLimit closes each indexed input before processing the next.
func TestSubseqDescriptorLimit(t *testing.T) {
	dir := t.TempDir()
	files := make([]string, 80)
	for i := range files {
		files[i] = auditInput(t, dir, fmt.Sprintf("%03d.fa", i), ">r\nACGT\n>s\nTGCA\n")
	}
	bed := auditInput(t, dir, "regions.bed", "r\t0\t2\n")
	gtf := auditInput(t, dir, "regions.gtf", "r\ttest\tCDS\t1\t2\t.\t+\t.\tgene_id \"r\";\n")
	for _, mode := range [][]string{{"-r", "1:2", "--chr", "r"}, {"--bed", bed}, {"--gtf", gtf}} {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		args := []string{"-c", `ulimit -n 64; exec "$@"`, "subseq-test", os.Args[0], "-test.run=^TestCLIProcess$", "--", "subseq", "--quiet"}
		args = append(args, mode...)
		args = append(args, files...)
		command := exec.CommandContext(ctx, "sh", args...)
		command.Env = append(os.Environ(), "SEQKIT_CLI_TEST_HELPER=1")
		var out, stderr bytes.Buffer
		command.Stdout, command.Stderr = &out, &stderr
		err := command.Run()
		cancel()
		if err != nil || strings.Count(out.String(), ">r") != len(files) {
			t.Fatalf("descriptor budget: %v %s", err, stderr.String())
		}
	}
}
