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
	"regexp"
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

const auFlagFg au.Color = 1 << 14 // presence flag (14th bit)
const auFlagBg au.Color = 1 << 15 // presence flag (15th bit)
const auShiftFg au.Color = 16
const auShiftBg au.Color = 24
const auStart au.Color = 19
const auEnd au.Color = 216
const auSkip au.Color = 5

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
	for i := auStart; i <= auEnd; i += auSkip {
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

// SeqColorizer is a sequence colorizer object.
type SeqColorizer struct {
	NucPalette    map[byte]au.Color
	ProtPalette   map[byte]au.Color
	QualPalette   map[byte]au.Color
	QualBgPalette map[byte]au.Color
	Alphabet      string
}

// NewSeqColorizer return a new sequence colorizer object.
func NewSeqColorizer(alphabet string) *SeqColorizer {
	res := new(SeqColorizer)
	res.NucPalette = make(map[byte]au.Color)
	res.ProtPalette = make(map[byte]au.Color)
	res.QualPalette = make(map[byte]au.Color)
	res.QualBgPalette = make(map[byte]au.Color)
	switch alphabet {
	case "nucleic":
	case "amino":
	case "dummy":
	default:
		panic("Invalid alphabet: " + alphabet)
	}
	res.Alphabet = alphabet
	i := auStart
	for base, _ := range IUPACBases {
		switch base {
		case 'A', 'a':
			res.NucPalette[base] = au.GreenFg
		case 'C', 'c':
			res.NucPalette[base] = au.BlueFg
		case 'G', 'g':
			res.NucPalette[base] = au.YellowFg
		case 'T', 't':
			res.NucPalette[base] = au.RedFg
		case 'U', 'u':
			res.NucPalette[base] = au.RedFg
		case '-', '*':
			res.NucPalette[base] = au.WhiteFg
		default:
			res.NucPalette[base] = i<<auShiftFg | auFlagFg
			i += auSkip
		}
	}

	// The Lesk color scheme from http://www.bioinformatics.nl/~berndb/aacolour.html
	for aa, _ := range IUPACAminoAcids {
		switch aa {
		case 'G', 'A', 'S', 'T': // Small nonpolar
			res.ProtPalette[aa] = au.YellowFg
		case 'C', 'V', 'I', 'L', 'P', 'F', 'Y', 'M', 'W': //  Hydrophobic
			res.ProtPalette[aa] = au.GreenFg
		case 'N', 'Q', 'H': //  Polar
			res.ProtPalette[aa] = au.MagentaFg
		case 'D', 'E': //  Negatively charged
			res.ProtPalette[aa] = au.RedFg
		case 'K', 'R': //  Positively charged
			res.ProtPalette[aa] = au.BlueFg
		case 'X', 'B', 'Z': // Special
			res.ProtPalette[aa] = au.WhiteFg
		case '-', '*': // Gap
			res.ProtPalette[aa] = au.WhiteFg
		}

	}

	gb := uint8(239)
	for i := 33; i < 90; i++ {
		res.QualPalette[byte(i)] = ((au.Color(gb) << auShiftFg) | auFlagFg)
		if gb < 254 {
			gb++
		}
	}

	gb = uint8(232)
	for i := 90; i >= 33; i-- {
		res.QualBgPalette[byte(i)] = ((au.Color(gb) << auShiftBg) | auFlagBg)
		if i <= 53 && gb < 254 {
			if i%2 == 0 {
				gb++
			}
		}
	}
	return res
}

// ColorNucleic adds ANSI colors to DNA/RNA sequences.
func (p *SeqColorizer) ColorNucleic(seq []byte) []byte {
	res := make([]byte, 0, len(seq)*4)
	for _, base := range seq {
		if color, ok := p.NucPalette[base]; ok {
			res = append(res, []byte(au.Sprintf("%s", au.Colorize(string(base), color)))...)
		} else {
			res = append(res, base)
		}
	}
	return res
}

// ColorNucleic adds ANSI colors to DNA/RNA, use quality palette as background.
func (p *SeqColorizer) ColorNucleicWithQuals(seq []byte, quals []byte) []byte {
	res := make([]byte, 0, len(seq)*4)
	qIdx := 0
	for _, base := range seq {
		if color, ok := p.NucPalette[base]; ok {
			res = append(res, []byte(au.Sprintf("%s", au.Colorize(string(base), color|p.QualBgPalette[quals[qIdx]])))...)
			qIdx++
		} else {
			res = append(res, base)
		}
	}
	return res
}

// ColorAmino adds ANSI colors to protein sequences.
func (p *SeqColorizer) ColorAmino(seq []byte) []byte {
	res := make([]byte, 0, len(seq)*4)
	for _, base := range seq {
		if color, ok := p.ProtPalette[base]; ok {
			res = append(res, []byte(au.Sprintf("%s", au.Colorize(string(base), color)))...)
		} else {
			res = append(res, base)
		}
	}
	return res
}

// ColorAmino adds ANSI colors to DNA/RNA or protein sequences.
func (p *SeqColorizer) Color(seq []byte) []byte {
	switch p.Alphabet {
	case "nucleic":
		return p.ColorNucleic(seq)
	case "amino":
		return p.ColorAmino(seq)
	case "dummy":
		return seq
	default:
		return seq
	}
	return seq
}

// ColorAmino adds ANSI colors to DNA/RNA or protein sequences, use quality palette as background.
func (p *SeqColorizer) ColorWithQuals(seq []byte, quals []byte) []byte {
	switch p.Alphabet {
	case "nucleic":
		return p.ColorNucleicWithQuals(seq, quals)
	default:
		return seq
	}
	return seq
}

// ColorAmino adds grayscale colors to DNA/RNA or protein sequences.
func (p *SeqColorizer) ColorQuals(quals []byte) []byte {
	res := make([]byte, 0, len(quals)*4)
	for _, base := range quals {
		if color, ok := p.QualPalette[base]; ok {
			res = append(res, []byte(au.Sprintf("%s", au.Colorize(string(base), color)))...)
		} else {
			res = append(res, base)
		}
	}
	return res
}

// WrapWriter wraps a file into am go-colorable object if necessary.
func (p *SeqColorizer) WrapWriter(fh *os.File) io.Writer {
	if !isatty.IsTerminal(fh.Fd()) {
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

// reFilterName matches a file name to a regular expression.
func reFilterName(name string, re *regexp.Regexp) bool {
	return re.MatchString(name)
}

// checkFileFormat complains if the file format is not valid.
func checkFileFormat(format string) {
	switch format {
	case "fasta":
	case "fastq":
	case "":
	default:
		log.Fatal("Invalid format specified:", format)
	}
}
