package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/biogo/hts/sam"
)

// TestFishBAMDuplicateReferenceIDs rejects ambiguous BAM dictionaries before
// replacing an existing BAM, including duplicates supplied through separate files.
func TestFishBAMDuplicateReferenceIDs(t *testing.T) {
	for _, fullHeader := range []bool{false, true} {
		for _, separateFiles := range []bool{false, true} {
			dir := t.TempDir()
			output := auditInput(t, dir, "hits.bam", "keep\n")
			args := []string{"fish", "-F", "ACGT", "-s", "-b", output}
			if fullHeader {
				args = append(args, "-D")
			}
			input := ">r\nACGT\n>r\nACGTT\n"
			if separateFiles {
				first := auditInput(t, dir, "first.fa", ">r\nACGT\n")
				second := auditInput(t, dir, "second.fa", ">r\nACGTT\n")
				args = append(args, first, second)
				input = ""
			}
			_, stderr, err := runCLI(t, input, args...)
			data, readErr := os.ReadFile(output)
			if err == nil || !strings.Contains(stderr, "duplicate reference ID") || readErr != nil || string(data) != "keep\n" {
				t.Fatalf("duplicate reference IDs corrupted BAM: %v %s", err, stderr)
			}
		}
	}
}

// TestFishBAMEmptyReferences skips zero-length BAM references without shifting
// later reference IDs or dropping empty records from pass-through output.
func TestFishBAMEmptyReferences(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		wantRefs    []string
	}{
		{"no records", "", nil},
		{"empty records", ">empty\n\n", nil},
		{"mixed", ">empty\n\n>r1\nACGT\n>empty2\n\n>r2\nACGT\n", []string{"r1", "r2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "hits.bam")
			output, stderr, err := runCLI(t, tc.input, "fish", "-F", "ACGT", "-s", "-x", "-b", path)
			if err != nil {
				t.Fatalf("empty reference rejected: %v %s", err, stderr)
			}
			if output != tc.input {
				t.Fatal("pass-through records changed")
			}
			header, records := readBAMFixture(t, path)
			if len(header.Refs()) != len(tc.wantRefs) || len(records) != len(tc.wantRefs) {
				t.Fatalf("got %d references and %d records, want %d", len(header.Refs()), len(records), len(tc.wantRefs))
			}
			for i, name := range tc.wantRefs {
				if header.Refs()[i].Name() != name || records[i].Ref.Name() != name || records[i].Ref.ID() != i {
					t.Errorf("incorrect BAM reference for record %d", i)
				}
			}
		})
	}
}

// TestFishBAMPrimaryAlignments requires one best alignment per query across
// references, search ranges, and recursive searches for repeated matches.
func TestFishBAMPrimaryAlignments(t *testing.T) {
	for _, tc := range []struct {
		name, input, queries string
		flags                []string
		wantRecords          int
		wantPrimary          map[string]string
	}{
		{"repeated", ">r\nACGTAAAAACGT\n", "ACGT", []string{"-a", "-s", "-q", "5"}, 2, map[string]string{"q0": "r"}},
		{"references", ">partial\nACGA\n>exact\nACGT\n", "ACGT", []string{"-s"}, 2, map[string]string{"q0": "exact"}},
		{"queries", ">r\nACGTAAAA\n", "ACGT,AAAA", []string{"-s"}, 2, map[string]string{"q0": "r", "q1": "r"}},
		{"ranges", ">r\nACGTAAAAACGT\n", "ACGT", []string{"-s", "-r", "0:4,8:12"}, 2, map[string]string{"q0": "r"}},
		{"strands", ">r\nACGT\n", "ACGT", nil, 2, map[string]string{"q0": "r"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "hits.bam")
			args := append([]string{"fish", "-F", tc.queries, "-q", "0", "-b", output}, tc.flags...)
			_, stderr, err := runCLI(t, tc.input, args...)
			if err != nil {
				t.Fatalf("fish failed: %v %s", err, stderr)
			}
			_, records := readBAMFixture(t, output)
			if len(records) != tc.wantRecords {
				t.Fatalf("got %d alignments, want %d", len(records), tc.wantRecords)
			}
			if tc.name != "references" && records[0].Flags&sam.Secondary != 0 {
				t.Error("the first equally scoring alignment should remain primary")
			}
			primary := make(map[string]int)
			for _, record := range records {
				if record.Flags&sam.Secondary == 0 {
					primary[record.Name]++
					if record.Ref.Name() != tc.wantPrimary[record.Name] {
						t.Errorf("primary for %s maps to %s, want %s", record.Name, record.Ref.Name(), tc.wantPrimary[record.Name])
					}
				}
			}
			for query := range tc.wantPrimary {
				if primary[query] != 1 {
					t.Errorf("query %s has %d primary alignments, want 1", query, primary[query])
				}
			}
		})
	}
}

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

// TestFishValidateLeadingEmptyRecords keeps base validation active after empty reads.
func TestFishValidateLeadingEmptyRecords(t *testing.T) {
	assertLeadingEmptyValidation(t, "fish", "-v", "-F", "AC", "-r", "0:2")
}
