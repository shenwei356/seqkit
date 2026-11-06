package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRenameMixedFileFormats preserves each input file's format.
func TestRenameMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "rename")
}

// TestRenameProtectsInputList checks generated names and recursive cleanup.
func TestRenameProtectsInputList(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(map[bool]string{false: "generated output", true: "directory cleanup"}[force], func(t *testing.T) {
			dir := t.TempDir()
			input := auditInput(t, dir, "in.fa", ">r1\nACGT\n")
			outdir := filepath.Join(dir, "out")
			name := "in.fa"
			if force {
				name = "inputs.list"
			}
			list := auditInput(t, outdir, name, input+"\n")
			keep := auditInput(t, outdir, "keep", "keep")
			args := []string{"rename", "-m", "-O", outdir, "--infile-list", list}
			if force {
				args = append(args, "-f")
			}
			_, stderr, err := runCLI(t, "", args...)
			data, readErr := os.ReadFile(list)
			marker, markerErr := os.ReadFile(keep)
			if readErr != nil || string(data) != input+"\n" || markerErr != nil || string(marker) != "keep" {
				t.Fatalf("input list modified: err=%v preserved=%v stderr=%s", err, string(data) == input+"\n", stderr)
			}
			if force && err != nil || !force && (err == nil || !strings.Contains(stderr, "input")) {
				t.Fatalf("unexpected rename result: %v %s", err, stderr)
			}
		})
	}
}

// TestRenameForceProtectsSymlinkChain keeps intermediate input links intact.
func TestRenameForceProtectsSymlinkChain(t *testing.T) {
	dir := t.TempDir()
	input := auditInput(t, dir, "source/in.fa", ">r1\nACGT\n")
	outdir := filepath.Join(dir, "out")
	keep := auditInput(t, outdir, "keep", "keep")
	link := filepath.Join(outdir, "input-link")
	if err := os.Symlink(filepath.Dir(input), link); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, "alias")
	if err := os.Symlink(link, alias); err != nil {
		t.Fatal(err)
	}
	_, stderr, err := runCLI(t, "", "rename", "-m", "-f", "-O", outdir, filepath.Join(alias, "in.fa"))
	_, linkErr := os.Lstat(link)
	data, readErr := os.ReadFile(keep)
	if err != nil || linkErr != nil || readErr != nil || string(data) != "keep" {
		t.Fatalf("input symlink removed: err=%v link=%v marker=%q stderr=%s", err, linkErr, data, stderr)
	}
}
