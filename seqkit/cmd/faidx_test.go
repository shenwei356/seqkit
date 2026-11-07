package cmd

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestFaidxCompressedPreflight preserves outputs and indexes for unsupported inputs.
func TestFaidxCompressedPreflight(t *testing.T) {
	for _, extension := range []string{".gz", ".xz", ".zst", ".bz2", ".lz4"} {
		encoded := compressedInput(t, extension, ">r\nACGT\n")
		for _, suffix := range []string{extension, strings.ToUpper(extension)} {
			for _, fullHead := range []bool{false, true} {
				t.Run(suffix+"/"+map[bool]string{false: "id", true: "full"}[fullHead], func(t *testing.T) {
					dir := t.TempDir()
					input := auditInput(t, dir, "input.fa"+suffix, string(encoded))
					output := auditInput(t, dir, "output.fa", "keep output\n")
					indexSuffix := ".fai"
					flags := []string{"faidx", input, "-U", "--skip-file-check", "-o", output}
					if fullHead {
						indexSuffix = ".seqkit.fai"
						flags = append(flags, "-f")
					}
					index := auditInput(t, dir, "input.fa"+suffix+indexSuffix, "keep index\n")
					_, stderr, err := runCLI(t, "", flags...)
					if err == nil || !strings.Contains(stderr, "not supported") {
						t.Errorf("compressed input was not rejected early: %v %s", err, stderr)
					}
					for path, want := range map[string]string{output: "keep output\n", index: "keep index\n"} {
						data, err := os.ReadFile(path)
						if err != nil || string(data) != want {
							t.Errorf("file changed before compression validation: %s: %v", path, err)
						}
					}
				})
			}
		}
	}
}

// TestFaidxRegionStdin validates index paths before the region reader closes stdin.
func TestFaidxRegionStdin(t *testing.T) {
	dir := t.TempDir()
	input := auditInput(t, dir, "input.fa", ">r\nACGT\n")
	for _, flags := range [][]string{nil, nil, {"-U"}} {
		args := append([]string{"faidx", input, "-l", "-"}, flags...)
		output, stderr, err := runCLI(t, "r:1-2\n", args...)
		if err != nil || output != ">r:1-2\nAC\n" {
			t.Fatalf("region stdin failed with index creation/reuse/update: %v %s", err, stderr)
		}
	}
	// A redirected region stream must still be protected from index replacement.
	index := auditInput(t, dir, "input.fa.fai", "r:1-2\n")
	output := auditInput(t, dir, "out.fa", "keep\n")
	f, err := os.Open(index)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCLIProcess$", "--", "faidx", input, "-U", "-l", "-", "-o", output)
	command.Env = append(os.Environ(), "SEQKIT_CLI_TEST_HELPER=1")
	command.Stdin = f
	message, err := command.CombinedOutput()
	data, readErr := os.ReadFile(index)
	out, outErr := os.ReadFile(output)
	if err == nil || !strings.Contains(string(message), "cannot be the same") || readErr != nil || string(data) != "r:1-2\n" || outErr != nil || string(out) != "keep\n" {
		t.Fatalf("region stdin was not protected: %v %s", err, message)
	}
}

// TestFaidxInvalidIDRegexp rejects invalid parsers before touching outputs or indexes.
func TestFaidxInvalidIDRegexp(t *testing.T) {
	for _, fullHead := range []bool{false, true} {
		for _, reuse := range []bool{false, true} {
			for _, pattern := range []string{"[", "r"} {
				dir := t.TempDir()
				input := auditInput(t, dir, "input.fa", ">r\nACGT\n")
				output := auditInput(t, dir, "out.fa", "keep\n")
				index := input + ".fai"
				args := []string{"faidx"}
				if fullHead {
					args = append(args, "-f")
					index = input + ".seqkit.fai"
				}
				const indexData = "r\t4\t3\t4\t5\n"
				if reuse {
					if err := os.WriteFile(index, []byte(indexData), 0600); err != nil {
						t.Fatal(err)
					}
					args = append(args, "-U")
				}
				args = append(args, "--id-regexp", pattern, "-o", output, input, "r")
				_, stderr, err := runCLI(t, "", args...)
				if err == nil || strings.Contains(stderr, "panic:") || !strings.Contains(stderr, "[ERRO]") {
					t.Fatalf("invalid parser %q (full=%v reuse=%v): %v %s", pattern, fullHead, reuse, err, stderr)
				}
				data, readErr := os.ReadFile(output)
				if readErr != nil || string(data) != "keep\n" {
					t.Fatalf("output changed: %q %v", data, readErr)
				}
				data, readErr = os.ReadFile(index)
				if reuse && (readErr != nil || string(data) != indexData) || !reuse && !os.IsNotExist(readErr) {
					t.Fatalf("index changed: %q %v", data, readErr)
				}
			}
		}
	}
}

