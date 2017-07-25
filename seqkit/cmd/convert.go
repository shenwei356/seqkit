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
	"bytes"
	"fmt"
	"io"
	"runtime"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "covert FASTQ quality",
	Long: `covert FASTQ quality

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		outFile := config.OutFile
		seq.AlphabetGuessSeqLenghtThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = true
		runtime.GOMAXPROCS(config.Threads)

		from := getFlagNonNegativeInt(cmd, "from")
		if from >= seq.NQualityEncoding {
			checkError(fmt.Errorf("unsupported quality encoding code: %d", from))
		}
		to := getFlagNonNegativeInt(cmd, "to")
		if to >= seq.NQualityEncoding {
			checkError(fmt.Errorf("unsupported quality encoding code: %d", to))
		}
		var fromEncoding seq.QualityEncoding
		toEncoding := seq.QualityEncoding(to)

		nrecords := getFlagPositiveInt(cmd, "nrecords")

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		guessing := from <= 0

		var records []*fastx.Record // records for guessing quality encoding
		var n int
		var encodings, encodingsGuessed []seq.QualityEncoding
		var encoding seq.QualityEncoding
		// for compute intersection of potential quality encoding from leading N records
		var encodingsMark []int
		if guessing {
			records = make([]*fastx.Record, nrecords+1)
			encodingsGuessed = make([]seq.QualityEncoding, 0, seq.NQualityEncoding)
			encodingsMark = make([]int, seq.NQualityEncoding)
		} else {
			fromEncoding = seq.QualityEncoding(from)
			log.Infof("converting %s -> %s", fromEncoding, toEncoding)
		}

		var record *fastx.Record
		var fastxReader *fastx.Reader
		for _, file := range files {
			fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)

			for {
				record, err = fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						if guessing && n < nrecords {
							for e, n2 := range encodingsMark {
								if n2 == n {
									encodingsGuessed = append(encodingsGuessed, seq.QualityEncoding(e))
								}
							}
							log.Infof("possible quality encodings: %v", encodingsGuessed)
							switch len(encodingsGuessed) {
							case 0:
								checkError(fmt.Errorf("quality encoding not consistent"))
							case 1:
								fromEncoding = encodingsGuessed[0]
								log.Infof("guessed quality encoding: %s", fromEncoding)
							default:
								same := true
								isSolexa := encodingsGuessed[0].IsSolexa()
								offset := encodingsGuessed[0].Offset()
								for i := 1; i < len(encodingsGuessed); i++ {
									if encodingsGuessed[i].IsSolexa() != isSolexa ||
										encodingsGuessed[i].Offset() != offset {
										same = false
										break
									}
									isSolexa = encodingsGuessed[i].IsSolexa()
									offset = encodingsGuessed[i].Offset()
								}
								if same {
									fromEncoding = encodingsGuessed[len(encodingsGuessed)-1]
									log.Infof("guessed quality encoding: %s", fromEncoding)
								} else {
									checkError(fmt.Errorf("fail to guess the source quality encoding, please specify it"))
								}
							}

							log.Infof("converting %s -> %s", fromEncoding, toEncoding)
							for _, record = range records {
								if record == nil {
									break
								}
								record.Seq.Qual, err = seq.QualityConvert(fromEncoding, toEncoding, record.Seq.Qual)
								checkError(err)
								record.FormatToWriter(outfh, config.LineWidth)
							}
						}
						break
					}
					checkError(err)
					break
				}
				if fastxReader.IsFastq {
					config.LineWidth = 0
				} else {
					checkError(fmt.Errorf("this command only works for FASTQ format"))
				}

				if guessing {
					if n < nrecords {
						records[n] = record.Clone()
						encodings = seq.GuessQualityEncoding(record.Seq.Qual)
						for _, encoding = range encodings {
							encodingsMark[encoding]++
						}
					} else if n == nrecords {
						for e, n2 := range encodingsMark {
							if n2 == n {
								encodingsGuessed = append(encodingsGuessed, seq.QualityEncoding(e))
							}
						}
						log.Infof("possible quality encodings: %v", encodingsGuessed)
						switch len(encodingsGuessed) {
						case 0:
							checkError(fmt.Errorf("quality encoding not consistent"))
						case 1:
							fromEncoding = encodingsGuessed[0]
							log.Infof("guessed quality encoding: %s", fromEncoding)
						default:
							same := true
							isSolexa := encodingsGuessed[0].IsSolexa()
							offset := encodingsGuessed[0].Offset()
							for i := 1; i < len(encodingsGuessed); i++ {
								if encodingsGuessed[i].IsSolexa() != isSolexa ||
									encodingsGuessed[i].Offset() != offset {
									same = false
									break
								}
								isSolexa = encodingsGuessed[i].IsSolexa()
								offset = encodingsGuessed[i].Offset()
							}
							if same {
								// choose the latest encoding
								fromEncoding = encodingsGuessed[len(encodingsGuessed)-1]
								log.Infof("guessed quality encoding: %s", fromEncoding)
							} else {
								checkError(fmt.Errorf("fail to guess the source quality encoding, please specify it"))
							}
						}

						log.Infof("converting %s -> %s", fromEncoding, toEncoding)
						records[n] = record.Clone()
						for _, record = range records {
							record.Seq.Qual, err = seq.QualityConvert(fromEncoding, toEncoding, record.Seq.Qual)
							checkError(err)
							record.FormatToWriter(outfh, config.LineWidth)
						}
						guessing = false
					}
					n++
					continue
				}

				record.Seq.Qual, err = seq.QualityConvert(fromEncoding, toEncoding, record.Seq.Qual)
				checkError(err)
				record.FormatToWriter(outfh, config.LineWidth)
			}
		}
	},
}

var qualityEncodingCode string

func init() {
	RootCmd.AddCommand(convertCmd)

	var buf bytes.Buffer
	for i := 0; i < seq.NQualityEncoding; i++ {
		buf.WriteString(fmt.Sprintf("%d for %s, ", i, seq.QualityEncoding(i)))
	}
	qualityEncodingCode = buf.String()

	convertCmd.Flags().IntP("from", "", 0,
		fmt.Sprintf(`source quality encoding. available value: %s`, qualityEncodingCode))
	convertCmd.Flags().IntP("to", "", int(seq.Illumina1p8),
		fmt.Sprintf(`target quality encoding. available value: %s`, qualityEncodingCode))
	convertCmd.Flags().IntP("nrecords", "n", 1000, "number of records for guessing quality encoding")
}
