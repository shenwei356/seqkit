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
	"reflect"
	"runtime"
	"strings"

	"github.com/brentp/xopen"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/breader"
	"github.com/shenwei356/util/byteutil"
	"github.com/spf13/cobra"
)

// tab2faCmd represents the seq command
var tab2faCmd = &cobra.Command{
	Use:   "tab2fa",
	Short: "covert tabular format to FASTA format",
	Long: `covert tabular format (first two columns) to FASTA format

`,
	Run: func(cmd *cobra.Command, args []string) {
		chunkSize := getFlagPositiveInt(cmd, "chunk-size")
		threads := getFlagPositiveInt(cmd, "threads")
		outFile := getFlagString(cmd, "out-file")
		lineWidth := getFlagNonNegativeInt(cmd, "line-width")
		seq.AlphabetGuessSeqLenghtThreshold = getFlagalphabetGuessSeqLength(cmd, "alphabet-guess-seq-length")
		runtime.GOMAXPROCS(threads)

		files := getFileList(args)

		commentPrefixes := getFlagStringSlice(cmd, "comment-line-prefix")

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		fn := func(line string) (interface{}, bool, error) {
			line = strings.TrimRight(line, "\n")

			// check comment line
			isCommentLine := false
			for _, p := range commentPrefixes {
				if strings.HasPrefix(line, p) {
					isCommentLine = true
					break
				}
			}
			if isCommentLine {
				return "", false, nil
			}

			items := strings.Split(line, "\t")
			if len(items) < 2 {
				return items, false, fmt.Errorf("at least two columns needed: %s", line)
			}
			return items[0:2], true, nil
		}

		for _, file := range files {
			reader, err := breader.NewBufferedReader(file, threads, chunkSize, fn)
			checkError(err)

			for chunk := range reader.Ch {
				for _, data := range chunk.Data {
					switch reflect.TypeOf(data).Kind() {
					case reflect.Slice:
						s := reflect.ValueOf(data)
						items := make([]string, s.Len())
						for i := 0; i < s.Len(); i++ {
							items[i] = s.Index(i).String()
						}
						outfh.WriteString(fmt.Sprintf(">%s\n%s\n", items[0], byteutil.WrapByteSlice([]byte(items[1]), lineWidth)))
					}
				}
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(tab2faCmd)
	tab2faCmd.Flags().StringSliceP("comment-line-prefix", "p", []string{"#", "//"}, "comment line prefix")
}
