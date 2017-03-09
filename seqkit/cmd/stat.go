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
	"sort"

	"github.com/cznic/sortutil"
	"github.com/dustin/go-humanize"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/util/byteutil"
	"github.com/shenwei356/util/math"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
	"github.com/tatsushid/go-prettytable"
)

// statCmd represents the stat command
var statCmd = &cobra.Command{
	Use:     "stats",
	Aliases: []string{"stat"},
	Short:   "simple statistics of FASTA files",
	Long: `simple statistics of FASTA files

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		outFile := config.OutFile
		seq.AlphabetGuessSeqLenghtThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		gapLetters := []byte(getFlagString(cmd, "gap-letters"))
		if len(gapLetters) == 0 {
			checkError(fmt.Errorf("value of flag -G (--gap-letters) should not be empty"))
		}
		all := getFlagBool(cmd, "all")

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var fastxReader *fastx.Reader
		var num, l, lenMin, lenMax, lenSum, gapSum int64
		var n50, sum, l50 int64
		var lens sortutil.Int64Slice
		var seqFormat, t string
		statInfos := []statInfo{}
		for _, file := range files {
			fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)

			seqFormat = ""
			num, lenMin, lenMax, lenSum, gapSum = 0, int64(^uint64(0)>>1), 0, 0, 0
			n50, sum, l50 = 0, 0, 0
			if all {
				lens = make(sortutil.Int64Slice, 0, 1000)
			}
			for {
				record, err := fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}

				num++
				if seqFormat == "" {
					if len(record.Seq.Qual) > 0 {
						seqFormat = "FASTQ"
					} else {
						seqFormat = "FASTA"
					}
				}
				l = int64(len(record.Seq.Seq))
				if all {
					lens = append(lens, l)
				}
				lenSum += l
				if l < lenMin {
					lenMin = l
				}
				if l > lenMax {
					lenMax = l
				}
				gapSum += int64(byteutil.CountBytes(record.Seq.Seq, gapLetters))
			}

			if fastxReader.Alphabet() == seq.DNAredundant {
				t = "DNA"
			} else if fastxReader.Alphabet() == seq.RNAredundant {
				t = "RNA"
			} else {
				t = fmt.Sprintf("%s", fastxReader.Alphabet())
			}

			if all {
				sort.Sort(lens)
				for i := num - 1; i >= 0; i-- {
					sum += lens[i]
					if (sum << 1) >= lenSum {
						n50 = lens[i]
						l50 = num - i
						break
					}
				}
			}

			if num == 0 {
				statInfos = append(statInfos, statInfo{file, seqFormat, t,
					num, lenSum, gapSum, lenMin,
					0, lenMax, n50, l50})

			} else {
				statInfos = append(statInfos, statInfo{file, seqFormat, t,
					num, lenSum, gapSum, lenMin,
					math.Round(float64(lenSum)/float64(num), 1), lenMax, n50, l50})
			}
		}

		// format output
		columns := []prettytable.Column{
			{Header: "file"},
			{Header: "format"},
			{Header: "type"},
			{Header: "num_seqs", AlignRight: true},
			{Header: "sum_len", AlignRight: true},
			{Header: "min_len", AlignRight: true},
			{Header: "avg_len", AlignRight: true},
			{Header: "max_len", AlignRight: true}}

		if all {
			columns = append(columns, []prettytable.Column{
				{Header: "sum_gap", AlignRight: true},
				{Header: "N50", AlignRight: true},
				{Header: "L50", AlignRight: true},
			}...)
		}

		tbl, err := prettytable.NewTable(columns...)

		checkError(err)
		tbl.Separator = "  "

		for _, info := range statInfos {
			if !all {
				tbl.AddRow(
					info.file,
					info.format,
					info.t,
					humanize.Comma(info.num),
					humanize.Comma(info.lenSum),
					humanize.Comma(info.lenMin),
					humanize.Commaf(info.lenAvg),
					humanize.Comma(info.lenMax))
			} else {
				tbl.AddRow(
					info.file,
					info.format,
					info.t,
					humanize.Comma(info.num),
					humanize.Comma(info.lenSum),
					humanize.Comma(info.lenMin),
					humanize.Commaf(info.lenAvg),
					humanize.Comma(info.lenMax),
					humanize.Comma(info.gapSum),
					humanize.Comma(info.N50),
					humanize.Comma(info.L50))
			}
		}
		outfh.Write(tbl.Bytes())
	},
}

type statInfo struct {
	file   string
	format string
	t      string
	num    int64
	lenSum int64
	gapSum int64
	lenMin int64
	lenAvg float64
	lenMax int64
	N50    int64
	L50    int64
}

func init() {
	RootCmd.AddCommand(statCmd)

	statCmd.Flags().StringP("gap-letters", "G", "- ", "gap letters")
	statCmd.Flags().BoolP("all", "a", false, "all statistics, including sum_gap, N50, L50")
}
