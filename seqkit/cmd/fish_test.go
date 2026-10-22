package cmd

import (
	"fmt"
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
