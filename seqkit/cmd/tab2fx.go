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
	"bufio"
	"fmt"
	"runtime"
	"strings"

	"github.com/shenwei356/util/byteutil"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// tab2faCmd represents the tab2fx command
var tab2faCmd = &cobra.Command{
	GroupID: "format",

	Use:   "tab2fx",
	Short: "convert tabular format to FASTA/Q format",
	Long: `convert tabular format (first two/three columns) to FASTA/Q format

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		lineWidth := config.LineWidth
		outFile := config.OutFile
		runtime.GOMAXPROCS(config.Threads)

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", !config.SkipFileCheck)
		if !config.SkipFileCheck {
			for _, file := range files {
				checkIfFilesAreTheSame(file, outFile, "input", "output")
			}
		}

		commentPrefixes := getFlagStringSlice(cmd, "comment-line-prefix")
		bufferSizeS := getFlagString(cmd, "buffer-size")
		if bufferSizeS == "" {
			checkError(fmt.Errorf("value of buffer size. supported unit: K, M, G"))
		}
		bufferSize, err := ParseByteSize(bufferSizeS)
		if err != nil {
			checkError(fmt.Errorf("invalid value of buffer size. supported unit: K, M, G"))
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var line, p string
		var items []string
		var isCommentLine, isFastq bool
		var scanner *bufio.Scanner
		var fh *xopen.Reader
		buf := make([]byte, bufferSize)
		for _, file := range files {
			fh, err = xopen.Ropen(file)
			checkError(err)

			scanner = bufio.NewScanner(fh)
			scanner.Buffer(buf, int(bufferSize))

			isFastq = false
			for scanner.Scan() {
				line = strings.TrimRight(scanner.Text(), "\r\n")

				if line == "" {
					continue
				}
				// check comment line
				isCommentLine = false
				for _, p = range commentPrefixes {
					if strings.HasPrefix(line, p) {
						isCommentLine = true
						break
					}
				}
				if isCommentLine {
					continue
				}

				items = strings.Split(line, "\t")
				if len(items) < 2 {
					checkError(fmt.Errorf("at least two columns needed: %s", line))
				}

				if len(items) == 3 && (len(items[2]) > 0 || isFastq) { // fastq
					isFastq = true
					outfh.WriteString(fmt.Sprintf("@%s\n", items[0]))
					outfh.WriteString(items[1]) // seq
					outfh.WriteString("\n+\n")
					outfh.WriteString(items[2]) // qual

					outfh.WriteString("\n")
				} else {
					outfh.WriteString(fmt.Sprintf(">%s\n", items[0]))
					outfh.Write(byteutil.WrapByteSlice([]byte(items[1]), lineWidth))
					outfh.WriteString("\n")
				}

			}
			checkError(scanner.Err())
		}
	},
}

func init() {
	RootCmd.AddCommand(tab2faCmd)
	tab2faCmd.Flags().StringSliceP("comment-line-prefix", "p", []string{"#", "//"}, "comment line prefix")
	tab2faCmd.Flags().StringP("buffer-size", "b", "1G", `size of buffer, supported unit: K, M, G. You need increase the value when "bufio.Scanner: token too long" error reported`)
}
