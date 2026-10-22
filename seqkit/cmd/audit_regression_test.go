package cmd

import (
	"bytes"
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

// auditInput writes a disposable CLI fixture, including any parent directories.
func auditInput(t *testing.T, dir, name, data string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// auditCLI runs a CLI helper in an isolated directory with a bounded lifetime.
func auditCLI(t *testing.T, dir, input string, args ...string) (string, string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)...)
	cmd.Env = append(os.Environ(), "SEQKIT_CLI_TEST_HELPER=1")
	cmd.Dir, cmd.Stdin = dir, strings.NewReader(input)
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("CLI did not terminate: %v", args)
	}
	return out.String(), stderr.String(), err
}

// TestAuditTranslateProtectsInputs verifies rejection precedes truncation.
func TestAuditTranslateProtectsInputs(t *testing.T) {
	for _, alias := range []string{"same", "symlink", "hardlink"} {
		t.Run(alias, func(t *testing.T) {
			dir := t.TempDir()
			const data = ">a\nATGGCC\n"
			in := auditInput(t, dir, "in.fa", data)
			out := in
			if alias != "same" {
				out = filepath.Join(dir, "alias.fa")
				var err error
				if alias == "symlink" {
					err = os.Symlink(in, out)
				} else {
					err = os.Link(in, out)
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			_, stderr, err := runCLI(t, "", "translate", in, "-o", out)
			if err == nil || !strings.Contains(stderr, "cannot be the same") {
				t.Fatalf("expected collision: %v %s", err, stderr)
			}
			got, err := os.ReadFile(in)
			if err != nil || string(got) != data {
				t.Fatalf("input changed: %q %v", got, err)
			}
		})
	}
}

// TestAuditTempFilesProtectExisting exercises every two-pass split branch too.
func TestAuditTempFilesProtectExisting(t *testing.T) {
	for _, args := range [][]string{{"sort", "-2"}, {"shuffle", "-2"}, {"split", "-2", "-s", "1"}, {"split", "-2", "-p", "1"}, {"split", "-2", "-i"}, {"split", "-2", "-r", "1:2"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			dir := t.TempDir()
			path := auditInput(t, dir, "stdin.fastx", "keep this file\n")
			_, stderr, err := auditCLI(t, dir, ">a\nACGT\n", append(args, "--quiet")...)
			if err != nil {
				t.Fatalf("%v: %s", err, stderr)
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != "keep this file\n" {
				t.Fatalf("existing file changed: %q %v", got, err)
			}
			leftovers, err := filepath.Glob(filepath.Join(dir, "seqkit-*.fastx*"))
			if err != nil || len(leftovers) != 0 {
				t.Fatalf("temporary files remain: %v %v", leftovers, err)
			}
		})
	}
}

// TestAuditGeneratedOutputs checks generated paths before writers or force cleanup.
func TestAuditGeneratedOutputs(t *testing.T) {
	t.Run("rename input", func(t *testing.T) {
		dir := t.TempDir()
		data := strings.Repeat(">a\nACGT\n", 20000)
		in := auditInput(t, dir, "in.fa", data)
		_, stderr, err := auditCLI(t, dir, "", "rename", "-m", "-O", ".", in)
		if err == nil {
			t.Fatalf("expected collision: %s", stderr)
		}
		got, err := os.ReadFile(in)
		if err != nil || string(got) != data {
			t.Fatalf("input changed: %v", err)
		}
	})
	for _, command := range []string{"pair", "rename"} {
		t.Run(command+" basename", func(t *testing.T) {
			dir := t.TempDir()
			left := auditInput(t, dir, "left/reads.fq", "@a\nAAAA\n+\nIIII\n")
			right := auditInput(t, dir, "right/reads.fq", "@a\nCCCC\n+\nJJJJ\n")
			out := filepath.Join(dir, "out")
			args := []string{"pair", "-1", left, "-2", right, "-O", out}
			if command == "rename" {
				args = []string{"rename", "-m", "-O", out, left, right}
			}
			_, stderr, err := runCLI(t, "", args...)
			if err == nil {
				t.Fatalf("expected output collision: %s", stderr)
			}
		})
	}
}

// TestAuditRmdupOutputAliases includes paths that do not exist yet.
func TestAuditRmdupOutputAliases(t *testing.T) {
	for _, flags := range [][2]string{{"-o", "-d"}, {"-o", "-D"}, {"-d", "-D"}} {
		t.Run(strings.Join(flags[:], " "), func(t *testing.T) {
			dir := t.TempDir()
			out := filepath.Join(dir, "out.fa")
			args := []string{"rmdup", flags[0], out, flags[1], filepath.Join(dir, ".", "out.fa")}
			_, stderr, err := runCLI(t, ">a\nAC\n>a\nGT\n", args...)
			if err == nil || !strings.Contains(stderr, "cannot be the same") {
				t.Fatalf("expected collision: %v %s", err, stderr)
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatalf("output opened before validation: %v", err)
			}
		})
	}
}

// TestAuditShuffleDuplicateHeaders verifies shuffling preserves whole records.
func TestAuditShuffleDuplicateHeaders(t *testing.T) {
	for _, input := range []string{">x\nAAAA\n>x\nCCCC\n", "@x\nAAAA\n+\n1234\n@x\nCCCC\n+\n5678\n"} {
		out, stderr, err := runCLI(t, input, "shuffle", "--quiet", "-w", "0")
		if err != nil || strings.Count(out, "AAAA") != 1 || strings.Count(out, "CCCC") != 1 {
			t.Fatalf("record loss: %q %v %s", out, err, stderr)
		}
		if strings.Contains(input, "1234") && (!strings.Contains(out, "AAAA\n+\n1234") || !strings.Contains(out, "CCCC\n+\n5678")) {
			t.Fatalf("qualities changed: %q", out)
		}
	}
}

// TestAuditIndexedDuplicates rejects lossy indexes, including reused ones.
func TestAuditIndexedDuplicates(t *testing.T) {
	dir := t.TempDir()
	in := auditInput(t, dir, "in.fa", ">x\nAAAA\n>x\nCCCC\n")
	for attempt := 0; attempt < 2; attempt++ {
		for _, args := range [][]string{{"sort", "-2"}, {"shuffle", "-2"}, {"split", "-2", "-s", "1", "-O", filepath.Join(dir, "out")}} {
			out, stderr, err := runCLI(t, "", append(args, in)...)
			if err == nil || out != "" || !strings.Contains(stderr, "duplicat") {
				t.Fatalf("duplicate accepted: %v %q %s", err, out, stderr)
			}
		}
	}
}

// TestAuditConvertBoundaries covers exact sample sizes and independent files.
func TestAuditConvertBoundaries(t *testing.T) {
	for _, n := range []int{1, 2, 3} {
		input := strings.Repeat("@r\nAC\n+\n!!\n", n)
		for _, sample := range []int{1, 2, 3} {
			out, stderr, err := runCLI(t, input, "convert", "--quiet", "-n", fmt.Sprint(sample))
			if err != nil || out != input {
				t.Fatalf("records=%d sample=%d: %q %v %s", n, sample, out, err, stderr)
			}
		}
	}
	dir := t.TempDir()
	a := auditInput(t, dir, "a.fq", "@a\nAC\n+\n!!\n")
	b := auditInput(t, dir, "b.fq", "@b\nAC\n+\n!!\n")
	out, stderr, err := runCLI(t, "", "convert", "--quiet", a, b)
	if err != nil || out != "@a\nAC\n+\n!!\n@b\nAC\n+\n!!\n" {
		t.Fatalf("multiple files: %q %v %s", out, err, stderr)
	}
}

// TestAuditReplaceQualityMapping uses distinct qualities to reveal wrong positions.
func TestAuditReplaceQualityMapping(t *testing.T) {
	for _, tt := range []struct{ pattern, replacement, seq, qual string }{
		{"A(?P<b>C)", "$b", "CGT", "234"},
		{"A(?P<b>C)", "${b}", "CGT", "234"},
		{"(A)(C)", "$2$1", "CAGT", "2134"},
		{"(A)C", "$1$1", "AAGT", "1134"},
		{"A(C)", "$1T", "GT", "34"},
		{"AC", "T", "TGT", "134"},
		{"AC", "$$", "$GT", "134"},
		{"(A)?C", "$1", "AGT", "134"},
	} {
		t.Run(tt.pattern+"/"+tt.replacement, func(t *testing.T) {
			out, stderr, err := runCLI(t, "@r\nACGT\n+\n1234\n", "replace", "-s", "-p", tt.pattern, "-r", tt.replacement)
			want := "@r\n" + tt.seq + "\n+\n" + tt.qual + "\n"
			if err != nil || out != want {
				t.Fatalf("got %q want %q: %v %s", out, want, err, stderr)
			}
		})
	}
}

// TestAuditSumConcurrency drives the real worker and progress-channel lifecycle.
func TestAuditSumConcurrency(t *testing.T) {
	dir := t.TempDir()
	in := auditInput(t, dir, "in.fa", ">a\nACGT\n")
	for _, quiet := range []bool{false, true} {
		args := []string{"sum", "-j", "4"}
		if quiet {
			args = append(args, "--quiet")
		}
		for i := 0; i < 16; i++ {
			args = append(args, in)
		}
		out, stderr, err := runCLI(t, "", args...)
		if err != nil || strings.Count(out, "\n") != 16 {
			t.Fatalf("sum failed: %v %q %s", err, out, stderr)
		}
	}
}

// TestAuditSumNormalizedLength rejects short circular sequences without panics.
func TestAuditSumNormalizedLength(t *testing.T) {
	for _, input := range []string{">g\nA---\n", ">g\n----\n"} {
		_, stderr, err := runCLI(t, input, "sum", "--quiet", "-c", "-g", "-k", "3")
		if err == nil || strings.Contains(stderr, "panic:") || !strings.Contains(stderr, "too big") {
			t.Fatalf("bad normalized-length error: %s", stderr)
		}
	}
}

// TestAuditLocateEmptyMatches requires finite output on either strand.
func TestAuditLocateEmptyMatches(t *testing.T) {
	for _, pattern := range []string{"A*", "^", "$", "C?"} {
		out, stderr, err := auditCLI(t, t.TempDir(), ">r\nCCCC\n", "locate", "-r", "-G", "-p", pattern)
		wantLines := 1
		if pattern == "C?" {
			wantLines = 5
		}
		if err != nil || strings.Count(out, "\n") != wantLines {
			t.Fatalf("empty match failed: %q %v %s", out, err, stderr)
		}
	}
}

// TestAuditSlidingOversized rejects undefined multi-wrap windows.
func TestAuditSlidingOversized(t *testing.T) {
	for _, input := range []string{">a\nACGT\n", "@a\nACGT\n+\n1234\n"} {
		out, stderr, err := runCLI(t, input, "sliding", "-c", "-s", "1", "-W", "10")
		if err == nil || out != "" || strings.Contains(stderr, "panic:") {
			t.Fatalf("oversized window accepted: %q %v %s", out, err, stderr)
		}
	}
}

// TestAuditMergeSlidesErrors covers empty inputs and scanner failures.
func TestAuditMergeSlidesErrors(t *testing.T) {
	for _, input := range []string{"", "# test\n"} {
		out, stderr, err := runCLI(t, input, "merge-slides", "-b", "1K")
		if err != nil || out != "" {
			t.Fatalf("empty input: %q %v %s", out, err, stderr)
		}
	}
	_, stderr, err := runCLI(t, "ref_sliding:1-10\nref_sliding:1000000000-1000000010\n", "merge-slides", "-b", "20")
	if err == nil || !strings.Contains(stderr, "token too long") {
		t.Fatalf("scanner failure lost: %v %s", err, stderr)
	}
	for _, coordinates := range []string{"999999999999999999999999-10", "1-999999999999999999999999"} {
		_, stderr, err = runCLI(t, "ref_sliding:"+coordinates+"\n", "merge-slides", "-b", "1K")
		if err == nil || !strings.Contains(stderr, "invalid") {
			t.Fatalf("coordinate overflow accepted (%s): %v %s", coordinates, err, stderr)
		}
	}
}

// TestAuditCoordinateOverflows rejects coordinates that do not fit in int.
func TestAuditCoordinateOverflows(t *testing.T) {
	const tooLarge = "999999999999999999999999"
	input := filepath.Join(t.TempDir(), "input.fa")
	if err := os.WriteFile(input, []byte(">chr1\nACGT\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var stderr string
	var err error
	for _, region := range []string{"chr1:" + tooLarge, "chr1:1-" + tooLarge, "chr1:" + tooLarge + "-"} {
		_, stderr, err = runCLI(t, "", "faidx", input, region)
		if err == nil || !strings.Contains(stderr, "invalid region") {
			t.Fatalf("faidx coordinate overflow accepted (%s): %v %s", region, err, stderr)
		}
	}
	for _, test := range []struct{ flag, value string }{
		{"-p", tooLarge + ":A"},
		{"-d", tooLarge + ":1"},
		{"-d", "1:" + tooLarge},
		{"-i", tooLarge + ":A"},
	} {
		args := []string{"mutate", test.flag, test.value}
		_, stderr, err = runCLI(t, ">a\nACGT\n", args...)
		if err == nil || !strings.Contains(stderr, "invalid") {
			t.Fatalf("mutate %s coordinate overflow accepted: %v %s", test.flag, err, stderr)
		}
	}
}

// TestScatSequenceQueueBoundsPendingRecordsAcrossDirectories.
func TestScatSequenceQueueBoundsPendingRecordsAcrossDirectories(t *testing.T) {
	channels := newWatchSequenceChannels(10)
	if len(channels) != 10 {
		t.Fatalf("got %d channels, want 10", len(channels))
	}
	for _, channel := range channels {
		if channel != channels[0] {
			t.Fatal("watched directories do not share the bounded sequence queue")
		}
	}
	if got := cap(channels[0]); got != watchSequenceQueueSize {
		t.Fatalf("queue capacity = %d, want %d", got, watchSequenceQueueSize)
	}
}

// TestAppendAmpliconMismatchName preserves names while reusing its byte buffer.
func TestAppendAmpliconMismatchName(t *testing.T) {
	name := []byte("read/1")
	got := appendAmpliconMismatchName(name, name, 3, 1, 2)
	if string(got) != "read/1 mismatches=3(1+2)" {
		t.Fatalf("formatted name = %q", got)
	}
	got = appendAmpliconMismatchName(got, name, 0, 0, 0)
	if string(got) != "read/1 mismatches=0(0+0)" {
		t.Fatalf("reused formatted name = %q", got)
	}
}

// TestAmpliconMismatchAnnotation preserves CLI mismatch labels in both search paths.
func TestAmpliconMismatchAnnotation(t *testing.T) {
	tests := []struct {
		name, sequence, mismatches string
		maxMismatch                string
	}{
		{"exact", "acgcccactgaaatga", "mismatches=0(0+0)", "0"},
		{"primer mismatches", "acgcacactgaaatga", "mismatches=2(1+1)", "1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := ">seq\n" + test.sequence + "\n"
			out, stderr, err := runCLI(t, input, "amplicon", "-F", "ccc", "-R", "ttt", "-P", "-M", "-m", test.maxMismatch)
			if err != nil || !strings.Contains(out, test.mismatches) {
				t.Fatalf("output=%q, err=%v, stderr=%s", out, err, stderr)
			}
		})
	}
}

// TestAmpliconWorkerState preserves formatting and order across concurrent input reads.
func TestAmpliconWorkerState(t *testing.T) {
	const sequence = "acgcacactgaaatga"
	const extracted = "cgcacactgaaat"
	dir := t.TempDir()
	fasta := auditInput(t, dir, "in.fa", strings.Repeat(">seq\n"+sequence+"\n", 64))
	fastq := auditInput(t, dir, "in.fq", strings.Repeat("@seq\n"+sequence+"\n+\n"+strings.Repeat("I", len(sequence))+"\n", 64))
	want := strings.Repeat(">seq mismatches=2(1+1)\ncgca\ncact\ngaaa\nt\n", 64) +
		strings.Repeat("@seq mismatches=2(1+1)\n"+extracted+"\n+\n"+strings.Repeat("I", len(extracted))+"\n", 64)
	for _, threads := range []string{"1", "4"} {
		out, stderr, err := runCLI(t, "", "amplicon", "-F", "ccc", "-R", "ttt", "-P", "-M", "-m", "1", "-w", "4", "-j", threads, fasta, fastq)
		if err != nil || out != want {
			t.Fatalf("threads=%s: err=%v stderr=%s output=%q", threads, err, stderr, out)
		}
	}
}

// TestTranslateOutputHeaders preserves FASTA headers in whole and split outputs.
func TestTranslateOutputHeaders(t *testing.T) {
	input := ">r desc\nATGTAAATG\n"
	out, stderr, err := runCLI(t, input, "translate", "-f", "1")
	if err != nil || out != ">r desc\nM*M\n" {
		t.Fatalf("whole translation = %q, err=%v, stderr=%s", out, err, stderr)
	}
	out, stderr, err = runCLI(t, input, "translate", "-f", "1", "-s")
	want := ">r_frame=1_begin=1_end=3 desc\nM\n>r_frame=1_begin=7_end=9 desc\nM\n"
	if err != nil || out != want {
		t.Fatalf("split translation = %q, want %q, err=%v, stderr=%s", out, want, err, stderr)
	}
}

// TestAuditRenameGeneratedIDs checks both orders of original/generated collisions.
func TestAuditRenameGeneratedIDs(t *testing.T) {
	for _, input := range []string{">a\nAC\n>a\nGT\n>a_2\nTT\n", ">a_2\nTT\n>a\nAC\n>a\nGT\n"} {
		out, stderr, err := runCLI(t, input, "rename")
		if err != nil {
			t.Fatalf("%v %s", err, stderr)
		}
		seen := make(map[string]bool)
		for _, line := range strings.Split(out, "\n") {
			if !strings.HasPrefix(line, ">") {
				continue
			}
			if seen[line] {
				t.Fatalf("duplicate output ID: %q", out)
			}
			seen[line] = true
		}
		if len(seen) != 3 {
			t.Fatalf("record loss: %q", out)
		}
	}
}

// TestAuditOutputErrors checks buffered finalization failures through real commands.
func TestAuditOutputErrors(t *testing.T) {
	if _, err := os.Stat("/dev/full"); err != nil {
		t.Skip("/dev/full is unavailable on this platform")
	}
	for _, command := range []string{"fx2tab", "rename", "sort", "shuffle", "rmdup", "fq2fa", "translate", "sana"} {
		t.Run(command, func(t *testing.T) {
			input := "@r\nATGGCC\n+\nIIIIII\n"
			_, stderr, err := runCLI(t, input, command, "--quiet", "-o", "/dev/full")
			if err == nil || !strings.Contains(stderr, "no space left") {
				t.Fatalf("output failure lost: %v %s", err, stderr)
			}
		})
	}
}

// TestAuditSplit2PairedForce checks shared directory and FASTQ worker lifecycles.
func TestAuditSplit2PairedForce(t *testing.T) {
	dir := t.TempDir()
	leftData := strings.Repeat("@a\nAAAA\n+\n1234\n", 20)
	rightData := strings.Repeat("@a\nCCCC\n+\n5678\n", 20)
	left := auditInput(t, dir, "left.fq", leftData)
	right := auditInput(t, dir, "right.fq", rightData)
	outDir := filepath.Join(dir, "out")
	auditInput(t, outDir, "old.txt", "old output\n")
	_, stderr, err := runCLI(t, "", "split2", "-s", "1", "-1", left, "-2", right, "-O", outDir, "-f", "--quiet")
	if err != nil {
		t.Fatalf("split2 failed: %v %s", err, stderr)
	}
	for _, tt := range []struct{ prefix, seq, qual string }{{"left", "AAAA", "1234"}, {"right", "CCCC", "5678"}} {
		paths, err := filepath.Glob(filepath.Join(outDir, tt.prefix+".*.fq"))
		if err != nil || len(paths) != 20 {
			t.Fatalf("lost %s parts: %d %v", tt.prefix, len(paths), err)
		}
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil || string(data) != "@a\n"+tt.seq+"\n+\n"+tt.qual+"\n" {
				t.Fatalf("corrupt part: %s %q %v", path, data, err)
			}
		}
	}
}

// TestAuditSplitRegionsConcurrent checks independent region-worker output state.
func TestAuditSplitRegionsConcurrent(t *testing.T) {
	dir := t.TempDir()
	var input strings.Builder
	for i, region := range []string{"AA", "AC", "AG", "AT", "CA", "CC", "CG", "CT"} {
		fmt.Fprintf(&input, ">r%d\n%sGT\n", i, region)
	}
	in := auditInput(t, dir, "in.fa", input.String())
	outDir := filepath.Join(dir, "out")
	_, stderr, err := runCLI(t, "", "split", "-2", "-r", "1:2", "-j", "4", "-O", outDir, in)
	if err != nil {
		t.Fatalf("split regions: %v %s", err, stderr)
	}
	paths, err := filepath.Glob(filepath.Join(outDir, "*.fa"))
	if err != nil || len(paths) != 8 {
		t.Fatalf("lost regions: %d %v", len(paths), err)
	}
}

// TestAuditRangeBeforeStart checks an interval entirely before a short input.
func TestAuditRangeBeforeStart(t *testing.T) {
	out, stderr, err := runCLI(t, ">a\nACGT\n", "range", "-r=-3:-2")
	if err != nil || out != "" {
		t.Fatalf("out-of-range records: %q %v %s", out, err, stderr)
	}
}

// TestAuditFASTQFormatting preserves explicit wrapping and empty FASTQ records.
func TestAuditFASTQFormatting(t *testing.T) {
	out, stderr, err := runCLI(t, "@r\nACGT\n+\n1234\n", "convert", "--from", "Sanger", "-w", "2")
	if err != nil || out != "@r\nAC\nGT\n+\n12\n34\n" {
		t.Fatalf("wrapped conversion: %q %v %s", out, err, stderr)
	}
	out, stderr, err = runCLI(t, "@r\n\n+\n\n", "convert", "--from", "Sanger")
	if err != nil || out != "@r\n\n+\n\n" {
		t.Fatalf("empty conversion: %q %v %s", out, err, stderr)
	}
	for _, tt := range []struct {
		input, want string
		width       string
	}{
		{"@r\nACGT\n+\n1234\n", "@r\nAC\nGT\n+\n12\n34\n", "2"},
		{"@r\n\n+\n\n", "@r\n\n+\n\n", "0"},
	} {
		dir := t.TempDir()
		in := auditInput(t, dir, "in.fq", tt.input)
		outDir := filepath.Join(dir, "out")
		_, stderr, err := runCLI(t, "", "split2", "-s", "1", "-w", tt.width, "-O", outDir, in)
		if err != nil {
			t.Fatalf("split2 format: %v %s", err, stderr)
		}
		paths, err := filepath.Glob(filepath.Join(outDir, "*.fq"))
		if err != nil || len(paths) != 1 {
			t.Fatalf("missing split part: %v %v", paths, err)
		}
		data, err := os.ReadFile(paths[0])
		if err != nil || string(data) != tt.want {
			t.Fatalf("split format: %q want %q: %v", data, tt.want, err)
		}
	}
}

// TestAuditFASTAIndexValidation retains empty/BOM/CRLF and unterminated input support.
func TestAuditFASTAIndexValidation(t *testing.T) {
	for i, data := range []string{"", "\n\n", ">", ">a\n", ">a\nACGT", "\n\n>a\nAC\nGT\n>b\n\n", "\xef\xbb\xbf>a\r\nAC\r\nGT\r\n>b\r\nA\r\n"} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			dir := t.TempDir()
			in := auditInput(t, dir, "in.fa", data)
			index, err := fai.CreateWithFullHead(in, in+".fai")
			if err != nil {
				t.Fatal(err)
			}
			if err := validateFASTAIndex(in, index); err != nil {
				t.Fatal(err)
			}
		})
	}
	dir := t.TempDir()
	in := auditInput(t, dir, "in.fa", ">a\nACGT\n")
	index, err := fai.CreateWithFullHead(in, in+".fai")
	if err != nil {
		t.Fatal(err)
	}
	r := index["a"]
	r.Length += 100
	index["a"] = r
	if err := validateFASTAIndex(in, index); err == nil {
		t.Fatal("accepted index extending past EOF")
	}
}

