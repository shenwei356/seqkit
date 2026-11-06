package cmd

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// TestConcatFullSingleNonemptyInput preserves duplicate records and qualities.
func TestConcatFullSingleNonemptyInput(t *testing.T) {
	for _, input := range []string{">r\nAC\n>r\nGT\n", "@r\nAC\n+\nIJ\n@r\nGT\n+\nKL\n"} {
		dir := t.TempDir()
		file := auditInput(t, dir, "records.fx", input)
		empty := auditInput(t, dir, "empty.fx", "")
		for _, files := range [][]string{{empty, file}, {file, empty}, {empty, file, empty}} {
			out, stderr, err := runCLI(t, "", append([]string{"concat", "-f", "-F", "N"}, files...)...)
			if err != nil || out != input {
				t.Fatalf("single nonempty input changed: %q %v %s", out, err, stderr)
			}
		}
	}
}

// TestConcatCustomIDDescription excludes text already captured as part of the ID.
func TestConcatCustomIDDescription(t *testing.T) {
	description := strings.Repeat("x", 3) + " " + strings.Repeat("y", 2)
	header := "r " + description
	file := auditInput(t, t.TempDir(), "input.fa", ">"+header+"\nAC\n")
	for _, test := range []struct{ pattern, header string }{
		{`^(.+)$`, header},
		{`^(\S+)`, header + "|" + description},
		{`^(\S+ \S+)`, header + "|" + strings.Repeat("y", 2)},
		{`^(\S+) \S+`, header + "|" + description},
		{`^(missing)`, header},
	} {
		out, stderr, err := runCLI(t, "", "concat", "--id-regexp", test.pattern, file, file)
		if err != nil || out != ">"+test.header+"\nACAC\n" {
			t.Fatalf("custom ID/description mismatch for %q: %v %s", test.pattern, err, stderr)
		}
	}
}

// TestConcatEmptyFASTQ preserves file format independently of the first quality.
func TestConcatEmptyFASTQ(t *testing.T) {
	dir := t.TempDir()
	first := auditInput(t, dir, "first.fq", "@empty\n\n+\n\n@r\nAC\n+\nIJ\n")
	second := auditInput(t, dir, "second.fq", "@empty\n\n+\n\n@r\nGT\n+\nKL\n")
	nonempty := auditInput(t, dir, "third.fq", "@r\nGT\n+\nKL\n")
	for _, file := range []string{second, nonempty} {
		out, stderr, err := runCLI(t, "", "concat", first, file)
		if err != nil || !strings.Contains(out, "@r\nACGT\n+\nIJKL\n") || strings.Contains(out, ">") {
			t.Fatalf("FASTQ format lost: %v %s", err, stderr)
		}
		if file == second && !strings.Contains(out, "@empty\n\n+\n\n") {
			t.Fatal("empty FASTQ record lost")
		}
	}
}

// TestConcatFillValidation rejects malformed padding before touching output.
func TestConcatFillValidation(t *testing.T) {
	dir := t.TempDir()
	a := auditInput(t, dir, "a.fq", "@a\nAC\n+\nII\n")
	b := auditInput(t, dir, "b.fq", "@b\nGT\n+\nJJ\n")
	for _, fill := range []string{"NN", "é", "\n", " "} {
		output := auditInput(t, dir, "out.fq", "keep\n")
		_, stderr, err := runCLI(t, "", "concat", "-f", "-F", fill, "-o", output, a, b)
		data, readErr := os.ReadFile(output)
		if err == nil || !strings.Contains(stderr, "--fill") || readErr != nil || string(data) != "keep\n" {
			t.Fatalf("invalid fill changed output: %v %s", err, stderr)
		}
	}
	out, stderr, err := runCLI(t, "", "concat", "-f", "-F", "N", a, b)
	if err != nil || !strings.Contains(out, "@a\nACNN\n+\nIIBB\n") || !strings.Contains(out, "@b\nNNGT\n+\nBBJJ\n") {
		t.Fatalf("valid padding changed: %v %s", err, stderr)
	}
}

// TestConcatProduct streams duplicates in the same order and supports early stop.
func TestConcatProduct(t *testing.T) {
	var got [][]int
	for values := range product([]int{1, 1}, []int{-1}, []int{2, 3}) {
		got = append(got, append([]int(nil), values...))
	}
	want := [][]int{{1, -1, 2}, {1, -1, 3}, {1, -1, 2}, {1, -1, 3}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("combinations: %v", got)
	}
	for range product([]int{1}, nil) {
		t.Fatal("empty list yielded a combination")
	}
	count := 0
	for range product([]int{1, 2}, []int{3, 4}) {
		count++
		break
	}
	if count != 1 {
		t.Fatal("iterator ignored early stop")
	}
}
