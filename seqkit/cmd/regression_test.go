package cmd

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shenwei356/bio/util"
	"github.com/shenwei356/bwt/fmi"
	"github.com/shenwei356/go-logging"
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
	// Match main's log levels so CLI tests distinguish errors from warnings.
	logging.SetBackend(logging.NewBackendFormatter(
		logging.NewLogBackend(os.Stderr, "", 0),
		logging.MustStringFormatter(`[%{level:.4s}] %{message}`),
	))
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)...)
	cmd.Env = append(os.Environ(), "SEQKIT_CLI_TEST_HELPER=1")
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// TestDuplicateStdin rejects repeated streams before opening the output.
func TestDuplicateStdin(t *testing.T) {
	dir := t.TempDir()
	one := auditInput(t, dir, "one.list", "-\n")
	two := auditInput(t, dir, "two.list", "-\n-\n")
	empty := auditInput(t, dir, "empty.list", "")
	for _, command := range []string{"seq", "stats", "sum", "convert", "head", "concat", "sana", "tab2fx", "bam"} {
		for _, tt := range []struct {
			name string
			args []string
		}{
			{"arguments", []string{"-", "-"}},
			{"list", []string{"--infile-list", two}},
			{"arguments and list", []string{"-", "--infile-list", one}},
			{"empty list", []string{"-", "-", "--infile-list", empty}},
			{"skip file check", []string{"--skip-file-check", "--infile-list", two}},
		} {
			t.Run(command+"/"+tt.name, func(t *testing.T) {
				output := auditInput(t, dir, "out.txt", "keep\n")
				args := append([]string{command, "-o", output}, tt.args...)
				out, stderr, err := runCLI(t, ">a\nACGT\n", args...)
				if err == nil || !strings.Contains(stderr, "stdin may only be specified once") || strings.Count(stderr, "[ERRO]") != 1 || out != "" {
					t.Fatalf("err=%v out=%q stderr=%s", err, out, stderr)
				}
				data, err := os.ReadFile(output)
				if err != nil || string(data) != "keep\n" {
					t.Fatalf("output changed before validation: %q %v", data, err)
				}
			})
		}
	}
}

// TestStdinFileList ensures stdin cannot supply both the list and its input data.
func TestStdinFileList(t *testing.T) {
	for _, list := range []string{"-\n", ""} {
		_, stderr, err := runCLI(t, list, "seq", "--infile-list", "-")
		if err == nil || !strings.Contains(stderr, "stdin may only be specified once") {
			t.Fatalf("list=%q err=%v stderr=%s", list, err, stderr)
		}
	}
}

// TestInputListStdinCompatibility preserves a single stdin and repeated regular files.
func TestInputListStdinCompatibility(t *testing.T) {
	dir := t.TempDir()
	const input = ">a\nACGT\n"
	const disk = ">b\nTTTT\n"
	file := auditInput(t, dir, "input.fa", disk)
	one := auditInput(t, dir, "one.list", "-\n")
	files := auditInput(t, dir, "files.list", file+"\n")
	empty := auditInput(t, dir, "empty.list", "")
	for _, tt := range []struct {
		name, stdin, want string
		args              []string
	}{
		{"implicit stdin", input, input, nil},
		{"explicit stdin", input, input, []string{"-"}},
		{"stdin in list", input, input, []string{"-X", one}},
		{"replace implicit stdin", input, disk, []string{"-X", files}},
		{"preserve explicit stdin", input, input + disk, []string{"-", "-X", files}},
		{"empty list fallback", input, input, []string{"-X", empty}},
		{"duplicate regular files", "", disk + disk, []string{file, file}},
		{"list from stdin", file + "\n", disk, []string{"-X", "-"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, stderr, err := runCLI(t, tt.stdin, append([]string{"seq"}, tt.args...)...)
			if err != nil || out != tt.want {
				t.Fatalf("err=%v out=%q want=%q stderr=%s", err, out, tt.want, stderr)
			}
		})
	}
}

