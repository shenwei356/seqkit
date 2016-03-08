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

	"github.com/brentp/xopen"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fasta"
	"github.com/spf13/cobra"
)

// rmdupCmd represents the seq command
var rmdupCmd = &cobra.Command{
	Use:   "rmdup",
	Short: "remove duplicated sequences",
	Long: `remove duplicated sequences

`,
	Run: func(cmd *cobra.Command, args []string) {
		alphabet := getAlphabet(cmd, "seq-type")
		idRegexp := getFlagString(cmd, "id-regexp")
		chunkSize := getFlagInt(cmd, "chunk-size")
		threads := getFlagInt(cmd, "threads")
		lineWidth := getFlagInt(cmd, "line-width")
		outFile := getFlagString(cmd, "out-file")
		quiet := getFlagBool(cmd, "quiet")

		bySeq := getFlagBool(cmd, "by-seq")
		byName := getFlagBool(cmd, "by-name")
		ignoreCase := getFlagBool(cmd, "ignore-case")

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		readed := make(map[string]bool)
		var subject string
		removed := 0
		for _, file := range files {
			if alphabet == seq.Unlimit {
				alphabet, err = fasta.GuessAlphabet(file)
				checkError(err)
			}
			fastaReader, err := fasta.NewFastaReader(alphabet, file, chunkSize, threads, idRegexp)
			checkError(err)
			for chunk := range fastaReader.Ch {
				checkError(chunk.Err)

				for _, record := range chunk.Data {
					if bySeq {
						if ignoreCase {
							subject = MD5(bytes.ToLower(record.Seq.Seq))
						} else {
							subject = MD5(record.Seq.Seq)
						}
					} else if byName {
						subject = string(record.Name)
					} else { // byID
						subject = string(record.ID)
					}

					if _, ok := readed[subject]; ok { // duplicated
						removed++
					} else { // new one
						outfh.WriteString(fmt.Sprintf(">%s\n%s\n", record.Name, record.FormatSeq(lineWidth)))
						readed[subject] = true
					}
				}
			}
		}
		if !quiet {
			log.Info("%d duplicated records removed", removed)
		}
	},
}

func init() {
	RootCmd.AddCommand(rmdupCmd)

	rmdupCmd.Flags().BoolP("by-name", "n", false, "by full name instead of just id")
	rmdupCmd.Flags().BoolP("by-seq", "s", false, "by seq")
	rmdupCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
}
