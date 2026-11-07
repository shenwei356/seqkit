package cmd

import (
	"strings"
	"testing"
)

// TestTab2fxBufferValidation preserves existing output for invalid buffer limits.
func TestTab2fxBufferValidation(t *testing.T) { assertInvalidBufferSizes(t, "tab2fx") }

// TestTab2fxExtraColumns retains qualities when fx2tab adds sequence statistics.
func TestTab2fxExtraColumns(t *testing.T) {
	for _, input := range []string{"@r1\nACGT\n+\n!I?J\n", ">r1\nACGT\n"} {
		tab, stderr, err := runCLI(t, input, "fx2tab", "-l", "-g")
		if err != nil {
			t.Fatalf("fx2tab: %v %s", err, stderr)
		}
		out, stderr, err := runCLI(t, tab, "tab2fx")
		if err != nil || out != input {
			t.Fatalf("round trip lost format/qualities: got=%q want=%q err=%v stderr=%s", out, input, err, stderr)
		}
	}
}

// TestTab2fxQualityLength rejects records that would produce malformed FASTQ.
func TestTab2fxQualityLength(t *testing.T) {
	for _, input := range []string{"r1\tACGT\t!!!\n", "r1\tACGT\t!!!!!\t4\n"} {
		out, stderr, err := runCLI(t, input, "tab2fx")
		if err == nil || out != "" || !strings.Contains(stderr, "length") {
			t.Fatalf("invalid quality length accepted: %q %v %s", out, err, stderr)
		}
	}
}

// TestTab2fxEmptyInputs skips empty files and accepts an empty stdin stream.
func TestTab2fxEmptyInputs(t *testing.T) {
	dir := t.TempDir()
	empty := auditInput(t, dir, "empty.tsv", "")
	records := auditInput(t, dir, "records.tsv", "r1\tAC\t!!\n")
	for _, tc := range []struct {
		files []string
		want  string
	}{
		{nil, ""}, {[]string{empty, records, empty}, "@r1\nAC\n+\n!!\n"},
	} {
		out, stderr, err := runCLI(t, "", append([]string{"tab2fx"}, tc.files...)...)
		if err != nil || out != tc.want {
			t.Fatalf("empty input: %q %v %s", out, err, stderr)
		}
	}
}