// TestSplit2DuplicateStdin covers paired inputs assembled outside the shared list helper.
func TestSplit2DuplicateStdin(t *testing.T) {
	outdir := filepath.Join(t.TempDir(), "parts")
	_, stderr, err := runCLI(t, ">a\nACGT\n", "split2", "-1", "-", "-2", "-", "-s", "1", "-O", outdir)
	if err == nil || !strings.Contains(stderr, "stdin may only be specified once") {
		t.Fatalf("err=%v stderr=%s", err, stderr)
	}
	if _, err := os.Stat(outdir); !os.IsNotExist(err) {
		t.Fatalf("output directory created before validation: %v", err)
	}
}

// TestSplit2SingleStdin allows one paired input stream alongside a regular file.
func TestSplit2SingleStdin(t *testing.T) {
	dir := t.TempDir()
	const input = ">a\nACGT\n"
	file := auditInput(t, dir, "read.fa", input)
	outdir := filepath.Join(dir, "parts")
	_, stderr, err := runCLI(t, input, "split2", "-1", "-", "-2", file, "-s", "1", "-O", outdir)
	if err != nil {
		t.Fatalf("err=%v stderr=%s", err, stderr)
	}
	for _, name := range []string{"stdin.part_001.fasta", "read.part_001.fa"} {
		data, err := os.ReadFile(filepath.Join(outdir, name))
		if err != nil || string(data) != input {
			t.Fatalf("%s: got=%q err=%v", name, data, err)
		}
	}
}

// TestFaidxDashQueries treats query IDs as IDs, not additional input streams.
func TestFaidxDashQueries(t *testing.T) {
	dir := t.TempDir()
	file := auditInput(t, dir, "input.fa", ">-\nACGT\n")
	queries := auditInput(t, dir, "queries.list", "-\n-\n")
	for _, args := range [][]string{{file, "-", "-"}, {file, "-X", queries}} {
		out, stderr, err := runCLI(t, "", append([]string{"faidx"}, args...)...)
		if err != nil || out != ">-\nACGT\n>-\nACGT\n" {
			t.Fatalf("err=%v out=%q stderr=%s", err, out, stderr)
		}
	}
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

// TestSortInputFormats checks mixed-format rejection and empty-file handling.
func TestSortInputFormats(t *testing.T) {
	for _, test := range []struct {
		name   string
		inputs []string
		want   string
		mixed  bool
	}{
		{name: "FASTQ then FASTA", inputs: []string{"@a\nGG\n+\nII\n", ">b\nCC\n"}, mixed: true},
		{name: "FASTA then FASTQ", inputs: []string{">a\nGG\n", "@b\nCC\n+\nJJ\n"}, mixed: true},
		{name: "mixed with empty files", inputs: []string{"", "@a\nGG\n+\nII\n", "\n", ">b\nCC\n"}, mixed: true},
		{name: "empty FASTQ record then FASTA", inputs: []string{"@a\n\n+\n\n", ">b\nCC\n"}, mixed: true},
		{name: "empty FASTA record then FASTQ", inputs: []string{">a\n\n", "@b\nCC\n+\nJJ\n"}, mixed: true},
		{name: "FASTA", inputs: []string{"", ">b\nCC\n", "\n", ">a\nGG\n", ""}, want: ">a\nGG\n>b\nCC\n"},
		{name: "FASTQ", inputs: []string{"", "@b\nCC\n+\nJJ\n", "\n", "@a\nGG\n+\nII\n", ""}, want: "@a\nGG\n+\nII\n@b\nCC\n+\nJJ\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			args := []string{"sort", "--quiet"}
			for i, input := range test.inputs {
				path := filepath.Join(dir, fmt.Sprintf("input%d.fx", i))
				if err := os.WriteFile(path, []byte(input), 0600); err != nil {
					t.Fatal(err)
				}
				args = append(args, path)
			}
			stdout, stderr, err := runCLI(t, "", args...)
			if test.mixed {
				if err == nil || !strings.Contains(stderr, "cannot mix FASTA and FASTQ") || strings.Contains(stderr, "panic:") {
					t.Fatalf("expected format error, got %v: %s", err, stderr)
				}
				if stdout != "" {
					t.Fatalf("mixed input produced output: %q", stdout)
				}
			} else if err != nil || stdout != test.want {
				t.Fatalf("sort output = %q, want %q, err = %v: %s", stdout, test.want, err, stderr)
			}
		})
	}
}

