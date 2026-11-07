package cmd

import (
	"os"
	"strings"
	"testing"
)

// TestMutateColonContent keeps edit content separate from its position delimiter.
func TestMutateColonContent(t *testing.T) {
	for _, tc := range []struct {
		flag, edit, want string
	}{
		{"-p", "1::", ":CGT"},
		{"-i", "0::", ":ACGT"},
		{"-i", "2:G:T", "ACG:TGT"},
		{"-i", "-1:::", "ACGT::"},
	} {
		t.Run(tc.flag+tc.edit, func(t *testing.T) {
			out, stderr, err := runCLI(t, ">s\nACGT\n", "mutate", "--quiet", tc.flag, tc.edit)
			if err != nil || out != ">s\n"+tc.want+"\n" {
				t.Fatalf("edit failed: %q %v %s", out, err, stderr)
			}
		})
	}
}

// TestMutateNonASCIIPoint rejects a multibyte replacement before opening output.
func TestMutateNonASCIIPoint(t *testing.T) {
	output := auditInput(t, t.TempDir(), "out.fa", "keep\n")
	_, stderr, err := runCLI(t, ">s\nACGT\n", "mutate", "-p", "1:é", "-o", output)
	data, readErr := os.ReadFile(output)
	if err == nil || !strings.Contains(stderr, "invalid value") || readErr != nil || string(data) != "keep\n" {
		t.Fatalf("invalid point did not fail safely: %v %s", err, stderr)
	}
}

// TestMutateDamagedAuxiliaryInput rejects incomplete auxiliary data.
func TestMutateDamagedAuxiliaryInput(t *testing.T) {
	assertDamagedTextInput(t, "r\n", "mutate", "-p", "1:T", "-f")
}
