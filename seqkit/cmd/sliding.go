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
	"bytes"
	"fmt"
	"io"
	"runtime"

	"github.com/shenwei356/xopen"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/util/byteutil"
	"github.com/spf13/cobra"
)

// slidingCmd represents the sliding command
var slidingCmd = &cobra.Command{
	Use:   "sliding",
	Short: "sliding sequences, circular genome supported",
	Long: `sliding sequences, circular genome supported

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		lineWidth := config.LineWidth
		outFile := config.OutFile
		seq.AlphabetGuessSeqLenghtThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.BufferSize)

		files := getFileList(args)

		circular := getFlagBool(cmd, "circular-genome")
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
		var text []byte
		var b *bytes.Buffer
		for _, file := range files {
			fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)
			for {
				record, err := fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}

				originalLen = len(record.Seq.Seq)
				sequence = record.Seq.Seq
				if circular {
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
					outfh.WriteString(fmt.Sprintf(">%s_sliding:%d-%d\n",
						record.ID, i+1, e))

					// outfh.Write(byteutil.WrapByteSlice(sequence[i:i+window], lineWidth))
					if window <= pageSize {
						outfh.Write(byteutil.WrapByteSlice(sequence[i:i+window], lineWidth))
					} else {
						if bufferedByteSliceWrapper == nil {
							bufferedByteSliceWrapper = byteutil.NewBufferedByteSliceWrapper2(1, window, lineWidth)
						}
						text, b = bufferedByteSliceWrapper.Wrap(sequence[i:i+window], lineWidth)
						outfh.Write(text)
						outfh.Flush()
						bufferedByteSliceWrapper.Recycle(b)
					}

					outfh.WriteString("\n")
				}
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(slidingCmd)

	slidingCmd.Flags().IntP("step", "s", 0, "step size")
	slidingCmd.Flags().IntP("window", "W", 0, "window size")
	slidingCmd.Flags().BoolP("circular-genome", "C", false, "circular genome")
}
