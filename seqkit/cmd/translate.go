// Copyright © 2018
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
	"io"
	"runtime"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/util/byteutil"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// translateCmd represents the translate command
var translateCmd = &cobra.Command{
	Use:   "translate",
	Short: "translate DNA sequences",
	Long:  `translate DNA sequences into amino acids (proteins)`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		runtime.GOMAXPROCS(config.Threads)
		seq.ComplementThreads = config.Threads

		inputFiles := getFileList(args)
		outFileName := config.OutFile
		outFile, err := xopen.Wopen(outFileName)
		checkError(err)
		defer outFile.Close()

		var fastxReader *fastx.Reader
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp

		lowerCase := getFlagBool(cmd, "lower-case")
		upperCase := getFlagBool(cmd, "upper-case")
		translationTable := getFlagInt(cmd, "transl_table")

		for _, file := range inputFiles {
			fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)
			var fastxRecord *fastx.Record
			for {
				fastxRecord, err = fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}

				var protein []byte
				var outputBufferedText []byte
				var outputBuffer *bytes.Buffer
				sequence := fastxRecord.Seq

				if lowerCase {
					sequence.Seq = bytes.ToLower(sequence.Seq)
				} else if upperCase {
					sequence.Seq = bytes.ToUpper(sequence.Seq)
				}

				protein, err := seq.DNAToProtein(sequence.Seq, translationTable)
				if protein != nil && err == nil {
					if len(protein) <= pageSize {
						outFile.Write(byteutil.WrapByteSlice(protein, config.LineWidth))
					} else {
						if bufferedByteSliceWrapper == nil {
							bufferedByteSliceWrapper = byteutil.NewBufferedByteSliceWrapper2(1, len(protein), config.LineWidth)
						}
						outputBufferedText, outputBuffer = bufferedByteSliceWrapper.Wrap(protein, config.LineWidth)
						outFile.Write(outputBufferedText)
						outFile.Flush()
						bufferedByteSliceWrapper.Recycle(outputBuffer)
					}
					outFile.WriteString("\n")
				}
			}
		}
		outFile.Close()
	},
}

func init() {
	RootCmd.AddCommand(translateCmd)
	translateCmd.Flags().BoolP("lower-case", "l", false, "handle sequences in lower case")
	translateCmd.Flags().BoolP("upper-case", "u", false, "handle sequences in upper case")
	translateCmd.Flags().IntP("transl_table", "s", 1, "standard translation tables from NIBH standards")
}
