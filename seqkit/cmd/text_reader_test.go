package cmd

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// assertDamagedTextInput exercises a command's auxiliary-file read before output.
func assertDamagedTextInput(t *testing.T, content string, args ...string) {
	t.Helper()
	data := compressedInput(t, ".gz", strings.Repeat(content, 10000))
	data[len(data)-8] ^= 0xff
	dir := t.TempDir()
	aux := auditInput(t, dir, "damaged.txt.gz", string(data))
	input := auditInput(t, dir, "input.fa", ">r\nACGT\n")
	output := auditInput(t, dir, "out.txt", "keep\n")
	_, stderr, err := runCLI(t, "", append(args, aux, "-o", output, input)...)
	got, readErr := os.ReadFile(output)
	if err == nil || !strings.Contains(stderr, "[ERRO]") || strings.Contains(stderr, "panic:") || readErr != nil || string(got) != "keep\n" {
		t.Fatalf("damaged auxiliary input did not fail safely: %v %s", err, stderr)
	}
}

// TestReadTextLines preserves long, blank, CRLF, and unterminated lines.
func TestReadTextLines(t *testing.T) {
	for _, content := range []string{"", "a\r\n\nb", strings.Repeat("x", 128<<10) + "\n"} {
		path := auditInput(t, t.TempDir(), "input.txt", content)
		var got []string
		for line, err := range readTextLines(path) {
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, line)
		}
		var want []string
		if content != "" {
			want = strings.Split(strings.TrimSuffix(content, "\n"), "\n")
			for i := range want {
				want[i] = strings.TrimRight(want[i], "\r")
			}
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatal("line contents changed")
		}
	}
}
