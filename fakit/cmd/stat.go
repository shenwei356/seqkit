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
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fasta"
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
		chunkSize := getFlagInt(cmd, "chunk-size")
		threads := getFlagInt(cmd, "threads")
		outFile := getFlagString(cmd, "out-file")

		if chunkSize <= 0 || threads <= 0 {
			checkError(fmt.Errorf("value of flag -c, -j, -w should be greater than 0"))
		}
		runtime.GOMAXPROCS(threads)

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		outfh.WriteString(fmt.Sprintf("file\ttype\tnum_seqs\tmin_len\tavg_len\tmax_len\n"))
		var num, l, lenMin, lenMax, lenSum uint64
		var t string
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
			outfh.WriteString(fmt.Sprintf("%s\t%s\t%d\t%d\t%.1f\t%d\n",
				file, t, num, lenMin, float64(lenSum)/float64(num), lenMax))
		}
	},
}

func init() {
	RootCmd.AddCommand(statCmd)
}