// TestSortCaseDistinctRecords preserves records whose IDs differ only in case.
func TestSortCaseDistinctRecords(t *testing.T) {
	input := ">r1\nACGT\n>R1\nGGGG\n>x\nTTTT\n"
	path := filepath.Join(t.TempDir(), "input.fa")
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"", "-l", "-s", "-n", "-N"} {
		for _, twoPass := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/twoPass%v", mode, twoPass), func(t *testing.T) {
				args := []string{"sort", "--quiet", "-i"}
				if mode != "" {
					args = append(args, mode)
				}
				if twoPass {
					args = append(args, "-2")
				}
				stdout, stderr, err := runCLI(t, "", append(args, path)...)
				want := input
				if mode == "-l" {
					want = ">R1\nGGGG\n>r1\nACGT\n>x\nTTTT\n"
				}
				if err != nil || stdout != want {
					t.Fatalf("sort output=%q, want %q, err=%v, stderr=%s", stdout, want, err, stderr)
				}
			})
		}
	}
}

// TestSortModeParity checks sequence keys, prefix ties, empty records, and reverse order.
func TestSortModeParity(t *testing.T) {
	for _, test := range []struct {
		name, input, want string
		args              []string
	}{
		{"sequence without ignore case", ">b\nCCCC\n>a\nAAAA\n", ">a\nAAAA\n>b\nCCCC\n", []string{"-s"}},
		{"sequence ignore case", ">x\nbbb\n>y\nAAA\n>z\nCCC\n", ">y\nAAA\n>x\nbbb\n>z\nCCC\n", []string{"-s", "-i"}},
		{"reverse sequence", ">x\nbbb\n>y\nAAA\n>z\nCCC\n", ">z\nCCC\n>x\nbbb\n>y\nAAA\n", []string{"-s", "-i", "-r"}},
		{"prefix ties", ">s1\nGGGGA\n>s2\nGGGGC\n>s3\nGGGGA\n>s4\nGGGGC\n", ">s1\nGGGGA\n>s2\nGGGGC\n>s3\nGGGGA\n>s4\nGGGGC\n", []string{"-s", "-L", "4"}},
		{"whole sequences", ">s1\nGGGGA\n>s2\nGGGGC\n>s3\nGGGGA\n>s4\nGGGGC\n", ">s1\nGGGGA\n>s3\nGGGGA\n>s2\nGGGGC\n>s4\nGGGGC\n", []string{"-s", "-L", "0"}},
		{"empty record by ID", ">e\n\n>a\nAC\n", ">a\nAC\n>e\n\n", nil},
		{"empty record by length", ">e\n\n>a\nAC\n", ">e\n\n>a\nAC\n", []string{"-l"}},
		{"empty record by sequence", ">e\n\n>a\nAC\n", ">e\n\n>a\nAC\n", []string{"-s"}},
		{"natural reverse ties", ">r2\nAA\n>R2\nCC\n>r10\nGG\n", ">r10\nGG\n>r2\nAA\n>R2\nCC\n", []string{"-i", "-N", "-r"}},
		{"natural numeric ties", ">r02\nAA\n>r2\nCC\n>r10\nGG\n", ">r02\nAA\n>r2\nCC\n>r10\nGG\n", []string{"-N"}},
		{"natural reverse numeric ties", ">r02\nAA\n>r2\nCC\n>r10\nGG\n", ">r10\nGG\n>r02\nAA\n>r2\nCC\n", []string{"-N", "-r"}},
		{"case-distinct full names", ">a same\nAA\n>a SAME\nCC\n", ">a same\nAA\n>a SAME\nCC\n", []string{"-n", "-i"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "input.fa")
			if err := os.WriteFile(path, []byte(test.input), 0600); err != nil {
				t.Fatal(err)
			}
			for _, twoPass := range []bool{false, true} {
				args := append([]string{"sort", "--quiet"}, test.args...)
				if twoPass {
					args = append(args, "-2")
				}
				stdout, stderr, err := runCLI(t, "", append(args, path)...)
				if err != nil || stdout != test.want {
					t.Errorf("twoPass=%v: output=%q, want %q, err=%v, stderr=%s", twoPass, stdout, test.want, err, stderr)
				}
			}
		})
	}
}