// TestAuditOutputAliasResolution covers dangling symlinks and hard-linked outputs.
func TestAuditOutputAliasResolution(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "new.fa")
	link := filepath.Join(dir, "alias.fa")
	if err := os.Symlink("new.fa", link); err != nil {
		t.Fatal(err)
	}
	same, err := isTheSameFile(target, link)
	if err != nil || !same {
		t.Fatalf("dangling aliases not recognized: %v %v", same, err)
	}
	for _, alias := range []string{"symlink", "hardlink"} {
		t.Run(alias, func(t *testing.T) {
			dir := t.TempDir()
			out := auditInput(t, dir, "out.fa", "keep output\n")
			other := filepath.Join(dir, "other.fa")
			var err error
			if alias == "symlink" {
				err = os.Symlink(out, other)
			} else {
				err = os.Link(out, other)
			}
			if err != nil {
				t.Fatal(err)
			}
			_, stderr, err := runCLI(t, ">a\nAC\n>a\nGT\n", "rmdup", "-o", out, "-d", other)
			if err == nil {
				t.Fatalf("accepted output aliases: %s", stderr)
			}
			data, err := os.ReadFile(out)
			if err != nil || string(data) != "keep output\n" {
				t.Fatalf("output changed: %q %v", data, err)
			}
		})
	}
}

