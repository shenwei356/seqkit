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
	"math"
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
	Short: "convert FASTA/Q to tabular format (with length/GC content/GC skew)",
	Long: `convert FASTA/Q to tabular format, and provide various information,
like sequence length, GC content/GC skew.

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		outFile := config.OutFile
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		infileList := getFlagString(cmd, "infile-list")
		files := getFileList(args, true)
		if infileList != "" {
			_files, err := getListFromFile(infileList, true)
			checkError(err)
			files = append(files, _files...)
		}

		onlyID := getFlagBool(cmd, "only-id")
		printLength := getFlagBool(cmd, "length")
		printGC := getFlagBool(cmd, "gc")
		printGCSkew := getFlagBool(cmd, "gc-skew")
		baseContents := getFlagStringSlice(cmd, "base-content")
		onlyName := getFlagBool(cmd, "name")
		printTitle := getFlagBool(cmd, "header-line")
		printAlphabet := getFlagBool(cmd, "alphabet")
		printAvgQual := getFlagBool(cmd, "avg-qual")
		qBase := getFlagPositiveInt(cmd, "qual-ascii-base")

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		if printTitle {
			outfh.WriteString("#name\tseq\tqual")
			if printLength {
				outfh.WriteString("\tlength")
			}
			if printGC {
				outfh.WriteString("\tGC")
			}
			if printGCSkew {
				outfh.WriteString("\tGC-Skew")
			}
			if len(baseContents) > 0 {
				for _, bc := range baseContents {
					outfh.WriteString(fmt.Sprintf("\t%s", bc))
				}
			}
			if printAlphabet {
				outfh.WriteString("\talphabet")
			}

			outfh.WriteString("\n")
		}

		var name []byte
		var g, c float64
		var record *fastx.Record
		var fastxReader *fastx.Reader
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
					outfh.WriteString(fmt.Sprintf("%s\t%s\t%s", name, "", ""))
				} else {
					//outfh.WriteString(fmt.Sprintf("%s\t%s\t%s", name,
					//	record.Seq.Seq, record.Seq.Qual))
					outfh.WriteString(fmt.Sprintf("%s\t", name))
					outfh.Write(record.Seq.Seq)
					outfh.WriteString("\t")
					outfh.Write(record.Seq.Qual)

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

				if len(baseContents) > 0 {
					for _, bc := range baseContents {
						outfh.WriteString(fmt.Sprintf("\t%.2f", record.Seq.BaseContent(bc)*100))
					}
				}

				if printAlphabet {
					outfh.WriteString(fmt.Sprintf("\t%s", alphabetStr(record.Seq.Seq)))
				}

				if printAvgQual {
					outfh.WriteString(fmt.Sprintf("\t%.2f", avgQual(record.Seq, qBase)))
				}
				outfh.WriteString("\n")
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
	fx2tabCmd.Flags().BoolP("only-id", "i", false, "print ID instead of full head")
	fx2tabCmd.Flags().BoolP("name", "n", false, "only print names (no sequences and qualities)")
	fx2tabCmd.Flags().BoolP("header-line", "H", false, "print header line")
	fx2tabCmd.Flags().BoolP("alphabet", "a", false, "print alphabet letters")
	fx2tabCmd.Flags().BoolP("avg-qual", "q", false, "print average quality of a read")
	fx2tabCmd.Flags().IntP("qual-ascii-base", "b", 33, "ASCII BASE, 33 for Phred+33")

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

func avgQual(s *seq.Seq, base int) float64 {
	if len(s.Qual) == 0 {
		return 0
	}
	s.ParseQual(base)
	var sum float64
	for _, q := range s.QualValue {
		sum += math.Pow(10, float64(q)/-10)
	}
	return -10 * math.Log10(sum/float64(len(s.QualValue)))
}