// TestSortDefaultSequencePrefix checks the default cutoff against whole-sequence sorting.
func TestSortDefaultSequencePrefix(t *testing.T) {
	common := strings.Repeat("G", 10000)
	input := ">s1\n" + common + "A\n>s2\n" + common + "C\n>s3\n" + common + "A\n>s4\n" + common + "C\n"
	path := filepath.Join(t.TempDir(), "input.fa")
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	for _, whole := range []bool{false, true} {
		for _, twoPass := range []bool{false, true} {
			args := []string{"sort", "--quiet", "-s"}
			if whole {
				args = append(args, "-L", "0")
			}
			if twoPass {
				args = append(args, "-2")
			}
			stdout, stderr, err := runCLI(t, "", append(args, path)...)
			if err != nil {
				t.Fatalf("whole=%v, twoPass=%v: %v: %s", whole, twoPass, err, stderr)
			}
			var headers []string
			for _, line := range strings.Split(stdout, "\n") {
				if strings.HasPrefix(line, ">") {
					headers = append(headers, line)
				}
			}
			want := ">s1,>s2,>s3,>s4"
			if whole {
				want = ">s1,>s3,>s2,>s4"
			}
			if got := strings.Join(headers, ","); got != want {
				t.Errorf("whole=%v, twoPass=%v: headers=%s, want %s", whole, twoPass, got, want)
			}
		}
	}
}

// TestSortFASTQCaseDistinctRecords checks that sequence and quality pairs remain intact.
func TestSortFASTQCaseDistinctRecords(t *testing.T) {
	input := "@r1\nACGT\n+\nIIII\n@R1\nGGGG\n+\nJJJJ\n@x\nTTTT\n+\nKKKK\n"
	for _, args := range [][]string{{"sort", "--quiet", "-i"}, {"sort", "--quiet", "-s", "-i"}} {
		stdout, stderr, err := runCLI(t, input, args...)
		if err != nil || stdout != input {
			t.Fatalf("%v: output=%q, want %q, err=%v, stderr=%s", args, stdout, input, err, stderr)
		}
	}
}

// TestSortDuplicateIDs retains rejection of exact duplicate IDs with different headers.
func TestSortDuplicateIDs(t *testing.T) {
	input := ">same first\nAC\n>same second\nGT\n"
	path := filepath.Join(t.TempDir(), "input.fa")
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	for _, twoPass := range []bool{false, true} {
		args := []string{"sort", "--quiet", "-i"}
		if twoPass {
			args = append(args, "-2")
		}
		_, stderr, err := runCLI(t, "", append(args, path)...)
		if err == nil || !strings.Contains(stderr, "duplicated sequences found: same") || strings.Contains(stderr, "panic:") {
			t.Fatalf("twoPass=%v: expected duplicate-ID error, got %v: %s", twoPass, err, stderr)
		}
	}
}

