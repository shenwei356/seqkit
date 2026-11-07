package cmd

import (
	"reflect"
	"strings"
	"testing"
)

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
