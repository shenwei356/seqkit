package cmd

import (
	"bytes"
	"compress/gzip"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestAuxiliaryStdinPreflight rejects competing stdin consumers before reading
// auxiliary data or truncating output, while permitting one auxiliary stream.
func TestAuxiliaryStdinPreflight(t *testing.T) {
	for _, tc := range []struct {
		name, data string
		args       []string
	}{
		{"grep", "r\n", []string{"grep", "-f", "-"}},
		{"locate", ">p\nAC\n", []string{"locate", "-f", "-"}},
		{"fish", ">p\nAC\n", []string{"fish", "-f", "-"}},
		{"seq", "r\n", []string{"seq", "--f-pattern-file", "-"}},
		{"replace-filter", "r\n", []string{"replace", "-p", "r", "-r", "s", "--f-pattern-file", "-"}},
		{"replace-kv", "r\ts\n", []string{"replace", "-p", "(r)", "-r", "{kv}", "-k", "-"}},
		{"mutate", "r\n", []string{"mutate", "-p", "1:T", "-f", "-"}},
		{"fa2fq", ">r\nAC\n", []string{"fa2fq", "-f", "-"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := auditInput(t, dir, "input.fq", "@r\nACGT\n+\nIIII\n")
			if tc.name == "mutate" {
				input = auditInput(t, dir, "input.fa", ">r\nACGT\n")
			}
			list := auditInput(t, dir, "inputs.list", "-\n")
			for _, extra := range [][]string{nil, {"-"}, {"--infile-list", list}, {"--skip-file-check"}, {input, "--infile-list", "-"}} {
				output := auditInput(t, dir, "output", "keep\n")
				args := append(append([]string(nil), tc.args...), extra...)
				args = append(args, "-o", output)
				_, stderr, err := runCLI(t, tc.data, args...)
				data, readErr := os.ReadFile(output)
				if err == nil || !strings.Contains(stderr, "stdin may only be specified once") || readErr != nil || string(data) != "keep\n" {
					t.Errorf("stdin conflict %v was not rejected before writing: %v %s", extra, err, stderr)
				}
			}
			args := append(append([]string(nil), tc.args...), input)
			if _, stderr, err := runCLI(t, tc.data, args...); err != nil {
				t.Fatalf("single auxiliary stdin rejected: %v %s", err, stderr)
			}
		})
	}
}

// TestAuxiliaryStdinConflicts also covers multiple auxiliaries and faidx's
// region-list stream, without treating positional faidx query IDs as inputs.
func TestAuxiliaryStdinConflicts(t *testing.T) {
	dir := t.TempDir()
	input := auditInput(t, dir, "input.fa", ">r\nACGT\n")
	for _, args := range [][]string{
		{"replace", input, "-p", "(r)", "-r", "{kv}", "-k", "-", "--f-pattern-file", "-"},
		{"faidx", input, "--region-file", "-", "--infile-list", "-"},
	} {
		output := auditInput(t, dir, "output", "keep\n")
		_, stderr, err := runCLI(t, "", append(args, "-o", output)...)
		data, readErr := os.ReadFile(output)
		if err == nil || !strings.Contains(stderr, "stdin may only be specified once") || readErr != nil || string(data) != "keep\n" {
			t.Errorf("auxiliary conflict not rejected: %v %s", err, stderr)
		}
	}
}

// TestRegionSyntax rejects extra coordinate fields before writing results.
func TestRegionSyntax(t *testing.T) {
	commands := [][]string{
		{"subseq", "-r"}, {"range", "-r"}, {"grep", "-p", "AC", "-R"},
		{"amplicon", "-F", "AC", "-r"}, {"split", "-r"},
	}
	for _, command := range commands {
		for _, region := range []string{"1:2:3", "1:2:", "1:2:invalid"} {
			t.Run(command[0]+"/"+region, func(t *testing.T) {
				dir := t.TempDir()
				output := filepath.Join(dir, "output")
				args := append(append([]string(nil), command...), region)
				if command[0] == "split" {
					args = append(args, "-O", output)
				} else {
					auditInput(t, dir, "output", "keep\n")
					args = append(args, "-o", output)
				}
				_, stderr, err := runCLI(t, ">r\nACGT\n", args...)
				if err == nil || !strings.Contains(stderr, "invalid") || strings.Contains(stderr, "panic:") {
					t.Errorf("extra coordinate fields accepted: %v %s", err, stderr)
				}
				if command[0] == "split" {
					if _, err := os.Stat(output); !os.IsNotExist(err) {
						t.Errorf("output directory created before validation: %v", err)
					}
				} else if data, err := os.ReadFile(output); err != nil || string(data) != "keep\n" {
					t.Errorf("output changed before validation: %v", err)
				}
			})
		}
	}
}

// TestInvalidIDRegexpPreflight rejects invalid global parsers before opening
// outputs or replacing indexes, even on paths that bypass the FASTX reader.
func TestInvalidIDRegexpPreflight(t *testing.T) {
	commands := [][]string{
		{"seq"}, {"head"}, {"grep", "-p", "r"}, {"translate"},
		{"sort", "-2", "-U"}, {"shuffle", "-2", "-U"},
		{"subseq", "-r", "1:2", "-U"}, {"faidx", "-U"}, {"sum"},
	}
	for _, command := range commands {
		for _, pattern := range []string{"[", "^r$"} {
			t.Run(command[0]+"/"+pattern, func(t *testing.T) {
				dir := t.TempDir()
				input := auditInput(t, dir, "input.fa", ">r\nACGT\n")
				output := auditInput(t, dir, "output.fa", "keep output\n")
				index := auditInput(t, dir, "input.fa.seqkit.fai", "keep index\n")
				if command[0] == "faidx" {
					index = auditInput(t, dir, "input.fa.fai", "keep index\n")
				}
				args := append(append([]string(nil), command...), input, "--id-regexp", pattern, "-o", output)
				_, stderr, err := runCLI(t, "", args...)
				if err == nil || !strings.Contains(stderr, "[ERRO]") || strings.Contains(stderr, "panic:") {
					t.Errorf("invalid ID parser accepted: %v %s", err, stderr)
				}
				for path, want := range map[string]string{output: "keep output\n", index: "keep index\n"} {
					data, err := os.ReadFile(path)
					if err != nil || string(data) != want {
						t.Errorf("file changed before ID validation: %s: %v", path, err)
					}
				}
			})
		}
	}
}

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
