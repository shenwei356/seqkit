package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/botond-sipos/thist"
	"golang.org/x/term"
	"gonum.org/v1/plot/vg/draw"
)

// newHistogram applies terminal fitting with a positive bin limit.
func newHistogram(data []float64, title, mode string, bins int, normalize bool) *thist.Hist {
	if mode == "termfit" {
		mode, bins = "fit", terminalHistogramBins()
	}
	return thist.NewHist(data, title, mode, bins, normalize)
}

// histogramText avoids the dependency's undefined empty-data mode calculation.
func histogramText(h *thist.Hist, dump bool) string {
	if h.DataCount == 0 {
		return ""
	}
	if dump {
		return h.Dump()
	}
	return h.Draw()
}

// validateHistogramImage uses the same registered formats as plot.Save.
func validateHistogramImage(path string) error {
	if path == "" {
		return nil
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	for _, format := range draw.Formats() {
		if ext == format {
			return nil
		}
	}
	return fmt.Errorf("unsupported image format: %q", path)
}

// saveHistogram publishes a complete image and preserves an existing target on failure.
func saveHistogram(h *thist.Hist, path string) (err error) {
	if h.DataCount == 0 {
		return nil
	}
	if err = validateHistogramImage(path); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".seqkit-image-*"+filepath.Ext(path))
	if err != nil {
		return fmt.Errorf("saving image %s: %w", path, err)
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Close(); err != nil {
		return err
	}
	// SaveImage reports plot.Save errors by panic; contain that legacy interface.
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("saving image %s: %v", path, value)
		}
	}()
	h.SaveImage(tmp)
	if err = os.Rename(tmp, path); err != nil {
		return fmt.Errorf("saving image %s: %w", path, err)
	}
	return nil
}

// histogramClear leaves the terminal untouched when there is nothing to draw.
func histogramClear(h *thist.Hist) string {
	if h.DataCount == 0 {
		return ""
	}
	return thist.ClearScreenString()
}

// terminalHistogramBins clamps the available plotting width, including after resize.
func terminalHistogramBins() int {
	columns, _, err := term.GetSize(int(os.Stderr.Fd()))
	if err != nil {
		columns = 80
	}
	return max(1, columns-10)
}

// updateHistogram retains automatic terminal fitting as the terminal is resized.
func updateHistogram(h *thist.Hist, value float64) {
	if h.BinMode == "fit" {
		h.MaxBins = terminalHistogramBins()
	}
	h.Update(value)
}
