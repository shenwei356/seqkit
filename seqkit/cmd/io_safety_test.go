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

	"github.com/shenwei356/bio/seqio/fai"
)

// TestFASTAIndexProtectsInputList covers every two-pass index update path.
func TestFASTAIndexProtectsInputList(t *testing.T) {
	commands := [][]string{
		{"sort", "-2"}, {"shuffle", "-2"}, {"split", "-2", "-s", "1"},
		{"split", "-2", "-p", "2"}, {"split", "-2", "-i"}, {"split", "-2", "-r", "1:2"},
	}
	for _, command := range commands {
		for _, alias := range []string{"same", "symlink", "hardlink"} {
			t.Run(strings.Join(command, "_")+"/"+alias, func(t *testing.T) {
				dir := t.TempDir()
				input := auditInput(t, dir, "input.fa", ">a\nACGT\n>b\nTGCA\n")
				index := input + ".seqkit.fai"
				list := index
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
					if err := link(list, index); err != nil {
						t.Skipf("links unavailable: %v", err)
					}
				}
				args := append(append([]string(nil), command...), "-U", "--infile-list", list, "--skip-file-check")
				output := filepath.Join(dir, "output")
				if command[0] == "split" {
					args = append(args, "-O", output)
				} else {
					auditInput(t, dir, "output", "keep\n")
					args = append(args, "-o", output)
				}
				_, stderr, err := runCLI(t, "", args...)
				got, readErr := os.ReadFile(list)
				indexData, indexErr := os.ReadFile(index)
				if err == nil || !strings.Contains(stderr, "cannot be the same") || readErr != nil || indexErr != nil || string(got) != data || string(indexData) != data {
					t.Fatalf("index update changed input list: %v %s; read=%v index=%v", err, stderr, readErr, indexErr)
				}
				if command[0] != "split" {
					got, readErr = os.ReadFile(output)
					if readErr != nil || string(got) != "keep\n" {
						t.Fatalf("output opened before index validation: %v", readErr)
					}
				}
			})
		}
	}
}

// TestFASTAIndexUpdateDirectories protects direct paths and directory symlinks.
func TestFASTAIndexUpdateDirectories(t *testing.T) {
	commands := [][]string{
		{"faidx"}, {"subseq", "-r", "1:2"}, {"sort", "-2"}, {"shuffle", "-2"},
		{"split", "-2", "-s", "1"}, {"split", "-2", "-p", "2"},
		{"split", "-2", "-i"}, {"split", "-2", "-r", "1:2"},
	}
	for _, command := range commands {
		for _, alias := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/symlink=%v", strings.Join(command, "_"), alias), func(t *testing.T) {
				dir := t.TempDir()
				input := auditInput(t, dir, "input.fa", ">a\nACGT\n>b\nTGCA\n")
				index := input + ".seqkit.fai"
				if command[0] == "faidx" {
					index = input + ".fai"
				}
				target := index
				if alias {
					target = filepath.Join(dir, "target")
				}
				if err := os.Mkdir(target, 0700); err != nil {
					t.Fatal(err)
				}
				marker := auditInput(t, target, "keep", "keep\n")
				if alias {
					if err := os.Symlink(target, index); err != nil {
						t.Skipf("symlinks unavailable: %v", err)
					}
				}
				args := append(append([]string(nil), command...), "-U", input)
				output := filepath.Join(dir, "output")
				if command[0] == "split" {
					args = append(args, "-O", output)
				} else {
					auditInput(t, dir, "output", "keep\n")
					args = append(args, "-o", output)
				}
				_, stderr, err := runCLI(t, "", args...)
				got, readErr := os.ReadFile(marker)
				if err == nil || !strings.Contains(stderr, "regular file") || readErr != nil || string(got) != "keep\n" {
					t.Fatalf("index directory was not protected: %v %s; marker=%v", err, stderr, readErr)
				}
				if command[0] != "split" {
					got, readErr = os.ReadFile(output)
					if readErr != nil || string(got) != "keep\n" {
						t.Fatalf("output changed before index validation: %v", readErr)
					}
				}
			})
		}
	}
}

// TestFASTAIndexOutputAliases checks index identity before any output is opened.
func TestFASTAIndexOutputAliases(t *testing.T) {
	for _, command := range [][]string{{"faidx"}, {"subseq", "-r", "1:2"}, {"sort", "-2"}, {"shuffle", "-2"}} {
		for _, alias := range []string{"same", "symlink", "hardlink", "new"} {
			t.Run(command[0]+"/"+alias, func(t *testing.T) {
				dir := t.TempDir()
				input := auditInput(t, dir, "input.fa", ">a\nACGT\n")
				index := input + ".seqkit.fai"
				build := []string{"faidx", "-f", input}
				if command[0] == "faidx" {
					index = input + ".fai"
					build = []string{"faidx", input}
				}
				var before []byte
				if alias != "new" {
					if _, stderr, err := runCLI(t, "", build...); err != nil {
						t.Fatalf("build index: %v %s", err, stderr)
					}
					var err error
					before, err = os.ReadFile(index)
					if err != nil {
						t.Fatal(err)
					}
				}
				output := index
				if alias == "symlink" || alias == "hardlink" {
					output = filepath.Join(dir, "alias")
					link := os.Link
					if alias == "symlink" {
						link = os.Symlink
					}
					if err := link(index, output); err != nil {
						t.Skipf("links unavailable: %v", err)
					}
				}
				for _, update := range [][]string{nil, {"-U"}} {
					args := append(append([]string(nil), command...), input, "--skip-file-check", "-o", output)
					_, stderr, err := runCLI(t, "", append(args, update...)...)
					after, readErr := os.ReadFile(index)
					if err == nil || !strings.Contains(stderr, "cannot be the same") || string(after) != string(before) {
						t.Fatalf("index/output conflict accepted: %v %s", err, stderr)
					}
					if alias == "new" && !os.IsNotExist(readErr) || alias != "new" && readErr != nil {
						t.Fatalf("index existence changed: %v", readErr)
					}
				}
			})
		}
	}
}

// TestFASTAIndexLeadingComments follows the index builder's leading-comment rules.
func TestFASTAIndexLeadingComments(t *testing.T) {
	for i, prefix := range []string{";comment\n", "\n;first\n;second\r\n", "\xef\xbb\xbf;comment\n"} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			file := auditInput(t, t.TempDir(), "input.fa", prefix+">a\nACGT\n>b\nTGCA\n")
			index, err := fai.CreateWithIDRegexp(file, file+".seqkit.fai", `^(.+)$`)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateFASTAIndex(file, index); err != nil {
				t.Fatalf("new index rejected: %v", err)
			}
			for _, args := range [][]string{{"sort", "-2", file}, {"shuffle", "-2", file}} {
				out, stderr, err := runCLI(t, "", args...)
				if err != nil || strings.Count(out, ">") != 2 {
					t.Fatalf("reused index rejected: %v %s", err, stderr)
				}
			}
		})
	}
}

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
		{"seq-filter", []string{"seq", "-r", "--f-pattern-file", "AUX"}, "r1\n"},
		{"mutate-filter", []string{"mutate", "-p", "1:T", "-f", "AUX"}, "r1\n"},
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
