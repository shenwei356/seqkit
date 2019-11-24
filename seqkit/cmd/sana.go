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
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

type SeqStreamCtrl int

const (
	StreamTry SeqStreamCtrl = iota
	StreamQuit
	StreamEOF
	StreamExited
)

// sanaCmd represents the sana command
var sanaCmd = &cobra.Command{
	Use:   "sana",
	Short: "sanitize broken single line fastq files",
	Long:  "sanitize broken single line fastq files",

	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		outFile := config.OutFile
		qBase := getFlagPositiveInt(cmd, "qual-ascii-base")
		runtime.GOMAXPROCS(config.Threads)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Flush()
		defer outfh.Close()

		files := getFileList(args)

		for _, file := range files {
			rawSeqChan := NewRawSeqStreamFromFile(file, 1000, qBase, "fasta")

			for rawSeq := range rawSeqChan {
				switch rawSeq.Err {
				case nil:
					outfh.WriteString(rawSeq.String() + "\n")
				default:
					os.Stderr.WriteString(rawSeq.String() + "\n")
				}
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(sanaCmd)
	sanaCmd.Flags().IntP("qual-ascii-base", "b", 33, "ASCII BASE, 33 for Phred+33")
}

// simpleSeq is a structure holding basic sequnce information with qualities.
type simpleSeq struct {
	Id        string
	Seq       string
	Qual      []int
	QBase     int
	Err       error
	StartLine int
}

// String generates a string representation of a pointer to simpleSeq.
func (s *simpleSeq) String() string {
	if s.Err != nil {
		return fmt.Sprintf("%s\t%d:\t %s", s.Err, s.StartLine, s.Seq)
	}
	if len(s.Qual) == 0 {
		return fmt.Sprintf(">%s\n%s", s.Id, s.Seq)
	}
	qs := make([]string, len(s.Qual))
	for i, qq := range s.Qual {
		qs[i] = string(qq + s.QBase)
	}
	return fmt.Sprintf("@%s\n%s\n+\n%s", s.Id, s.Seq, strings.Join(qs, ""))
}

// IUPACBases is a map of valid IUPAC bases.
var IUPACBases map[byte]bool

func init() {
	IUPACBases = map[byte]bool{
		'A': true,
		'C': true,
		'G': true,
		'T': true,
		'R': true,
		'Y': true,
		'S': true,
		'W': true,
		'K': true,
		'M': true,
		'B': true,
		'D': true,
		'H': true,
		'V': true,
		'N': true,
		'U': true,
		'a': true,
		'c': true,
		'g': true,
		't': true,
		'r': true,
		'y': true,
		's': true,
		'w': true,
		'k': true,
		'm': true,
		'b': true,
		'd': true,
		'h': true,
		'v': true,
		'n': true,
		'u': true,
	}

}

// validateSeqBytes check for illegal bases.
func validateSeqBytes(dna []byte) error {
	for i, base := range dna {
		if !IUPACBases[base] {
			return errors.New(fmt.Sprintf("Illegal base '%s' at position %d", string(base), i))
		}
	}
	return nil
}

// validateSeqString check for illegal bases.
func validateSeqString(dna string) error {
	return validateSeqBytes([]byte(dna))
}

// validateQuals checks for negative quality values.
func validateQuals(quals []int) error {
	for i, qual := range quals {
		if qual < 0 {
			return errors.New(fmt.Sprintf("Illegal quality value '%d' at position %d", qual, i))
		}
	}
	return nil
}

// ValidateSeq validates simpleSeq objects.
func ValidateSeq(seq *simpleSeq) error {
	if len(seq.Seq) != len(seq.Qual) {
		return errors.New(fmt.Sprintf("Sequence (%d) and quality (%d) length mismatch", len(seq.Seq), len(seq.Qual)))
	}
	if seqErr := validateSeqString(seq.Seq); seqErr != nil {
		return seqErr
	}
	if qualErr := validateQuals(seq.Qual); qualErr != nil {
		return qualErr
	}
	return nil
}

// NewRawSeqStream initializes a new channel for reading fastq records from a file in a robust way.
func NewRawSeqStreamFromFile(inFastq string, chanSize int, qBase int, format string) chan *simpleSeq {
	r, err := xopen.Ropen(inFastq)
	checkError(err)
	buffSize := 100 * 1024
	ctrlChan := make(chan SeqStreamCtrl, 0)
	go func() {
		for {
			select {
			case i := <-ctrlChan:
				if i == StreamEOF {
					ctrlChan <- StreamQuit
					i = <-ctrlChan
					if i == StreamExited {
						return
					}
				}
			case ctrlChan <- StreamTry:
			default:
			}
		}
	}()
	switch format {
	case "fastq":
		return NewRawFastqStream(bufio.NewReaderSize(r, buffSize), chanSize, qBase, inFastq, ctrlChan)
	case "fasta":
		return NewRawFastaStream(bufio.NewReaderSize(r, buffSize), chanSize, inFastq, ctrlChan)
	}
	return nil
}

type FqlState struct {
	Header  bool
	Seq     bool
	Plus    bool
	Qual    bool
	Partial bool
	Invalid bool
}

type FqLine struct {
	Line     string
	FqlState FqlState
}
type FqLines []FqLine

func guessFqlState(line []byte) FqlState {
	state := FqlState{}
	switch line[0] {
	case '@':
		state.Header = true
		state.Qual = true
	case '+':
		if len(line) == 1 {
			state.Plus = true
		} else {
			state.Qual = true
		}
	default:
		state.Seq = true
		state.Qual = true
	}
	return state
}

func guessFasState(line []byte) FqlState {
	state := FqlState{}
	switch line[0] {
	case '>':
		if len(line) > 1 {
			state.Header = true
		} else {
			state.Invalid = true
		}
	default:
		if validateSeqBytes(line) == nil {
			state.Seq = true
		} else {
			state.Invalid = true
		}
	}
	return state
}

func FqLinesToSimpleSeq(lines FqLines, qBase int) (*simpleSeq, error) {
	if len(lines) != 4 {
		return nil, errors.New("Line buffer must have 4 lines!")
	}
	lh, ls, lp, lq := &lines[0], &lines[1], &lines[2], &lines[3]
	if lh.FqlState.Header && ls.FqlState.Seq && lp.FqlState.Plus && lq.FqlState.Qual {
		seq := &simpleSeq{lh.Line[1:], ls.Line, parseQuals(lq.Line, qBase), qBase, nil, -1}
		seq.Err = ValidateSeq(seq)
		if seq.Err != nil {
			return nil, seq.Err
		}
		return seq, seq.Err
	} else {
		return nil, errors.New("Invalid line states!")
	}
	return nil, nil
}

func FasLinesToSimpleSeq(lines FqLines) (*simpleSeq, error) {
	if len(lines) < 2 {
		return nil, errors.New("Line buffer must have 4 lines!")
	}
	if !lines[0].FqlState.Header {
		return nil, errors.New("Missing header line!")
	}
	s := &simpleSeq{Id: lines[0].Line[1:]}
	for i := 1; i < len(lines); i++ {
		if lines[i].FqlState.Invalid && !lines[i].FqlState.Seq {
			return nil, errors.New("Invalid line structure!")
		}
		s.Seq += lines[i].Line
	}
	return s, nil
}

func streamFastq(r *bufio.Reader, sbuff FqLines, out chan *simpleSeq, ctrlChan chan SeqStreamCtrl, lineCounter *int, qBase int) (FqLines, error) {
	var line []byte
	var lastLine *FqLine
	if len(sbuff) > 0 {
		lastLine = &sbuff[len(sbuff)-1]
	}
	var err error
	for {
		line, err = r.ReadBytes('\n')
		switch err {
		case nil:
			line = bytes.Trim(line, "\n\t ")
			if len(line) == 0 {
				*lineCounter++
				continue
			}
			if lastLine != nil && lastLine.FqlState.Partial {
				*lineCounter++
				lastLine.Line += string(line)
				lastLine.FqlState = guessFqlState([]byte(lastLine.Line))
			} else {
				*lineCounter++
				lineStr := string(line)
				sbuff = append(sbuff, FqLine{lineStr, guessFqlState(line)})
				lastLine = &sbuff[len(sbuff)-1]
			}
			if len(sbuff) == 4 && !lastLine.FqlState.Partial {
				seq, err := FqLinesToSimpleSeq(sbuff, qBase)
				if err == nil {
					seq.StartLine = *lineCounter - 4
					out <- seq
					sbuff = sbuff[:0]
				} else {
					h := -1
					for i := 1; i < len(sbuff); i++ {
						if sbuff[i].FqlState.Header {
							h = i
							break
						}
					}
					if h < 0 {
						h = len(sbuff)
					}
					for j := 0; j < h; j++ {
						serr := &simpleSeq{StartLine: *lineCounter - j, Err: errors.New("Discarded line"), Seq: sbuff[j].Line}
						out <- serr
					}
					sbuff = sbuff[h:]

				}
			} //sbuff == 4
		case io.EOF:
			line = bytes.TrimRight(line, "\n")
			if len(line) > 0 {
				sbuff = append(sbuff, FqLine{string(line), FqlState{Partial: true}})
			}
			ctrlChan <- StreamEOF
			return sbuff, nil
		default:
			return sbuff, err
		}

	}
	return sbuff, nil
}

func streamFasta(r *bufio.Reader, sbuff FqLines, out chan *simpleSeq, ctrlChan chan SeqStreamCtrl, lineCounter *int, final bool) (FqLines, error) {
	var line []byte
	var lastLine *FqLine
	if len(sbuff) > 0 {
		lastLine = &sbuff[len(sbuff)-1]
	}
	var err error
	for {
		line, err = r.ReadBytes('\n')
		switch err {
		case nil:
			line = bytes.TrimRight(line, "\n\t ")
			if len(line) == 0 {
				*lineCounter++
				continue
			}
			if lastLine != nil && lastLine.FqlState.Partial {
				*lineCounter++
				lastLine.Line += string(line)
				lastLine.FqlState = guessFasState([]byte(lastLine.Line))
			} else {
				*lineCounter++
				lineStr := string(line)
				sbuff = append(sbuff, FqLine{lineStr, guessFasState(line)})
				lastLine = &sbuff[len(sbuff)-1]
			}
			if len(sbuff) > 2 && !lastLine.FqlState.Partial {
				if sbuff[0].FqlState.Header && (lastLine.FqlState.Header || final) {
					seq, err := FasLinesToSimpleSeq(sbuff[:len(sbuff)-1])
					if err == nil {
						seq.StartLine = *lineCounter - 4
						out <- seq
						sbuff = sbuff[len(sbuff)-1:]
					} else {
						for j := 0; j < len(sbuff); j++ {
							serr := &simpleSeq{StartLine: *lineCounter - j, Err: errors.New("Discarded line"), Seq: sbuff[j].Line}
							out <- serr
						}
					}
					sbuff = sbuff[len(sbuff)-1:]
				}
			}
		case io.EOF:
			line = bytes.TrimRight(line, "\n")
			if len(line) > 0 {
				sbuff = append(sbuff, FqLine{string(line), FqlState{Partial: true}})
			}
			ctrlChan <- StreamEOF
			return sbuff, nil
		default:
			return sbuff, err
		}

	}
	return sbuff, nil
}

// NewRawSeqStream initializes a new channel for reading fastq records in a robust way.
func NewRawFastqStream(inReader *bufio.Reader, chanSize int, qBase int, id string, ctrlChan chan SeqStreamCtrl) chan *simpleSeq {
	seqChan := make(chan *simpleSeq, chanSize)
	lineCounter := 0

	go func() {
		sbuff := make(FqLines, 0, 100)
		var err error
		_ = err

		for {
			select {
			case cmd := <-ctrlChan:
				if cmd == StreamTry {
					sbuff, err = streamFastq(inReader, sbuff, seqChan, ctrlChan, &lineCounter, qBase)
					if err != nil {
						panic(err)
					}

				} else if cmd == StreamQuit {
					sbuff, err = streamFastq(inReader, sbuff, seqChan, ctrlChan, &lineCounter, qBase)
					for i, l := range sbuff {
						serr := &simpleSeq{Err: errors.New("Discarded line"), StartLine: lineCounter - i, Seq: l.Line}
						seqChan <- serr
					}
					close(seqChan)
					ctrlChan <- StreamExited
					return
				} else {
					panic("Invalid stream control command!")
				}
			}
		}
	}()

	return seqChan
}

// NewRawSeqStream initializes a new channel for reading fastq records in a robust way.
func NewRawFastaStream(inReader *bufio.Reader, chanSize int, id string, ctrlChan chan SeqStreamCtrl) chan *simpleSeq {
	seqChan := make(chan *simpleSeq, chanSize)
	lineCounter := 0

	go func() {
		sbuff := make(FqLines, 0, 100)
		var err error
		_ = err

		for {
			select {
			case cmd := <-ctrlChan:
				if cmd == StreamTry {
					sbuff, err = streamFasta(inReader, sbuff, seqChan, ctrlChan, &lineCounter, false)
					if err != nil {
						panic(err)
					}

				} else if cmd == StreamQuit {
					sbuff, err = streamFasta(inReader, sbuff, seqChan, ctrlChan, &lineCounter, true)
					for i, l := range sbuff {
						serr := &simpleSeq{Err: errors.New("Discarded line"), StartLine: lineCounter - i, Seq: l.Line}
						seqChan <- serr
					}
					close(seqChan)
					ctrlChan <- StreamExited
					return
				} else {
					panic("Invalid stream control command!")
				}
			}
		}
	}()

	return seqChan
}

// parseQuals parses quality string into a slice of integers.
func parseQuals(qualString string, qBase int) []int {
	quals := make([]int, len(qualString))
	for i, char := range qualString {
		quals[i] = int(char) - qBase
	}
	return quals
}
