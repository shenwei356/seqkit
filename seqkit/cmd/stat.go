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
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cznic/sortutil"
	"github.com/dustin/go-humanize"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/bio/util"
	"github.com/shenwei356/stable"
	"github.com/shenwei356/util/byteutil"
	"github.com/shenwei356/util/math"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
)

// statCmd represents the stat command
var statCmd = &cobra.Command{
	Use:     "stats",
	Aliases: []string{"stat"},
	Short:   "simple statistics of FASTA/Q files",
	Long: `simple statistics of FASTA/Q files

Columns:

  1.  file      input file, "-" for STDIN
  2.  format    FASTA or FASTQ
  3.  type      DNA, RNA, Protein or Unlimit
  4.  num_seqs  number of sequences
  5.  sum_len   number of bases or residues       , with gaps or spaces counted
  6.  min_len   minimal sequence length           , with gaps or spaces counted
  7.  avg_len   average sequence length           , with gaps or spaces counted
  8.  max_len   miximal sequence length           , with gaps or spaces counted
  9.  Q1        first quartile of sequence length , with gaps or spaces counted
  10. Q2        median of sequence length         , with gaps or spaces counted
  11. Q3        third quartile of sequence length , with gaps or spaces counted
  12. sum_gap   number of gaps
  13. N50       N50. https://en.wikipedia.org/wiki/N50,_L50,_and_related_statistics#N50
  14. Q20(%)    percentage of bases with the quality score greater than 20
  15. Q30(%)    percentage of bases with the quality score greater than 30
  16. GC(%)     percentage of GC content
  
Attentions:
  1. Sequence length metrics (sum_len, min_len, avg_len, max_len, Q1, Q2, Q3)
     count the number of gaps or spaces. You can remove them with "seqkit seq -g":
         seqkit seq -g input.fasta | seqkit stats

Tips:
  1. For lots of small files (especially on SDD), use a big value of '-j' to
     parallelize counting.
  2. Extract one metric with csvtk (https://github.com/shenwei356/csvtk):
         seqkit stats -Ta input.fastq.gz | csvtk cut -t -f "Q30(%)" | csvtk del-header 

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		outFile := config.OutFile
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
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
		gcLettersBytes := []byte{'g', 'c', 'G', 'C'}

		all := getFlagBool(cmd, "all")
		tabular := getFlagBool(cmd, "tabular")
		skipErr := getFlagBool(cmd, "skip-err")
		fqEncoding := parseQualityEncoding(getFlagString(cmd, "fq-encoding"))
		basename := getFlagBool(cmd, "basename")
		stdinLabel := getFlagString(cmd, "stdin-label")
		replaceStdinLabel := stdinLabel != "-"
		// NX := getFlagFloat64Slice(cmd, "N")

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		style := &stable.TableStyle{
			Name: "plain",

			HeaderRow: stable.RowStyle{"", "  ", ""},
			DataRow:   stable.RowStyle{"", "  ", ""},
			Padding:   "",
		}

		// process bar
		var pbs *mpb.Progress
		var bar *mpb.Bar
		var chDuration chan time.Duration
		var doneDuration chan int

		if !config.Quiet && len(files) > 1 {
			pbs = mpb.New(mpb.WithWidth(40), mpb.WithOutput(os.Stderr))
			bar = pbs.AddBar(int64(len(files)),
				mpb.BarStyle("[=>-]<+"),
				mpb.PrependDecorators(
					decor.Name("processed files: ", decor.WC{W: len("processed files: "), C: decor.DidentRight}),
					decor.Name("", decor.WCSyncSpaceR),
					decor.CountersNoUnit("%d / %d", decor.WCSyncWidth),
				),
				mpb.AppendDecorators(
					decor.Name("ETA: ", decor.WC{W: len("ETA: ")}),
					decor.EwmaETA(decor.ET_STYLE_GO, 5),
					decor.OnComplete(decor.Name(""), ". done"),
				),
			)

			chDuration = make(chan time.Duration, config.Threads)
			doneDuration = make(chan int)
			go func() {
				for t := range chDuration {
					bar.Increment()
					bar.DecoratorEwmaUpdate(t)
				}
				doneDuration <- 1
			}()
		}
		// process bar

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
				colnames = append(colnames, []string{"Q1", "Q2", "Q3", "sum_gap", "N50", "Q20(%)", "Q30(%)", "GC(%)"}...)
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
						fmt.Fprintf(outfh, "%s\t%s\t%s\t%d\t%d\t%d\t%.1f\t%d",
							info.file,
							info.format,
							info.t,
							info.num,
							info.lenSum,
							info.lenMin,
							info.lenAvg,
							info.lenMax)
						if all {
							fmt.Fprintf(outfh, "\t%.1f\t%.1f\t%.1f\t%d\t%d\t%.2f\t%.2f\t%.2f",
								info.Q1,
								info.Q2,
								info.Q3,
								info.gapSum,
								info.N50,
								info.q20,
								info.q30,
								info.gc)
						}
						outfh.WriteString("\n")
						outfh.Flush()
					}
					id++
					continue
				}

				buf[info.id] = info // save for later check

				// check bufferd results
				if info1, ok := buf[id]; ok {
					if !tabular {
						statInfos = append(statInfos, info1)
					} else {
						fmt.Fprintf(outfh, "%s\t%s\t%s\t%d\t%d\t%d\t%.1f\t%d",
							info.file,
							info.format,
							info.t,
							info.num,
							info.lenSum,
							info.lenMin,
							info.lenAvg,
							info.lenMax)
						if all {
							fmt.Fprintf(outfh, "\t%.1f\t%.1f\t%.1f\t%d\t%d\t%.2f\t%.2f\t%.2f",
								info.Q1,
								info.Q2,
								info.Q3,
								info.gapSum,
								info.N50,
								info.q20,
								info.q30,
								info.gc)
						}
						outfh.WriteString("\n")
						outfh.Flush()
					}

					delete(buf, info1.id)
					id++
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
						fmt.Fprintf(outfh, "%s\t%s\t%s\t%d\t%d\t%d\t%.1f\t%d",
							info.file,
							info.format,
							info.t,
							info.num,
							info.lenSum,
							info.lenMin,
							info.lenAvg,
							info.lenMax)
						if all {
							fmt.Fprintf(outfh, "\t%.1f\t%.1f\t%.1f\t%d\t%d\t%.2f\t%.2f\t%.2f",
								info.Q1,
								info.Q2,
								info.Q3,
								info.gapSum,
								info.N50,
								info.q20,
								info.q30,
								info.gc)
						}
						outfh.WriteString("\n")
						outfh.Flush()
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
		threadsFloat := float64(config.Threads) // just avoid repeated type conversion
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
				startTime := time.Now()
				defer func() {
					wg.Done()
					<-token

					if !config.Quiet && len(files) > 1 {
						chDuration <- time.Duration(float64(time.Since(startTime)) / threadsFloat)
					}
				}()

				var gapSum uint64
				var gcSum uint64

				lensStats := util.NewLengthStats()

				var q20, q30 int64
				var q byte
				var encodeOffset int = fqEncoding.Offset()
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
					if replaceStdinLabel && isStdin(file) {
						file = stdinLabel
					}
					ch <- statInfo{file: file, err: err, id: id}
					return
				}

				seqFormat = ""

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
							if replaceStdinLabel && isStdin(file) {
								file = stdinLabel
							}
							ch <- statInfo{file: file, err: err, id: id}
							return
						}
						break
					}

					if seqFormat == "" {
						if len(record.Seq.Qual) > 0 {
							seqFormat = "FASTQ"
						} else {
							seqFormat = "FASTA"
						}
					}

					lensStats.Add(uint64(len(record.Seq.Seq)))

					if all {
						if fastxReader.IsFastq {
							for _, q = range record.Seq.Qual {
								if int(q)-encodeOffset >= 20 {
									q20++
									if int(q)-encodeOffset >= 30 {
										q30++
									}
								}
							}
						}

						gapSum += uint64(byteutil.CountBytes(record.Seq.Seq, gapLettersBytes))
						gcSum += uint64(byteutil.CountBytes(record.Seq.Seq, gcLettersBytes))
					}
				}

				if fastxReader.Alphabet() == seq.DNAredundant {
					t = "DNA"
				} else if fastxReader.Alphabet() == seq.RNAredundant {
					t = "RNA"
				} else if seqFormat == "" && fastxReader.Alphabet() == seq.Unlimit {
					t = ""
				} else {
					t = fastxReader.Alphabet().String()
				}

				var n50 uint64
				var l50 int
				var q1, q2, q3 float64
				if all {
					n50 = lensStats.N50()
					l50 = lensStats.L50()
					q1, q2, q3 = lensStats.Q1(), lensStats.Q2(), lensStats.Q3()
				}

				select {
				case <-cancel:
					return
				default:
				}
				if lensStats.Count() == 0 {
					if basename {
						file = filepath.Base(file)
					}
					if replaceStdinLabel && isStdin(file) {
						file = stdinLabel
					}
					ch <- statInfo{file, seqFormat, t,
						0, 0, 0, 0,
						0, 0, 0, 0,
						0, 0, 0,
						0, 0, 0,
						nil, id}
				} else {
					if basename {
						file = filepath.Base(file)
					}
					if replaceStdinLabel && isStdin(file) {
						file = stdinLabel
					}
					ch <- statInfo{file, seqFormat, t,
						lensStats.Count(), lensStats.Sum(), gapSum, lensStats.Min(),
						math.Round(lensStats.Mean(), 1), lensStats.Max(), n50, l50,
						q1, q2, q3,
						math.Round(float64(q20)/float64(lensStats.Sum())*100, 2), math.Round(float64(q30)/float64(lensStats.Sum())*100, 2),
						math.Round(float64(gcSum)/float64(lensStats.Sum())*100, 2),
						nil, id}
				}
			}(file, id)
		}

		<-doneSendFile
		wg.Wait()

		if !config.Quiet && len(files) > 1 {
			close(chDuration)
			<-doneDuration
			pbs.Wait()
		}

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
		columns := []stable.Column{
			{Header: "file"},
			{Header: "format"},
			{Header: "type"},
			{Header: "num_seqs", Align: stable.AlignRight, HumanizeNumbers: true},
			{Header: "sum_len", Align: stable.AlignRight, HumanizeNumbers: true},
			{Header: "min_len", Align: stable.AlignRight, HumanizeNumbers: true},
			{Header: "avg_len", Align: stable.AlignRight, HumanizeNumbers: true},
			{Header: "max_len", Align: stable.AlignRight, HumanizeNumbers: true},
		}

		if all {
			columns = append(columns, []stable.Column{
				{Header: "Q1", Align: stable.AlignRight, HumanizeNumbers: true},
				{Header: "Q2", Align: stable.AlignRight, HumanizeNumbers: true},
				{Header: "Q3", Align: stable.AlignRight, HumanizeNumbers: true},
				{Header: "sum_gap", Align: stable.AlignRight, HumanizeNumbers: true},
				{Header: "N50", Align: stable.AlignRight, HumanizeNumbers: true},
				{Header: "Q20(%)", Align: stable.AlignRight, HumanizeNumbers: true},
				{Header: "Q30(%)", Align: stable.AlignRight, HumanizeNumbers: true},
				{Header: "GC(%)", Align: stable.AlignRight, HumanizeNumbers: true},
				// {Header: "L50", AlignRight: true},
			}...)
		}

		tbl := stable.New()
		tbl.HeaderWithFormat(columns)

		checkError(err)
		row := make([]interface{}, 0, len(columns))
		for _, info := range statInfos {
			row = append(row, info.file)
			row = append(row, info.format)
			row = append(row, info.t)
			row = append(row, humanize.Comma(int64(info.num)))
			row = append(row, info.lenSum)
			row = append(row, info.lenMin)
			row = append(row, info.lenAvg)
			row = append(row, info.lenMax)

			if all {
				row = append(row, info.Q1)
				row = append(row, info.Q2)
				row = append(row, info.Q3)
				row = append(row, info.gapSum)
				row = append(row, info.N50)
				row = append(row, info.q20)
				row = append(row, info.q30)
				row = append(row, info.gc)
			}

			tbl.AddRow(row)
		}
		outfh.Write(tbl.Render(style))
	},
}

type statInfo struct {
	file   string
	format string
	t      string

	num    uint64
	lenSum uint64
	gapSum uint64
	lenMin uint64

	lenAvg float64
	lenMax uint64
	N50    uint64
	L50    int

	Q1 float64
	Q2 float64
	Q3 float64

	q20 float64
	q30 float64

	gc float64

	// nx []float64

	err error
	id  uint64
}

func init() {
	RootCmd.AddCommand(statCmd)

	statCmd.Flags().BoolP("tabular", "T", false, "output in machine-friendly tabular format")
	statCmd.Flags().StringP("gap-letters", "G", "- .", "gap letters")
	statCmd.Flags().BoolP("all", "a", false, "all statistics, including quartiles of seq length, sum_gap, N50")
	statCmd.Flags().BoolP("skip-err", "e", false, "skip error, only show warning message")
	statCmd.Flags().StringP("fq-encoding", "E", "sanger", `fastq quality encoding. available values: 'sanger', 'solexa', 'illumina-1.3+', 'illumina-1.5+', 'illumina-1.8+'.`)
	statCmd.Flags().BoolP("basename", "b", false, "only output basename of files")
	statCmd.Flags().StringP("stdin-label", "i", "-", `label for replacing default "-" for stdin`)
	statCmd.Flags().Float64SliceP("N", "N", []float64{}, `other N50-like stats. range [0, 100]`)
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
