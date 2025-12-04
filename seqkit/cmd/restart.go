// Copyright © 2016-2026 Wei Shen <shenwei356@gmail.com>
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
	"bytes"
	"fmt"
	"io"
	"runtime"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// restartCmd represents the sliding command
var restartCmd = &cobra.Command{
	GroupID: "edit",

	Use:     "restart",
	Aliases: []string{"rotate"},
	Short:   "reset start position for circular genome",
	Long: `reset start position for circular genome

Examples

    $ echo -e ">seq\nacgtnACGTN"
    >seq
    acgtnACGTN

    $ echo -e ">seq\nacgtnACGTN" | seqkit restart -i 2
    >seq
    cgtnACGTNa

    $ echo -e ">seq\nacgtnACGTN" | seqkit restart -i -2
    >seq
    TNacgtnACG

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		lineWidth := config.LineWidth
		outFile := config.OutFile
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.BufferSize)

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", !config.SkipFileCheck)
		if !config.SkipFileCheck {
			for _, file := range files {
				checkIfFilesAreTheSame(file, outFile, "input", "output")
			}
		}

		newstart := getFlagInt(cmd, "new-start")
		if newstart == 0 {
			checkError(fmt.Errorf("value of flag -s (--start) should not be 0"))
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var sequence, qual []byte
		var bufSeq, bufQual bytes.Buffer
		var l int
		var record *fastx.Record
		for _, file := range files {
			fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)
			for {
				record, err = fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}
				if fastxReader.IsFastq {
					config.LineWidth = 0
					fastx.ForcelyOutputFastq = true
				}

				l = len(record.Seq.Seq)
				if newstart > l || newstart < -l {
					checkError(fmt.Errorf("new start (%d) exceeds length of sequence (%d)", newstart, l))
				}

				sequence = record.Seq.Seq
				bufSeq.Reset()
				if newstart > 0 {
					bufSeq.Write(sequence[newstart-1:])
					bufSeq.Write(sequence[0 : newstart-1])
				} else {
					bufSeq.Write(sequence[l+newstart:])
					bufSeq.Write(sequence[0 : l+newstart])
				}
				record.Seq.Seq = bufSeq.Bytes()

				if len(record.Seq.Qual) > 0 {
					qual = record.Seq.Qual
					bufQual.Reset()
					if newstart > 0 {
						bufQual.Write(qual[newstart-1:])
						bufQual.Write(qual[0 : newstart-1])
					} else {
						bufQual.Write(qual[l+newstart:])
						bufQual.Write(qual[0 : l+newstart])

					}
					record.Seq.Qual = bufQual.Bytes()
				}

				record.FormatToWriter(outfh, config.LineWidth)
			}
			fastxReader.Close()
			config.LineWidth = lineWidth
		}
	},
}

func init() {
	RootCmd.AddCommand(restartCmd)

	restartCmd.Flags().IntP("new-start", "i", 1, "new start position (1-base, supporting negative value counting from the end)")
}
