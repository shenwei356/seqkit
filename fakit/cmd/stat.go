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
	"runtime"

	"github.com/brentp/xopen"
	"github.com/dustin/go-humanize"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fasta"
	"github.com/shenwei356/util/math"
	"github.com/spf13/cobra"
)

// statCmd represents the seq command
var statCmd = &cobra.Command{
	Use:   "stat",
	Short: "simple statistics of FASTA files",
	Long: `simple statistics of FASTA files

`,
	Run: func(cmd *cobra.Command, args []string) {
		alphabet := getAlphabet(cmd, "seq-type")
		idRegexp := getFlagString(cmd, "id-regexp")
		chunkSize := getFlagPositiveInt(cmd, "chunk-size")
		threads := getFlagPositiveInt(cmd, "threads")
		outFile := getFlagString(cmd, "out-file")
		seq.AlphabetGuessSeqLenghtThreshold = getFlagalphabetGuessSeqLength(cmd, "alphabet-guess-seq-length")
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(threads)

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var num, l, lenMin, lenMax, lenSum uint64
		var t string
		statInfos := []statInfo{}
		for _, file := range files {
			fastaReader, err := fasta.NewFastaReader(alphabet, file, threads, chunkSize, idRegexp)
			checkError(err)

			num, lenMin, lenMax, lenSum = 0, ^uint64(0), 0, 0
			for chunk := range fastaReader.Ch {
				checkError(chunk.Err)

				num += uint64(len(chunk.Data))
				for _, record := range chunk.Data {
					l = uint64(len(record.Seq.Seq))
					lenSum += l
					if l < lenMin {
						lenMin = l
					}
					if l > lenMax {
						lenMax = l
					}
				}
			}
			if fastaReader.Alphabet() == seq.DNAredundant {
				t = "DNA"
			} else if fastaReader.Alphabet() == seq.RNAredundant {
				t = "RNA"
			} else {
				t = fmt.Sprintf("%s", fastaReader.Alphabet())
			}

			statInfos = append(statInfos, statInfo{file, t, int64(num), int64(lenMin),
				math.Round(float64(lenSum)/float64(num), 1), int64(lenMax)})
		}

		// format output
		fileLen, tLen, numLen, lenMinLen, lenAvgLen, lenMaxLen := len("file"),
			len("seq_type"), len("num_seqs"), len("min_len"), len("avg_len"), len("max_len")

		l2 := 0
		for _, info := range statInfos {
			if len(info.file) > fileLen {
				fileLen = len(info.file)
			}
			if len(info.t) > tLen {
				tLen = len(info.t)
			}
			l2 = len(humanize.Comma(info.num))
			if l2 > numLen {
				numLen = l2
			}
			l2 = len(humanize.Comma(info.lenMin))
			if l2 > lenMinLen {
				lenMinLen = l2
			}
			l2 = len(humanize.Commaf(info.lenAvg))
			if l2 > lenAvgLen {
				lenAvgLen = l2
			}
			l2 = len(humanize.Comma(info.lenMax))
			if l2 > lenMaxLen {
				lenMaxLen = l2
			}
		}

		format := "%" + fmt.Sprintf("-%d", fileLen) + "s"
		for _, d := range []int{tLen, numLen, lenMinLen, lenAvgLen, lenMaxLen} {
			format += "    %" + fmt.Sprintf("%d", d) + "s"
		}
		format += "\n"

		outfh.WriteString(fmt.Sprintf(format, "file", "seq_type", "num_seqs",
			"min_len", "avg_len", "max_len"))
		for _, info := range statInfos {
			outfh.WriteString(fmt.Sprintf(format, info.file, info.t,
				humanize.Comma(info.num),
				humanize.Comma(info.lenMin),
				humanize.Commaf(info.lenAvg),
				humanize.Comma(info.lenMax)))
		}
	},
}

type statInfo struct {
	file   string
	t      string
	num    int64
	lenMin int64
	lenAvg float64
	lenMax int64
}

func init() {
	RootCmd.AddCommand(statCmd)
}
