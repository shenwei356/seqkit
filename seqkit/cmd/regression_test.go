package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shenwei356/bio/util"
)

func TestL50FromLengths(t *testing.T) {
	tests := []struct {
		name    string
		lengths []uint64
		want    int
	}{
		{"equal lengths", []uint64{4, 4, 4, 4}, 2},
		{"first record reaches half", []uint64{10, 5, 5}, 1},
		{"repeated N50 length", []uint64{7, 7, 1, 1}, 2},
		{"zero total", []uint64{0, 0}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := util.NewLengthStats()
			counts := make(map[uint64]uint64)
			for _, length := range tt.lengths {
				stats.Add(length)
				counts[length]++
			}
			if got := l50FromLengths(counts, stats.N50(), stats.Sum()); got != tt.want {
				t.Fatalf("L50 = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCLIProcess(t *testing.T) {
	if os.Getenv("SEQKIT_CLI_TEST_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			RootCmd.SetArgs(os.Args[i+1:])
			if err := RootCmd.Execute(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
	}
	os.Exit(1)
}

func runCLI(t *testing.T, input string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(os.Args[0], append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)...)
	cmd.Env = append(os.Environ(), "SEQKIT_CLI_TEST_HELPER=1")
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// TestRmdupRejectsInputOutputs checks all input paths before any output is opened.
func TestRmdupRejectsInputOutputs(t *testing.T) {
	for _, flag := range []string{"-o", "-d", "-D"} {
		for inputIndex := 0; inputIndex < 2; inputIndex++ {
			for _, alias := range []string{"same path", "symlink", "hard link"} {
				t.Run(fmt.Sprintf("%s/input%d/%s", flag, inputIndex, alias), func(t *testing.T) {
					dir := t.TempDir()
					inputs := []string{filepath.Join(dir, "in.fa"), filepath.Join(dir, "dup.fa")}
					contents := []string{">a\nAAAA\n>b\nCCCC\n", ">a\nTTTT\n>b\nTTTT\n"}
					for i, path := range inputs {
						if err := os.WriteFile(path, []byte(contents[i]), 0600); err != nil {
							t.Fatal(err)
						}
					}
					target := inputs[inputIndex]
					switch alias {
					case "symlink":
						target = filepath.Join(dir, "alias.fa")
						if err := os.Symlink(inputs[inputIndex], target); err != nil {
							t.Fatal(err)
						}
					case "hard link":
						target = filepath.Join(dir, "alias.fa")
						if err := os.Link(inputs[inputIndex], target); err != nil {
							t.Fatal(err)
						}
					}
					out := filepath.Join(dir, "out.fa")
					const originalOutput = "existing output\n"
					if err := os.WriteFile(out, []byte(originalOutput), 0600); err != nil {
						t.Fatal(err)
					}
					args := []string{"rmdup", "-s", "--quiet"}
					if flag != "-o" {
						args = append(args, "-o", out)
					}
					args = append(args, flag, target)
					stdout, stderr, err := runCLI(t, "", append(args, inputs...)...)
					if err == nil || !strings.Contains(stderr, "input and output files cannot be the same") || strings.Contains(stderr, "panic:") {
						t.Errorf("expected file collision error, got %v: %s", err, stderr)
					}
					if stdout != "" {
						t.Errorf("unexpected stdout: %q", stdout)
					}
					for i, path := range append(inputs, out) {
						want := originalOutput
						if i < len(contents) {
							want = contents[i]
						}
						got, err := os.ReadFile(path)
						if err != nil || string(got) != want {
							t.Errorf("file %s changed: got %q, want %q, err %v", path, got, want, err)
						}
					}
				})
			}
		}
	}
}

// TestRmdupAuxiliaryOutputs checks normal duplicate sequence and count outputs.
func TestRmdupAuxiliaryOutputs(t *testing.T) {
	dir := t.TempDir()
	dup := filepath.Join(dir, "duplicates.fa")
	num := filepath.Join(dir, "counts.tsv")
	input := ">a\nAAAA\n>b\nCCCC\n>c\nAAAA\n"
	stdout, stderr, err := runCLI(t, input, "rmdup", "-s", "-P", "--quiet", "-d", dup, "-D", num)
	if err != nil || stdout != ">a\nAAAA\n>b\nCCCC\n" {
		t.Fatalf("rmdup failed: stdout=%q, err=%v, stderr=%q", stdout, err, stderr)
	}
	for path, want := range map[string]string{dup: ">c\nAAAA\n", num: "2\ta, c\n"} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Errorf("file %s: got %q, want %q, err %v", path, got, want, err)
		}
	}
}

func TestStatsL50CLI(t *testing.T) {
	input := ">a\nAAAA\n>b\nAAAA\n>c\nAAAA\n>d\nAAAA\n"
	stdout, stderr, err := runCLI(t, input, "stats", "-Ta", "--quiet")
	if err != nil {
		t.Fatalf("stats failed: %v: %s", err, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Fatalf("unexpected stats output: %q", stdout)
	}
	fields := strings.Split(lines[1], "\t")
	if fields[12] != "4" || fields[13] != "2" {
		t.Fatalf("N50/N50_num = %s/%s, want 4/2", fields[12], fields[13])
	}
}

func TestWatchRegressions(t *testing.T) {
	input := ">a\nAAAA\n>b\nCCCC\n"
	for _, args := range [][]string{
		{"watch", "-p", "0", "-Q", "-y"},
		{"watch", "-f", "ReadLen,GC", "-Q", "-y"},
		{"watch", "-f", "GCSkew", "-L", "-Q", "-y"},
	} {
		_, stderr, err := runCLI(t, input, args...)
		if err == nil || strings.Contains(stderr, "panic:") {
			t.Fatalf("%v: expected a regular error, got %v: %s", args, err, stderr)
		}
	}
	stdout, stderr, err := runCLI(t, input, "watch", "-f", "GCSkew", "-y", "-Q", "-x")
	if err != nil {
		t.Fatalf("watch failed: %v: %s", err, stderr)
	}
	if stdout != input || strings.Contains(stderr, "NaN") || !strings.Contains(stderr, "\t1\n") {
		t.Fatalf("unexpected GC-skew output: stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestBAMRegressions(t *testing.T) {
	first := filepath.Join("..", "..", "tests", "pcs109_5k.bam")
	second := filepath.Join("..", "..", "tests", "pcs109_5k_prim.bam")
	for _, args := range [][]string{
		{"bam", "-f", "ReadLen", "-p", "0", "-Q", "-y", first},
		{"bam", "-c", "-", "-p", "0", first},
		{"bam", "-f", "ReadLen,MapQual", "-m", "100000", "-Q", first},
		{"bam", "-f", "ReadLen", "-Q", "-y", first, second},
		{"bam", "--idx-count", first, second},
	} {
		_, stderr, err := runCLI(t, "", args...)
		if err == nil || strings.Contains(stderr, "panic:") {
			t.Fatalf("%v: expected a regular error, got %v: %s", args, err, stderr)
		}
	}
}

func TestSanaMultipleFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input.fa")
	input := ">a\nACGT\n"
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runCLI(t, "", "sana", "-i", "fasta", "--quiet", path, path)
	if err != nil || stdout != input+input {
		t.Fatalf("sana output = %q, err = %v, stderr = %q", stdout, err, stderr)
	}
}

func TestQuietCommands(t *testing.T) {
	_, stderr, err := runCLI(t, ">x\nACGT\n", "seq", "-t", "unlimit", "-p", "--quiet")
	if err != nil || stderr != "" {
		t.Fatalf("seq --quiet: err = %v, stderr = %q", err, stderr)
	}
	_, stderr, err = runCLI(t, "", "scat", "-f", "-g", "-i", "fasta", "--quiet", filepath.Join("..", "..", "tests"))
	if err != nil || stderr != "" {
		t.Fatalf("scat --quiet: err = %v, stderr = %q", err, stderr)
	}
}

func TestAllCommandHelp(t *testing.T) {
	commands := strings.Fields(`faidx scat seq sliding stats subseq translate watch
		convert fa2fq fq2fa fx2tab tab2fx amplicon fish grep locate
		common duplicate head head-genome pair range rmdup sample sample2 split split2
		concat mutate rename replace restart sana shuffle sort bam merge-slides sum
		genautocomplete version`)
	if len(commands) != 41 {
		t.Fatalf("expected 41 commands, got %d", len(commands))
	}
	for _, name := range commands {
		t.Run(name, func(t *testing.T) {
			stdout, stderr, err := runCLI(t, "", name, "--help")
			if err != nil || !strings.Contains(stdout, "Usage:") {
				t.Fatalf("%s --help failed: %v: stdout=%q stderr=%q", name, err, stdout, stderr)
			}
		})
	}
}
