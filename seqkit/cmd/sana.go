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
	"time"

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

const NAP_SLEEP = 10 * time.Millisecond
const BIG_SLEEP = 100 * time.Millisecond

// sanaCmd represents the sana command
var sanaCmd = &cobra.Command{
	Use:   "sana",
	Short: "sanitize broken single line fastq files",
	Long:  "sanitize broken single line fastq files",

	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		outFile := config.OutFile
		qBase := getFlagPositiveInt(cmd, "qual-ascii-base")
		inFmt := getFlagString(cmd, "in-format")
		outFmt := getFlagString(cmd, "out-format")
		allowGaps := getFlagBool(cmd, "allow-gaps")
		runtime.GOMAXPROCS(config.Threads)

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Flush()
		defer outfh.Close()

		for _, file := range files {
			rawSeqChan, ctrlChanIn, ctrlChanOut := NewRawSeqStreamFromFile(file, 10000, qBase, inFmt, allowGaps)
			go func() {
				for {
					select {
					case i := <-ctrlChanOut:
						if i == StreamEOF {
							ctrlChanIn <- StreamQuit
							for j := range ctrlChanOut {
								if j == StreamExited {
									return
								} else if j == StreamEOF {
									continue
								} else {
									panic(int(i))
								}
							}
						} else if i != StreamExited {
							panic(i)
						} else {
							return
						}
					default:
						ctrlChanIn <- StreamTry
					}
				}
			}()

			pass, fail := 0, 0
			ctrlChanIn <- StreamTry
			for rawSeq := range rawSeqChan {
				switch rawSeq.Err {
				case nil:
					pass++
					outfh.WriteString(rawSeq.Format(outFmt) + "\n")
				default:
					fail++
					os.Stderr.WriteString("File: " + file + "\t" + rawSeq.String() + "\n")
				}
			}
			os.Stderr.WriteString(fmt.Sprintf("File: %s\tPass chunks: %d\tFail chunks: %d\n", file, pass, fail))
		}

	},
}

