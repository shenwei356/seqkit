package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFishCaseAndBAMMetadata checks statistics and the reference/program links.
func TestFishCaseAndBAMMetadata(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "fish.bam")
	_, stderr, err := runCLI(t, ">ref1 first ref\nACGTACGTAC\n", "fish", "-F", "acgtacgtac", "-s", "-q", "0", "-D", "-b", output)
	if err != nil {
		t.Fatalf("fish: %v %s", err, stderr)
	}
	if !strings.Contains(stderr, "ref1 first ref") || !strings.Contains(stderr, "100.00\t100.00") {
		t.Fatalf("case-sensitive accuracy: %s", stderr)
	}
	header, records := readBAMFixture(t, output)
	if len(header.Refs()) != 1 || header.Refs()[0].Name() != "ref1" || len(records) != 1 {
		t.Fatalf("invalid references/records: %v %d", header, len(records))
	}
	headerText, marshalErr := header.MarshalText()
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if strings.Contains(string(headerText), "PP:-") {
		t.Fatalf("missing parent reference: %v", header)
	}
	pg, ok := records[0].Tag([]byte("PG"))
	if !ok || pg.Type() != 'Z' || pg.Value() != "seqkit" {
		t.Fatalf("invalid PG tag: %v", pg)
	}
	nm, ok := records[0].Tag([]byte("NM"))
	if !ok || fmt.Sprint(nm.Value()) != "0" {
		t.Fatalf("case-sensitive NM: %v", nm)
	}
}

// TestFishInvalidParameters checks all formats and score sign constraints.
func TestFishInvalidParameters(t *testing.T) {
	for _, flags := range [][]string{{"-p", "4,-4,2"}, {"-p", "4,4,2,1"}, {"-p", "0,-4,-2,-1"}, {"-r", "5-30"}, {"-r", "30:5"}} {
		_, stderr, err := runCLI(t, ">s\nACGT\n", append([]string{"fish", "-F", "AC"}, flags...)...)
		if err == nil || !strings.Contains(stderr, "[ERRO]") || strings.Contains(stderr, "panic:") {
			t.Fatalf("invalid parameters accepted: %v %s", err, stderr)
		}
	}
}

// TestFishBAMOutputAliases protects sequence, query, list, and pass-through files.
func TestFishBAMOutputAliases(t *testing.T) {
	for _, target := range []string{"input", "query", "input list", "pass-through"} {
		t.Run(target, func(t *testing.T) {
			dir := t.TempDir()
			input := auditInput(t, dir, "in.fa", ">r1\nACGT\n")
			query := auditInput(t, dir, "query.fa", ">q1\nACGT\n")
			list := auditInput(t, dir, "inputs.list", input+"\n")
			output := auditInput(t, dir, "out.fa", "keep")
			bam := map[string]string{"input": input, "query": query, "input list": list, "pass-through": output}[target]
			before, err := os.ReadFile(bam)
			if err != nil {
				t.Fatal(err)
			}
			_, stderr, err := runCLI(t, "", "fish", "-f", query, "--infile-list", list, "-x", "-o", output, "-b", bam)
			after, readErr := os.ReadFile(bam)
			if err == nil || readErr != nil || string(before) != string(after) || !strings.Contains(stderr, "cannot be the same") {
				t.Fatalf("BAM alias accepted: err=%v preserved=%v stderr=%s", err, string(before) == string(after), stderr)
			}
		})
	}
}

// TestFishBAMDashAlias protects inputs named like the literal BAM destination.
func TestFishBAMDashAlias(t *testing.T) {
	dir := t.TempDir()
	const data = ">r1\nACGT\n"
	in := auditInput(t, dir, "-", data)
	_, stderr, err := auditCLI(t, dir, "", "fish", in, "-F", "ACGT", "-b", "-")
	after, readErr := os.ReadFile(in)
	if err == nil || readErr != nil || string(after) != data || !strings.Contains(stderr, "cannot be the same") {
		t.Fatalf("literal dash output alias: err=%v preserved=%v stderr=%s", err, string(after) == data, stderr)
	}
}

// TestFishRangeBounds skips disjoint ranges and rejects noninteger coordinates.
func TestFishRangeBounds(t *testing.T) {
	for _, region := range []string{"10:20", "-20:-10", "4:", ":-4"} {
		_, stderr, err := runCLI(t, ">r\nACGT\n", "fish", "-F", "ACGT", "-r", region)
		if err != nil || stderr != "" {
			t.Fatalf("out-of-range search %q produced a hit: %v %s", region, err, stderr)
		}
	}
	for _, region := range []string{"Inf:Inf", "-Inf:4", "NaN:4", "0.5:4"} {
		_, stderr, err := runCLI(t, ">r\nACGT\n", "fish", "-F", "ACGT", "-r", region)
		if err == nil || !strings.Contains(stderr, "[ERRO]") || strings.Contains(stderr, "panic:") {
			t.Fatalf("invalid range %q: %v %s", region, err, stderr)
		}
	}
}

// TestFishAmbiguousDNA scores uncertain bases as mismatches without panicking.
func TestFishAmbiguousDNA(t *testing.T) {
	for _, all := range []bool{false, true} {
		args := []string{"fish", "-F", "ACGT", "-q", "0", "-s"}
		if all {
			args = append(args, "-a")
		}
		_, stderr, err := runCLI(t, ">r\nNNNACGTNNN\n", args...)
		if err != nil || !strings.Contains(stderr, "r\t3\t7\t") {
			t.Fatalf("ambiguous flank all=%v: %v %s", all, err, stderr)
		}
		_, stderr, err = runCLI(t, ">r\nNNNN\n", args...)
		if err != nil || stderr != "" {
			t.Fatalf("ambiguous bases produced a hit: %v %s", err, stderr)
		}
	}
}

// TestFishQueryIDWhitespace uses the parsed ID for query names in result columns.
func TestFishQueryIDWhitespace(t *testing.T) {
	query := auditInput(t, t.TempDir(), "query.fa", ">q1\t\nACGT\n")
	_, stderr, err := runCLI(t, ">r1\nACGT\n", "fish", "-f", query, "-s", "-q", "0")
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	if err != nil || len(lines) != 2 {
		t.Fatalf("query ID: %v %s", err, stderr)
	}
	fields := strings.Split(lines[1], "\t")
	if len(fields) != 12 || fields[3] != "q1" {
		t.Fatalf("query ID broke result columns: %q", fields)
	}
}
