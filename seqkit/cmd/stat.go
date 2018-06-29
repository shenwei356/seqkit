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
	"sync"

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

Tips:
  1. For lots of small files (especially on SDD), use big value of '-j' to
     parallelize counting.

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
		skipErr := getFlagBool(cmd, "skip-err")

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

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
		}

		ch := make(chan statInfo, config.Threads)
		statInfos := make([]statInfo, 0, 1000)

		cancel := make(chan struct{})

		done := make(chan int)
		go func() {
			var id uint64 = 1 // for keepping order
			buf := make(map[uint64]statInfo)

			for info := range ch {
				if info.err != nil {
					if skipErr {
						log.Warningf("%s: %s", info.file, info.err)
						continue
					} else {
						log.Errorf("%s: %s", info.file, info.err)
						close(cancel)
						break
					}
				}

				if id == info.id { // right the one
					if !tabular {
						statInfos = append(statInfos, info)
					} else {
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
					id++
				} else { // check bufferd result
					for true {
						if info1, ok := buf[id]; ok {
							if !tabular {
								statInfos = append(statInfos, info1)
							} else {
								if !all {
									outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%.1f\t%d\n",
										info1.file,
										info1.format,
										info1.t,
										info1.num,
										info1.lenSum,
										info1.lenMin,
										info1.lenAvg,
										info1.lenMax))
								} else {
									outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%.1f\t%d\t%d\t%d\t%d\t%d\t%d\n",
										info1.file,
										info1.format,
										info1.t,
										info1.num,
										info1.lenSum,
										info1.lenMin,
										info1.lenAvg,
										info1.lenMax,
										info1.Q1,
										info1.Q2,
										info1.Q3,
										info1.gapSum,
										info1.N50))
								}
							}

							delete(buf, info1.id)
							id++
						} else {
							break
						}
					}
					buf[info.id] = info
				}
			}

			if len(buf) > 0 {
				ids := make(sortutil.Uint64Slice, len(buf))
				i := 0
				for id := range buf {
					ids[i] = id
					i++
				}
				sort.Sort(ids)
				for _, id := range ids {
					info := buf[id]
					if !tabular {
						statInfos = append(statInfos, info)
					} else {
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
				}
			}

			done <- 1
		}()

		chFile := make(chan string, config.Threads)
		doneSendFile := make(chan int)
		go func() {
			for _, file := range files {
				select {
				case <-cancel:
					break
				default:
				}
				chFile <- file
			}
			close(chFile)
			doneSendFile <- 1
		}()

		var wg sync.WaitGroup
		token := make(chan int, config.Threads)
		var id uint64
		for file := range chFile {
			select {
			case <-cancel:
				break
			default:
			}

			token <- 1
			wg.Add(1)
			id++
			go func(file string, id uint64) {
				defer func() {
					wg.Done()
					<-token
				}()

				var num, l, lenMin, lenMax, lenSum, gapSum int64
				var n50, sum, l50 int64
				var q1, q2, q3 int64
				var lens sortutil.Int64Slice
				var seqFormat, t string
				var record *fastx.Record
				var fastxReader *fastx.Reader
				var err error

				fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
				if err != nil {
					select {
					case <-cancel:
						return
					default:
					}
					ch <- statInfo{file: file, err: err, id: id}
					return
				}

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
						if err != nil {
							select {
							case <-cancel:
								return
							default:
							}
							ch <- statInfo{file: file, err: err, id: id}
							return
						}
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

				select {
				case <-cancel:
					return
				default:
				}
				if num == 0 {
					ch <- statInfo{file, seqFormat, t,
						0, 0, 0, 0,
						0, lenMax, 0, 0,
						q1, q2, q3,
						nil, id}
				} else {
					ch <- statInfo{file, seqFormat, t,
						num, lenSum, gapSum, lenMin,
						math.Round(float64(lenSum)/float64(num), 1), lenMax, n50, l50,
						q1, q2, q3,
						nil, id}
				}
			}(file, id)
		}

		<-doneSendFile
		wg.Wait()
		close(ch)
		<-done

		select {
		case <-cancel:
			return
		default:
		}

		if tabular {
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

	err error
	id  uint64
}

func init() {
	RootCmd.AddCommand(statCmd)

	statCmd.Flags().BoolP("tabular", "T", false, "output in machine-friendly tabular format")
	statCmd.Flags().StringP("gap-letters", "G", "- .", "gap letters")
	statCmd.Flags().BoolP("all", "a", false, "all statistics, including quartiles of seq length, sum_gap, N50")
	statCmd.Flags().BoolP("skip-err", "e", false, "skip error, only show warning message")
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
