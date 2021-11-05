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
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// fx2tabCmd represents the fx2tab command
var fx2tabCmd = &cobra.Command{
	Use:   "fx2tab",
	Short: "convert FASTA/Q to tabular format (and length, GC content, average quality...)",
	Long: `convert FASTA/Q to tabular format, and provide various information,
like sequence length, GC content/GC skew.

Attention:
  1. Fixed three columns (ID, sequence, quality) are outputted for either FASTA
     or FASTQ, except when flag -n/--name is on. This is for format compatibility.

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		outFile := config.OutFile
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		onlyID := getFlagBool(cmd, "only-id")
		printLength := getFlagBool(cmd, "length")
		printGC := getFlagBool(cmd, "gc")
		printGCSkew := getFlagBool(cmd, "gc-skew")
		baseContents := getFlagStringSlice(cmd, "base-content")
		baseCounts := getFlagStringSlice(cmd, "base-count")
		caseSensitive := getFlagBool(cmd, "case-sensitive")
		onlyName := getFlagBool(cmd, "name")
		printTitle := getFlagBool(cmd, "header-line")
		printAlphabet := getFlagBool(cmd, "alphabet")
		printAvgQual := getFlagBool(cmd, "avg-qual")
		qBase := getFlagPositiveInt(cmd, "qual-ascii-base")
		printSeqHash := getFlagBool(cmd, "seq-hash")
		noQual := getFlagBool(cmd, "no-qual")

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		if printTitle {
			if onlyName {
				if onlyID {
					outfh.WriteString("#id")
				} else {
					outfh.WriteString("#name")
				}
			} else {
				if onlyID {
					if noQual {
						outfh.WriteString("#id\tseq")
					} else {
						outfh.WriteString("#id\tseq\tqual")
					}
				} else {
					if noQual {
						outfh.WriteString("#name\tseq")
					} else {
						outfh.WriteString("#name\tseq\tqual")
					}

				}
			}
			if printLength {
				outfh.WriteString("\tlength")
			}
			if printGC {
				outfh.WriteString("\tGC")
			}
			if printGCSkew {
				outfh.WriteString("\tGC-Skew")
			}
			if len(baseCounts) > 0 {
				for _, bc := range baseCounts {
					outfh.WriteString(fmt.Sprintf("\t%s", bc))
				}
			}
			if len(baseContents) > 0 {
				for _, bc := range baseContents {
					outfh.WriteString(fmt.Sprintf("\t%s", bc))
				}
			}
			if printAlphabet {
				outfh.WriteString("\talphabet")
			}
			if printAvgQual {
				outfh.WriteString("\tavg.qual")
			}
			if printSeqHash {
				outfh.WriteString("\tseq.hash")
			}

			outfh.WriteString("\n")
		}

		var name []byte
		var g, c float64
		var record *fastx.Record
		var fastxReader *fastx.Reader
		var sum [md5.Size]byte
		for _, file := range files {
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
				if onlyID {
					name = record.ID
				} else {
					name = record.Name
				}
				if onlyName {
					outfh.Write(name)
				} else {
					//outfh.WriteString(fmt.Sprintf("%s\t%s\t%s", name,
					//	record.Seq.Seq, record.Seq.Qual))

					// outfh.WriteString(fmt.Sprintf("%s\t", name))
					outfh.Write(name)
					outfh.Write(_tab)
					outfh.Write(record.Seq.Seq)
					if !noQual {
						outfh.Write(_tab)
						outfh.Write(record.Seq.Qual)
					}
				}

				if printLength {
					outfh.WriteString(fmt.Sprintf("\t%d", len(record.Seq.Seq)))
				}
				if printGC || printGCSkew {
					g = record.Seq.BaseContent("G")
					c = record.Seq.BaseContent("C")
				}

				if printGC {
					outfh.WriteString(fmt.Sprintf("\t%.2f", (g+c)*100))
				}
				if printGCSkew {
					outfh.WriteString(fmt.Sprintf("\t%.2f", (g-c)/(g+c)*100))
				}

				if len(baseCounts) > 0 {
					for _, bc := range baseCounts {
						if caseSensitive {
							outfh.WriteString(fmt.Sprintf("\t%d", record.Seq.BaseCountCaseSensitive(bc)))
						} else {
							outfh.WriteString(fmt.Sprintf("\t%d", record.Seq.BaseCount(bc)))
						}
					}
				}

				if len(baseContents) > 0 {
					for _, bc := range baseContents {
						if caseSensitive {
							outfh.WriteString(fmt.Sprintf("\t%.2f", record.Seq.BaseContentCaseSensitive(bc)*100))
						} else {
							outfh.WriteString(fmt.Sprintf("\t%.2f", record.Seq.BaseContent(bc)*100))
						}
					}
				}

				if printAlphabet {
					outfh.WriteString(fmt.Sprintf("\t%s", alphabetStr(record.Seq.Seq)))
				}

				if printAvgQual {
					outfh.WriteString(fmt.Sprintf("\t%.2f", record.Seq.AvgQual(qBase)))
				}

				if printSeqHash {
					if caseSensitive {
						sum = md5.Sum(record.Seq.Seq)
						outfh.WriteString(fmt.Sprintf("\t%s", hex.EncodeToString(sum[:])))
					} else {
						sum = md5.Sum(bytes.ToLower(record.Seq.Seq))
						outfh.WriteString(fmt.Sprintf("\t%s", hex.EncodeToString(sum[:])))
					}

				}

				// outfh.WriteString("\n")
				outfh.Write(_mark_newline)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(fx2tabCmd)

	fx2tabCmd.Flags().BoolP("length", "l", false, "print sequence length")
	fx2tabCmd.Flags().BoolP("gc", "g", false, "print GC content")
	fx2tabCmd.Flags().BoolP("gc-skew", "G", false, "print GC-Skew")
	fx2tabCmd.Flags().StringSliceP("base-content", "B", []string{}, "print base content. (case ignored, multiple values supported) e.g. -B AT -B N")
	fx2tabCmd.Flags().StringSliceP("base-count", "C", []string{}, "print base count. (case ignored, multiple values supported) e.g. -C AT -C N")
	fx2tabCmd.Flags().BoolP("case-sensitive", "I", false, "calculate case sensitive base content/sequence hash")
	fx2tabCmd.Flags().BoolP("only-id", "i", false, "print ID instead of full head")
	fx2tabCmd.Flags().BoolP("name", "n", false, "only print names (no sequences and qualities)")
	fx2tabCmd.Flags().BoolP("header-line", "H", false, "print header line")
	fx2tabCmd.Flags().BoolP("alphabet", "a", false, "print alphabet letters")
	fx2tabCmd.Flags().BoolP("avg-qual", "q", false, "print average quality of a read")
	fx2tabCmd.Flags().IntP("qual-ascii-base", "b", 33, "ASCII BASE, 33 for Phred+33")
	fx2tabCmd.Flags().BoolP("seq-hash", "s", false, "print hash (MD5) of sequence")
	fx2tabCmd.Flags().BoolP("no-qual", "Q", false, "only output two column even for FASTQ file")

}

func alphabetStr(s []byte) string {
	m := make(map[byte]struct{})
	for _, b := range s {
		m[b] = struct{}{}
	}
	alphabet := make([]string, len(m))
	i := 0
	for a := range m {
		alphabet[i] = string([]byte{a})
		i++
	}
	sort.Strings(alphabet)
	return strings.Join(alphabet, "")
}

var _tab = []byte{'\t'}
