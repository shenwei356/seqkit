// Copyright © 2016-2019 Wei Shen <shenwei356@gmail.com>
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
	"runtime"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// slidingCmd represents the sliding command
var slidingCmd = &cobra.Command{
	Use:   "sliding",
	Short: "extract subsequences in sliding windows",
	Long: `extract subsequences in sliding windows

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

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		greedy := getFlagBool(cmd, "greedy")
		circular := getFlagBool(cmd, "circular-genome") || getFlagBool(cmd, "circular")
		step := getFlagInt(cmd, "step")
		window := getFlagInt(cmd, "window")
		if step == 0 || window == 0 {
			checkError(fmt.Errorf("both flags -s (--step) and -W (--window) needed"))
		}
		if step < 1 {
			checkError(fmt.Errorf("value of flag -s (--step) should be greater than 0: %d ", step))
		}
		if window < 1 {
			checkError(fmt.Errorf("value of flag -W (--window) should be greater than 0: %d ", window))
		}

		suffix := getFlagString(cmd, "suffix")

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var sequence, s, qual, q []byte
		var r *fastx.Record
		var originalLen, l, end, e int
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

				originalLen = len(record.Seq.Seq)
				sequence = record.Seq.Seq
				qual = record.Seq.Qual
				l = len(sequence)
				end = l - 1
				if end < 0 {
					end = 0
				}
				for i := 0; i <= end; i += step {
					e = i + window
					if e > originalLen {
						if circular {
							e = e - originalLen
							s = sequence[i:]
							s = append(s, sequence[0:e]...)
							if len(qual) > 0 {
								q = qual[i:]
								q = append(q, qual[0:e]...)
							}
						} else if greedy {
							s = sequence[i:]
							if len(qual) > 0 {
								q = qual[i:]
							}
							e = l
						} else {
							break
						}
					} else {
						s = sequence[i : i+window]
						if len(qual) > 0 {
							q = qual[i : i+window]
						}
					}

					if len(qual) > 0 {
						r, _ = fastx.NewRecordWithQualWithoutValidation(record.Seq.Alphabet,
							[]byte{}, []byte(fmt.Sprintf("%s%s:%d-%d", record.ID, suffix, i+1, e)), []byte{}, s, q)
					} else {
						r, _ = fastx.NewRecordWithoutValidation(record.Seq.Alphabet,
							[]byte{}, []byte(fmt.Sprintf("%s%s:%d-%d", record.ID, suffix, i+1, e)), []byte{}, s)
					}
					r.FormatToWriter(outfh, config.LineWidth)
				}
			}
			fastxReader.Close()

			config.LineWidth = lineWidth
		}
	},
}

func init() {
	RootCmd.AddCommand(slidingCmd)

	slidingCmd.Flags().IntP("step", "s", 0, "step size")
	slidingCmd.Flags().IntP("window", "W", 0, "window size")
	slidingCmd.Flags().BoolP("greedy", "g", false, "greedy mode, i.e., exporting last subsequences even shorter than the windows size")
	slidingCmd.Flags().BoolP("circular-genome", "C", false, "circular genome (same to -c/--circular)")
	slidingCmd.Flags().BoolP("circular", "c", false, "circular genome (same to -C/--circular-genome)")
	slidingCmd.Flags().StringP("suffix", "S", "_sliding", "suffix added to the sequence ID")
}
