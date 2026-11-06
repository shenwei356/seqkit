package cmd

import (
	"fmt"
	"strings"
	"testing"
)

// TestHeaderDescriptions locks the separately approved default and custom-ID rules.
func TestHeaderDescriptions(t *testing.T) {
	for _, separator := range []string{" ", "  ", "\t\t", " \t  "} {
		out, stderr, err := runCLI(t, ">a"+separator+"even\nAC\n>a"+separator+"even\nGG\n", "rename")
		if err != nil || !strings.Contains(out, ">a_2 even\nGG\n") {
			t.Fatalf("description cut: %q %v %s", out, err, stderr)
		}
	}
	out, stderr, err := runCLI(t, ">a even\nAC\n>a even\nGG\n", "rename", "--id-regexp", `^(\S+)\s`)
	if err != nil || out != ">a even\nAC\n>a_2\nGG\n" {
		t.Fatalf("custom rename changed: %q %v %s", out, err, stderr)
	}
	out, stderr, err = runCLI(t, ">a1  XA B\nAC\n>a2  YA B\nGG\n", "head-genome")
	if err != nil || out != ">a1  XA B\nAC\n" {
		t.Fatalf("word boundary mismatch: %q %v %s", out, err, stderr)
	}
	_, stderr, err = runCLI(t, ">a1 desc\nAC\n", "head-genome", "--id-regexp", `^(\S+)\s`)
	if err == nil || !strings.Contains(stderr, "no description") {
		t.Fatalf("custom head-genome changed: %v %s", err, stderr)
	}
}

// TestMutateEmptyIntersections preserves valid edits and insertion at zero.
func TestMutateEmptyIntersections(t *testing.T) {
	for _, edit := range [][]string{{"-p", "-11:x"}, {"-i", "-11:GG"}, {"-d", "1:-11"}} {
		out, stderr, err := runCLI(t, ">s\nACGTACGTAC\n", append([]string{"mutate"}, edit...)...)
		if err != nil || out != ">s\nACGTACGTAC\n" || !strings.Contains(stderr, "WARN") {
			t.Fatalf("invalid edit applied: %q %v %s", out, err, stderr)
		}
	}
	out, stderr, err := runCLI(t, ">s\nACGT\n", "mutate", "-i", "0:GG")
	if err != nil || out != ">s\nGGACGT\n" {
		t.Fatalf("insertion at zero changed: %q %v %s", out, err, stderr)
	}
}

// TestTranslateContracts covers truncation, failed frames and preflight errors.
func TestTranslateContracts(t *testing.T) {
	out, stderr, err := runCLI(t, ">s\nATGTAANNNAAA\n", "translate", "--trim", "-x", "-m", "99")
	if err != nil || out != ">s\nM\n" {
		t.Fatalf("trim/min-len changed: %q %v %s", out, err, stderr)
	}
	out, stderr, err = runCLI(t, ">b\nN\n", "translate", "-e", "-F", "-f", "1,3")
	if err != nil || out != ">b_frame=1\n\n>b_frame=3\n\n" {
		t.Fatalf("failed frame headers: %q %v %s", out, err, stderr)
	}
	for _, flags := range [][]string{{"-l", "99"}, {"-L", "99"}, {"-f", ""}} {
		out, stderr, err = runCLI(t, "", append([]string{"translate"}, flags...)...)
		if err == nil || out != "" || !strings.Contains(stderr, "[ERRO]") {
			t.Fatalf("invalid parameter accepted: %v %s", err, stderr)
		}
	}
	for _, frame := range []string{"-1", "-2", "-3"} {
		out, stderr, err = runCLI(t, ">s\nTTATTTCATTTATTTCAT\n", "translate", "-s", "-f", frame)
		if err != nil {
			t.Fatalf("negative frame: %v %s", err, stderr)
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		for i := 0; i+1 < len(lines); i += 2 {
			var begin, end int
			_, scanErr := fmt.Sscanf(strings.Split(lines[i], "_begin=")[1], "%d_end=%d", &begin, &end)
			if scanErr != nil || end-begin+1 != 3*len(lines[i+1]) {
				t.Fatalf("bad translated interval: %q", lines[i:i+2])
			}
		}
	}
}

// TestReplaceLocalQuality rejects borrowing qualities outside the current match.
func TestReplaceLocalQuality(t *testing.T) {
	out, stderr, err := runCLI(t, "@s\nACGTTT\n+\nABCDEF\n", "replace", "-s", "-p", "T+|A", "-r", "GG")
	if err == nil || out != "" || !strings.Contains(stderr, "length-increasing") {
		t.Fatalf("quality borrowed: %q %v %s", out, err, stderr)
	}
}

// TestSlidingEmptyRecords skips empty records while retaining subsequent windows.
func TestSlidingEmptyRecords(t *testing.T) {
	out, stderr, err := runCLI(t, ">empty\n\n>s\nACGT\n", "sliding", "-c", "-s", "4", "-W", "4")
	if err != nil || out != ">s_sliding:1-4\nACGT\n" {
		t.Fatalf("empty circular record: %q %v %s", out, err, stderr)
	}
}
