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
	"runtime"

	"github.com/brentp/xopen"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/util/byteutil"
	"github.com/spf13/cobra"
)

// seqCmd represents the seq command
var seqCmd = &cobra.Command{
	Use:   "seq",
	Short: "transform sequences (revserse, complement, extract ID...)",
	Long: `transform sequences (revserse, complement, extract ID...)

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		chunkSize := config.ChunkSize
		threads := config.Threads
		lineWidth := config.LineWidth
		outFile := config.OutFile
		seq.AlphabetGuessSeqLenghtThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = true
		runtime.GOMAXPROCS(threads)

		reverse := getFlagBool(cmd, "reverse")
		complement := getFlagBool(cmd, "complement")
		onlyName := getFlagBool(cmd, "name")
		onlySeq := getFlagBool(cmd, "seq")
		onlyQual := getFlagBool(cmd, "qual")
		onlyID := getFlagBool(cmd, "only-id")
		removeGaps := getFlagBool(cmd, "remove-gaps")
		gapLetters := getFlagString(cmd, "gap-letter")
		lowerCase := getFlagBool(cmd, "lower-case")
		upperCase := getFlagBool(cmd, "upper-case")
		dna2rna := getFlagBool(cmd, "dna2rna")
		rna2dna := getFlagBool(cmd, "rna2dna")

		if lowerCase && upperCase {
			checkError(fmt.Errorf("could not give both flags -l (--lower-case) and -u (--upper-case)"))
		}
		runtime.GOMAXPROCS(threads)

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var isFastq bool
		checkSeqType := true
		var printName, printSeq, printQual bool
		var head []byte
		var sequence *seq.Seq
		for _, file := range files {
			fastxReader, err := fastx.NewReader(alphabet, file, threads, chunkSize, idRegexp)
			checkError(err)

			once := true
			for chunk := range fastxReader.Ch {
				checkError(chunk.Err)

				for _, record := range chunk.Data {
					if checkSeqType {
						if len(record.Seq.Qual) > 0 {
							isFastq = true
						}
						checkSeqType = false
					}
					printQual = false
					if isFastq {
						printQual = true
					}
					printName, printSeq = true, true
					if onlyName && onlySeq {
						printName, printSeq = true, true
					} else if onlyName {
						printName, printSeq, printQual = true, false, false
					} else if onlySeq {
						printName, printSeq, printQual = false, true, false
					} else if onlyQual {
						if !isFastq {
							checkError(fmt.Errorf("FASTA format has no quality. So do not just use flag -q (--qual)"))
						}
						printName, printSeq, printQual = false, false, true
					}
					if printName {
						if onlyID {
							head = record.ID
						} else {
							head = record.Name
						}

						if printSeq {
							if isFastq {
								outfh.WriteString(fmt.Sprintf("@%s\n", head))
							} else {
								outfh.WriteString(fmt.Sprintf(">%s\n", head))
							}
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
						if dna2rna {
							ab := fastxReader.Alphabet()
							if ab == seq.RNA || ab == seq.RNAredundant {
								if once {
									log.Warningf("it's already RNA, no need to convert")
									once = false
								}
							} else {
								for i, b := range sequence.Seq {
									switch b {
									case 't':
										sequence.Seq[i] = 'u'
									case 'T':
										sequence.Seq[i] = 'U'
									}
								}
							}
						}
						if rna2dna {
							ab := fastxReader.Alphabet()
							if ab == seq.DNA || ab == seq.DNAredundant {
								if once {
									log.Warningf("it's already DNA, no need to convert")
									once = false
								}
							} else {
								for i, b := range sequence.Seq {
									switch b {
									case 'u':
										sequence.Seq[i] = 't'
									case 'U':
										sequence.Seq[i] = 'T'
									}
								}
							}
						}
						if lowerCase {
							sequence.Seq = bytes.ToLower(sequence.Seq)
						} else if upperCase {
							sequence.Seq = bytes.ToUpper(sequence.Seq)
						}

						if isFastq {
							outfh.WriteString(fmt.Sprintf("%s\n", sequence.Seq))
						} else {
							outfh.WriteString(fmt.Sprintf("%s\n", byteutil.WrapByteSlice(sequence.Seq, lineWidth)))
						}
					}

					if printQual {
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
						if onlyQual {
							outfh.WriteString(fmt.Sprintf("%s\n", sequence.Qual))
						} else {
							outfh.WriteString(fmt.Sprintf("+\n%s\n", sequence.Qual))
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
	seqCmd.Flags().BoolP("qual", "q", false, "only print qualities")
	seqCmd.Flags().BoolP("only-id", "i", false, "print ID instead of full head")
	seqCmd.Flags().BoolP("remove-gaps", "g", false, "remove gaps")
	seqCmd.Flags().StringP("gap-letter", "G", "- ", "gap letters")
	seqCmd.Flags().BoolP("lower-case", "l", false, "print sequences in lower case")
	seqCmd.Flags().BoolP("upper-case", "u", false, "print sequences in upper case")
	seqCmd.Flags().BoolP("dna2rna", "", false, "DNA to RNA")
	seqCmd.Flags().BoolP("rna2dna", "", false, "RNA to DNA")
}
