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

// TestOutputProtectsRedirectedStdin compares the open descriptor before truncation.
func TestOutputProtectsRedirectedStdin(t *testing.T) {
	for _, skip := range []bool{false, true} {
		for _, alias := range []string{"same", "symlink", "hardlink"} {
			t.Run(fmt.Sprintf("%s/skip=%v", alias, skip), func(t *testing.T) {
				dir := t.TempDir()
				const data = ">r1\nACGT\n"
				input := auditInput(t, dir, "input.fa", data)
				output := input
				if alias != "same" {
					output = filepath.Join(dir, "output.fa")
					var err error
					if alias == "symlink" {
						err = os.Symlink(input, output)
					} else {
						err = os.Link(input, output)
					}
					if err != nil {
						t.Skipf("cannot create %s: %v", alias, err)
					}
				}
				fh, err := os.Open(input)
				if err != nil {
					t.Fatal(err)
				}
				defer fh.Close()
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCLIProcess$", "--", "seq", "-o", output)
				if skip {
					command.Args = append(command.Args, "--skip-file-check")
				}
				command.Env = append(os.Environ(), "SEQKIT_CLI_TEST_HELPER=1")
				command.Stdin = fh
				message, err := command.CombinedOutput()
				got, readErr := os.ReadFile(input)
				if err == nil || !strings.Contains(string(message), "cannot be the same") || readErr != nil || string(got) != data {
					t.Fatalf("redirected input not protected: %v %s read=%v", err, message, readErr)
				}
			})
		}
	}
}

// TestSkippedFileChecksProtectInputs keeps identical path checks across commands.
func TestSkippedFileChecksProtectInputs(t *testing.T) {
	commands := [][]string{
		{"seq"}, {"fx2tab"}, {"fq2fa"}, {"tab2fx"}, {"convert"},
		{"head"}, {"head-genome"}, {"range", "-r", "1:1"},
		{"sliding", "-s", "1", "-W", "2"}, {"duplicate", "-n", "2"},
		{"grep", "-p", "r1"}, {"locate", "-p", "AC"}, {"amplicon", "-F", "AC"},
		{"replace", "-p", "r1", "-r", "r2"}, {"mutate", "-p", "1:A"},
		{"restart", "-i", "1"}, {"rename"}, {"rmdup"}, {"sort"}, {"shuffle"},
		{"sample", "-p", "1"}, {"sample2", "-p", "1"}, {"translate"},
		{"subseq", "-r", "1:2"}, {"faidx"}, {"stats"}, {"sum"},
		{"sana", "-i", "fasta"}, {"bam", "-s"}, {"watch"},
		{"fish", "-F", "AC"}, {"merge-slides"}, {"split2", "-s", "1"},
		{"common", "EXTRA"}, {"concat", "EXTRA"}, {"fa2fq", "-f", "EXTRA"},
	}
	for _, args := range commands {
		t.Run(args[0], func(t *testing.T) {
			dir := t.TempDir()
			const data = ">r1\nACGTACGT\n"
			input := auditInput(t, dir, "input.fa", data)
			extra := auditInput(t, dir, "extra.fa", data)
			command := append([]string(nil), args...)
			for i := range command {
				if command[i] == "EXTRA" {
					command[i] = extra
				}
			}
			_, stderr, err := runCLI(t, "", append(command, "--skip-file-check", input, "-o", input)...)
			got, readErr := os.ReadFile(input)
			if err == nil || !strings.Contains(stderr, "cannot be the same") || readErr != nil || string(got) != data {
				t.Fatalf("input not protected: err=%v stderr=%s read=%v changed=%v", err, stderr, readErr, string(got) != data)
			}
		})
	}
}

// TestSkippedFileChecksAvoidInputMetadata leaves unused inputs unopened and
// unprobed, while normal preflight still reports missing files and aliases.
func TestSkippedFileChecksAvoidInputMetadata(t *testing.T) {
	for _, kind := range []string{"missing", "symlink-loop", "symlink-output", "hardlink-output"} {
		for _, fromList := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/list=%v", kind, fromList), func(t *testing.T) {
				dir := t.TempDir()
				const data = ">r1\nACGT\n"
				auditInput(t, dir, "input.fa", data)
				output := auditInput(t, dir, "output.fa", "untouched\n")
				unused := filepath.Join(dir, "unused.fa")
				var err error
				switch kind {
				case "symlink-loop":
					err = os.Symlink("unused.fa", unused)
				case "symlink-output":
					err = os.Symlink("output.fa", unused)
				case "hardlink-output":
					err = os.Link(output, unused)
				}
				if err != nil {
					t.Skipf("cannot create %s: %v", kind, err)
				}
				inputs := []string{"input.fa", "unused.fa"}
				if fromList {
					list := auditInput(t, dir, "inputs.list", strings.Join(inputs, "\n")+"\n")
					inputs = []string{"--infile-list", list}
				}
				args := append([]string{"head", "-n", "1", "-o", output}, inputs...)
				_, stderr, err := auditCLI(t, dir, "", args...)
				got, readErr := os.ReadFile(output)
				if err == nil || readErr != nil || string(got) != "untouched\n" {
					t.Fatalf("normal preflight: err=%v stderr=%s output=%q read=%v", err, stderr, got, readErr)
				}
				_, stderr, err = auditCLI(t, dir, "", append(args, "--skip-file-check")...)
				got, readErr = os.ReadFile(output)
				if err != nil || readErr != nil || string(got) != data {
					t.Fatalf("skipped preflight: err=%v stderr=%s output=%q read=%v", err, stderr, got, readErr)
				}
			})
		}
	}
}

