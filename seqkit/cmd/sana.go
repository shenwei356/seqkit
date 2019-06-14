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
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// sanaCmd represents the concatenate command
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
			rawSeqChan := NewRawSeqStream(file, 1000, qBase)

			prevRec := "None"
			for rawSeq := range rawSeqChan {
				if err := ValidateSeq(rawSeq); err != nil {
					fmt.Fprintf(os.Stderr, "Discarded record\t%s\t: %s\tPrevious record:%s\n", rawSeq.Id, err, prevRec)
				} else {
					prevRec = rawSeq.Id
					outfh.WriteString(rawSeq.String() + "\n")
				}
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(sanaCmd)
	sanaCmd.Flags().IntP("qual-ascii-base", "b", 33, "ASCII BASE, 33 for Phred+33")
}

type simpleSeq struct {
	Id    string
	Seq   string
	Qual  []int
	QBase int
}

func (s *simpleSeq) String() string {
	qs := make([]string, len(s.Qual))
	for i, qq := range s.Qual {
		qs[i] = string(qq + s.QBase)
	}
	return fmt.Sprintf("@%s\n%s\n+\n%s", s.Id, s.Seq, strings.Join(qs, ""))
}

var IUPACBases map[rune]bool

func init() {
	IUPACBases = map[rune]bool{
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
	}

}

// Validate sequence:
func validateSeqString(dna string) error {
	for i, base := range dna {
		if !IUPACBases[base] {
			return errors.New(fmt.Sprintf("Illegal base '%s' at position %d", string(base), i))
		}
	}
	return nil
}

// Validate quality string:
func validateQuals(quals []int) error {
	for i, qual := range quals {
		if qual < 0 {
			return errors.New(fmt.Sprintf("Illegal quality value '%d' at position %d", qual, i))
		}
	}
	return nil
}

// Validate Seq object:
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

// Robust parsing of single line fastqs:
func NewRawSeqStream(inFastq string, chanSize int, qBase int) chan *simpleSeq {
	seqChan := make(chan *simpleSeq, chanSize)
	r, err := xopen.Ropen(inFastq)
	checkError(err)

	go func() {
		var token string
		var err error
		var inRecord int
		var lineNr int
		var s *simpleSeq

		for {

			// Read in the next line:
			token, err = r.ReadString('\n')
			token = strings.TrimRight(token, "\n")

			// Reached end of file,
			// send last record:
			if err == io.EOF {
				seqChan <- s
				close(seqChan)
				break
			} else if err != nil {
				panic(err)
			} else {

				if token[0] == '@' && inRecord == 0 {
					// Found a new header line:
					if s != nil {
						seqChan <- s
					}
					inRecord++
					s = new(simpleSeq)
					s.Id = string(token[1:])

				} else if inRecord == 1 {
					// Sequence line:
					s.Seq = token
					inRecord++
				} else if inRecord == 2 {
					// Skip separator line:
					inRecord++
				} else if inRecord == 3 {
					// Quality line:
					s.Qual = parseQuals(token, qBase)
					s.QBase = qBase
					inRecord = 0 // Expect header next time.
				} else {
					fmt.Fprintf(os.Stderr, "Skipping input line %d!\n", lineNr)
				}
			}

			lineNr++
		}
	}()

	return seqChan
}

// Parse quality string into a slice of integers.
func parseQuals(qualString string, qBase int) []int {
	quals := make([]int, len(qualString))
	for i, char := range qualString {
		quals[i] = int(char) - qBase
	}
	return quals
}
