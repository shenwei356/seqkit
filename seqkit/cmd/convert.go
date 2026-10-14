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
	GroupID: "format",

	Use:   "convert",
	Short: "convert FASTQ quality encoding between Sanger, Solexa and Illumina",
	Long: `convert FASTQ quality encoding between Sanger, Solexa and Illumina

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		outFile := config.OutFile
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = true
		quiet := config.Quiet
		runtime.GOMAXPROCS(config.Threads)

		dryRun := getFlagBool(cmd, "dry-run")
		force := getFlagBool(cmd, "force")

		from := parseQualityEncoding(getFlagString(cmd, "from"))

		toEncoding := parseQualityEncoding(getFlagString(cmd, "to"))

		var fromEncoding seq.QualityEncoding

		nrecords := getFlagPositiveInt(cmd, "nrecords")

		seq.NMostCommonThreshold = getFlagPositiveInt(cmd, "thresh-B-in-n-most-common")
		threshIllumina1p5Frac := getFlagFloat64(cmd, "thresh-illumina1.5-frac")

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", !config.SkipFileCheck)
		if !config.SkipFileCheck {
			for _, file := range files {
				checkIfFilesAreTheSame(file, outFile, "input", "output")
			}
		}
		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer closeOutput(outfh)
		lineWidth := config.LineWidth
		if !config.LineWidthChanged {
			lineWidth = 0
		}
		fastx.ForcelyOutputFastq = true // This command accepts only FASTQ, including empty records.

		for _, file := range files {
			func() {
				fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
				checkError(err)
				defer fastxReader.Close()

				guessing := from <= 0
				fromEncoding = from
				var records []*fastx.Record // Only the current file's unconverted sample.
				marks := make([]int, seq.NQualityEncoding)
				reportEncoding := func() {
					if !quiet {
						log.Infof("converting %s -> %s", fromEncoding, toEncoding)
					}
					if encodingsMatch(fromEncoding, toEncoding, force) {
						log.Warningf("source and target quality encoding match.")
					}
				}
				finishGuessing := func() {
					fromEncoding, err = guessQualityEncoding(marks, len(records), threshIllumina1p5Frac)
					checkError(err)
					if !quiet {
						log.Infof("guessed quality encoding: %s", fromEncoding)
					}
					reportEncoding()
					if !dryRun {
						for _, record := range records {
							record.Seq.Qual, err = seq.QualityConvert(fromEncoding, toEncoding, record.Seq.Qual, force)
							checkError(err)
							record.FormatToWriter(outfh, lineWidth)
						}
					}
					records = nil
					guessing = false
				}
				if !guessing {
					reportEncoding()
				}
				for {
					record, err := fastxReader.Read()
					if err == io.EOF {
						if guessing && len(records) > 0 {
							finishGuessing()
						}
						break
					}
					checkError(err)
					if !fastxReader.IsFastq {
						checkError(fmt.Errorf("this command only works for FASTQ format"))
					}
					if guessing {
						records = append(records, record.Clone())
						for _, encoding := range seq.GuessQualityEncoding(record.Seq.Qual) {
							marks[encoding]++
						}
						if len(records) == nrecords {
							finishGuessing()
							if dryRun {
								break
							}
						}
						continue
					}
					if dryRun {
						break
					}
					record.Seq.Qual, err = seq.QualityConvert(fromEncoding, toEncoding, record.Seq.Qual, force)
					checkError(err)
					record.FormatToWriter(outfh, lineWidth)
				}
			}()
		}
	},
}

// guessQualityEncoding intersects sample candidates and resolves equivalent encodings.
func guessQualityEncoding(marks []int, n int, illuminaFraction float64) (seq.QualityEncoding, error) {
	candidates := make([]seq.QualityEncoding, 0, seq.NQualityEncoding)
	for e, count := range marks {
		if count == n {
			candidates = append(candidates, seq.QualityEncoding(e))
		}
	}
	if float64(marks[seq.Illumina1p5])/float64(n) > illuminaFraction {
		candidates = []seq.QualityEncoding{seq.Illumina1p5}
	}
	if len(candidates) == 0 {
		return seq.Unknown, fmt.Errorf("quality encoding not consistent")
	}
	first := candidates[0]
	for _, e := range candidates[1:] {
		if e.IsSolexa() != first.IsSolexa() || e.Offset() != first.Offset() {
			return seq.Unknown, fmt.Errorf("fail to guess the source quality encoding, please specify it")
		}
	}
	encoding := candidates[len(candidates)-1]
	if encoding == seq.Illumina1p8 {
		encoding = seq.Sanger
	}
	return encoding, nil
}

var qualityEncodingCode string

func init() {
	RootCmd.AddCommand(convertCmd)

	convertCmd.Flags().StringP("from", "", "", `source quality encoding. if not given, we'll guess it`)
	convertCmd.Flags().StringP("to", "", "Sanger", `target quality encoding`)
	convertCmd.Flags().BoolP("dry-run", "d", false, `dry run`)
	convertCmd.Flags().BoolP("force", "f", false, `for Illumina-1.8+ -> Sanger, truncate scores > 40 to 40`)
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
		checkError(fmt.Errorf("unsupported quality encoding: %s. available values: 'sanger', 'solexa', 'illumina-1.3+', 'illumina-1.5+', 'illumina-1.8+'", s))
		return -1
	}
}

func encodingsMatch(source, target seq.QualityEncoding, force bool) bool {
	if source == target {
		return true
	}
	if !force {
		if source == seq.Sanger && target == seq.Illumina1p8 {
			return true
		}
		if source == seq.Illumina1p8 && target == seq.Sanger {
			return true
		}
	}
	return false
}
