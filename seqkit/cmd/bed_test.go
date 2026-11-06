package cmd

import (
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
