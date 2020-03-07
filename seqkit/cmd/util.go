// Copyright © 2019 Oxford Nanopore Technologies.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	au "github.com/logrusorgru/aurora"
	colorable "github.com/mattn/go-colorable"
	isatty "github.com/mattn/go-isatty"
)

// ColorCycler is a utilty object to cycle between colors and colorize text.
type ColorCycler struct {
	Dummy   bool
	Index   int
	Palette []au.Color
}

// NewColorCycler return a new color cycler object.
func NewColorCycler(dummy bool) *ColorCycler {
	self := new(ColorCycler)
	self.Index = 0
	self.Dummy = dummy
	self.Palette = []au.Color{
		au.RedFg,
		au.GreenFg,
		au.YellowFg,
		au.BlueFg,
		au.MagentaFg,
		au.CyanFg,
	}
	const flagFg au.Color = 1 << 14 // presence flag (14th bit)
	const shiftFg = 16              // shift for foreground (starting from 16th bit)
	for i := uint8(19); i <= 216; i += 5 {
		self.Palette = append(self.Palette, au.Color(i)<<shiftFg|flagFg)
	}
	return self
}

// Next swiches to the next color.
func (p *ColorCycler) Next() {
	if p.Dummy {
		return
	}
	p.Index++
	if p.Index >= len(p.Palette)-1 {
		p.Index = 0
	}
}

// Colorize adds the current ANSI color to the text.
func (p *ColorCycler) Colorize(s string) string {
	if p.Dummy {
		return s
	}
	return au.Sprintf(au.Colorize(s, p.Palette[p.Index]))
}

// Colorize adds the current ANSI color to the text with a header style.
func (p *ColorCycler) Header(s string) string {
	if p.Dummy {
		return s
	}
	return au.Sprintf(au.BgGray(5, au.Colorize(s, p.Palette[p.Index]|au.BoldFm)))
}

// Fancy colorizes text with normal or header styles.
func (p *ColorCycler) Fancy(s string, head bool) string {
	switch head {
	case false:
		return p.Colorize(s)
	case true:
		return p.Header(s)
	}
	return s
}

// WrapWriter wraps a file into am go-colorable object if necessary.
func (p *ColorCycler) WrapWriter(fh *os.File) io.Writer {
	if p.Dummy || !isatty.IsTerminal(fh.Fd()) {
		return fh
	}
	return colorable.NewColorable(fh)
}

// BashExec executes a command via bash.
func BashExec(command string) {
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed running command: %s - %s\n", command, err)
		os.Exit(1)
	}

}

// FileSize gets size of a file by calling os.Stat.
func FileSize(file string) int {
	info, err := os.Stat(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not stat file %s: %s\n", file, err)
		os.Exit(1)
	}
	return int(info.Size())
}

// ReverseInt revsrees a slice of integers.
func ReverseInt(d []int) []int {
	s := make([]int, len(d))
	copy(d, s)
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

// FileExists checks if a file exists by calling os.Stat.
func FileExists(fn string) bool {
	_, err := os.Stat(fn)
	if err == nil {
		return true
	}
	return false
}

// MinInts calculates the minimum of a slice of integers.
func MinInts(s []int) (m int) {
	m = s[0]
	for _, e := range s {
		if e < m {
			m = e
		}
	}
	return
}

// MaxInts calculates the maximum of a slice of integers.
func MaxInts(s []int) (m int) {
	for _, e := range s {
		if e > m {
			m = e
		}
	}
	return
}

// SumInts calculates the sum of a slice of integers.
func SumInts(s []int) (r int) {
	for _, e := range s {
		r += e
	}
	return
}

// RevCompDNA reverse complements a DNA sequence string.
func RevCompDNA(s string) string {
	size := len(s)
	s = strings.ToUpper(s)
	tmp := make([]byte, size)
	var inBase byte
	var outBase byte
	for i := 0; i < size; i++ {
		inBase = s[i]
		switch inBase {
		case 'A':
			outBase = 'T'
		case 'T':
			outBase = 'A'
		case 'G':
			outBase = 'C'
		case 'C':
			outBase = 'G'
		default:
			outBase = 'N'
		}
		tmp[size-1-i] = outBase
	}
	return string(tmp)
}

func maxStrLen(slice []string) int {
	l := 0
	for _, s := range slice {
		if len(s) > l {
			l = len(s)
		}
	}
	return l
}

// PrettyPrintTsv pretty prints and optionally colorizes a "data frame".
func PrettyPrintTsv(cols []string, fields [][]string, width int, color bool) (string, *ColorCycler) {
	brush := NewColorCycler(!color)
	nrCols := len(cols)
	if nrCols != len(fields) {
		panic("Length mismatch!")
	}
	out := make([][]string, nrCols)
	for i := 0; i < nrCols; i++ {
		out[i] = []string{cols[i]}
		out[i] = append(out[i], fields[i]...)
	}
	auto := false
	if width < 0 {
		auto = true
	}
	prevCol := 0
	for i := 0; i < nrCols; i++ {
		width := 0
		if auto {
			width = maxStrLen(out[i])
		}
		if i > prevCol {
			brush.Next()
			prevCol++
		}
		for j := 0; j < len(out[i]); j++ {
			head := false
			if j == 0 {
				head = true
			}
			sep := "\t"
			if width > 0 {
				out[i][j] = brush.Fancy(fmt.Sprintf("%-"+strconv.Itoa(width)+"s"+sep, out[i][j]), head)
			} else {
				out[i][j] = brush.Fancy(fmt.Sprintf("%s"+sep, out[i][j]), head)
			}
		}
	}
	outStr := ""
	rows := len(out[0])
	for i := 0; i < rows; i++ {
		tmp := make([]string, len(out))
		for j := 0; j < len(out); j++ {
			tmp[j] = out[j][i]
		}
		outStr += strings.Join(tmp, "") + "\n"
	}
	return outStr, brush
}
