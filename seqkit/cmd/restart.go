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
	"github.com/shenwei356/bwt/fmi"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// restartCmd represents the sliding command
var restartCmd = &cobra.Command{
	GroupID: "edit",

	Use:     "restart",
	Aliases: []string{"rotate"},
	Short:   "reset start position (rotate) for circular genomes",
	Long: `reset start position (rotate) for circular genomes

Examples
  1. Specify a new start position.

    $ echo -e ">seq1\natggcCACTG"
    >seq1
    atggcCACTG

    $ echo -e ">seq1\natggcCACTG" | seqkit restart -i 2
    >seq1
    tggcCACTGa

    $ echo -e ">seq1\natggcCACTG" | seqkit restart -i -2
    >seq1
    TGatggcCAC

  2. Specify a starting subsequence.

    $ echo -e ">seq1\natggcCACTG" | seqkit restart -I -s GGCC
    >seq1
    ggcCACTGat

    # on the negative strand
    $ echo -e ">seq1\natggcCACTG" | seqkit restart -I -s AGTG
    >seq1
    AGTGgccatC

    # allo 1 mismatch
    $ echo -e ">seq1\natggcCACTG" | seqkit restart -I -s GGCCT -m 1
    >seq1
    ggcCACTGat

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
		startSeq := []byte(getFlagString(cmd, "start-with"))
		if !cmd.Flags().Lookup("new-start").Changed && len(startSeq) == 0 {
			checkError(fmt.Errorf("one of the flags needed: -i/--new-start or -s/--start-with"))
		}
		if newstart == 0 {
			checkError(fmt.Errorf("the value of flag -s (--start) should not be 0"))
		}
		mismatches := getFlagNonNegativeInt(cmd, "max-mismatch")
		ignoreCase := getFlagBool(cmd, "ignore-case")

		// -------------------------------------------------------------------------------

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var sequence0, sequence, qual []byte
		var bufSeq, bufQual bytes.Buffer
		var l int
		var record *fastx.Record
		var i int
		var sfmi *fmi.FMIndex
		sfmi = fmi.NewFMIndex()
		positions := make([]int, 0, 8)
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

				sequence0 = record.Seq.Seq
				sequence = record.Seq.Seq
				l = len(record.Seq.Seq)

				// ----------------------------------------------------
				// find the position of subsequence
				if len(startSeq) > 0 {
					if ignoreCase {
						sequence = bytes.ToUpper(sequence0)
						startSeq = bytes.ToUpper(startSeq)
					}

					positions = positions[:0]
					if mismatches > 0 {
						_, err = sfmi.Transform(sequence)
						if err != nil {
							checkError(fmt.Errorf("fail to build FMIndex for sequence: %s", record.ID))
						}
						positions, err = sfmi.Locate(startSeq, mismatches)
						if err != nil {
							checkError(fmt.Errorf("fail to search subsequence on seq '%s': %s", record.ID, err))
						}

						if len(positions) > 1 {
							log.Warningf("the given subsequence is found at %d positions in sequence: %s, please give a longer one to increase the specificity", len(positions), record.ID)
							continue
						}

						if len(positions) == 0 { // not found in the current strand
							record.Seq.RevComInplace()
							sequence0 = record.Seq.Seq
							sequence = record.Seq.Seq
							if ignoreCase {
								sequence = bytes.ToUpper(sequence0)
							}

							_, err = sfmi.Transform(sequence)
							if err != nil {
								checkError(fmt.Errorf("fail to build FMIndex for sequence: %s", record.ID))
							}
							positions, err = sfmi.Locate(startSeq, mismatches)
							if err != nil {
								checkError(fmt.Errorf("fail to search subsequence on seq '%s': %s", record.ID, err))
							}

							if len(positions) > 1 {
								log.Warningf("the given subsequence is found at %d positions in the reverse complement sequence of %s, please give a longer one to increase the specificity", len(positions), record.ID)
								continue
							}
						}
					} else {
						for {
							newstart = bytes.Index(sequence[i:], startSeq)
							if newstart < 0 {
								break
							}
							positions = append(positions, newstart)
							i += newstart + len(startSeq)
						}

						if len(positions) > 1 {
							log.Warningf("the given subsequence is found at %d positions in sequence: %s, please give a longer one to increase the specificity", len(positions), record.ID)
							continue
						}

						if len(positions) == 0 { // not found in the current strand
							record.Seq.RevComInplace()
							sequence0 = record.Seq.Seq
							sequence = record.Seq.Seq
							if ignoreCase {
								sequence = bytes.ToUpper(sequence0)
							}

							for {
								newstart = bytes.Index(sequence[i:], startSeq)
								if newstart < 0 {
									break
								}
								positions = append(positions, newstart)
								i += newstart + len(startSeq)
							}
							if len(positions) > 1 {
								log.Warningf("the given subsequence is found at %d positions in the reverse complement sequence of %s, please give a longer one to increase the specificity", len(positions), record.ID)
								continue
							}
						}
					}

					if len(positions) == 0 {
						log.Warningf("the given subsequence is not found in both strands of sequence: %s", record.ID)
						continue
					}

					newstart = positions[0] + 1 // convert to 1-based
				} else {
					if newstart > l || newstart < -l {
						checkError(fmt.Errorf("new start (%d) exceeds length of sequence (%d)", newstart, l))
					}
				}

				// ----------------------------------------------------

				bufSeq.Reset()
				if newstart > 0 {
					bufSeq.Write(sequence0[newstart-1:])
					bufSeq.Write(sequence0[0 : newstart-1])
				} else {
					bufSeq.Write(sequence0[l+newstart:])
					bufSeq.Write(sequence0[0 : l+newstart])
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

	restartCmd.Flags().IntP("new-start", "i", 1, "new start position (1-based, supporting negative value counting from the end)")

	restartCmd.Flags().StringP("start-with", "s", "", "rotate the genome to make it starting with the given subsequence")
	restartCmd.Flags().BoolP("ignore-case", "I", false, "ignore case when searching the custom starting subsequence")
	restartCmd.Flags().IntP("max-mismatch", "m", 0, "max mismatch when searching the custom starting subsequence")

}