// TestSkippedFileChecksNormalizePaths compares relative and absolute spellings.
func TestSkippedFileChecksNormalizePaths(t *testing.T) {
	for _, input := range []string{"input.fa", "./input.fa", "sub/../input.fa"} {
		t.Run(input, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dir, "sub"), 0700); err != nil {
				t.Fatal(err)
			}
			const data = ">r1\nACGT\n"
			output := auditInput(t, dir, "input.fa", data)
			_, stderr, err := auditCLI(t, dir, "", "seq", "--skip-file-check", input, "-o", output)
			got, readErr := os.ReadFile(output)
			if err == nil || !strings.Contains(stderr, "cannot be the same") || readErr != nil || string(got) != data {
				t.Fatalf("normalized input not protected: err=%v stderr=%s read=%v changed=%v", err, stderr, readErr, string(got) != data)
			}
		})
	}
}

// TestInputListStdinOutput keeps stdin-based lists usable with a file output.
func TestInputListStdinOutput(t *testing.T) {
	for _, skip := range []bool{false, true} {
		dir := t.TempDir()
		const data = ">r1\nACGT\n"
		input := auditInput(t, dir, "input.fa", data)
		output := filepath.Join(dir, "output.fa")
		args := []string{"seq", "--infile-list", "-", "-o", output}
		if skip {
			args = append(args, "--skip-file-check")
		}
		_, stderr, err := runCLI(t, input+"\n", args...)
		got, readErr := os.ReadFile(output)
		if err != nil || readErr != nil || string(got) != data {
			t.Fatalf("skip=%v: err=%v stderr=%s output=%q read=%v", skip, err, stderr, got, readErr)
		}
	}
}

// TestAuxiliaryInputOutputAliases checks metadata inputs before any output is opened.
func TestAuxiliaryInputOutputAliases(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		data string
	}{
		{"grep", []string{"grep", "-f", "AUX"}, "r1\n"},
		{"locate", []string{"locate", "-f", "AUX"}, ">p1\nAC\n"},
		{"fa2fq", []string{"fa2fq", "-f", "AUX"}, ">r1\nAC\n"},
		{"bed", []string{"subseq", "--bed", "AUX"}, "r1\t0\t2\n"},
		{"gtf", []string{"subseq", "--gtf", "AUX"}, "r1\tsrc\tgene\t1\t2\t.\t+\t.\tgene_id \"g1\";\n"},
		{"primers", []string{"amplicon", "-p", "AUX"}, "p1\tAC\tGT\n"},
		{"replace-kv", []string{"replace", "-p", "^(r1)$", "-r", "{kv}", "-k", "AUX"}, "r1\tr2\n"},
		{"replace-filter", []string{"replace", "-p", "r1", "-r", "r2", "--f-pattern-file", "AUX"}, "r1\n"},
		{"faidx-regions", []string{"faidx", "-l", "AUX"}, "r1\n"},
		{"fish-queries", []string{"fish", "-f", "AUX"}, ">q1\nACGT\n"},
		{"input-list", []string{"seq", "--infile-list", "AUX"}, "INPUT\n"},
	} {
		for _, alias := range []string{"same", "symlink", "hardlink"} {
			t.Run(tt.name+"/"+alias, func(t *testing.T) {
				dir := t.TempDir()
				input := auditInput(t, dir, "input.fa", ">r1\nACGT\n")
				data := strings.ReplaceAll(tt.data, "INPUT", input)
				auxiliary := auditInput(t, dir, "auxiliary", data)
				output := auxiliary
				if alias != "same" {
					output = filepath.Join(dir, "alias")
					var err error
					if alias == "symlink" {
						err = os.Symlink(auxiliary, output)
					} else {
						err = os.Link(auxiliary, output)
					}
					if err != nil {
						t.Skipf("cannot create %s: %v", alias, err)
					}
				}
				args := append([]string(nil), tt.args...)
				for i := range args {
					if args[i] == "AUX" {
						args[i] = auxiliary
					}
				}
				for _, skip := range []bool{false, true} {
					command := append(append([]string(nil), args...), input, "-o", output)
					if skip {
						command = append(command, "--skip-file-check")
					}
					_, stderr, err := runCLI(t, "", command...)
					got, readErr := os.ReadFile(auxiliary)
					if err == nil || !strings.Contains(stderr, "cannot be the same") || readErr != nil || string(got) != data {
						t.Fatalf("skip=%v: auxiliary input not protected: %v %s read=%v changed=%v", skip, err, stderr, readErr, string(got) != data)
					}
				}
			})
		}
	}
}

// TestSkippedFileChecksProtectAuxiliaryOutputs covers rmdup's extra writers too.
func TestSkippedFileChecksProtectAuxiliaryOutputs(t *testing.T) {
	for _, flag := range []string{"-d", "-D"} {
		dir := t.TempDir()
		const data = ">r1\nAC\n>r1\nAC\n"
		input := auditInput(t, dir, "input.fa", data)
		_, stderr, err := runCLI(t, "", "rmdup", "--skip-file-check", input, flag, input)
		got, _ := os.ReadFile(input)
		if err == nil || !strings.Contains(stderr, "cannot be the same") || string(got) != data {
			t.Fatalf("%s: input not protected: %v %s", flag, err, stderr)
		}
	}
}
