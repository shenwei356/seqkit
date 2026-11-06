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

// TestSplitOutputRegistryHardlinks protects existing and newly created output identities.
func TestSplitOutputRegistryHardlinks(t *testing.T) {
	for _, existing := range []bool{false, true} {
		dir := t.TempDir()
		output := filepath.Join(dir, "output.fa")
		if existing {
			auditInput(t, dir, "output.fa", "old\n")
		}
		registry := splitOutputRegistry{}
		w, err := registry.open(output)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.WriteString("keep\n"); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		alias := filepath.Join(dir, "alias.fa")
		if err := os.Link(output, alias); err != nil {
			t.Skipf("hard links unavailable: %v", err)
		}
		if w, err := registry.open(alias); err == nil {
			w.Close()
			t.Fatal("output hard link accepted")
		}
		got, err := os.ReadFile(output)
		if err != nil || string(got) != "keep\n" {
			t.Fatalf("previous output changed: %v", err)
		}
		// An unrelated existing file must remain writable after a rejected alias.
		next := auditInput(t, dir, "next.fa", "old\n")
		w, err = registry.open(next)
		if err != nil {
			t.Fatalf("distinct output rejected: %v", err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

// TestSplitProtectsInputList checks generated output aliases even with skipped input checks.
func TestSplitProtectsInputList(t *testing.T) {
	for _, name := range []string{"split", "split2"} {
		for _, alias := range []string{"same", "symlink", "hardlink"} {
			for _, force := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/force=%v", name, alias, force), func(t *testing.T) {
					dir := t.TempDir()
					input := auditInput(t, dir, "input.fa", ">a\nACGT\n")
					outdir := filepath.Join(dir, "out")
					if err := os.Mkdir(outdir, 0700); err != nil {
						t.Fatal(err)
					}
					output := filepath.Join(outdir, "input.part_001.fa")
					list := output
					if alias != "same" {
						list = filepath.Join(dir, "inputs.txt")
					}
					data := input + "\n"
					if err := os.WriteFile(list, []byte(data), 0600); err != nil {
						t.Fatal(err)
					}
					if alias != "same" {
						link := os.Link
						if alias == "symlink" {
							link = os.Symlink
						}
						if err := link(list, output); err != nil {
							t.Skipf("links unavailable: %v", err)
						}
					}
					args := []string{name, "-s", "1", "-O", outdir, "--infile-list", list, "--skip-file-check"}
					if force {
						args = append(args, "--force")
					}
					_, stderr, err := runCLI(t, "", args...)
					got, readErr := os.ReadFile(list)
					if err == nil || readErr != nil || string(got) != data {
						t.Fatalf("input list overwritten: %v %s; read=%v", err, stderr, readErr)
					}
				})
			}
		}
	}
}

// TestSplitProtectsRedirectedInputList retains the identity of list stdin.
func TestSplitProtectsRedirectedInputList(t *testing.T) {
	for _, name := range []string{"split", "split2"} {
		dir := t.TempDir()
		input := auditInput(t, dir, "input.fa", ">a\nACGT\n")
		data := input + "\n"
		list := auditInput(t, dir, "input.part_001.fa", data)
		fh, err := os.Open(list)
		if err != nil {
			t.Fatal(err)
		}
		defer fh.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCLIProcess$", "--", name, "-s", "1", "-O", dir, "--infile-list", "-", "--skip-file-check")
		command.Env = append(os.Environ(), "SEQKIT_CLI_TEST_HELPER=1")
		command.Stdin = fh
		message, err := command.CombinedOutput()
		got, readErr := os.ReadFile(list)
		if err == nil || !strings.Contains(string(message), "cannot be the same") || readErr != nil || string(got) != data {
			t.Fatalf("stdin list overwritten: %v %s; read=%v", err, message, readErr)
		}
	}
}

// TestSplitOutputRegistrySkippedChecks skips input metadata but keeps lexical
// input protection and filesystem identity checks between generated outputs.
func TestSplitOutputRegistrySkippedChecks(t *testing.T) {
	dir := t.TempDir()
	input := auditInput(t, dir, "input.fa", ">r1\nACGT\n")
	loop := filepath.Join(dir, "loop.fa")
	if err := os.Symlink("loop.fa", loop); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	output := filepath.Join(dir, "output.fa")
	registry := splitOutputRegistry{inputs: []string{loop, input}}
	if w, err := registry.open(output); err == nil {
		w.Close()
		t.Fatal("normal checks accepted a symlink loop")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("output opened before validation: %v", err)
	}
	registry.skipInputFileCheck = true
	w, err := registry.open(output)
	if err != nil {
		t.Fatalf("skipped checks probed unused input: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, "output-alias.fa")
	if err := os.Symlink(output, alias); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{input, alias} {
		if w, err := registry.open(path); err == nil {
			w.Close()
			t.Fatalf("accepted conflicting output: %s", path)
		} else if !strings.Contains(err.Error(), "cannot be the same") {
			t.Fatalf("unexpected collision error: %v", err)
		}
	}
	got, err := os.ReadFile(input)
	if err != nil || string(got) != ">r1\nACGT\n" {
		t.Fatalf("input changed: read=%v", err)
	}
}

// TestSplitRedirectedStdin protects the source file even after a buffered reader
// has consumed and closed stdin while discovering generated output names.
func TestSplitRedirectedStdin(t *testing.T) {
	for _, name := range []string{"split", "split2"} {
		for _, skip := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/skip=%v", name, skip), func(t *testing.T) {
				dir := t.TempDir()
				const data = ">r1\nACGT\n"
				input := auditInput(t, dir, "stdin.part_001.fasta", data)
				args := []string{name, "-s", "1", "-O", dir}
				if name == "split2" {
					mate := auditInput(t, dir, "mate.fa", data)
					args = append(args, "-1", "-", "-2", mate)
				}
				if skip {
					args = append(args, "--skip-file-check")
				}
				fh, err := os.Open(input)
				if err != nil {
					t.Fatal(err)
				}
				defer fh.Close()
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				command := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)...)
				command.Env = append(os.Environ(), "SEQKIT_CLI_TEST_HELPER=1")
				command.Stdin = fh
				message, err := command.CombinedOutput()
				got, readErr := os.ReadFile(input)
				if err == nil || !strings.Contains(string(message), "cannot be the same") || readErr != nil || string(got) != data {
					t.Fatalf("redirected input not protected: err=%v message=%s read=%v", err, message, readErr)
				}
			})
		}
	}
}
