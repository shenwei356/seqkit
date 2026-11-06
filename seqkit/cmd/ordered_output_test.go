package cmd

import (
	"testing"
	"time"
)

// TestConsumeOrderedDrainsReadyResults keeps input open after filling a gap.
func TestConsumeOrderedDrainsReadyResults(t *testing.T) {
	input := make(chan uint64, 3)
	output := make(chan uint64, 3)
	done := make(chan struct{})
	go func() {
		consumeOrdered(input, func(id uint64) uint64 { return id }, func(id uint64) { output <- id })
		close(done)
	}()
	defer func() { close(input); <-done }()
	input <- 2
	input <- 3
	input <- 1
	for want := uint64(1); want <= 3; want++ {
		select {
		case got := <-output:
			if got != want {
				t.Fatalf("result = %d, want %d", got, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("ready result %d waited for more input", want)
		}
	}
}
