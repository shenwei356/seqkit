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
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
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
	mathutil "github.com/shenwei356/util/math"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
)

// statCmd represents the stat command
var statCmd = &cobra.Command{
	GroupID: "basic",

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
  14. N50_num   N50_num or L50. https://en.wikipedia.org/wiki/N50,_L50,_and_related_statistics#L50
  15. Q20(%)    percentage of bases with the quality score greater than 20
  16. Q30(%)    percentage of bases with the quality score greater than 30
  17. AvgQual   average quality
  18. GC(%)     percentage of GC content
  19. sum_n     number of ambitious letters (N, n, X, x)
  
Attention:
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
		nLettersBytesNucl := []byte{'N', 'n'}
		nLettersBytesProt := []byte{'X', 'x'}

		skipFileCheck := getFlagBool(cmd, "skip-file-check")
		all := getFlagBool(cmd, "all")
		tabular := getFlagBool(cmd, "tabular")
		skipErr := getFlagBool(cmd, "skip-err")
		fqEncoding := parseQualityEncoding(getFlagString(cmd, "fq-encoding"))
		basename := getFlagBool(cmd, "basename")
		stdinLabel := getFlagString(cmd, "stdin-label")
		replaceStdinLabel := stdinLabel != "-"
		_NX := getFlagStringSlice(cmd, "N")
		hasNX := len(_NX) > 0

		NX := make([]float64, len(_NX))
		var err error
		for i, x := range _NX {
			NX[i], err = strconv.ParseFloat(x, 64)
			if err != nil {
				checkError(fmt.Errorf("the value of -N/--N should be a float: %s", x))
			}
			if NX[i] < 0 || NX[i] > 100 {
				checkError(fmt.Errorf("the value of -N/--N should be in the range of [0, 100]: %s", x))
			}
		}

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", !(skipFileCheck || config.SkipFileCheck))
		for _, file := range files {
			checkIfFilesAreTheSame(file, outFile, "input", "output")
		}

		style := &stable.TableStyle{
			Name: "plain",

			HeaderRow: stable.RowStyle{Begin: "", Sep: "  ", End: ""},
			DataRow:   stable.RowStyle{Begin: "", Sep: "  ", End: ""},
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
				colnames = append(colnames, []string{"Q1", "Q2", "Q3", "sum_gap", "N50", "N50_num", "Q20(%)", "Q30(%)", "AvgQual", "GC(%)", "sum_n"}...)
			}

			if hasNX {
				for _, x := range _NX {
					colnames = append(colnames, "N"+x)
				}
			}
			outfh.WriteString(strings.Join(colnames, "\t") + "\n")
		}

		ch := make(chan statInfo, config.Threads)
		statInfos := make([]statInfo, 0, 1024)

		cancel := make(chan struct{})

		done := make(chan int)
		var x float64
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
							fmt.Fprintf(outfh, "\t%.0f\t%.0f\t%.0f\t%d\t%d\t%d\t%.0f\t%.0f\t%.2f\t%.2f\t%d",
								info.Q1,
								info.Q2,
								info.Q3,
								info.gapSum,
								info.N50,
								info.L50,
								info.q20,
								info.q30,
								info.avgQual,
								info.gc,
								info.nSum,
							)
						}
						if hasNX {
							for _, x = range info.nx {
								fmt.Fprintf(outfh, "\t%.0f", x)
							}
						}
						outfh.WriteString("\n")
						outfh.Flush()
					}
					id++
					continue
				}

				buf[info.id] = info // save for later check

				// check bufferd results
				if info, ok := buf[id]; ok {
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
							fmt.Fprintf(outfh, "\t%.0f\t%.0f\t%.0f\t%d\t%d\t%d\t%.0f\t%.0f\t%.2f\t%.2f\t%d",
								info.Q1,
								info.Q2,
								info.Q3,
								info.gapSum,
								info.N50,
								info.L50,
								info.q20,
								info.q30,
								info.avgQual,
								info.gc,
								info.nSum,
							)
						}
						if hasNX {
							for _, x = range info.nx {
								fmt.Fprintf(outfh, "\t%.0f", x)
							}
						}
						outfh.WriteString("\n")
						outfh.Flush()
					}

					delete(buf, info.id)
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
							fmt.Fprintf(outfh, "\t%.0f\t%.0f\t%.0f\t%d\t%d\t%d\t%.0f\t%.0f\t%.2f\t%.2f\t%d",
								info.Q1,
								info.Q2,
								info.Q3,
								info.gapSum,
								info.N50,
								info.L50,
								info.q20,
								info.q30,
								info.avgQual,
								info.gc,
								info.nSum,
							)
						}
						if hasNX {
							for _, x = range info.nx {
								fmt.Fprintf(outfh, "\t%.0f", x)
							}
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
				var nSum uint64

				lensStats := util.NewLengthStats()

				var errSum, avgQual float64
				qual_map := seq.QUAL_MAP
				var q20, q30 int64
				var q byte
				var qual int
				var encodeOffset int = fqEncoding.Offset()
				var seqFormat, t string
				var record *fastx.Record
				var fastxReader *fastx.Reader
				var err error
				checkSeqType := true
				var isNucleotide bool

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

					if checkSeqType {
						checkSeqType = false

						if len(record.Seq.Qual) > 0 {
							seqFormat = "FASTQ"
						} else {
							seqFormat = "FASTA"
						}

						isNucleotide = fastxReader.Alphabet() == seq.DNA ||
							fastxReader.Alphabet() == seq.DNAredundant ||
							fastxReader.Alphabet() == seq.RNA || fastxReader.Alphabet() == seq.RNAredundant
					}

					lensStats.Add(uint64(len(record.Seq.Seq)))

					if all {
						if fastxReader.IsFastq {
							for _, q = range record.Seq.Qual {
								qual = int(q) - encodeOffset
								if qual >= 20 {
									q20++
									if qual >= 30 {
										q30++
									}
								}

								errSum += qual_map[qual]
							}
						}

						gapSum += uint64(byteutil.CountBytes(record.Seq.Seq, gapLettersBytes))
						if isNucleotide {
							gcSum += uint64(byteutil.CountBytes(record.Seq.Seq, gcLettersBytes))
							nSum += uint64(byteutil.CountBytes(record.Seq.Seq, nLettersBytesNucl))
						} else {
							nSum += uint64(byteutil.CountBytes(record.Seq.Seq, nLettersBytesProt))
						}
					}
				}

				fastxReader.Close()

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

					if errSum == 0 {
						avgQual = 0
					} else {
						avgQual = -10 * math.Log10(errSum/float64(lensStats.Sum()))
					}
				}
				var nx []float64
				if hasNX {
					nx = make([]float64, len(NX))
					for i, x := range NX {
						nx[i] = float64(lensStats.NX(x))
					}
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
						0, 0, 0, 0, 0,
						0, 0, 0, 0,
						0, 0, 0,
						0, 0, 0, 0,
						nx,
						nil, id}
				} else {
					if basename {
						file = filepath.Base(file)
					}
					if replaceStdinLabel && isStdin(file) {
						file = stdinLabel
					}
					ch <- statInfo{file, seqFormat, t,
						lensStats.Count(), lensStats.Sum(), gapSum, lensStats.Min(), nSum,
						mathutil.Round(lensStats.Mean(), 1), lensStats.Max(), n50, l50,
						q1, q2, q3,
						mathutil.Round(float64(q20)/float64(lensStats.Sum())*100, 2),
						mathutil.Round(float64(q30)/float64(lensStats.Sum())*100, 2),
						mathutil.Round(avgQual, 2),
						mathutil.Round(float64(gcSum)/float64(lensStats.Sum())*100, 2),
						nx,
						nil, id}
				}
			}(file, id)
		}

		<-doneSendFile
		wg.Wait()

		close(ch)
		<-done

		if !config.Quiet && len(files) > 1 {
			close(chDuration)
			<-doneDuration
			pbs.Wait()
		}

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
				{Header: "N50_num", Align: stable.AlignRight, HumanizeNumbers: true},
				{Header: "Q20(%)", Align: stable.AlignRight, HumanizeNumbers: true},
				{Header: "Q30(%)", Align: stable.AlignRight, HumanizeNumbers: true},
				{Header: "AvgQual", Align: stable.AlignRight, HumanizeNumbers: true},
				{Header: "GC(%)", Align: stable.AlignRight, HumanizeNumbers: true},
				{Header: "sum_n", Align: stable.AlignRight, HumanizeNumbers: true},
				// {Header: "L50", AlignRight: true},
			}...)
		}
		if hasNX {
			for _, x := range _NX {
				columns = append(columns, stable.Column{Header: "N" + x, Align: stable.AlignRight, HumanizeNumbers: true})
			}
		}

		tbl := stable.New()
		tbl.HeaderWithFormat(columns)
		for _, info := range statInfos {
			row := make([]interface{}, 0, len(columns))
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
				row = append(row, info.L50)
				row = append(row, info.q20)
				row = append(row, info.q30)
				row = append(row, info.avgQual)
				row = append(row, info.gc)
				row = append(row, info.nSum)
			}
			if hasNX {
				for _, x = range info.nx {
					row = append(row, x)
				}
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
	nSum   uint64

	lenAvg float64
	lenMax uint64
	N50    uint64
	L50    int

	Q1 float64
	Q2 float64
	Q3 float64

	q20     float64
	q30     float64
	avgQual float64

	gc float64

	nx []float64

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
	statCmd.Flags().StringSliceP("N", "N", []string{}, `append other N50-like stats as new columns. value range [0, 100], multiple values supported, e.g., -N 50,90 or -N 50 -N 90`)
	statCmd.Flags().BoolP("skip-file-check", "S", false, `skip input file checking when given files or a file list.`)

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
