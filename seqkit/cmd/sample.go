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
	"math/rand"
	"runtime"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// sampleCmd represents the sample command
var sampleCmd = &cobra.Command{
	Use:   "sample",
	Short: "sample sequences by number or proportion",
	Long: `sample sequences by number or proportion.

`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			checkError(fmt.Errorf("no more than one file needed (%d)", len(args)))
		}

		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		// lineWidth := config.LineWidth
		outFile := config.OutFile
		quiet := config.Quiet
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		files := getFileList(args)

		seed := getFlagInt64(cmd, "rand-seed")
		twoPass := getFlagBool(cmd, "two-pass")
		number := getFlagInt64(cmd, "number")
		proportion := getFlagFloat64(cmd, "proportion")

		if number == 0 && proportion == 0 {
			checkError(fmt.Errorf("one of flags -n (--number) and -p (--proportion) needed"))
		}

		if number < 0 {
			checkError(fmt.Errorf("value of -n (--number) and should be greater than 0"))
		}
		if proportion < 0 || proportion > 1 {
			checkError(fmt.Errorf("value of -p (--proportion) (%f) should be in range of [0, 1]", proportion))
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		rand.Seed(seed)

		file := files[0]

		n := int64(0)
		var record *fastx.Record
		var fastxReader *fastx.Reader
		if number > 0 { // by number
			if !quiet {
				log.Info("sample by number")
			}

			if twoPass {
				if xopen.IsStdin() {
					checkError(fmt.Errorf("two-pass mode (-2) will failed when reading from stdin. please disable flag: -2"))
				}
				// first pass, get seq number
				if !quiet {
					log.Info("first pass: counting seq number")
				}
				seqNum, err := fastx.GetSeqNumber(file)
				checkError(err)

				if !quiet {
					log.Infof("seq number: %d", seqNum)
				}

				proportion = float64(number) / float64(seqNum) * 1.1

				// second pass
				if !quiet {
					log.Info("second pass: reading and sampling")
				}
				fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
				checkError(err)
			LOOP:
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

					if rand.Float64() <= proportion {
						n++
						record.FormatToWriter(outfh, config.LineWidth)
						if n == number {
							break LOOP
						}
					}
				}
			} else {
				records, err := fastx.GetSeqs(file, alphabet, config.Threads, 10, idRegexp)
				checkError(err)

				if len(records) > 0 && len(records[0].Seq.Qual) > 0 {
					config.LineWidth = 0
				}

				proportion = float64(number) / float64(len(records))

				for _, record := range records {
					if rand.Float64() <= proportion {
						n++
						record.FormatToWriter(outfh, config.LineWidth)
						if n == number {
							break
						}
					}
				}
			}
		} else {
			if !quiet {
				log.Info("sample by proportion")
			}

			fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
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

				if rand.Float64() <= proportion {
					n++
					record.FormatToWriter(outfh, config.LineWidth)
				}
			}
		}

		if !quiet {
			log.Infof("%d sequences outputted", n)
		}
	},
}

func init() {
	RootCmd.AddCommand(sampleCmd)

	sampleCmd.Flags().Int64P("rand-seed", "s", 11, "rand seed")
	sampleCmd.Flags().Int64P("number", "n", 0, "sample by number (result may not exactly match)")
	sampleCmd.Flags().Float64P("proportion", "p", 0, "sample by proportion")
	sampleCmd.Flags().BoolP("two-pass", "2", false, "2-pass mode read files twice to lower memory usage. Not allowed when reading from stdin")
}
