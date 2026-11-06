package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestScatRetiredIdentity deterministically reuses an identity after its reader closes.
// Moving the old file outside the watched root retains its inode for the replacement.
func TestScatRetiredIdentity(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "input")
	file := auditInput(t, root, "first.fq", "@r1\nAC\n+\nII\n")
	stdout, err := os.Create(filepath.Join(dir, "stdout"))
	if err != nil {
		t.Fatal(err)
	}
	defer stdout.Close()
	stderr, err := os.Create(filepath.Join(dir, "stderr"))
	if err != nil {
		t.Fatal(err)
	}
	defer stderr.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCLIProcess$", "--", "scat", "-d", "0", "-D", "0ms", root)
	cmd.Env = append(os.Environ(), "SEQKIT_CLI_TEST_HELPER=1")
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	waitFor := func(label string, ready func() bool) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if ready() {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		logs, _ := os.ReadFile(stderr.Name())
		t.Fatalf("timed out waiting for %s: %s", label, logs)
	}
	want := ""
	for round := 1; round <= 3; round++ {
		want += fmt.Sprintf("@r%d\nAC\n+\nII\n", round)
		waitFor("new records", func() bool {
			data, _ := os.ReadFile(stdout.Name())
			return string(data) == want
		})
		if round == 3 {
			break
		}
		retired := filepath.Join(dir, "retired.fq")
		if err := os.Rename(file, retired); err != nil {
			t.Fatal(err)
		}
		waitFor("reader close", func() bool {
			fds, err := os.ReadDir(fmt.Sprintf("/proc/%d/fd", cmd.Process.Pid))
			if err != nil {
				t.Fatal(err)
			}
			for _, fd := range fds {
				path, _ := os.Readlink(fmt.Sprintf("/proc/%d/fd/%s", cmd.Process.Pid, fd.Name()))
				if strings.TrimSuffix(path, " (deleted)") == retired {
					return false
				}
			}
			return true
		})
		if err := os.WriteFile(retired, []byte(fmt.Sprintf("@r%d\nAC\n+\nII\n", round+1)), 0600); err != nil {
			t.Fatal(err)
		}
		file = filepath.Join(root, fmt.Sprintf("next%d.fq", round))
		if err := os.Rename(retired, file); err != nil {
			t.Fatal(err)
		}
	}
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		logs, _ := os.ReadFile(stderr.Name())
		t.Fatalf("watch exit: %v %s", err, logs)
	}
	data, err := os.ReadFile(stdout.Name())
	if err != nil || string(data) != want {
		t.Fatalf("lost or duplicated records: %q %v", data, err)
	}
}
