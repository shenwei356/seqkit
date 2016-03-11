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

	"github.com/brentp/xopen"
	"github.com/shenwei356/bio/seqio/fasta"
	"github.com/shenwei356/util/byteutil"
	"github.com/spf13/cobra"
)

// slidingCmd represents the seq command
var slidingCmd = &cobra.Command{
	Use:   "sliding",
	Short: "sliding sequences, circle genome supported",
	Long: `sliding sequences, circle genome supported

`,
	Run: func(cmd *cobra.Command, args []string) {
		alphabet := getAlphabet(cmd, "seq-type")
		idRegexp := getFlagString(cmd, "id-regexp")
		chunkSize := getFlagInt(cmd, "chunk-size")
		threads := getFlagInt(cmd, "threads")
		lineWidth := getFlagInt(cmd, "line-width")
		outFile := getFlagString(cmd, "out-file")

		files := getFileList(args)

		circle := getFlagBool(cmd, "circle-genome")
		step := getFlagInt(cmd, "step")
		window := getFlagInt(cmd, "window")
		if step == 0 || window == 0 {
			checkError(fmt.Errorf("both flags -s (--step) and -W (--window) needed"))
		}
		if step < 1 {
			checkError(fmt.Errorf("value of flag -s (--step) should be greater than 0: %d ", step))
		}
		if window < 1 {
			checkError(fmt.Errorf("value of flag -W (--window) should be greater than 0: %d ", window))
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var sequence []byte
		var originalLen, l, end, e int
		for _, file := range files {
			fastaReader, err := fasta.NewFastaReader(alphabet, file, chunkSize, threads, idRegexp)
			checkError(err)
			for chunk := range fastaReader.Ch {
				checkError(chunk.Err)

				for _, record := range chunk.Data {
					originalLen = len(record.Seq.Seq)
					sequence = record.Seq.Seq
					if circle {
						sequence = append(sequence, sequence[0:window-1]...)
					}

					l = len(sequence)
					end = l - window
					if end < 0 {
						end = 0
					}
					for i := 0; i <= end; i += step {
						e = i + window
						if e > originalLen {
							e = e - originalLen
						}
						outfh.WriteString(fmt.Sprintf(">%s sliding:%d-%d\n%s\n",
							record.Name, i+1, e,
							byteutil.WrapByteSlice(sequence[i:i+window], lineWidth)))
					}
				}
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(slidingCmd)

	slidingCmd.Flags().IntP("step", "s", 0, "step size")
	slidingCmd.Flags().IntP("window", "W", 0, "window size")
	slidingCmd.Flags().BoolP("circle-genome", "C", false, "circle genome")
}
