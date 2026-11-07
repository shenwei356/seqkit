package cmd

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestScatCompressedFileDiscovery covers every supported codec and --gz-only.
func TestScatCompressedFileDiscovery(t *testing.T) {
	for _, format := range []string{"fasta", "fastq"} {
		for _, extension := range []string{".gz", ".xz", ".zst", ".bz2", ".lz4"} {
			t.Run(format+extension, func(t *testing.T) {
				dir := t.TempDir()
				input, suffix := ">r\nACGT\n", ".fa"
				if format == "fastq" {
					input, suffix = "@r\nACGT\n+\n!I?J\n", ".fq"
				}
				file := filepath.Join(dir, "input"+suffix+extension)
				if err := os.WriteFile(file, compressedInput(t, extension, input), 0600); err != nil {
					t.Fatal(err)
				}
				for _, gzipOnly := range []bool{false, true} {
					args := []string{"scat", "-f", "-i", format, dir, "--quiet"}
					want := input
					if gzipOnly {
						args = append(args, "-g")
						if extension != ".gz" {
							want = ""
						}
					}
					out, stderr, err := runCLI(t, "", args...)
					if err != nil || out != want {
						t.Fatalf("gz-only=%v output=%q want=%q: %v %s", gzipOnly, out, want, err, stderr)
					}
				}
			})
		}
	}
}

// TestScatScanErrorsAndDirectoryAliases retains good records and reports bad links.
func TestScatScanErrorsAndDirectoryAliases(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	nested := filepath.Join(root, "nested")
	auditInput(t, nested, "good.fq", "@a\nAC\n+\nII\n")
	if err := os.Symlink(filepath.Join(dir, "missing"), filepath.Join(root, "bad.fq")); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	list := auditInput(t, dir, "dirs", alias+"\n"+nested+"\n")
	out, stderr, err := runCLI(t, "", "scat", "-f", "--quiet", root, "-X", list)
	if err == nil || out != "@a\nAC\n+\nII\n" || !strings.Contains(stderr, "bad.fq") || !strings.Contains(stderr, "[ERRO]") {
		t.Fatalf("scan error/dedup: %q %v %s", out, err, stderr)
	}
}

// TestScatLiveAppendRenameAndThrottle observes output before finalization.
func TestScatLiveAppendRenameAndThrottle(t *testing.T) {
	for _, compressed := range []bool{false, true} {
		t.Run(fmt.Sprint(compressed), func(t *testing.T) { testScatLiveAppend(t, compressed) })
	}
}

// testScatLiveAppend also exercises temporary EOF inside a compressed member.
func testScatLiveAppend(t *testing.T, compressed bool) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	encode := func(text string) string {
		if !compressed {
			return text
		}
		var data bytes.Buffer
		w := gzip.NewWriter(&data)
		if _, err := w.Write([]byte(text)); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		return data.String()
	}
	suffix := ".fq"
	delay := "150ms"
	if compressed {
		suffix = ".fq.gz"
		delay = "0ms"
	}
	file := auditInput(t, root, "n"+suffix, encode("@a\nAC\n+\nII\n"))
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
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCLIProcess$", "--", "scat", "-d", "0", "-D", delay, root, root)
	cmd.Env = append(os.Environ(), "SEQKIT_CLI_TEST_HELPER=1")
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	waitFor := func(text string) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			data, _ := os.ReadFile(stdout.Name())
			if strings.Contains(string(data), text) {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		logs, _ := os.ReadFile(stderr.Name())
		t.Fatalf("live output missing %q: %s", text, logs)
	}
	appendText := func(path, text string) {
		t.Helper()
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		text = encode(text)
		split := len(text)
		if compressed {
			split = len(text) / 2
		}
		_, err = f.WriteString(text[:split])
		if compressed && err == nil {
			time.Sleep(40 * time.Millisecond)
			_, err = f.WriteString(text[split:])
		}
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	waitFor("@a\nAC\n+\nII\n")
	// A new alias of redirected stdout must never become a watched input.
	if err := os.Link(stdout.Name(), filepath.Join(root, "output.fq")); err != nil {
		t.Fatal(err)
	}
	appendText(file, "@b\nGG\n+\n")
	appendText(file, "JJ\n")
	waitFor("@b\nGG\n+\nJJ\n")
	renamed := filepath.Join(root, "m"+suffix)
	if err := os.Rename(file, renamed); err != nil {
		t.Fatal(err)
	}
	appendText(renamed, "@c\nTT\n+\nKK\n")
	waitFor("@c\nTT\n+\nKK\n")
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		logs, _ := os.ReadFile(stderr.Name())
		t.Fatalf("watch exit: %v %s", err, logs)
	}
	data, _ := os.ReadFile(stdout.Name())
	if string(data) != "@a\nAC\n+\nII\n@b\nGG\n+\nJJ\n@c\nTT\n+\nKK\n" {
		t.Fatalf("duplicate/lost live record: %q", data)
	}
}

// TestScatWaitPid checks an already exited PID and parser finalization.
func TestScatWaitPid(t *testing.T) {
	if !IsPidAlive(os.Getpid()) {
		t.Fatal("current process reported dead")
	}
	dir := t.TempDir()
	auditInput(t, dir, "a.fa", ">a\nAC")
	out, stderr, err := runCLI(t, "", "scat", "-i", "fasta", "-p", "2147483647", dir)
	if err != nil || out != ">a\nAC\n" {
		t.Fatalf("PID finalization failed: %q %v %s", out, err, stderr)
	}
}

// TestScatOutputAliases protects matching files before opening the output.
func TestScatOutputAliases(t *testing.T) {
	for _, alias := range []string{"same", "symlink", "hardlink"} {
		t.Run(alias, func(t *testing.T) {
			dir := t.TempDir()
			root := filepath.Join(dir, "root")
			const data = "@r1\nAC\n+\nII\n"
			input := auditInput(t, root, "input.fq", data)
			output := input
			if alias != "same" {
				output = filepath.Join(dir, "alias")
				link := os.Link
				if alias == "symlink" {
					link = os.Symlink
				}
				if err := link(input, output); err != nil {
					t.Skipf("links unavailable: %v", err)
				}
			}
			_, stderr, err := runCLI(t, "", "scat", "--find-only", root, "-o", output)
			after, _ := os.ReadFile(input)
			if err == nil || !strings.Contains(stderr, "cannot be the same") || string(after) != data {
				t.Fatalf("input overwritten: %v %s", err, stderr)
			}
		})
	}
}

// TestScatExcludesNewOutput prevents reading the growing output as another input.
func TestScatExcludesNewOutput(t *testing.T) {
	for _, live := range []bool{false, true} {
		t.Run(fmt.Sprint(live), func(t *testing.T) {
			root := t.TempDir()
			const data = "@r1\nAC\n+\nII\n"
			auditInput(t, root, "input.fq", data)
			output := filepath.Join(root, "output.fq")
			args := []string{"scat", root, "-o", output, "--time-limit", "150ms", "-D", "0ms", "-d", "0"}
			if !live {
				args = append(args, "--find-only")
			}
			_, stderr, err := runCLI(t, "", args...)
			after, _ := os.ReadFile(output)
			if err != nil || string(after) != data || strings.Contains(stderr, "file: "+output) {
				t.Fatalf("output was read as input: %v %s bytes=%d", err, stderr, len(after))
			}
		})
	}
}