// TestFaidxLongIndexRows keeps creation and reuse results identical.
func TestFaidxLongIndexRows(t *testing.T) {
	file := auditInput(t, t.TempDir(), "input.fa", ">short1\nACGT\n>"+strings.Repeat("x", 70_001)+"\nCCCC\n>short2\nTGCA\n")
	for attempt := 0; attempt < 2; attempt++ {
		out, stderr, err := runCLI(t, "", "faidx", file, "short1", "short2")
		if err != nil || out != ">short1\nACGT\n>short2\nTGCA\n" {
			t.Fatalf("attempt=%d: records lost across index reuse: %v %s", attempt, err, stderr)
		}
	}
}

// TestFaidxUpdateProtectsWorkingDirectory rejects an index path equal to cwd.
func TestFaidxUpdateProtectsWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	input := auditInput(t, dir, "input.fa", ">a\nACGT\n")
	index := input + ".fai"
	if err := os.Mkdir(index, 0700); err != nil {
		t.Fatal(err)
	}
	marker := auditInput(t, index, "keep", "keep\n")
	_, stderr, err := auditCLI(t, index, "", "faidx", "-U", "../input.fa")
	data, readErr := os.ReadFile(marker)
	if err == nil || readErr != nil || string(data) != "keep\n" {
		t.Fatalf("working directory was not protected: %v %s; marker=%v", err, stderr, readErr)
	}
}

// TestFaidxClippedStrands clips both orientations without turning empty ranges into hits.
func TestFaidxClippedStrands(t *testing.T) {
	file := auditInput(t, t.TempDir(), "input.fa", ">r\nACGTAACCGA\n>rna\nACGUAACCGA\n")
	for _, test := range []struct{ query, sequence string }{
		{"r:-12--1", "ACGTAACCGA"},
		{"r:-1--12", "TCGGTTACGT"},
		{"r:-12--3", "ACGTAACC"},
		{"r:-3--12", "GGTTACGT"},
		{"r:1-12", "ACGTAACCGA"},
		{"r:12-1", "TCGGTTACGT"},
		{"r:-20--12", ""},
		{"r:-12--20", ""},
		{"r:12-20", ""},
		{"r:20-12", ""},
		{"r:2--2", "CGTAACCG"},
		{"rna:-1--12", "UCGGUUACGU"},
	} {
		out, stderr, err := runCLI(t, "", "faidx", "--quiet", file, test.query)
		if err != nil || out != ">"+test.query+"\n"+test.sequence+"\n" {
			t.Fatalf("query %s: %q %v %s", test.query, out, err, stderr)
		}
	}
}

// TestFaidxIndexAuxiliaryAliases protects region and input lists during index updates.
func TestFaidxIndexAuxiliaryAliases(t *testing.T) {
	for _, flag := range []string{"-l", "--infile-list"} {
		dir := t.TempDir()
		input := auditInput(t, dir, "input.fa", ">r\nACGT\n")
		data := "r\n"
		if flag == "--infile-list" {
			data = input + "\n"
		}
		index := auditInput(t, dir, "input.fa.fai", data)
		output := auditInput(t, dir, "out.fa", "keep\n")
		args := []string{"faidx", "-U", flag, index, "-o", output}
		if flag == "-l" {
			args = append(args, input)
		}
		_, stderr, err := runCLI(t, "", args...)
		after, _ := os.ReadFile(index)
		out, _ := os.ReadFile(output)
		if err == nil || string(after) != data || string(out) != "keep\n" {
			t.Fatalf("auxiliary input overwritten: %v %s", err, stderr)
		}
	}
}

// TestFaidxDamagedAuxiliaryInput rejects incomplete auxiliary data.
func TestFaidxDamagedAuxiliaryInput(t *testing.T) {
	assertDamagedTextInput(t, "r\n", "faidx", "-l")
}

// TestFaidxEmptyIndexQueries keeps missing-ID failures consistent across index reuse.
func TestFaidxEmptyIndexQueries(t *testing.T) {
	for _, fullHead := range []bool{false, true} {
		input := auditInput(t, t.TempDir(), "empty.fa", "")
		args := []string{"faidx", "--quiet"}
		if fullHead {
			args = append(args, "-f")
		}
		args = append(args, input)
		for attempt := 0; attempt < 2; attempt++ {
			out, stderr, err := runCLI(t, "", append(args, "missing")...)
			if err == nil || out != "" || !strings.Contains(stderr, "sequence not found: missing") {
				t.Fatalf("full=%v attempt=%d: missing query succeeded: %q %v %s", fullHead, attempt, out, err, stderr)
			}
		}
		// Creating or reusing an empty index without queries is still valid.
		out, stderr, err := runCLI(t, "", args...)
		if err != nil || out != "" {
			t.Fatalf("index-only command failed: %q %v %s", out, err, stderr)
		}
	}
}
