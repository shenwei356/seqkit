// Copyright © 2020 Oxford Nanopore Technologies.
// Copyright © 2020 Botond Sipos.
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

const MICRO_SLEEP = time.Millisecond
const NAP_SLEEP = 10 * time.Millisecond
const BIG_SLEEP = 100 * time.Millisecond

// sanaCmd represents the sana command
var sanaCmd = &cobra.Command{
	GroupID: "edit",

	Use:   "sana",
	Short: "sanitize broken single line FASTQ files",
	Long: `sanitize broken single line FASTQ files

Sana is a resilient FASTQ/FASTA parser. Unlike many parsers,
it won't stop at the first error. Instead, it skips malformed records
and continues processing the file.

Sana currently supports this FASTQ dialect:

  - One line for each sequence and quality value

	`,

	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		outFile := config.OutFile
		quiet := config.Quiet // FIXME: add quiet mode
		qBase := getFlagPositiveInt(cmd, "qual-ascii-base")
		inFmt := getFlagString(cmd, "in-format")
		checkFileFormat(inFmt)
		outFmt := getFlagString(cmd, "out-format")
		checkFileFormat(outFmt)
		inOutFmt := getFlagString(cmd, "format")
		checkFileFormat(inOutFmt)
		if inFmt == "" {
			inFmt = inOutFmt
		}
		if outFmt == "" {
			outFmt = inOutFmt
		}

		allowGaps := getFlagBool(cmd, "allow-gaps")
		runtime.GOMAXPROCS(config.Threads)

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", !config.SkipFileCheck)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Flush()
		defer outfh.Close()

		for _, file := range files {
			rawSeqChan := make(chan *simpleSeq, 10000)
			ctrlChanIn, ctrlChanOut := NewRawSeqStreamFromFile(file, rawSeqChan, qBase, inFmt, allowGaps)
			go func() {
			IT:
				for {
					select {
					case i := <-ctrlChanOut:
						if i == StreamExited {
							break IT
						} else {
							log.Fatal("Invalid command when trying to exit:", int(i))
						}
					default:
						time.Sleep(BIG_SLEEP)
					}
				}
				close(rawSeqChan)
			}()

			pass, fail := 0, 0
			ctrlChanIn <- StreamQuit
			for rawSeq := range rawSeqChan {
				switch rawSeq.Err {
				case nil:
					pass++
					outfh.WriteString(rawSeq.Format(outFmt) + "\n")
				default:
					fail++
					if !quiet {
						log.Info("File: " + rawSeq.File + "\t" + rawSeq.String() + "\n")
					}
				}
			}
			if !quiet {
				log.Info(fmt.Sprintf("File: %s\tPass records: %d\tDiscarded lines: %d\n", file, pass, fail))
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(sanaCmd)
	sanaCmd.Flags().StringP("in-format", "I", "", "input format: fastq or fasta")
	sanaCmd.Flags().StringP("out-format", "O", "", "output format: fastq or fasta")
	sanaCmd.Flags().StringP("format", "i", "fastq", "input and output format: fastq or fasta")
	sanaCmd.Flags().BoolP("allow-gaps", "A", false, "allow gap character (-) in sequences")
	sanaCmd.Flags().IntP("qual-ascii-base", "b", 33, "ASCII BASE, 33 for Phred+33")
}

// simpleSeq is a structure holding basic sequnce information with qualities.
type simpleSeq struct {
	Id        string
	Seq       string
	Sep       string
	Qual      []int
	QBase     int
	Err       error
	StartLine int
	File      string
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
	return fmt.Sprintf("@%s\n%s\n%s\n%s", s.Id, s.Seq, s.Sep, strings.Join(qs, ""))
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
	return fmt.Sprintf("@%s\n%s\n%s\n%s", s.Id, s.Seq, s.Sep, strings.Join(qs, ""))
}

// String generates a fasta string representation of a pointer to simpleSeq.
func (s *simpleSeq) FastaString() string {
	if s.Err != nil {
		return fmt.Sprintf("%s\t%d:\t %s", s.Err, s.StartLine, s.Seq)
	}
	return fmt.Sprintf(">%s\n%s", s.Id, s.Seq)
}

// validateSeqBytes check for illegal bases.
func validateSeqBytes(dna []byte, gaps bool) error {
	for i, base := range dna {
		if base == '-' && gaps {
		} else if !IUPACBases.Contains(base) {
			return fmt.Errorf("illegal base '%s' at position %d", string(base), i)
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
			return fmt.Errorf("illegal quality value '%d' at position %d", qual, i)
		}
	}
	return nil
}

// ValidateSeq validates simpleSeq objects.
func ValidateSeq(seq *simpleSeq, gaps bool) error {
	if len(seq.Seq) != len(seq.Qual) {
		return fmt.Errorf("sequence (%d) and quality (%d) length mismatch", len(seq.Seq), len(seq.Qual))
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
func NewRawSeqStreamFromFile(inFastq string, seqChan chan *simpleSeq, qBase int, format string, allowGaps bool) (chan SeqStreamCtrl, chan SeqStreamCtrl) {
	rio, err := xopen.Ropen(inFastq)
	var bio *bufio.Reader
	if err == nil {
		buffSize := 128 * 1024
		bio = bufio.NewReaderSize(rio, buffSize)
	}
	ctrlChanIn := make(chan SeqStreamCtrl, 1000)
	ctrlChanOut := make(chan SeqStreamCtrl)

	switch format {
	case "fastq":
		NewRawFastqStream(inFastq, rio, bio, seqChan, qBase, inFastq, ctrlChanIn, ctrlChanOut, allowGaps)
		return ctrlChanIn, ctrlChanOut
	case "fasta":
		NewRawFastaStream(inFastq, rio, bio, seqChan, inFastq, ctrlChanIn, ctrlChanOut, allowGaps)
		return ctrlChanIn, ctrlChanOut
	}
	return nil, nil
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
	LineNr   int
}
type FqLines []FqLine

// guessFqlState tries to infer the type of a fastq line.
func guessFqlState(line []byte, prevLine *FqLine) FqlState {
	state := FqlState{}
	if len(line) == 0 {
		state.Invalid = true
		return state
	}
	switch line[0] {
	case '@':
		state.Header = true
		state.Qual = true
	case '+':
		if len(line) == 1 {
			state.Plus = true
		} else if prevLine != nil && prevLine.FqlState.Seq {
			state.Plus = true
		} else {
			state.Qual = true
			state.Plus = true
		}
	default:
		state.Seq = true
		state.Qual = true
	}
	return state
}

// guessFasState tries to infer the type of a fasta line.
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

// FqLinesToSimpleSeq attempts to construct a valid fastq record from a buffer of parsed lines.
func FqLinesToSimpleSeq(lines FqLines, qBase int, gaps bool) (*simpleSeq, error) {
	if len(lines) != 4 {
		return nil, errors.New("Line buffer must have 4 lines!")
	}
	lh, ls, lp, lq := &lines[0], &lines[1], &lines[2], &lines[3]
	if lh.FqlState.Header && ls.FqlState.Seq && lp.FqlState.Plus && lq.FqlState.Qual {
		seq := &simpleSeq{lh.Line[1:], ls.Line, lp.Line, parseQuals(lq.Line, qBase), qBase, nil, -1, ""}
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

// FasLinesToSimpleSeq attempts to construct a valid sequence record from a buffer of parsed lines.
func FasLinesToSimpleSeq(lines FqLines) (*simpleSeq, error) {
	if len(lines) < 2 {
		return nil, fmt.Errorf("fasta record requires at least 2 lines, got %d", len(lines))
	}
	if !lines[0].FqlState.Header {
		return nil, fmt.Errorf("line %d: expected header line, got: %s", lines[0].LineNr, lines[0].Line)
	}
	s := &simpleSeq{Id: lines[0].Line[1:]}
	for i := 1; i < len(lines); i++ {
		if lines[i].FqlState.Invalid && !lines[i].FqlState.Seq {
			return nil, fmt.Errorf("line %d: invalid sequence line: %s", lines[i].LineNr, lines[i].Line)
		}
		s.Seq += lines[i].Line
	}
	return s, nil
}

// streamFastq reads records from a potentially incomplete fastq file.
func streamFastq(name string, r *bufio.Reader, sbuff FqLines, out chan *simpleSeq, ctrlChanIn, ctrlChanOut chan SeqStreamCtrl, lineCounter *int, qBase int, gaps bool, final bool) (FqLines, error) {
	var line []byte
	var lastLine *FqLine
	var prevLine *FqLine
	if len(sbuff) > 0 {
		lastLine = &sbuff[len(sbuff)-1]
	}
	if len(sbuff) > 1 {
		prevLine = &sbuff[len(sbuff)-2]
	}
	var err error
	for {
		if r == nil {
			return sbuff, nil
		}
		line, err = r.ReadBytes('\n')
		switch err {
		case nil:
			line = bytes.Trim(line, "\r\n\t ")
			(*lineCounter)++
			if lastLine != nil && lastLine.FqlState.Partial {
				lastLine.Line += string(line)
				lastLine.FqlState = guessFqlState([]byte(lastLine.Line), prevLine)
			} else {
				lineStr := string(line)
				if len(sbuff) > 0 {
					prevLine = &sbuff[len(sbuff)-1]
				}
				sbuff = append(sbuff, FqLine{lineStr, guessFqlState(line, prevLine), *lineCounter})
				lastLine = &sbuff[len(sbuff)-1]
			}
			if len(sbuff) == 4 && !lastLine.FqlState.Partial {
				seq, err := FqLinesToSimpleSeq(sbuff, qBase, gaps)
				if err == nil {
					if lineCounter != nil {
						seq.StartLine = *lineCounter - 4
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
						serr := &simpleSeq{
							StartLine: sbuff[j].LineNr,
							Err:       fmt.Errorf("discarded line: %w", err),
							Seq:       sbuff[j].Line,
							File:      name,
						}
						out <- serr
					}
					sbuff = sbuff[h:]

				}
			} //sbuff == 4
		case io.EOF:
			line = bytes.TrimRight(line, "\n")
			if len(line) > 0 {
				(*lineCounter)++
				sbuff = append(sbuff, FqLine{string(line), FqlState{Partial: true}, *lineCounter})
			}
			if !final {
				ctrlChanOut <- StreamEOF
				return sbuff, nil
			}
			if final && len(sbuff) == 4 {
				last := len(sbuff) - 1
				prevLine = &sbuff[len(sbuff)-2]
				sbuff[last].FqlState.Partial = false
				sbuff[last].FqlState = guessFqlState([]byte(sbuff[last].Line), prevLine)
				seq, err := FqLinesToSimpleSeq(sbuff, qBase, gaps)
				if err != nil {
					for il, l := range sbuff {
						ems := fmt.Sprintf("Discarded line: %s", err)
						serr := &simpleSeq{StartLine: (*lineCounter - 4 + il + 1), Err: errors.New(ems), Seq: l.Line, File: name}
						out <- serr
						sbuff = sbuff[:0]
					}
				} else {
					if seq == nil {
						panic("Sequence is nil!")
					}
					seq.StartLine = *lineCounter - 4
					out <- seq
					sbuff = sbuff[:0]
				}
			}
			return sbuff, nil
		default:
			return sbuff, err
		}

	}
	return sbuff[:0], nil
}

// streamFastq reads records from a potentially incomplete fasta file.
func streamFasta(name string, r *bufio.Reader, sbuff FqLines, out chan *simpleSeq, ctrlChanIn, ctrlChanOut chan SeqStreamCtrl, lineCounter *int, gaps bool, final bool) (FqLines, error) {
	var line []byte
	var spaceShift int
	var lastLine *FqLine
	if len(sbuff) > 0 {
		lastLine = &sbuff[len(sbuff)-1]
	}
	var err error
	for {
		if r == nil {
			return sbuff, nil
		}
		line, err = r.ReadBytes('\n')
		switch err {
		case nil:
			line = bytes.TrimRight(line, "\n\t ")
			if len(line) == 0 {
				(*lineCounter)++
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
				sbuff = append(sbuff, FqLine{lineStr, guessFasState(line, gaps), *lineCounter})
				lastLine = &sbuff[len(sbuff)-1]
			}
			if len(sbuff) >= 2 && !lastLine.FqlState.Partial {
				if sbuff[0].FqlState.Header && (lastLine.FqlState.Header) {
					seq, err := FasLinesToSimpleSeq(sbuff[:len(sbuff)-1])
					if err == nil {
						seq.StartLine = spaceShift + *lineCounter - len(sbuff) - 1
						out <- seq
						sbuff = sbuff[len(sbuff)-1:]
					} else {
						for j := 0; j < len(sbuff)-1; j++ {
							ems := fmt.Sprintf("Discarded line: %s", err)
							serr := &simpleSeq{StartLine: spaceShift + *lineCounter - len(sbuff) - 1 + j, Err: errors.New(ems), Seq: sbuff[j].Line, File: name}
							out <- serr
						}
					}
					sbuff = sbuff[len(sbuff)-1:]
				}
			}
		case io.EOF:
			line = bytes.TrimRight(line, "\n")
			if len(line) == 0 && !final {
				spaceShift++
			}

			if len(line) > 0 {
				state := guessFasState(line, gaps)
				if !final {
					state.Partial = true
				}
				sbuff = append(sbuff, FqLine{string(line), state, *lineCounter})
			}

			if !final {
				ctrlChanOut <- StreamEOF
				return sbuff, nil
			}

			if final && len(sbuff) > 1 {
				last := len(sbuff) - 1
				sbuff[last].FqlState.Partial = false
				sbuff[last].FqlState = guessFasState([]byte(sbuff[last].Line), gaps)
				seq, err := FasLinesToSimpleSeq(sbuff)
				if err == nil {
					seq.StartLine = spaceShift + *lineCounter - len(sbuff) - 1
					out <- seq
					sbuff = sbuff[:0]
				} else {
					for j := 0; j < len(sbuff)-1; j++ {
						ems := fmt.Sprintf("Discarded line: %s", err)
						serr := &simpleSeq{StartLine: spaceShift + *lineCounter - len(sbuff) - 1 + j, Err: errors.New(ems), Seq: sbuff[j].Line, File: name}
						out <- serr
					}
				}
				sbuff = sbuff[:0]
			}
			return sbuff, nil
		default:
			return sbuff, err
		}

	}
	return sbuff, nil
}

// NewRawSeqStream initializes a new channel for reading fastq records in a robust way.
func NewRawFastqStream(name string, inFh *xopen.Reader, inReader *bufio.Reader, seqChan chan *simpleSeq, qBase int, id string, ctrlChanIn, ctrlChanOut chan SeqStreamCtrl, gaps bool) chan *simpleSeq {
	lineCounter := 0

	go func() {
		sbuff := make(FqLines, 0, 1000)
		var err error

	MAIN_FQ:
		for {
			select {
			case cmd := <-ctrlChanIn:
				if inReader == nil {
					inFh, err = xopen.Ropen(name)
					if err == nil {
						buffSize := 128 * 1024
						inReader = bufio.NewReaderSize(inFh, buffSize)
					} else {
						if cmd == StreamQuit {
							ctrlChanOut <- StreamExited
							return
						}
						continue MAIN_FQ
					}

				}
				if cmd == StreamTry {
					sbuff, err = streamFastq(name, inReader, sbuff, seqChan, ctrlChanIn, ctrlChanOut, &lineCounter, qBase, gaps, false)
					if err != nil {
						log.Fatal(err)
					}

				} else if cmd == StreamQuit {
					sbuff, err = streamFastq(name, inReader, sbuff, seqChan, ctrlChanIn, ctrlChanOut, &lineCounter, qBase, gaps, true)
					for _, l := range sbuff {
						var ems string
						if err != nil {
							ems = fmt.Sprintf("Discarded line: %s", err)
						} else {
							ems = "Discarded final line"
						}
						serr := &simpleSeq{Err: errors.New(ems), StartLine: lineCounter, Seq: l.Line, File: name}
						seqChan <- serr
					}
					ctrlChanOut <- StreamExited
					inFh.Close()
					return
				} else {
					log.Fatal("Invalid command:", int(cmd))
				}
			default:
				time.Sleep(BIG_SLEEP)
			}
		}
	}()

	return seqChan
}

// NewRawSeqStream initializes a new channel for reading fastq records in a robust way.
func NewRawFastaStream(name string, inFh *xopen.Reader, inReader *bufio.Reader, seqChan chan *simpleSeq, id string, ctrlChanIn, ctrlChanOut chan SeqStreamCtrl, gaps bool) chan *simpleSeq {
	go func() {
		sbuff := make(FqLines, 0, 1000)
		var err error
		lineCounter := new(int)

	MAIN_FA:
		for {
			select {
			case cmd := <-ctrlChanIn:
				if inReader == nil {
					inFh, err = xopen.Ropen(name)
					if err == nil {
						buffSize := 128 * 1024
						inReader = bufio.NewReaderSize(inFh, buffSize)
					} else {
						if cmd == StreamQuit {
							ctrlChanOut <- StreamExited
							return
						}
						continue MAIN_FA
					}

				}
				if cmd == StreamTry {
					sbuff, err = streamFasta(name, inReader, sbuff, seqChan, ctrlChanIn, ctrlChanOut, lineCounter, gaps, false)
					if err != nil {
						log.Fatal(err)
					}

				} else if cmd == StreamQuit {
					sbuff, err = streamFasta(name, inReader, sbuff, seqChan, ctrlChanIn, ctrlChanOut, lineCounter, gaps, true)
					for i, l := range sbuff {
						ems := fmt.Sprintf("Discarded line: %s", err)
						serr := &simpleSeq{Err: errors.New(ems), StartLine: *lineCounter - i, Seq: l.Line, File: name}
						seqChan <- serr
					}
					ctrlChanOut <- StreamExited
					inFh.Close()
					return
				} else {
					log.Fatal("Invalid command:", int(cmd))
				}
			default:
				time.Sleep(BIG_SLEEP)
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
