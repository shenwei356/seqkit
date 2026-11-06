package cmd

import "testing"

// TestTab2fxBufferValidation preserves existing output for invalid buffer limits.
func TestTab2fxBufferValidation(t *testing.T) { assertInvalidBufferSizes(t, "tab2fx") }