// TestQualityErrorsCLI checks malformed qualities without panics or partial rows.
func TestQualityErrorsCLI(t *testing.T) {
	for _, args := range [][]string{
		{"seq", "-b", "64", "-Q", "20"},
		{"seq", "-b", "64", "-R", "20"},
		{"fx2tab", "-b", "64", "-q"},
		{"stats", "-a", "-E", "illumina-1.5+"},
		{"stats", "-Ta", "-E", "illumina-1.5+"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			stdout, stderr, err := runCLI(t, "@r1\nACGT\n+\nIII!\n", args...)
			if err == nil || strings.Contains(stderr, "panic:") || !strings.Contains(stderr, "r1") || !strings.Contains(stderr, "'!'") || !strings.Contains(stderr, "ASCII base 64") {
				t.Fatalf("expected a descriptive quality error, got %v: %s", err, stderr)
			}
			if args[0] != "stats" && stdout != "" {
				t.Fatalf("invalid record produced partial output: %q", stdout)
			}
		})
	}
	// A long row exposes writes that would otherwise remain buffered on error.
	length := 128 << 10
	input := "@long\n" + strings.Repeat("A", length) + "\n+\n" + strings.Repeat("I", length-1) + "!\n"
	stdout, stderr, err := runCLI(t, input, "fx2tab", "-b", "64", "-q")
	if err == nil || strings.Contains(stderr, "panic:") || stdout != "" {
		t.Fatalf("invalid long record: stdout length=%d, err=%v, stderr=%s", len(stdout), err, stderr)
	}
}

// TestQualityCalculationsCLI preserves valid Phred+33 and Phred+64 results.
func TestQualityCalculationsCLI(t *testing.T) {
	for _, test := range []struct {
		base, encoding, quality string
	}{
		{"33", "sanger", "IIII"},
		{"64", "illumina-1.5+", "hhhh"},
	} {
		t.Run(test.base, func(t *testing.T) {
			input := "@r1\nACGT\n+\n" + test.quality + "\n"
			stdout, stderr, err := runCLI(t, input, "seq", "-b", test.base, "-Q", "20")
			if err != nil || stdout != input {
				t.Fatalf("seq: stdout=%q, err=%v, stderr=%s", stdout, err, stderr)
			}
			stdout, stderr, err = runCLI(t, input, "fx2tab", "-b", test.base, "-q")
			if err != nil || stdout != "r1\tACGT\t"+test.quality+"\t40.00\n" {
				t.Fatalf("fx2tab: stdout=%q, err=%v, stderr=%s", stdout, err, stderr)
			}
			stdout, stderr, err = runCLI(t, input, "stats", "-Ta", "-E", test.encoding)
			lines := strings.Split(strings.TrimSpace(stdout), "\n")
			if err != nil || len(lines) != 2 {
				t.Fatalf("stats: stdout=%q, err=%v, stderr=%s", stdout, err, stderr)
			}
			fields := strings.Split(lines[1], "\t")
			if len(fields) != 19 || fields[14] != "100" || fields[15] != "100" || fields[16] != "40.00" {
				t.Fatalf("unexpected quality metrics: %q", lines[1])
			}
		})
	}
}

// TestStatsSkipsInvalidQuality preserves the explicit skip-error mode.
func TestStatsSkipsInvalidQuality(t *testing.T) {
	stdout, stderr, err := runCLI(t, "@r1\nACGT\n+\nIII!\n", "stats", "-Ta", "-e", "-E", "illumina-1.5+")
	if err != nil || strings.Contains(stderr, "panic:") || !strings.Contains(stderr, "ASCII base 64") || strings.Count(stdout, "\n") != 1 {
		t.Fatalf("stats --skip-err: stdout=%q, err=%v, stderr=%s", stdout, err, stderr)
	}
}

