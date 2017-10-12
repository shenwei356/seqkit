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
	"strings"

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
	Short:   "simple statistics of FASTA/Q files",
	Long: `simple statistics of FASTA/Q files

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		outFile := config.OutFile
		seq.AlphabetGuessSeqLenghtThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		gapLetters := getFlagString(cmd, "gap-letters")
		if len(gapLetters) == 0 {
			checkError(fmt.Errorf("value of flag -G (--gap-letters) should not be empty"))
		}
		for _, c := range gapLetters {
			if c > 127 {
				checkError(fmt.Errorf("value of -G (--gap-letters) contains non-ASCII characters"))
			}
		}
		gapLettersBytes := []byte(gapLetters)

		all := getFlagBool(cmd, "all")
		tabular := getFlagBool(cmd, "tabular")

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var num, l, lenMin, lenMax, lenSum, gapSum int64
		var n50, sum, l50 int64
		var q1, q2, q3 int64
		var lens sortutil.Int64Slice
		var seqFormat, t string
		statInfos := []statInfo{}
		var record *fastx.Record
		var fastxReader *fastx.Reader
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
				record, err = fastxReader.Read()
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
				gapSum += int64(byteutil.CountBytes(record.Seq.Seq, gapLettersBytes))
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
				q1, q2, q3 = quartile(lens)
			}

			if num == 0 {
				statInfos = append(statInfos, statInfo{file, seqFormat, t,
					0, 0, 0, 0,
					0, lenMax, 0, 0,
					q1, q2, q3})

			} else {
				statInfos = append(statInfos, statInfo{file, seqFormat, t,
					num, lenSum, gapSum, lenMin,
					math.Round(float64(lenSum)/float64(num), 1), lenMax, n50, l50,
					q1, q2, q3})
			}
		}

		// tabular output
		if tabular {
			colnames := []string{
				"file",
				"format",
				"type",
				"num_seqs",
				"sum_len",
				"min_len",
				"avg_len",
				"max_len",
			}
			if all {
				colnames = append(colnames, []string{"Q1", "Q2", "Q3", "sum_gap", "N50"}...)
			}
			outfh.WriteString(strings.Join(colnames, "\t") + "\n")

			for _, info := range statInfos {
				if !all {
					outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%.1f\t%d\n",
						info.file,
						info.format,
						info.t,
						info.num,
						info.lenSum,
						info.lenMin,
						info.lenAvg,
						info.lenMax))
				} else {
					outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%.1f\t%d\t%d\t%d\t%d\t%d\t%d\n",
						info.file,
						info.format,
						info.t,
						info.num,
						info.lenSum,
						info.lenMin,
						info.lenAvg,
						info.lenMax,
						info.Q1,
						info.Q2,
						info.Q3,
						info.gapSum,
						info.N50))
				}
			}
			return
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
				{Header: "Q1", AlignRight: true},
				{Header: "Q2", AlignRight: true},
				{Header: "Q3", AlignRight: true},
				{Header: "sum_gap", AlignRight: true},
				{Header: "N50", AlignRight: true},
				// {Header: "L50", AlignRight: true},
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
					humanize.Comma(info.Q1),
					humanize.Comma(info.Q2),
					humanize.Comma(info.Q3),
					humanize.Comma(info.gapSum),
					humanize.Comma(info.N50),
					// humanize.Comma(info.L50),
				)
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
	Q1     int64
	Q2     int64
	Q3     int64
}

func init() {
	RootCmd.AddCommand(statCmd)

	statCmd.Flags().BoolP("tabular", "T", false, "output in machine-friendly tabular format")
	statCmd.Flags().StringP("gap-letters", "G", "- .", "gap letters")
	statCmd.Flags().BoolP("all", "a", false, "all statistics, including quartiles of seq length, sum_gap, N50")
}

func median(sorted []int64) int64 {
	l := len(sorted)
	if l == 0 {
		return 0
	}
	if l%2 == 0 {
		return (sorted[l/2-1] + sorted[l/2]) / 2
	}
	return sorted[l/2]
}

func quartile(sorted []int64) (q1, q2, q3 int64) {
	l := len(sorted)
	if l == 0 {
		return
	}

	var c1, c2 int
	if l%2 == 0 {
		c1 = l / 2
		c2 = l / 2
	} else {
		c1 = (l - 1) / 2
		c2 = c1 + 1
	}
	q1 = median(sorted[:c1])
	q2 = median(sorted)
	q3 = median(sorted[c2:])
	return
}
