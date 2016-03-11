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
	"strconv"
	"strings"

	"github.com/brentp/xopen"
	"github.com/shenwei356/bio/seqio/fasta"
	"github.com/shenwei356/util/byteutil"
	"github.com/spf13/cobra"
)

// subseqCmd represents the seq command
var subseqCmd = &cobra.Command{
	Use:   "subseq",
	Short: "get subsequence by region",
	Long: fmt.Sprintf(`get subsequence by region.

The definition of region is 1-based and with some custom design.

Examples:
%s
`, regionExample),
	Run: func(cmd *cobra.Command, args []string) {
		alphabet := getAlphabet(cmd, "seq-type")
		idRegexp := getFlagString(cmd, "id-regexp")
		chunkSize := getFlagInt(cmd, "chunk-size")
		threads := getFlagInt(cmd, "threads")
		lineWidth := getFlagInt(cmd, "line-width")
		outFile := getFlagString(cmd, "out-file")

		files := getFileList(args)

		region := getFlagString(cmd, "region")

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		if region == "" {
			checkError(fmt.Errorf("flag -r (--region) needed"))
		}

		if region != "" {
			if !reRegion.MatchString(region) {
				checkError(fmt.Errorf(`invalid region: %s. type "fakit subseq -h" for more examples`, region))
			}
			r := strings.Split(region, ":")
			start, err := strconv.Atoi(r[0])
			checkError(err)
			end, err := strconv.Atoi(r[1])
			checkError(err)
			if start == 0 || end == 0 {
				checkError(fmt.Errorf("both start and end should not be 0"))
			}
			if start < 0 && end > 0 {
				checkError(fmt.Errorf("when start < 0, end should not > 0"))
			}

			var s, e int
			for _, file := range files {
				fastaReader, err := fasta.NewFastaReader(alphabet, file, chunkSize, threads, idRegexp)
				checkError(err)
				for chunk := range fastaReader.Ch {
					checkError(chunk.Err)

					for _, record := range chunk.Data {
						s, e = start, end
						if s > 0 {
							s--
						}
						if e < 0 {
							e++
						}
						outfh.WriteString(fmt.Sprintf(">%s\n%s\n", record.Name,
							byteutil.WrapByteSlice(byteutil.SubSlice(record.Seq.Seq, s, e), lineWidth)))
					}
				}
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(subseqCmd)

	subseqCmd.Flags().StringP("region", "r", "", "subsequence of given region. "+
		`e.g 1:12 for first 12 bases, -12:-1 for last 12 bases, 13:-1 for cutting first 12 bases. type "fakit subseq -h" for more examples`)
}