// TestAuditCompressedOutputErrors exercises compressor finalization, not just plain writes.
func TestAuditCompressedOutputErrors(t *testing.T) {
	if _, err := os.Stat("/dev/full"); err != nil {
		t.Skip("/dev/full is unavailable on this platform")
	}
	for _, ext := range []string{"gz", "xz", "zst", "bz2"} {
		t.Run(ext, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "out."+ext)
			if err := os.Symlink("/dev/full", path); err != nil {
				t.Fatal(err)
			}
			_, stderr, err := runCLI(t, ">r\nACGT\n", "fx2tab", "-o", path)
			if err == nil || !strings.Contains(stderr, "no space left") {
				t.Fatalf("compressed failure lost: %v %s", err, stderr)
			}
		})
	}
}

// TestAuditForceProtectsWorkingDirectory only exercises disposable directories.
func TestAuditForceProtectsWorkingDirectory(t *testing.T) {
	parent := t.TempDir()
	keep := auditInput(t, parent, "keep.txt", "keep this file\n")
	cwd := filepath.Join(parent, "work")
	if err := os.Mkdir(cwd, 0700); err != nil {
		t.Fatal(err)
	}
	_, stderr, err := auditCLI(t, cwd, ">a\nACGT\n", "split2", "-s", "1", "-O", parent, "-f")
	if err == nil || !strings.Contains(stderr, "working directory") {
		t.Fatalf("unsafe directory accepted: %v %s", err, stderr)
	}
	data, err := os.ReadFile(keep)
	if err != nil || string(data) != "keep this file\n" {
		t.Fatalf("working tree deleted: %q %v", data, err)
	}
}

