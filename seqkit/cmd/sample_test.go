package cmd

import (
	"os"
	"strings"
	"testing"
)

// TestSampleMultipleInputs compares a multi-file invocation with separate runs.
func TestSampleMultipleInputs(t *testing.T) {
	dir := t.TempDir()
	a := auditInput(t, dir, "a.fa", ">a\nAA\n>b\nCC\n>c\nGG\n>d\nTT\n")
	b := auditInput(t, dir, "b.fq", "@w\nAA\n+\n12\n@x\nCC\n+\n34\n@y\nGG\n+\n56\n@z\nTT\n+\n78\n")
	list := auditInput(t, dir, "inputs", b+"\n")
	for _, extra := range [][]string{nil, {"-2"}} {
		args := append([]string{"sample", "-p", "0.7", "-s", "42"}, extra...)
		one, e1, err1 := runCLI(t, "", append(append([]string{}, args...), a)...)
		two, e2, err2 := runCLI(t, "", append(append([]string{}, args...), b)...)
		out, stderr, err := runCLI(t, "", append(args, a, "-X", list)...)
		if err1 != nil || err2 != nil || err != nil || out != one+two || one == "" || two == "" {
			t.Fatalf("seed/format mismatch: %q != %q + %q; %v %v %v %s %s %s", out, one, two, err, err1, err2, stderr, e1, e2)
		}
	}
	output := auditInput(t, dir, "keep", "keep")
	_, stderr, err := runCLI(t, "", "sample", "-n", "2", a, "-X", list, "-o", output)
	data, _ := os.ReadFile(output)
	if err == nil || !strings.Contains(stderr, "one file") || string(data) != "keep" {
		t.Fatalf("number mode accepted multiple files: %v %s %q", err, stderr, data)
	}
}

// TestSampleReadErrors rejects a damaged suffix instead of sampling its prefix.
func TestSampleReadErrors(t *testing.T) {
	for _, input := range []string{"not a sequence\n", "@good\nAC\n+\nII\n@bad\nACGT\n+\nII\n"} {
		out, stderr, err := runCLI(t, input, "sample", "-n", "10")
		if err == nil || out != "" || !strings.Contains(stderr, "[ERRO]") {
			t.Fatalf("lost input error: %q %v %s", out, err, stderr)
		}
	}
}

// TestSamplingNaNAndDuplicateTitles keeps positional sampling independent of names.
func TestSamplingNaNAndDuplicateTitles(t *testing.T) {
	dir := t.TempDir()
	file := auditInput(t, dir, "dup.fa", ">a desc\nAAAA\n>a desc\nCCCC\n")
	for _, command := range []string{"sample", "sample2"} {
		for _, input := range []string{file, "-"} {
			out, stderr, err := runCLI(t, ">a\nAC\n", command, "-p", "NaN", input)
			if err == nil || out != "" || !strings.Contains(stderr, "range of") {
				t.Fatalf("NaN accepted: %v %s", err, stderr)
			}
		}
		for _, mode := range [][]string{{"-n", "2"}, {"-p", "1"}} {
			out, stderr, err := runCLI(t, "", append([]string{command, "-2", file}, mode...)...)
			if err != nil || strings.Count(out, ">a desc") != 2 || !strings.Contains(out, "AAAA") || !strings.Contains(out, "CCCC") {
				t.Fatalf("duplicate sampling changed: %q %v %s", out, err, stderr)
			}
		}
	}
}
