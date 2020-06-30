// Copyright © 2019 Oxford Nanopore Technologies.
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
	"math"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/bsipos/thist"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/util/byteutil"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// watchCmd represents the watch command
var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "monitoring and online histograms of sequence features",
	Long:  "monitoring and online histograms of sequence features",

	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		lineWidth := config.LineWidth
		outFile := config.OutFile

		printBins := getFlagInt(cmd, "bins")
		binMode := "termfit"
		if printBins > 0 {
			binMode = "fixed"
		}
		printFreq := getFlagInt(cmd, "print-freq")
		printDelay := getFlagInt(cmd, "delay")
		printQuiet := getFlagBool(cmd, "quiet-mode")
		printReset := getFlagBool(cmd, "reset")
		printDump := getFlagBool(cmd, "dump")
		printHelp := getFlagBool(cmd, "list-fields")
		printPdf := getFlagString(cmd, "img")

		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		runtime.GOMAXPROCS(config.Threads)

		pass := getFlagBool(cmd, "pass")
		logMode := getFlagBool(cmd, "log")
		_ = logMode
		fieldsText := getFlagString(cmd, "fields")
		validateSeq := getFlagBool(cmd, "validate-seq")
		validateSeqLength := getFlagValidateSeqLength(cmd, "validate-seq-length")
		qBase := getFlagPositiveInt(cmd, "qual-ascii-base")
		_ = qBase

		fields := strings.Split(fieldsText, ",")

		validFields := []string{"ReadLen", "MeanQual", "GC", "GCSkew"}

		type fieldInfo struct {
			Title    string
			Generate func(*fastx.Record) float64
		}

		fmap := make(map[string]fieldInfo)

		fmap["ReadLen"] = fieldInfo{
			"Read length",
			func(r *fastx.Record) float64 {
				return float64(len(r.Seq.Seq))
			},
		}

		fmap["MeanQual"] = fieldInfo{
			"Mean base quality",
			func(r *fastx.Record) float64 {
				return float64(r.Seq.AvgQual(qBase))
			},
		}

		fmap["GC"] = fieldInfo{
			"GC content",
			func(r *fastx.Record) float64 {
				g := r.Seq.BaseContent("G")
				c := r.Seq.BaseContent("C")
				return (g + c) * 100
			},
		}

		fmap["GCSkew"] = fieldInfo{
			"GC content",
			func(r *fastx.Record) float64 {
				g := r.Seq.BaseContent("G")
				c := r.Seq.BaseContent("C")
				return (g - c) / (g + c) * 100
			},
		}

		if printHelp {
			for _, f := range validFields {
				fmt.Printf("%-10s\t%s\n", f, fmap[f].Title)
			}
			os.Exit(0)
		}

		if len(fields) == 0 {
			fmt.Fprintf(os.Stderr, "No fields specified!")
			os.Exit(1)

		}

		for _, f := range fields {
			if fmap[f].Generate == nil {
				fmt.Fprintf(os.Stderr, "Invalid field: %s\n", f)
				os.Exit(1)
			}
		}

		transform := func(x float64) float64 { return x }
		if logMode {
			transform = func(x float64) float64 {
				return math.Log10(x + 1)
			}
		}

		seq.ValidateSeq = validateSeq
		seq.ValidateWholeSeq = false
		seq.ValidSeqLengthThreshold = validateSeqLength
		seq.ValidSeqThreads = config.Threads
		seq.ComplementThreads = config.Threads

		if !(alphabet == nil || alphabet == seq.Unlimit) {
			seq.ValidateSeq = true
		}

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var checkSeqType bool
		var isFastq bool
		var printQual bool
		var head []byte
		var sequence *seq.Seq
		var text []byte
		var b *bytes.Buffer
		var record *fastx.Record
		var fastxReader *fastx.Reader
		var count int

		field := fields[0]

		h := thist.NewHist([]float64{}, fmap[field].Title, binMode, printBins, true)

		for _, file := range files {
			fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)

			checkSeqType = true
			printQual = false
			for {
				record, err = fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}

				if checkSeqType {
					isFastq = fastxReader.IsFastq
					if isFastq {
						config.LineWidth = 0
						printQual = true
					}
					checkSeqType = false
				}

				p := transform(fmap[field].Generate(record))
				count++
				h.Update(p)

				if printFreq > 0 && count%printFreq == 0 {
					if printDump {
						os.Stderr.Write([]byte(h.Dump()))
					} else {
						if !printQuiet {
							os.Stderr.Write([]byte(thist.ClearScreenString()))
							os.Stderr.Write([]byte(h.Draw()))
						}
					}
					if printPdf != "" {
						h.SaveImage(printPdf)
					}
					time.Sleep(time.Duration(printDelay) * time.Second)
					if printReset {
						h = thist.NewHist([]float64{}, fmap[field].Title, binMode, printBins, true)
					}
				}

				if !pass {
					continue
				}

				head = record.Name

				if isFastq {
					outfh.WriteString("@")
					outfh.Write(head)
					outfh.WriteString("\n")
				} else {
					outfh.WriteString(">")
					outfh.Write(head)
					outfh.WriteString("\n")
				}

				sequence = record.Seq

				if len(sequence.Seq) <= pageSize {
					outfh.Write(byteutil.WrapByteSlice(sequence.Seq, config.LineWidth))
				} else {
					if bufferedByteSliceWrapper == nil {
						bufferedByteSliceWrapper = byteutil.NewBufferedByteSliceWrapper2(1, len(sequence.Seq), config.LineWidth)
					}
					text, b = bufferedByteSliceWrapper.Wrap(sequence.Seq, config.LineWidth)
					outfh.Write(text)
					outfh.Flush()
					bufferedByteSliceWrapper.Recycle(b)
				}

				outfh.WriteString("\n")

				if printQual {
					outfh.WriteString("+\n")

					if len(sequence.Qual) <= pageSize {
						outfh.Write(byteutil.WrapByteSlice(sequence.Qual, config.LineWidth))
					} else {
						if bufferedByteSliceWrapper == nil {
							bufferedByteSliceWrapper = byteutil.NewBufferedByteSliceWrapper2(1, len(sequence.Qual), config.LineWidth)
						}
						text, b = bufferedByteSliceWrapper.Wrap(sequence.Qual, config.LineWidth)
						outfh.Write(text)
						outfh.Flush()
						bufferedByteSliceWrapper.Recycle(b)
					}

					outfh.WriteString("\n")
				}

			} // record
			config.LineWidth = lineWidth

		} //file

		if printFreq < 0 || count%printFreq != 0 {
			if printDump {
				os.Stderr.Write([]byte(h.Dump()))
			} else {
				if !printQuiet {
					os.Stderr.Write([]byte(thist.ClearScreenString()))
					os.Stderr.Write([]byte(h.Draw()))
				}
			}
			if printPdf != "" {
				h.SaveImage(printPdf)
			}
		}

		outfh.Close()
	},
}