// TestAuditWorkingDirectoryAliases must never clear cwd, even with stdin input.
func TestAuditWorkingDirectoryAliases(t *testing.T) {
	for _, relative := range []bool{false, true} {
		dir := t.TempDir()
		keep := auditInput(t, dir, "keep.txt", "keep this file\n")
		if err := os.Mkdir(filepath.Join(dir, "child"), 0700); err != nil {
			t.Fatal(err)
		}
		outDir := dir
		if relative {
			outDir = "child/.."
		}
		_, stderr, err := auditCLI(t, dir, ">a\nACGT\n", "split2", "-s", "1", "-O", outDir, "-f")
		if err != nil {
			t.Fatalf("cwd alias: %v %s", err, stderr)
		}
		data, err := os.ReadFile(keep)
		if err != nil || string(data) != "keep this file\n" {
			t.Fatalf("cwd cleared: %q %v", data, err)
		}
	}
}

// TestAuditSymlinkParentResolution retains the filesystem meaning of link/.. paths.
func TestAuditSymlinkParentResolution(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "other", "child"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "other", "child"), filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "link") + string(filepath.Separator) + ".." + string(filepath.Separator) + "out.fa"
	got, err := resolvedPath(path)
	if err != nil || got != filepath.Join(dir, "other", "out.fa") {
		t.Fatalf("wrong resolved path: %q %v", got, err)
	}
}
