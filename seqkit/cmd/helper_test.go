package cmd

import (
	"bytes"
	"compress/gzip"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestAuxiliaryReaderErrors preserves errors delivered after reader construction.
func TestAuxiliaryReaderErrors(t *testing.T) {
	badIndex := auditInput(t, t.TempDir(), "bad.fai", "r\tinvalid\t3\t4\t5\n")
	if _, _, err := getSeqIDAndLengthFromFaidxFile(badIndex); err == nil {
		t.Error("invalid index length was accepted")
	}
	var compressed bytes.Buffer
	w := gzip.NewWriter(&compressed)
	if _, err := w.Write([]byte(strings.Repeat("r\treplacement\n", 10000))); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data := compressed.Bytes()
	data[len(data)-8] ^= 0xff // Corrupt the CRC after a valid compressed payload.
	path := auditInput(t, t.TempDir(), "bad.tsv.gz", string(data))
	if _, err := readKVs(path, false); err == nil {
		t.Fatal("key-value checksum error was swallowed")
	}
}

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