func init() {
	RootCmd.AddCommand(watchCmd)

	watchCmd.Flags().BoolP("validate-seq", "v", false, "validate bases according to the alphabet")
	watchCmd.Flags().BoolP("pass", "x", false, "pass through mode (write input to stdout)")
	watchCmd.Flags().BoolP("log", "L", false, "log10(x+1) transform numeric values")
	watchCmd.Flags().StringP("fields", "f", "ReadLen", "target fields, available values: ReadLen, MeanQual, GC, GCSkew")
	watchCmd.Flags().IntP("validate-seq-length", "V", 10000, "length of sequence to validate (0 for whole seq)")
	watchCmd.Flags().IntP("qual-ascii-base", "b", 33, "ASCII BASE, 33 for Phred+33")
	watchCmd.Flags().IntP("bins", "B", -1, "number of histogram bins")
	watchCmd.Flags().IntP("print-freq", "p", -1, "print/report after this many records (-1 for print after EOF)")
	watchCmd.Flags().BoolP("quiet-mode", "Q", false, "supress all plotting to stderr")
	watchCmd.Flags().BoolP("reset", "R", false, "reset histogram after every report")
	watchCmd.Flags().BoolP("dump", "y", false, "print histogram data to stderr instead of plotting")
	watchCmd.Flags().BoolP("list-fields", "H", false, "print out a list of available fields")
	watchCmd.Flags().IntP("delay", "W", 1, "sleep this many seconds after online plotting")
	watchCmd.Flags().StringP("img", "O", "", "save histogram to this PDF/image file")

}
