package cmd

import (
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestParseByteSizeBounds rejects nonfinite and overflowing values before conversion.
func TestParseByteSizeBounds(t *testing.T) {
	for _, input := range []string{"NaN", "+Inf", "InfK", "1e30", "9223372036854775808", "8388608T", "K"} {
		if n, err := ParseByteSize(input); err == nil {
			t.Fatalf("accepted %q as %d", input, n)
		}
	}
	for input, want := range map[string]int64{"1K": 1024, "1.5K": 1536, "-1": 0, "0": 0, "9223372036854775807": math.MaxInt64} {
		if n, err := ParseByteSize(input); err != nil || n != want {
			t.Fatalf("%q: %d %v", input, n, err)
		}
	}
}

// assertInvalidBufferSizes checks the CLI's native-int bound and output preflight.
func assertInvalidBufferSizes(t *testing.T, command string) {
	t.Helper()
	values := []string{"NaN", "Inf", "1e30", "0", "0.1", "-1", "K"}
	if strconv.IntSize == 32 {
		values = append(values, "2G")
	}
	for _, value := range values {
		dir := t.TempDir()
		output := auditInput(t, dir, "out.txt", "keep\n")
		_, stderr, err := runCLI(t, "", command, "-b", value, "-o", output)
		data, readErr := os.ReadFile(output)
		if err == nil || !strings.Contains(stderr, "invalid buffer size") || strings.Contains(stderr, "panic:") || readErr != nil || string(data) != "keep\n" {
			t.Fatalf("%s -b %s did not fail safely: %v %s", command, value, err, stderr)
		}
	}
}
