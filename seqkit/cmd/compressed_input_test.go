package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/shenwei356/xopen"
)

// compressedInput encodes a complete fixture before tests truncate or corrupt it.
func compressedInput(t *testing.T, extension, input string) []byte {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.fa"+extension)
	w, err := xopen.Wopen(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.WriteString(input); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// compressedRecords generates reproducible, distinct records and their table output.
func compressedRecords(fastq bool, count, length int) (string, string) {
	rng := rand.New(rand.NewSource(1))
	var records, table strings.Builder
	for i := 0; i < count; i++ {
		bases := make([]byte, length)
		for j := range bases {
			bases[j] = "ACGT"[rng.Intn(4)]
		}
		if fastq {
			fmt.Fprintf(&records, "@s%d\n%s\n+\n%s\n", i, bases, strings.Repeat("I", length))
		} else {
			fmt.Fprintf(&records, ">s%d\n%s\n", i, bases)
		}
		fmt.Fprintf(&table, "s%d\t%s\n", i, bases)
	}
	return records.String(), table.String()
}

// TestCompressedInputErrors exercises shared readers through the affected commands.
func TestCompressedInputErrors(t *testing.T) {
	input, table := compressedRecords(false, 10, 100)
	commands := []struct {
		args []string
		want string // Exact output for commands that print sequences or tables.
	}{
		{[]string{"seq", "-w", "0"}, input},
		{[]string{"head", "-n", "100", "-w", "0"}, input},
		{[]string{"fx2tab", "-Q"}, table},
		{[]string{"grep", "-r", "-p", "^s", "-w", "0"}, input},
		{[]string{"rmdup", "-w", "0"}, input},
		{[]string{"sort", "-w", "0"}, input},
		{[]string{"stats", "-T"}, ""},
	}
	for _, extension := range []string{".gz", ".xz", ".zst", ".bz2", ".lz4"} {
		encoded := compressedInput(t, extension, input)
		for _, command := range commands {
			for _, stdin := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/stdin=%v", extension, command.args[0], stdin), func(t *testing.T) {
					for _, damaged := range []bool{false, true} {
						data := encoded
						if damaged {
							// This cut is inside the first record, not at a FASTA boundary.
							data = data[:60]
						}
						args := append([]string(nil), command.args...)
						var source string
						if stdin {
							source = string(data)
							args = append(args, "-")
						} else {
							path := auditInput(t, t.TempDir(), "input.fa"+extension, string(data))
							args = append(args, path)
						}
						stdout, stderr, err := runCLI(t, source, args...)
						if damaged {
							if err == nil || !strings.Contains(stderr, "[ERRO]") {
								t.Fatalf("damage hidden: err=%v stdout=%q stderr=%s", err, stdout, stderr)
							}
							if command.args[0] == "stats" {
								if len(strings.Split(strings.TrimSpace(stdout), "\n")) > 1 {
									t.Fatalf("stats emitted a valid row for damaged input: %q", stdout)
								}
							} else if stdout != "" {
								t.Fatalf("emitted an incomplete record: %q", stdout)
							}
						} else {
							if err != nil {
								t.Fatalf("valid input: %v %s", err, stderr)
							}
							if command.want != "" && stdout != command.want {
								t.Fatalf("output changed: got %q, want %q", stdout, command.want)
							}
							if command.args[0] == "stats" && !strings.Contains(stdout, "\tFASTA\tDNA\t10\t1000\t100\t100.0\t100\n") {
								t.Fatalf("unexpected stats: %q", stdout)
							}
						}
					}
				})
			}
		}
	}
}

// TestCompressedFastqErrors checks quality preservation and checksum/trailer errors.
func TestCompressedFastqErrors(t *testing.T) {
	input, _ := compressedRecords(true, 10, 100)
	for _, extension := range []string{".gz", ".xz", ".zst", ".bz2", ".lz4"} {
		t.Run(extension, func(t *testing.T) {
			encoded := compressedInput(t, extension, input)
			stdout, stderr, err := runCLI(t, string(encoded), "seq", "-w", "0")
			if err != nil || stdout != input {
				t.Fatalf("valid FASTQ: err=%v stdout=%q stderr=%s", err, stdout, stderr)
			}
			corrupt := append([]byte(nil), encoded...)
			offset := 4 // Stored checksum location, measured from the end of the frame.
			switch extension {
			case ".gz":
				offset = 8
			case ".xz":
				offset = 12
			case ".bz2":
				offset = 2
			}
			corrupt[len(corrupt)-offset] ^= 0x80
			for name, data := range map[string][]byte{"truncated": encoded[:60], "trailer": encoded[:len(encoded)-1], "checksum": corrupt} {
				t.Run(name, func(t *testing.T) {
					stdout, stderr, err := runCLI(t, string(data), "seq", "-w", "0")
					if err == nil || !strings.Contains(stderr, "[ERRO]") {
						t.Fatalf("damage hidden: err=%v stdout=%q stderr=%s", err, stdout, stderr)
					}
					// Output may contain earlier complete records, never a partial one.
					if !strings.HasPrefix(input, stdout) || strings.Count(stdout, "\n")%4 != 0 {
						t.Fatalf("partial FASTQ output: %q", stdout)
					}
				})
			}
		})
	}
}

// TestCompressedStatsSkipError retains the explicit warning-and-continue mode.
func TestCompressedStatsSkipError(t *testing.T) {
	input, _ := compressedRecords(false, 10, 100)
	data := compressedInput(t, ".xz", input)
	dir := t.TempDir()
	bad := auditInput(t, dir, "bad.fa.xz", string(data[:60]))
	good := auditInput(t, dir, "good.fa", input)
	for _, skip := range []bool{false, true} {
		args := []string{"stats", "-T", bad, good}
		if skip {
			args = append(args, "--skip-err")
		}
		stdout, stderr, err := runCLI(t, "", args...)
		if (err == nil) != skip || !strings.Contains(stderr, bad) || !strings.Contains(stdout, good+"\t") || strings.Contains(stdout, bad+"\t") {
			t.Fatalf("skip=%v: err=%v stdout=%q stderr=%s", skip, err, stdout, stderr)
		}
		if skip && !strings.Contains(stderr, "[WARN]") {
			t.Fatalf("missing skip warning: %s", stderr)
		}
	}
}

// TestCompressedHeadEarlyExit does not validate unread tails after enough records.
func TestCompressedHeadEarlyExit(t *testing.T) {
	input, _ := compressedRecords(false, 20000, 100)
	firstEnd := strings.Index(input, ">s1\n")
	for _, extension := range []string{".gz", ".xz", ".zst", ".bz2", ".lz4"} {
		t.Run(extension, func(t *testing.T) {
			encoded := compressedInput(t, extension, input)
			// Keep enough complete blocks that head's initial read cannot reach the cut.
			data := encoded[:len(encoded)-1]
			stdout, stderr, err := runCLI(t, string(data), "head", "-n", "1", "-w", "0")
			if err != nil || stdout != input[:firstEnd] {
				t.Fatalf("early exit changed: err=%v stdout=%q stderr=%s", err, stdout, stderr)
			}
			_, stderr, err = runCLI(t, string(data), "head", "-n", strconv.Itoa(20001))
			if err == nil || !strings.Contains(stderr, "[ERRO]") {
				t.Fatalf("full read hid damage: err=%v stderr=%s", err, stderr)
			}
		})
	}
}
