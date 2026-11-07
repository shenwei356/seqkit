package cmd

import (
	"strconv"
	"strings"
	"testing"

	"github.com/shenwei356/bio/seqio/fastx"
)

// TestRangeMixedFileFormats preserves each input file's format.
func TestRangeMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "range", "-r", "1:-1")
}

// TestRangeStopsAtEnd does not parse records beyond the requested positive range.
func TestRangeStopsAtEnd(t *testing.T) {
	out, stderr, err := runCLI(t, "@r1\nAC\n+\n!!\n@r2\nAC\n+\n!\n", "range", "-r", "1:1")
	if err != nil || out != "@r1\nAC\n+\n!!\n" {
		t.Fatalf("read past selected range: %q %v %s", out, err, stderr)
	}
}

// TestRangeMinimumInteger clips negative ranges without overflowing their magnitude.
func TestRangeMinimumInteger(t *testing.T) {
	minimum := strconv.Itoa(-int(^uint(0)>>1) - 1)
	for _, tc := range []struct{ region, want string }{
		{minimum + ":-1", ">r1\nAC\n>r2\nGT\n"},
		{minimum + ":" + minimum, ""},
	} {
		out, stderr, err := runCLI(t, ">r1\nAC\n>r2\nGT\n", "range", "-r", tc.region)
		if err != nil || out != tc.want {
			t.Fatalf("range %s: %q %v %s", tc.region, out, err, stderr)
		}
	}
}

// TestRangeSkippedFastqRestoresWrapping keeps the next FASTA file's default width.
func TestRangeSkippedFastqRestoresWrapping(t *testing.T) {
	dir := t.TempDir()
	fq := auditInput(t, dir, "short.fq", "@q1\nAC\n+\n!!\n")
	fa := auditInput(t, dir, "long.fa", ">r1\n"+strings.Repeat("A", 70)+"\n>r2\nAC\n")
	out, stderr, err := runCLI(t, "", "range", "-r", "-2:-2", fq, fa)
	want := ">r1\n" + strings.Repeat("A", 60) + "\n" + strings.Repeat("A", 10) + "\n"
	if err != nil || out != want {
		t.Fatalf("FASTA wrapping after skipped FASTQ: %q %v %s", out, err, stderr)
	}
}

// TestRecordLoopBufferWrap preserves order and links after overwriting old records.
func TestRecordLoopBufferWrap(t *testing.T) {
	for _, capacity := range []int{1, 3} {
		buffer, err := NewRecordLoopBuffer(capacity)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 10; i++ {
			buffer.Add(&fastx.Record{ID: []byte(strconv.Itoa(i))})
		}
		if buffer.Size != capacity {
			t.Fatalf("size = %d", buffer.Size)
		}
		for i := 10 - capacity; i < 10; i++ {
			node := buffer.Next()
			if string(node.Value.ID) != strconv.Itoa(i) || node.next.prev != node || node.prev.next != node {
				t.Fatalf("invalid ring at record %d", i)
			}
		}
	}
}

// BenchmarkRecordLoopBufferFull isolates the steady-state node replacement cost.
func BenchmarkRecordLoopBufferFull(b *testing.B) {
	buffer, _ := NewRecordLoopBuffer(10)
	record := &fastx.Record{ID: []byte("s")}
	for i := 0; i < 10; i++ {
		buffer.Add(record)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer.Add(record)
	}
}