func init() {
	RootCmd.AddCommand(sanaCmd)
	sanaCmd.Flags().StringP("in-format", "I", "fastq", "input format: fastq or fasta (fastq)")
	sanaCmd.Flags().StringP("out-format", "O", "fastq", "output format: fastq or fasta (fastq)")
	sanaCmd.Flags().BoolP("allow-gaps", "A", false, "allow gap character (-) in sequences")
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

// Format generates a string representation in the specified format of a pointer to simpleSeq.
func (s *simpleSeq) Format(fmt string) string {
	if fmt == "fastq" {
		return s.FastqString()
	}
	return s.FastaString()
}

// FastqString generates a fastq string representation of a pointer to simpleSeq.
func (s *simpleSeq) FastqString() string {
	if s.Err != nil {
		return fmt.Sprintf("%s\t%d:\t %s", s.Err, s.StartLine, s.Seq)
	}
	if len(s.Qual) == 0 {
		return fmt.Sprintf("@%s\n%s\n+\n%s", s.Id, s.Seq, strings.Repeat("I", len(s.Seq)))
	}
	qs := make([]string, len(s.Qual))
	for i, qq := range s.Qual {
		qs[i] = string(qq + s.QBase)
	}
	return fmt.Sprintf("@%s\n%s\n+\n%s", s.Id, s.Seq, strings.Join(qs, ""))
}

// String generates a fasta string representation of a pointer to simpleSeq.
func (s *simpleSeq) FastaString() string {
	if s.Err != nil {
		return fmt.Sprintf("%s\t%d:\t %s", s.Err, s.StartLine, s.Seq)
	}
	return fmt.Sprintf(">%s\n%s", s.Id, s.Seq)
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
func validateSeqBytes(dna []byte, gaps bool) error {
	for i, base := range dna {
		if base == '-' && gaps {
		} else if !IUPACBases[base] {
			return errors.New(fmt.Sprintf("Illegal base '%s' at position %d", string(base), i))
		}
	}
	return nil
}

// validateSeqString check for illegal bases.
func validateSeqString(dna string, gaps bool) error {
	return validateSeqBytes([]byte(dna), gaps)
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
func ValidateSeq(seq *simpleSeq, gaps bool) error {
	if len(seq.Seq) != len(seq.Qual) {
		return errors.New(fmt.Sprintf("Sequence (%d) and quality (%d) length mismatch", len(seq.Seq), len(seq.Qual)))
	}
	if seqErr := validateSeqString(seq.Seq, gaps); seqErr != nil {
		return seqErr
	}
	if qualErr := validateQuals(seq.Qual); qualErr != nil {
		return qualErr
	}
	return nil
}

// NewRawSeqStream initializes a new channel for reading fastq records from a file in a robust way.
func NewRawSeqStreamFromFile(inFastq string, chanSize int, qBase int, format string, allowGaps bool) (chan *simpleSeq, chan SeqStreamCtrl, chan SeqStreamCtrl) {
	rio, err := os.Open(inFastq)
	checkError(err)
	buffSize := 128 * 1024
	bio := bufio.NewReaderSize(rio, buffSize)
	ctrlChanIn := make(chan SeqStreamCtrl, 10000)
	ctrlChanOut := make(chan SeqStreamCtrl, 0)

	switch format {
	case "fastq":
		return NewRawFastqStream(bio, chanSize, qBase, inFastq, ctrlChanIn, ctrlChanOut, allowGaps), ctrlChanIn, ctrlChanOut
	case "fasta":
		return NewRawFastaStream(bio, chanSize, inFastq, ctrlChanIn, ctrlChanOut, allowGaps), ctrlChanIn, ctrlChanOut
	}
	return nil, nil, nil
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

func guessFasState(line []byte, gaps bool) FqlState {
	state := FqlState{}
	switch line[0] {
	case '>':
		if len(line) > 1 {
			state.Header = true
		} else {
			state.Invalid = true
		}
	default:
		if validateSeqBytes(line, gaps) == nil {
			state.Seq = true
		} else {
			state.Invalid = true
		}
	}
	return state
}

func FqLinesToSimpleSeq(lines FqLines, qBase int, gaps bool) (*simpleSeq, error) {
	if len(lines) != 4 {
		return nil, errors.New("Line buffer must have 4 lines!")
	}
	lh, ls, lp, lq := &lines[0], &lines[1], &lines[2], &lines[3]
	if lh.FqlState.Header && ls.FqlState.Seq && lp.FqlState.Plus && lq.FqlState.Qual {
		seq := &simpleSeq{lh.Line[1:], ls.Line, parseQuals(lq.Line, qBase), qBase, nil, -1}
		seq.Err = ValidateSeq(seq, gaps)
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
		return nil, errors.New("Line buffer must have at least 2 lines!")
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

func streamFastq(r *bufio.Reader, sbuff FqLines, out chan *simpleSeq, ctrlChanIn, ctrlChanOut chan SeqStreamCtrl, lineCounter *int, qBase int, gaps bool) (FqLines, error) {
	var line []byte
	var spaceShift int
	var lastLine *FqLine
	if len(sbuff) > 0 {
		lastLine = &sbuff[len(sbuff)-1]
	}
	var err error
	for {
		if r == nil {
			log.Fatal("Buffered reader is nil!", err)
		}
		line, err = r.ReadBytes('\n')
		switch err {
		case nil:
			line = bytes.Trim(line, "\n\t ")
			if len(line) == 0 {
				*lineCounter++
				spaceShift++
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
				seq, err := FqLinesToSimpleSeq(sbuff, qBase, gaps)
				if err == nil {
					seq.StartLine = *lineCounter + spaceShift - 4
					if seq == nil {
						panic("Sequence is nil!")
					}
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
						ems := fmt.Sprintf("Discarded line: %s", err)
						serr := &simpleSeq{StartLine: (spaceShift + *lineCounter - h + j + 1), Err: errors.New(ems), Seq: sbuff[j].Line}
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
			ctrlChanOut <- StreamEOF
			return sbuff, nil
		default:
			return sbuff, err
		}

	}
	return sbuff[:0], nil
}

func streamFasta(r *bufio.Reader, sbuff FqLines, out chan *simpleSeq, ctrlChanIn, ctrlChanOut chan SeqStreamCtrl, lineCounter *int, gaps bool, final bool) (FqLines, error) {
	var line []byte
	var spaceShift int
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
				spaceShift++
				continue
			}
			if lastLine != nil && lastLine.FqlState.Partial {
				*lineCounter++
				lastLine.Line += string(line)
				lastLine.FqlState = guessFasState([]byte(lastLine.Line), gaps)
			} else {
				*lineCounter++
				lineStr := string(line)
				sbuff = append(sbuff, FqLine{lineStr, guessFasState(line, gaps)})
				lastLine = &sbuff[len(sbuff)-1]
			}
			if len(sbuff) > 2 && !lastLine.FqlState.Partial {
				if sbuff[0].FqlState.Header && (lastLine.FqlState.Header) {
					seq, err := FasLinesToSimpleSeq(sbuff[:len(sbuff)-1])
					if err == nil {
						seq.StartLine = spaceShift + *lineCounter - len(sbuff) - 1
						out <- seq
						sbuff = sbuff[len(sbuff)-1:]
					} else {
						for j := 0; j < len(sbuff)-1; j++ {
							ems := fmt.Sprintf("Discarded line: %s", err)
							serr := &simpleSeq{StartLine: spaceShift + *lineCounter - len(sbuff) - 1 + j, Err: errors.New(ems), Seq: sbuff[j].Line}
							out <- serr
						}
					}
					sbuff = sbuff[len(sbuff)-1:]
					if final {
						sbuff = sbuff[:0]
					}
				}
			}
		case io.EOF:
			line = bytes.TrimRight(line, "\n")
			if len(line) == 0 && final {
				spaceShift++
			}
			if len(line) > 0 {
				state := guessFasState(line, gaps)
				if !final {
					state.Partial = true
				}
				sbuff = append(sbuff, FqLine{string(line), state})
			}
			if final {
				seq, err := FasLinesToSimpleSeq(sbuff[:len(sbuff)])
				if err == nil {
					seq.StartLine = spaceShift + *lineCounter - len(sbuff) - 1
					out <- seq
					sbuff = sbuff[len(sbuff)-1:]
				} else {
					for j := 0; j < len(sbuff)-1; j++ {
						ems := fmt.Sprintf("Discarded line: %s", err)
						serr := &simpleSeq{StartLine: spaceShift + *lineCounter - len(sbuff) - 1 + j, Err: errors.New(ems), Seq: sbuff[j].Line}
						out <- serr
					}
				}
				sbuff = sbuff[:0]
			}
			ctrlChanOut <- StreamEOF
			return sbuff, nil
		default:
			return sbuff, err
		}

	}
	return sbuff, nil
}

// NewRawSeqStream initializes a new channel for reading fastq records in a robust way.
func NewRawFastqStream(inReader *bufio.Reader, chanSize int, qBase int, id string, ctrlChanIn, ctrlChanOut chan SeqStreamCtrl, gaps bool) chan *simpleSeq {
	seqChan := make(chan *simpleSeq, chanSize)
	lineCounter := 0

	go func() {
		sbuff := make(FqLines, 0, 500)
		var err error
		_ = err

		for {
			time.Sleep(NAP_SLEEP)
			select {
			case cmd := <-ctrlChanIn:
				if cmd == StreamTry {
					sbuff, err = streamFastq(inReader, sbuff, seqChan, ctrlChanIn, ctrlChanOut, &lineCounter, qBase, gaps)
					if err != nil {
						panic(err)
					}

				} else if cmd == StreamQuit {
					sbuff, err = streamFastq(inReader, sbuff, seqChan, ctrlChanIn, ctrlChanOut, &lineCounter, qBase, gaps)
					for _, l := range sbuff {
						ems := fmt.Sprintf("Discarded line: %s", err)
						serr := &simpleSeq{Err: errors.New(ems), StartLine: -1, Seq: l.Line}
						seqChan <- serr
					}
					close(seqChan)
					ctrlChanOut <- StreamExited
					close(ctrlChanOut)
					close(ctrlChanIn)
					return
				} else {
					panic(cmd)
				}
			}
		}
	}()

	return seqChan
}

// NewRawSeqStream initializes a new channel for reading fastq records in a robust way.
func NewRawFastaStream(inReader *bufio.Reader, chanSize int, id string, ctrlChanIn, ctrlChanOut chan SeqStreamCtrl, gaps bool) chan *simpleSeq {
	seqChan := make(chan *simpleSeq, chanSize)

	go func() {
		sbuff := make(FqLines, 0, 500)
		var err error
		_ = err
		lineCounter := new(int)

		for {
			select {
			case cmd := <-ctrlChanIn:
				if cmd == StreamTry {
					sbuff, err = streamFasta(inReader, sbuff, seqChan, ctrlChanIn, ctrlChanOut, lineCounter, gaps, false)
					if err != nil {
						log.Fatal(err)
					}

				} else if cmd == StreamQuit {
					sbuff, err = streamFasta(inReader, sbuff, seqChan, ctrlChanIn, ctrlChanOut, lineCounter, gaps, true)
					for i, l := range sbuff {
						ems := fmt.Sprintf("Discarded line: %s", err)
						serr := &simpleSeq{Err: errors.New(ems), StartLine: *lineCounter - i, Seq: l.Line}
						seqChan <- serr
					}
					close(seqChan)
					ctrlChanOut <- StreamExited
					close(ctrlChanOut)
					close(ctrlChanIn)
					return
				} else {
					panic(cmd)
				}
			default:
				time.Sleep(NAP_SLEEP)
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
