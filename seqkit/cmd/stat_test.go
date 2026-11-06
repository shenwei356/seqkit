package cmd

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestStatsZeroLengthRecords reports finite zero metrics without discarding records.
func TestStatsZeroLengthRecords(t *testing.T) {
	for _, test := range []struct {
		input string
		count int
	}{
		{"", 0}, {">a\n", 1}, {">a\n>b\n", 2},
		{"@a\n\n+\n\n", 1}, {"@a\n\n+\n\n@b\n\n+\n\n", 2},
	} {
		out, stderr, err := runCLI(t, test.input, "stats", "-Ta", "-N", "0,50,100")
		if err != nil {
			t.Fatalf("empty records rejected: %v %s", err, stderr)
		}
		rows := strings.Split(strings.TrimSpace(out), "\n")
		if len(rows) != 2 {
			t.Fatalf("expected one statistics row, got %d", len(rows)-1)
		}
		header, values := strings.Split(rows[0], "\t"), strings.Split(rows[1], "\t")
		for i := 3; i < len(header); i++ {
			want := float64(0)
			if header[i] == "num_seqs" {
				want = float64(test.count)
			}
			got, err := strconv.ParseFloat(values[i], 64)
			if err != nil || got != want {
				t.Fatalf("count=%d %s=%s, want %g", test.count, header[i], values[i], want)
			}
		}
	}
}

// TestStatsInvalidN rejects nonfinite percentiles before opening output.
func TestStatsInvalidN(t *testing.T) {
	for _, value := range []string{"NaN", "Inf", "-Inf"} {
		output := auditInput(t, t.TempDir(), "out.tsv", "keep\n")
		_, stderr, err := runCLI(t, ">r\nAC\n", "stats", "-N", value, "-o", output)
		data, readErr := os.ReadFile(output)
		if err == nil || !strings.Contains(stderr, "range of [0, 100]") || readErr != nil || string(data) != "keep\n" {
			t.Fatalf("invalid percentile changed output: %v %s", err, stderr)
		}
	}
}

// TestStatsSolexa computes Phred statistics from Solexa error probabilities.
func TestStatsSolexa(t *testing.T) {
	for _, qualities := range []string{";", "@", ";@ST]^"} {
		out, stderr, err := runCLI(t, "@r\n"+strings.Repeat("A", len(qualities))+"\n+\n"+qualities+"\n", "stats", "-Ta", "-E", "solexa")
		if err != nil {
			t.Fatalf("valid Solexa rejected: %v %s", err, stderr)
		}
		rows := strings.Split(strings.TrimSpace(out), "\n")
		if len(rows) != 2 {
			t.Fatalf("expected one statistics row")
		}
		header, values := strings.Split(rows[0], "\t"), strings.Split(rows[1], "\t")
		fields := make(map[string]string)
		for i, name := range header {
			fields[name] = values[i]
		}
		probability := 0.0
		q20, q30 := 0, 0
		for _, q := range []byte(qualities) {
			score := int(q) - 64
			probability += 1 / (1 + math.Pow(10, float64(score)/10))
			phred := 10 * math.Log10(1+math.Pow(10, float64(score)/10))
			if phred >= 20 {
				q20++
			}
			if phred >= 30 {
				q30++
			}
		}
		want := fmt.Sprintf("%.2f", -10*math.Log10(probability/float64(len(qualities))))
		if fields["AvgQual"] != want {
			t.Fatalf("AvgQual=%s want=%s", fields["AvgQual"], want)
		}
		for field, count := range map[string]int{"Q20(%)": q20, "Q30(%)": q30} {
			got, err := strconv.ParseFloat(fields[field], 64)
			if err != nil || got != math.Round(100*float64(count)/float64(len(qualities))) {
				t.Fatalf("%s=%s", field, fields[field])
			}
		}
	}
	_, stderr, err := runCLI(t, "@r\nA\n+\n:\n", "stats", "-Ta", "-E", "solexa")
	if err == nil || !strings.Contains(stderr, "invalid Solexa quality") {
		t.Fatalf("invalid Solexa accepted: %v %s", err, stderr)
	}
}
