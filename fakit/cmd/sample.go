// Copyright © 2016 Wei Shen <shenwei356@gmail.com>
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
	"math/rand"

	"github.com/brentp/xopen"
	"github.com/shenwei356/bio/seqio/fasta"
	"github.com/spf13/cobra"
)

// sampleCmd represents the seq command
var sampleCmd = &cobra.Command{
	Use:   "sample",
	Short: "sample sequences by number or proportion",
	Long: `sample sequences by number or proportion.

`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			checkError(fmt.Errorf("no more than one file needed (%d)", len(args)))
		}

		alphabet := getAlphabet(cmd, "seq-type")
		idRegexp := getFlagString(cmd, "id-regexp")
		chunkSize := getFlagInt(cmd, "chunk-size")
		threads := getFlagInt(cmd, "threads")
		lineWidth := getFlagInt(cmd, "line-width")
		outFile := getFlagString(cmd, "out-file")
		quiet := getFlagBool(cmd, "quiet")

		files := getFileList(args)

		seed := getFlagInt64(cmd, "rand-seed")
		twoPass := getFlagBool(cmd, "two-pass")
		number := getFlagInt64(cmd, "number")
		proportion := getFlagFloat64(cmd, "proportion")

		if number == 0 && proportion == 0 {
			checkError(fmt.Errorf("one of flags -n (--number) and -p (--proportion) needed"))
		}

		if number < 0 {
			checkError(fmt.Errorf("value of --number and should be greater than 0"))
		}
		if proportion < 0 || proportion > 1 {
			checkError(fmt.Errorf("value of --propotion (%f) should be in range of [0, 1]", proportion))
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		rand.Seed(seed)

		file := files[0]

		n := 0
		if number > 0 { // by number
			if !quiet {
				log.Info("sample by number")
			}

			if twoPass {
				if xopen.IsStdin() {
					checkError(fmt.Errorf("2-pass mode (-2) will failed when reading from stdin. please disable flag: -2"))
				}
				// first pass, get seq number
				if !quiet {
					log.Info("first pass: get seq number")
				}
				names, err := fasta.GetSeqNames(file)
				checkError(err)

				if !quiet {
					log.Infof("seq number: %d", len(names))
				}

				proportion = float64(number) / float64(len(names))

				// second pass
				if !quiet {
					log.Info("second pass: read and sample")
				}
				fastaReader, err := fasta.NewFastaReader(alphabet, file, chunkSize, threads, idRegexp)
				checkError(err)
				for chunk := range fastaReader.Ch {
					checkError(chunk.Err)

					for _, record := range chunk.Data {
						if rand.Float64() <= proportion {
							n++
							outfh.WriteString(fmt.Sprintf(">%s\n%s\n", record.Name, record.FormatSeq(lineWidth)))
						}
					}
				}
			} else {
				records, err := fasta.GetSeqs(file, alphabet, chunkSize, threads, idRegexp)
				checkError(err)

				proportion = float64(number) / float64(len(records))

				for _, record := range records {
					if rand.Float64() <= proportion {
						n++
						outfh.WriteString(fmt.Sprintf(">%s\n%s\n", record.Name, record.FormatSeq(lineWidth)))
					}
				}
			}
		} else {
			if !quiet {
				log.Info("sample by proportion")
			}

			fastaReader, err := fasta.NewFastaReader(alphabet, file, chunkSize, threads, idRegexp)
			checkError(err)
			for chunk := range fastaReader.Ch {
				checkError(chunk.Err)

				for _, record := range chunk.Data {
					if rand.Float64() <= proportion {
						n++
						outfh.WriteString(fmt.Sprintf(">%s\n%s\n", record.Name, record.FormatSeq(lineWidth)))
					}
				}
			}
		}

		if !quiet {
			log.Info("%d sequences outputed", n)
		}
	},
}

func init() {
	RootCmd.AddCommand(sampleCmd)

	sampleCmd.Flags().Int64P("rand-seed", "s", 11, "rand seed for shuffle")
	sampleCmd.Flags().Int64P("number", "n", 0, "sample by number (result may not exactly match)")
	sampleCmd.Flags().Float64P("proportion", "p", 0, "sample by proportion")
	sampleCmd.Flags().BoolP("two-pass", "2", false, "2-pass mode read files twice to lower memory usage. Not allowed when reading from stdin")
}
