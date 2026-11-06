package cmd

import (
	"strings"
	"testing"
)

// TestMergeSlidesIntervals retains nested extents and chains through merged spans.
func TestMergeSlidesIntervals(t *testing.T) {
	out, stderr, err := runCLI(t, "ref_sliding:1-10\nref_sliding:2-5\nref_sliding:8-20\n", "merge-slides")
	if err != nil || out != "ref\t0\t20\n" {
		t.Fatalf("merged extent: %q %v %s", out, err, stderr)
	}
	for _, input := range []string{"ref_sliding:2-5\nref_sliding:1-10\n", "a_sliding:1-4\nb_sliding:1-4\na_sliding:5-8\n", "ref_sliding:0-2\n", "ref_sliding:5-2\n"} {
		_, stderr, err = runCLI(t, input, "merge-slides")
		if err == nil || !strings.Contains(stderr, "-:") {
			t.Fatalf("bad coordinates/order accepted: %v %s", err, stderr)
		}
	}
	_, stderr, err = runCLI(t, "", "merge-slides", "-r", `^(.+)$`)
	if err == nil || !strings.Contains(stderr, "three capturing groups") {
		t.Fatalf("short regexp accepted: %v %s", err, stderr)
	}
}
