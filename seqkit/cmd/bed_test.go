package cmd

import (
	"fmt"
	"strings"
	"testing"
)

// TestBEDNegativeCoordinates rejects invalid absolute coordinates before conversion.
func TestBEDNegativeCoordinates(t *testing.T) {
	for _, row := range []string{"s\t-9\t-2\n", "s\t-1\t3\n", "s\t3\t-1\n"} {
		path := auditInput(t, t.TempDir(), "bad.bed", row)
		if _, err := ReadBedFeatures(path); err == nil || !strings.Contains(err.Error(), "non-negative") {
			t.Fatalf("invalid BED coordinates accepted: %v", err)
		}
	}
}

// TestBEDMetadataPrefixes keeps reference IDs that merely start with a metadata keyword.
func TestBEDMetadataPrefixes(t *testing.T) {
	const rows = "track name=regions\nbrowser position chr1:1-4\ntrack\tname=regions\nbrowser\tposition chr1:1-4\ntrack1\t0\t4\nbrowser1\t1\t3\n"
	dir := t.TempDir()
	bed := auditInput(t, dir, "regions.bed", rows)
	features, err := ReadBedFeatures(bed)
	if err != nil || len(features) != 2 {
		t.Fatalf("expected two BED features, got %d: %v", len(features), err)
	}
	for i, want := range []string{"track1:1-4", "browser1:2-3"} {
		f := features[i]
		if got := fmt.Sprintf("%s:%d-%d", f.Chr, f.Start, f.End); got != want {
			t.Fatalf("feature %d: got %s, want %s", i, got, want)
		}
	}
	out, stderr, err := runCLI(t, ">track1\nACGT\n>browser1\nACGT\n", "subseq", "--bed", bed)
	if err != nil || !strings.Contains(out, "\nACGT\n") || !strings.Contains(out, "\nCG\n") || strings.Count(out, ">") != 2 {
		t.Fatalf("BED extraction lost records: %q %v %s", out, err, stderr)
	}
}
