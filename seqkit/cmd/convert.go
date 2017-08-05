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
	"io"
	"runtime"
	"strings"

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

		dryRun := getFlagBool(cmd, "dry-run")
		force := getFlagBool(cmd, "force")

		from := parseQualityEncoding(getFlagString(cmd, "from"))

		toEncoding := parseQualityEncoding(getFlagString(cmd, "to"))

		var fromEncoding seq.QualityEncoding

		nrecords := getFlagPositiveInt(cmd, "nrecords")

		seq.NMostCommonThreshold = getFlagPositiveInt(cmd, "thresh-B-in-n-most-common")
		threshIllumina1p5Frac := getFlagFloat64(cmd, "thresh-illumina1.5-frac")

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
			fromEncoding = from
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

							if float64(encodingsMark[seq.Illumina1p5]/len(records)) > threshIllumina1p5Frac {
								encodingsGuessed = []seq.QualityEncoding{seq.Illumina1p5}
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
									if fromEncoding == seq.Illumina1p8 {
										fromEncoding = seq.Sanger
									}
									log.Infof("guessed quality encoding: %s", fromEncoding)
								} else {
									checkError(fmt.Errorf("fail to guess the source quality encoding, please specify it"))
								}
							}

							log.Infof("converting %s -> %s", fromEncoding, toEncoding)

							if encodingsMatch(fromEncoding, toEncoding) {
								if force {
									log.Warningf("source and target quality encoding match.")
								} else {
									log.Warningf("source and target quality encoding match. aborted.")
									break
								}
							}

							if dryRun {
								break
							}
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

						if float64(encodingsMark[seq.Illumina1p5]/len(records)) > threshIllumina1p5Frac {
							encodingsGuessed = []seq.QualityEncoding{seq.Illumina1p5}
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
								if fromEncoding == seq.Illumina1p8 {
									fromEncoding = seq.Sanger
								}
								log.Infof("guessed quality encoding: %s", fromEncoding)
							} else {
								checkError(fmt.Errorf("fail to guess the source quality encoding, please specify it"))
							}
						}

						log.Infof("converting %s -> %s", fromEncoding, toEncoding)

						if encodingsMatch(fromEncoding, toEncoding) {
							if force {
								log.Warningf("source and target quality encoding match.")
							} else {
								log.Warningf("source and target quality encoding match. aborted.")
								break
							}
						}

						if dryRun {
							break
						}
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

				if encodingsMatch(fromEncoding, toEncoding) {
					if force {
						log.Warningf("source and target quality encoding match.")
					} else {
						log.Warningf("source and target quality encoding match. aborted.")
						break
					}
				}

				if dryRun {
					break
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

	convertCmd.Flags().StringP("from", "", "", `source quality encoding`)
	convertCmd.Flags().StringP("to", "", "Sanger", `target quality encoding`)
	convertCmd.Flags().BoolP("dry-run", "d", false, `dry run`)
	convertCmd.Flags().BoolP("force", "f", false, `force convert even source and target encoding match`)
	convertCmd.Flags().IntP("nrecords", "n", 1000, "number of records for guessing quality encoding")

	convertCmd.Flags().IntP("thresh-B-in-n-most-common", "N", seq.NMostCommonThreshold, "threshold of 'B' in top N most common quality for guessing Illumina 1.5.")
	convertCmd.Flags().Float64P("thresh-illumina1.5-frac", "F", 0.1, "threshold of faction of Illumina 1.5 in the leading N records")
}

func parseQualityEncoding(s string) seq.QualityEncoding {
	switch strings.ToLower(s) {
	case "sanger":
		return seq.Sanger
	case "solexa":
		return seq.Solexa
	case "illumina-1.3+":
		return seq.Illumina1p3
	case "illumina-1.5+":
		return seq.Illumina1p5
	case "illumina-1.8+":
		return seq.Illumina1p8
	case "":
		return seq.Unknown
	default:
		checkError(fmt.Errorf("unsupported quality encoding: %s", s))
		return -1
	}
}

func encodingsMatch(source, target seq.QualityEncoding) bool {
	if source == target {
		return true
	}
	if source == seq.Sanger && target == seq.Illumina1p8 {
		return true
	}
	if source == seq.Illumina1p8 && target == seq.Sanger {
		return true
	}
	return false
}
