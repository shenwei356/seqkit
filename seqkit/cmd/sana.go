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

Attention:
  1. FASTA recovery discards pre-header junk and malformed records, preserving
     valid neighboring records. Empty IDs are allowed; records without sequence
     are discarded.
  2. File I/O errors stop processing with a nonzero exit status.

`,

	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		outFile := config.OutFile
		quiet := config.Quiet
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

		if inFmt == "" || outFmt == "" {
			log.Fatal("Invalid format specified: empty effective format")
		}

		allowGaps := getFlagBool(cmd, "allow-gaps")
		runtime.GOMAXPROCS(config.Threads)

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", !config.SkipFileCheck)
		if !config.SkipFileCheck {
			for _, file := range files {
				checkIfFilesAreTheSame(file, outFile, "input", "output")
			}
		}

		for _, file := range files {
			if !isStdin(file) {
				info, err := os.Stat(file)
				if err == nil && info.IsDir() {
					checkError(fmt.Errorf("%s: input is a directory", file))
				}
			}
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer closeOutput(outfh)

		for _, file := range files {
			rawSeqChan := make(chan *simpleSeq, 10000)
			ctrlChanIn, ctrlChanOut := NewRawSeqStreamFromFile(file, rawSeqChan, qBase, inFmt, allowGaps)
			go func() {
				if i := <-ctrlChanOut; i != StreamExited {
					log.Fatal("Invalid command when trying to exit:", int(i))
				}
				close(rawSeqChan)
			}()

			pass, fail := 0, 0
			ctrlChanIn <- StreamQuit
			for rawSeq := range rawSeqChan {
				switch rawSeq.Err {
				case nil:
					pass++
					outfh.WriteString(rawSeq.Format(outFmt))
					outfh.WriteByte('\n')
				default:
					if _, fatal := rawSeq.Err.(*rawFileError); fatal {
						closeOutput(outfh)
						checkError(rawSeq.Err)
					}
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
		qs[i] = string(rune(qq + s.QBase))
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
		qs[i] = string(rune(qq + s.QBase))
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
func NewRawSeqStreamFromFile(file string, seqChan chan *simpleSeq, qBase int, format string, allowGaps bool) (chan SeqStreamCtrl, chan SeqStreamCtrl) {
	in := make(chan SeqStreamCtrl, 1000)
	out := make(chan SeqStreamCtrl)
	go runRawSeqStream(file, nil, nil, seqChan, qBase, format, in, out, allowGaps)
	return in, out
}

// rawFileError distinguishes I/O failures from recoverable malformed records.
type rawFileError struct {
	file  string
	cause error
}

func (e *rawFileError) Error() string { return fmt.Sprintf("%s: %v", e.file, e.cause) }
func (e *rawFileError) Unwrap() error { return e.cause }

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
		state.Header = true
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
	var sequence strings.Builder
	for i := 1; i < len(lines); i++ {
		if lines[i].FqlState.Invalid && !lines[i].FqlState.Seq {
			return nil, fmt.Errorf("line %d: invalid sequence line: %s", lines[i].LineNr, lines[i].Line)
		}
		sequence.WriteString(lines[i].Line)
	}
	s.Seq = sequence.String()
	return s, nil
}

// streamFastq reads records from a potentially incomplete fastq file.
// streamRaw retains physical lines across temporary EOFs and emits complete records.
func streamRaw(name string, r *bufio.Reader, buffer FqLines, out chan *simpleSeq, feedback chan SeqStreamCtrl, lineCounter *int, qBase int, gaps, fastq, final bool) (FqLines, error) {
	discard := func(lines FqLines, cause error) {
		for _, line := range lines {
			if line.Line != "" {
				out <- &simpleSeq{File: name, StartLine: line.LineNr, Seq: line.Line, Err: fmt.Errorf("Discarded line: %w", cause)}
			}
		}
	}
	emitFasta := func(lines FqLines) {
		record, err := FasLinesToSimpleSeq(lines)
		if err != nil {
			discard(lines, err)
		} else {
			record.File = name
			record.StartLine = lines[0].LineNr
			out <- record
		}
	}
	process := func(text string, number int) {
		if !fastq {
			text = strings.TrimRight(text, "\r\n\t ")
			if text == "" {
				return
			}
			state := guessFasState([]byte(text), gaps)
			if state.Header && len(buffer) > 0 {
				emitFasta(buffer)
				buffer = buffer[:0]
			}
			if len(buffer) == 0 && !state.Header {
				discard(FqLines{{Line: text, LineNr: number}}, fmt.Errorf("expected FASTA header"))
				return
			}
			buffer = append(buffer, FqLine{text, state, number})
			return
		}
		text = strings.Trim(text, "\r\n\t ")
		var prev *FqLine
		if len(buffer) > 0 {
			prev = &buffer[len(buffer)-1]
		}
		buffer = append(buffer, FqLine{text, guessFqlState([]byte(text), prev), number})
		if len(buffer) == 4 {
			record, err := FqLinesToSimpleSeq(buffer, qBase, gaps)
			if err == nil {
				record.File = name
				record.StartLine = buffer[0].LineNr
				out <- record
				buffer = buffer[:0]
			} else {
				next := len(buffer)
				for i := 1; i < len(buffer); i++ {
					if buffer[i].FqlState.Header {
						next = i
						break
					}
				}
				discard(buffer[:next], err)
				buffer = buffer[next:]
			}
		}
	}
	for {
		if r == nil {
			return buffer, nil
		}
		line, err := r.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return buffer, err
		}
		number := *lineCounter
		partial := len(buffer) > 0 && buffer[len(buffer)-1].FqlState.Partial
		if len(line) > 0 || final && partial {
			if partial {
				last := buffer[len(buffer)-1]
				buffer = buffer[:len(buffer)-1]
				line = append([]byte(last.Line), line...)
				number = last.LineNr
			} else {
				(*lineCounter)++
				number = *lineCounter
			}
			if err == io.EOF && !final {
				buffer = append(buffer, FqLine{string(line), FqlState{Partial: true}, number})
			} else {
				process(string(line), number)
			}
		}
		if err == io.EOF {
			if !final {
				feedback <- StreamEOF
				return buffer, nil
			}
			if len(buffer) > 0 {
				if fastq {
					discard(buffer, fmt.Errorf("incomplete FASTQ record"))
				} else {
					emitFasta(buffer)
				}
			}
			return buffer[:0], nil
		}
	}
}

// streamFastq resumes a potentially incomplete single-line FASTQ stream.
func streamFastq(name string, r *bufio.Reader, buffer FqLines, out chan *simpleSeq, in, feedback chan SeqStreamCtrl, counter *int, qBase int, gaps, final bool) (FqLines, error) {
	return streamRaw(name, r, buffer, out, feedback, counter, qBase, gaps, true, final)
}

// streamFasta resumes multiline FASTA without discarding an unfinished record.
func streamFasta(name string, r *bufio.Reader, buffer FqLines, out chan *simpleSeq, in, feedback chan SeqStreamCtrl, counter *int, gaps, final bool) (FqLines, error) {
	return streamRaw(name, r, buffer, out, feedback, counter, 0, gaps, false, final)
}

// rawInputView leaves ownership of the source file with the stream runner.
type rawInputView struct{ io.Reader }

// rawGrowingReader preserves decoded offsets across temporary compressed EOFs.
type rawGrowingReader struct {
	source  io.Reader
	offset  *int64
	growing bool
}

// Read counts decoded bytes and distinguishes a temporary truncated member.
func (r *rawGrowingReader) Read(p []byte) (int, error) {
	n, err := r.source.Read(p)
	*r.offset += int64(n)
	if r.growing && errors.Is(err, io.ErrUnexpectedEOF) {
		err = io.EOF
	}
	return n, err
}

// runRawSeqStream reports file failures through the ordered output channel.
func runRawSeqStream(name string, fh *xopen.Reader, reader *bufio.Reader, sequences chan *simpleSeq, qBase int, format string, in, feedback chan SeqStreamCtrl, gaps bool) {
	var buffer FqLines
	var counter int
	var source *os.File
	seekable := false
	defer func() {
		if source != nil {
			source.Close()
		}
	}()
	var decodedOffset int64
	var tracked *rawGrowingReader
	compressed := false
	report := func(err error) { sequences <- &simpleSeq{File: name, Err: &rawFileError{name, err}} }
	finish := func() {
		if source != nil {
			if err := source.Close(); err != nil {
				report(err)
			}
			source = nil
		}
		feedback <- StreamExited
	}

	for command := range in {
		if reader == nil {
			var err error
			// Final-only streams retain xopen support for URLs, commands and FIFOs.
			if isStdin(name) || command == StreamQuit && source == nil || strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") || strings.HasPrefix(name, "|") {
				fh, err = xopen.Ropen(name)
			} else {
				if source == nil {
					source, err = os.Open(name)
					if err == nil {
						info, statErr := source.Stat()
						err = statErr
						if err == nil {
							seekable = info.Mode().IsRegular()
						}
					}
				}
				if err == nil && !seekable {
					fh, err = xopen.Buf(rawInputView{source})
				}
				if err == nil && seekable {
					var magic [6]byte
					n, probeErr := source.ReadAt(magic[:], 0)
					if probeErr != nil && probeErr != io.EOF {
						err = probeErr
					}
					// A short compression header is not yet malformed while the file grows.
					partialHeader := false
					for _, signature := range [][]byte{{0x1f, 0x8b}, {0xfd, 0x37, 0x7a, 0x58, 0x5a, 0x00}, {0x28, 0xb5, 0x2f, 0xfd}, {'B', 'Z', 'h'}, {0x04, 0x22, 0x4d, 0x18}, {0x02, 0x21, 0x4c, 0x18}} {
						if n > 0 && (bytes.HasPrefix(magic[:n], signature) || bytes.HasPrefix(signature, magic[:n])) {
							compressed = true
							partialHeader = n < len(signature)
							break
						}
					}
					if command == StreamTry && partialHeader {
						feedback <- StreamEOF
						continue
					}
					if err == nil {
						_, err = source.Seek(0, io.SeekStart)
					}
					if err == nil {
						fh, err = xopen.Buf(rawInputView{source})
					}
				}
			}
			if err == xopen.ErrNoContent || command == StreamTry && errors.Is(err, io.ErrUnexpectedEOF) {
				if command == StreamQuit {
					finish()
					return
				}
				feedback <- StreamEOF
				continue
			}
			if err != nil {
				report(err)
				finish()
				return
			}
			if decodedOffset > 0 {
				if _, err := io.CopyN(io.Discard, fh, decodedOffset); err != nil {
					fh.Close()
					report(err)
					finish()
					return
				}
			}
			tracked = &rawGrowingReader{source: fh, offset: &decodedOffset}
			reader = bufio.NewReaderSize(tracked, 128<<10)
		}
		if tracked != nil {
			tracked.growing = command == StreamTry
		}
		var err error
		buffer, err = streamRaw(name, reader, buffer, sequences, feedback, &counter, qBase, gaps, format == "fastq", command == StreamQuit)
		if err != nil || command == StreamQuit {
			if err != nil {
				report(err)
			}
			if closeErr := fh.Close(); closeErr != nil {
				report(closeErr)
			}
			finish()
			return
		}
		if compressed {
			// Decoders may cache EOF. Reopen them at the decoded offset, while keeping
			// the original source descriptor so renames do not change the input identity.
			// The decoder also reports a temporary truncated member from Close.
			if err := fh.Close(); err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
				report(err)
				finish()
				return
			}
			reader = nil
		}
	}
}

// NewRawFastqStream starts the shared parser with an already opened reader.
func NewRawFastqStream(name string, fh *xopen.Reader, reader *bufio.Reader, sequences chan *simpleSeq, qBase int, id string, in, feedback chan SeqStreamCtrl, gaps bool) chan *simpleSeq {
	go runRawSeqStream(name, fh, reader, sequences, qBase, "fastq", in, feedback, gaps)
	return sequences
}

// NewRawFastaStream starts the shared parser with an already opened reader.
func NewRawFastaStream(name string, fh *xopen.Reader, reader *bufio.Reader, sequences chan *simpleSeq, id string, in, feedback chan SeqStreamCtrl, gaps bool) chan *simpleSeq {
	go runRawSeqStream(name, fh, reader, sequences, 0, "fasta", in, feedback, gaps)
	return sequences
}

// parseQuals parses quality string into a slice of integers.
func parseQuals(qualString string, qBase int) []int {
	quals := make([]int, len(qualString))
	for i, char := range qualString {
		quals[i] = int(char) - qBase
	}
	return quals
}
