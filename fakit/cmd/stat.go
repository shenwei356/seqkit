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
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/util/math"
	"github.com/spf13/cobra"
	"github.com/tatsushid/go-prettytable"
)

// statCmd represents the seq command
var statCmd = &cobra.Command{
	Use:   "stat",
	Short: "simple statistics of FASTA files",
	Long: `simple statistics of FASTA files

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		chunkSize := config.ChunkSize
		bufferSize := config.BufferSize
		outFile := config.OutFile
		seq.AlphabetGuessSeqLenghtThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var fastxReader *fastx.Reader
		var num, l, lenMin, lenMax, lenSum uint64
		var seqFormat, t string
		statInfos := []statInfo{}
		for _, file := range files {
			fastxReader, err = fastx.NewReader(alphabet, file, bufferSize, chunkSize, idRegexp)
			checkError(err)

			seqFormat = ""
			num, lenMin, lenMax, lenSum = 0, ^uint64(0), 0, 0
			for chunk := range fastxReader.Ch {
				checkError(chunk.Err)

				num += uint64(len(chunk.Data))
				for _, record := range chunk.Data {
					if seqFormat == "" {
						if len(record.Seq.Qual) > 0 {
							seqFormat = "FASTQ"
						} else {
							seqFormat = "FASTA"
						}
					}
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
			if fastxReader.Alphabet() == seq.DNAredundant {
				t = "DNA"
			} else if fastxReader.Alphabet() == seq.RNAredundant {
				t = "RNA"
			} else {
				t = fmt.Sprintf("%s", fastxReader.Alphabet())
			}

			statInfos = append(statInfos, statInfo{file, seqFormat, t, int64(num), int64(lenMin),
				math.Round(float64(lenSum)/float64(num), 1), int64(lenMax)})
		}

		// format output
		tbl, err := prettytable.NewTable([]prettytable.Column{
			{Header: "file"},
			{Header: "seq_format"},
			{Header: "seq_type"},
			{Header: "num_seqs", AlignRight: true},
			{Header: "min_len", AlignRight: true},
			{Header: "avg_len", AlignRight: true},
			{Header: "max_len", AlignRight: true}}...)
		checkError(err)
		tbl.Separator = "   "

		for _, info := range statInfos {
			tbl.AddRow(
				info.file,
				info.format,
				info.t,
				humanize.Comma(info.num),
				humanize.Comma(info.lenMin),
				humanize.Commaf(info.lenAvg),
				humanize.Comma(info.lenMax))
		}
		outfh.Write(tbl.Bytes())
	},
}

type statInfo struct {
	file   string
	format string
	t      string
	num    int64
	lenMin int64
	lenAvg float64
	lenMax int64
}

func init() {
	RootCmd.AddCommand(statCmd)
}
