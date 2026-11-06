package cmd

import (
	"math/rand"
	"testing"
)

// TestFishReverseComplementScores checks ambiguity and gap scoring on both strands.
func TestFishReverseComplementScores(t *testing.T) {
	rng := rand.New(rand.NewSource(29))
	params := parseAlnParams("4,-4,-2,-1")
	bases := []byte("ACGTRYSWKMBDHVN")
	for n := 0; n < 100; n++ {
		ref, query := make([]byte, 20), make([]byte, 8)
		for i := range ref {
			ref[i] = bases[rng.Intn(len(bases))]
		}
		for i := range query {
			query[i] = bases[rng.Intn(len(bases))]
		}
		align := func(ref, query string) float64 {
			return PairwiseAlignSW(&Reference{Seq: ref, Ranges: Ranges{{0, float64(len(ref))}}}, &Query{Seq: query}, params).Score
		}
		forward := align(string(ref), string(query))
		reverse := align(RevCompDNA(string(ref)), RevCompDNA(string(query)))
		if forward != reverse {
			t.Fatalf("case %d: forward score %g != reverse score %g", n, forward, reverse)
		}
	}
}
