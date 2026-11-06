package cmd

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
)

// restoreDescription skips the entire header separator without consuming text.
// Callers opt in so custom-ID behavior in other commands remains unchanged.
func restoreDescription(record *fastx.Record) {
	i := bytes.IndexAny(record.Name, " \t")
	if i < 0 {
		record.Desc = nil
		return
	}
	for i < len(record.Name) && (record.Name[i] == ' ' || record.Name[i] == '\t') {
		i++
	}
	record.Desc = record.Name[i:]
}

// readRecordsChecked never returns a usable prefix of damaged input.
func readRecordsChecked(file string, alphabet *seq.Alphabet, idRegexp string) ([]*fastx.Record, error) {
	r, err := fastx.NewReader(alphabet, file, idRegexp)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var records []*fastx.Record
	for {
		record, err := r.Read()
		if err == io.EOF {
			return records, nil
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		records = append(records, record.Clone())
	}
}

// nonemptySubLocation clips a region once and rejects empty intersections.
func nonemptySubLocation(length, start, end int) (int, int, bool) {
	if length == 0 {
		return 0, 0, false
	}
	if start < 0 {
		start = length + start + 1
	}
	if end < 0 {
		end = length + end + 1
	}
	start, end = max(1, start), min(length, end)
	return start, end, start <= end
}

// parseSequenceLength uses decimal biological units, truncating fractional bases.
func parseSequenceLength(value string) (int64, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return 0, fmt.Errorf("invalid sequence length: %q", value)
	}

	multiplier := int64(1)
	unit := true
	switch v[len(v)-1] {
	case 'k', 'K':
		multiplier = 1000
	case 'm', 'M':
		multiplier = 1000000
	case 'g', 'G':
		multiplier = 1000000000
	case 't', 'T':
		multiplier = 1000000000000
	case 'b', 'B':
	default:
		unit = false
	}
	if unit {
		v = strings.TrimSpace(v[:len(v)-1])
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, fmt.Errorf("invalid sequence length: %q", value)
	}
	if n <= 0 {
		return 0, nil
	}
	// Exact arithmetic keeps MaxInt64 valid and avoids float-to-int overflow.
	amount, ok := new(big.Rat).SetString(v)
	if !ok {
		return 0, fmt.Errorf("invalid sequence length: %q", value)
	}
	amount.Mul(amount, new(big.Rat).SetInt64(multiplier))
	integer := new(big.Int).Quo(amount.Num(), amount.Denom())
	if !integer.IsInt64() {
		return 0, fmt.Errorf("invalid sequence length: %q", value)
	}
	return integer.Int64(), nil

}

// equalFoldBase compares ASCII sequence characters without changing their text.
func equalFoldBase(a, b byte) bool {
	if a >= 'a' && a <= 'z' {
		a -= 'a' - 'A'
	}
	if b >= 'a' && b <= 'z' {
		b -= 'a' - 'A'
	}
	return a == b
}