// TestStatsReadErrors checks ordered valid rows and exit status across output modes.
func TestStatsReadErrors(t *testing.T) {
	dir := t.TempDir()
	files := []string{filepath.Join(dir, "a.fa"), filepath.Join(dir, "bad.fq"), filepath.Join(dir, "p.fa")}
	for i, input := range []string{">a\nACGT\n", "@r1\nACGT\n+\nIII\n", ">p\nMEL\n"} {
		if err := os.WriteFile(files[i], []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, threads := range []string{"1", "4"} {
		for _, tabular := range []bool{false, true} {
			for _, skipErr := range []bool{false, true} {
				t.Run(fmt.Sprintf("threads%s/tabular%v/skip%v", threads, tabular, skipErr), func(t *testing.T) {
					args := []string{"stats", "-j", threads}
					if tabular {
						args = append(args, "-T")
					}
					if skipErr {
						args = append(args, "-e")
					}
					stdout, stderr, err := runCLI(t, "", append(args, files...)...)
					if (err == nil) != skipErr || strings.Contains(stderr, "panic:") || !strings.Contains(stderr, "unequal sequence and quality") || strings.Count(stderr, files[1]) != 1 {
						t.Errorf("unexpected failure status or diagnostic: err=%v, stderr=%s", err, stderr)
					}
					lines := strings.Split(strings.TrimSpace(stdout), "\n")
					if len(lines) != 3 {
						t.Fatalf("expected header and both valid rows, got %q", stdout)
					}
					averages := []string{"4", "3"}
					if tabular {
						averages = []string{"4.0", "3.0"}
					}
					for i, want := range [][]string{
						{files[0], "FASTA", "DNA", "1", "4", "4", averages[0], "4"},
						{files[2], "FASTA", "Protein", "1", "3", "3", averages[1], "3"},
					} {
						if got := strings.Join(strings.Fields(lines[i+1]), "\t"); got != strings.Join(want, "\t") {
							t.Errorf("row %d = %q, want %q", i, got, strings.Join(want, "\t"))
						}
					}
				})
			}
		}
	}
}

// TestStatsErrorOutputFiles checks draining failures and closing buffered/compressed output.
func TestStatsErrorOutputFiles(t *testing.T) {
	dir := t.TempDir()
	for name, input := range map[string]string{
		"format.fx":  "not FASTA\n",
		"quality.fq": "@bad\nA\n+\n!\n",
		"length.fq":  "@bad\nAC\n+\nI\n",
		"a.fa":       ">a\nACGT\n",
		"empty.fa":   "",
		"p.fa":       ">p\nMEL\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
	}
	inputs := []string{"format.fx", "quality.fq", "length.fq", "missing.fa", "a.fa", "empty.fa", "p.fa"}
	paths := make([]string, len(inputs))
	for i, name := range inputs {
		paths[i] = filepath.Join(dir, name)
	}
	list := filepath.Join(dir, "inputs.txt")
	if err := os.WriteFile(list, []byte(strings.Join(paths, "\n")+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, compressed := range []bool{false, true} {
		for _, skipErr := range []bool{false, true} {
			t.Run(fmt.Sprintf("compressed%v/skip%v", compressed, skipErr), func(t *testing.T) {
				out := filepath.Join(t.TempDir(), "stats.tsv")
				if compressed {
					out += ".gz"
				}
				args := []string{"stats", "-Ta", "-b", "-j", "2", "-S", "-E", "illumina-1.5+", "-o", out, "--infile-list", list}
				if skipErr {
					args = append(args, "-e")
				}
				_, stderr, err := runCLI(t, "", args...)
				if (err == nil) != skipErr || strings.Contains(stderr, "panic:") {
					t.Fatalf("unexpected failure status: err=%v, stderr=%s", err, stderr)
				}
				previous := -1
				for _, name := range inputs[:4] {
					position := strings.Index(stderr, filepath.Join(dir, name))
					if position <= previous {
						t.Fatalf("missing or unordered diagnostic for %s: %s", name, stderr)
					}
					previous = position
				}
				data, err := os.ReadFile(out)
				if err != nil {
					t.Fatal(err)
				}
				if compressed {
					reader, err := gzip.NewReader(bytes.NewReader(data))
					if err != nil {
						t.Fatal(err)
					}
					defer reader.Close()
					data, err = io.ReadAll(reader)
					if err != nil {
						t.Fatal(err)
					}
				}
				lines := strings.Split(strings.TrimSpace(string(data)), "\n")
				if len(lines) != 4 {
					t.Fatalf("expected header and all three valid rows: %q", data)
				}
				for i, name := range inputs[4:] {
					fields := strings.Split(lines[i+1], "\t")
					if len(fields) != 19 || fields[0] != name {
						t.Errorf("unexpected row %d: %q", i, lines[i+1])
					}
				}
			})
		}
	}
}

// TestSubseqGTFAttributes checks attribute comments through indexed and streaming input.
func TestSubseqGTFAttributes(t *testing.T) {
	for _, test := range []struct {
		name, attributes, tag, comment string
	}{
		{"last with semicolon", `gene_id "g1"; gene_name "A";`, "gene_name", "A"},
		{"last without semicolon", `gene_name "A"; gene_id "g1"`, "gene_id", "g1"},
		{"single attribute", `gene_id "g1";`, "gene_id", "g1"},
		{"missing value", `gene_id; gene_name "A";`, "gene_id", ""},
		{"valid after missing value", `gene_id; gene_name "A";`, "gene_name", "A"},
		{"quoted semicolon", `gene_name "alpha; beta";`, "gene_name", "alpha; beta"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			gtfFile := filepath.Join(dir, "input.gtf")
			gtfInput := "chr1\tsrc\tgene\t1\t4\t.\t+\t.\t" + test.attributes + "\n"
			if err := os.WriteFile(gtfFile, []byte(gtfInput), 0600); err != nil {
				t.Fatal(err)
			}
			fasta := filepath.Join(dir, "input.fa")
			input := ">chr1\nACGTACGTAC\n"
			if err := os.WriteFile(fasta, []byte(input), 0600); err != nil {
				t.Fatal(err)
			}
			for _, indexed := range []bool{false, true} {
				args := []string{"subseq", "--quiet", "--gtf", gtfFile, "--gtf-tag", test.tag}
				if indexed {
					args = append(args, fasta)
				}
				stdout, stderr, err := runCLI(t, input, args...)
				want := ">chr1_1-4:+ " + test.comment + "\nACGT\n"
				if err != nil || stdout != want {
					t.Fatalf("indexed=%v: stdout=%q, want %q, err=%v, stderr=%s", indexed, stdout, want, err, stderr)
				}
			}
		})
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

func TestShortMismatchMatcher(t *testing.T) {
	rng := rand.New(rand.NewSource(54821))
	alphabet := []byte("ACGTNRYacgt")
	for _, length := range []int{1, 2, 3, 4, 15, 31, 32, 63, 64} {
		for errors := 0; errors <= 4 && errors <= length; errors++ {
			for trial := 0; trial < 50; trial++ {
				pattern := make([]byte, length)
				target := make([]byte, length+40)
				for i := range pattern {
					pattern[i] = alphabet[rng.Intn(len(alphabet))]
				}
				for i := range target {
					target[i] = alphabet[rng.Intn(len(alphabet))]
				}
				if trial%2 == 0 {
					start := rng.Intn(41)
					copy(target[start:], pattern)
					for i := 0; i < errors && i < length; i++ {
						target[start+i] = 'Z'
					}
				}
				index := fmi.NewFMIndex()
				if _, err := index.Transform(target); err != nil {
					t.Fatal(err)
				}
				want, err := index.Match(pattern, errors)
				if err != nil {
					t.Fatal(err)
				}
				matcher := newShortMismatchMatcher(pattern, errors)
				if got := matcher.match(target); got != want {
					t.Fatalf("pattern %q, target %q, mismatches %d: got %v, want %v", pattern, target, errors, got, want)
				}
				if matcher.match(target[:length-1]) || matcher.match(nil) {
					t.Fatal("matched a target shorter than the pattern")
				}
			}
		}
	}
	for _, input := range []struct {
		pattern []byte
		errors  int
	}{
		{nil, 1}, {bytes.Repeat([]byte("A"), 65), 1}, {[]byte("ACGTACGT"), 5},
	} {
		if newShortMismatchMatcher(input.pattern, input.errors) != nil {
			t.Fatal("unsupported pattern should use the FM-index fallback")
		}
	}
}
