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
	"github.com/spf13/cobra"
)

// fa2tabCmd represents the seq command
var fa2tabCmd = &cobra.Command{
	Use:   "fa2tab",
	Short: "covert FASTA to tabular format, and provide various information",
	Long: `covert FASTA to tabular format, and provide various information,
like sequence length, GC content.

`,
	Run: func(cmd *cobra.Command, args []string) {
		alphabet := getAlphabet(cmd, "seq-type")
		idRegexp := getFlagString(cmd, "id-regexp")
		chunkSize := getFlagInt(cmd, "chunk-size")
		threads := getFlagInt(cmd, "threads")
		outFile := getFlagString(cmd, "out-file")

		files := getFileList(args)

		onlyID := getFlagBool(cmd, "only-id")
		printLength := getFlagBool(cmd, "length")
		printGC := getFlagBool(cmd, "gc")
		baseContents := getFlagStringSlice(cmd, "base-content")
		onlyName := getFlagBool(cmd, "name")
		printTitle := getFlagBool(cmd, "title")

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		if printTitle {
			outfh.WriteString("# name\tseq")
			if printLength {
				outfh.WriteString("\tlength")
			}
			if printGC {
				outfh.WriteString("\tGC")
			}
			if len(baseContents) > 0 {
				for _, bc := range baseContents {
					outfh.WriteString(fmt.Sprintf("\t%s", bc))
				}
			}
			outfh.WriteString("\n")
		}

		var name []byte
		for _, file := range files {
			fastaReader, err := fasta.NewFastaReader(alphabet, file, chunkSize, threads, idRegexp)
			checkError(err)
			for chunk := range fastaReader.Ch {
				checkError(chunk.Err)

				for _, record := range chunk.Data {
					if onlyID {
						name = record.ID
					} else {
						name = record.Name
					}
					if onlyName {
						outfh.WriteString(fmt.Sprintf("%s\t%s", name, ""))
					} else {
						outfh.WriteString(fmt.Sprintf("%s\t%s", name, record.Seq.Seq))
					}

					if printLength {
						outfh.WriteString(fmt.Sprintf("\t%d", record.Seq.Length()))
					}
					if printGC {
						outfh.WriteString(fmt.Sprintf("\t%.2f", record.Seq.GC()*100))
					}
					if len(baseContents) > 0 {
						for _, bc := range baseContents {
							outfh.WriteString(fmt.Sprintf("\t%.2f", record.Seq.BaseContent(bc)*100))
						}
					}
					outfh.WriteString("\n")
				}
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(fa2tabCmd)

	fa2tabCmd.Flags().BoolP("length", "l", false, "print sequence length")
	fa2tabCmd.Flags().BoolP("gc", "g", false, "print GC content")
	fa2tabCmd.Flags().StringSliceP("base-content", "b", []string{}, "print base content. (case ignored, multiple values supported) e.g. -b AT -b N")
	fa2tabCmd.Flags().BoolP("only-id", "i", false, "print ID instead of full head")
	fa2tabCmd.Flags().BoolP("name", "n", false, "only print names (no sequences)")
	fa2tabCmd.Flags().BoolP("title", "T", false, "print title line")
}
