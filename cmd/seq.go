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
	"github.com/shenwei356/util/byteutil"
	"github.com/spf13/cobra"
)

// seqCmd represents the seq command
var seqCmd = &cobra.Command{
	Use:   "seq",
	Short: "transform sequence (revserse, complement, extract ID...)",
	Long: `transform sequence (revserse, complement, extract ID...)

`,
	Run: func(cmd *cobra.Command, args []string) {
		alphabet := getAlphabet(cmd, "seq-type")
		idRegexp := getFlagString(cmd, "id-regexp")
		chunkSize := getFlagInt(cmd, "chunk-size")
		threads := getFlagInt(cmd, "threads")
		lineWidth := getFlagInt(cmd, "line-width")
		outFile := getFlagString(cmd, "out-file")

		reverse := getFlagBool(cmd, "reverse")
		complement := getFlagBool(cmd, "complement")
		onlyName := getFlagBool(cmd, "name")
		onlySeq := getFlagBool(cmd, "seq")
		onlyID := getFlagBool(cmd, "only-id")
		removeGaps := getFlagBool(cmd, "remove-gaps")
		gapLetters := getFlagString(cmd, "gap-letter")
		lowerCase := getFlagBool(cmd, "lower-case")
		upperCase := getFlagBool(cmd, "upper-case")

		if lowerCase && upperCase {
			checkError(fmt.Errorf("could not give both flags -l (--lower-case) and -u (--upper-case)"))
		}

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var printName, printSeq bool
		var head []byte
		var sequence *seq.Seq
		for _, file := range files {
			fastaReader, err := fasta.NewFastaReader(alphabet, file, chunkSize, threads, idRegexp)
			checkError(err)

			for chunk := range fastaReader.Ch {
				checkError(chunk.Err)

				for _, record := range chunk.Data {
					printName, printSeq = true, true
					if onlyName && onlySeq {
						printName, printSeq = true, true
					} else if onlyName {
						printName, printSeq = true, false
					} else if onlySeq {
						printName, printSeq = false, true
					}
					if printName {
						if onlyID {
							head = record.ID
						} else {
							head = record.Name
						}

						if printSeq {
							outfh.WriteString(fmt.Sprintf(">%s\n", head))
						} else {
							outfh.WriteString(fmt.Sprintf("%s\n", head))
						}
					}

					if printSeq {
						sequence = record.Seq
						if reverse {
							sequence = sequence.Reverse()
						}
						if complement {
							sequence = sequence.Complement()
						}
						if removeGaps {
							sequence = sequence.RemoveGaps(gapLetters)
						}
						if lowerCase {
							outfh.WriteString(fmt.Sprintf("%s\n", byteutil.WrapByteSlice(bytes.ToLower(sequence.Seq), lineWidth)))
						} else if upperCase {
							outfh.WriteString(fmt.Sprintf("%s\n", byteutil.WrapByteSlice(bytes.ToUpper(sequence.Seq), lineWidth)))
						} else {
							outfh.WriteString(fmt.Sprintf("%s\n", byteutil.WrapByteSlice(sequence.Seq, lineWidth)))
						}
					}
				}
			}

			outfh.Close()
		}
	},
}

func init() {
	RootCmd.AddCommand(seqCmd)

	seqCmd.Flags().BoolP("reverse", "r", false, "reverse sequence)")
	seqCmd.Flags().BoolP("complement", "p", false, "complement sequence (blank for Protein sequence)")
	seqCmd.Flags().BoolP("name", "n", false, "only print names")
	seqCmd.Flags().BoolP("seq", "s", false, "only print sequences")
	seqCmd.Flags().BoolP("only-id", "i", false, "print ID instead of full head")
	seqCmd.Flags().BoolP("remove-gaps", "g", false, "remove gaps")
	seqCmd.Flags().StringP("gap-letter", "G", "-", "gap letters")
	seqCmd.Flags().BoolP("lower-case", "l", false, "print sequences in lower case")
	seqCmd.Flags().BoolP("upper-case", "u", false, "print sequences in upper case")
}
