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
	"syscall"
	// "runtime/debug"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/util/byteutil"
	"github.com/shenwei356/xopen"
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
		lineWidth := config.LineWidth
		outFile := config.OutFile
		seq.AlphabetGuessSeqLenghtThreshold = config.AlphabetGuessSeqLength
		runtime.GOMAXPROCS(config.Threads)

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
		validateSeq := getFlagBool(cmd, "validate-seq")
		validateSeqLength := getFlagValidateSeqLength(cmd, "validate-seq-length")

		seq.ValidateSeq = validateSeq
		seq.ValidateWholeSeq = false
		seq.ValidSeqLengthThreshold = validateSeqLength
		seq.ValidSeqThreads = config.Threads
		seq.ComplementThreads = config.Threads

		if !(alphabet == nil || alphabet == seq.Unlimit) {
			log.Info("when flag -t (--seq-type) given, flag -v (--validate-seq) is automatically switched on")
			seq.ValidateSeq = true
		}

		if lowerCase && upperCase {
			checkError(fmt.Errorf("could not give both flags -l (--lower-case) and -u (--upper-case)"))
		}

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var isFastq bool
		checkSeqType := true
		var printName, printSeq, printQual bool
		var head []byte
		var sequence *seq.Seq
		var text []byte
		var b *bytes.Buffer
		for _, file := range files {
			fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)

			once := true
			for {
				record, err := fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}
				if fastxReader.IsFastq {
					config.LineWidth = 0
				}

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
							outfh.WriteString("@")
							outfh.Write(head)
							outfh.WriteString("\n")
						} else {
							outfh.WriteString(">")
							outfh.Write(head)
							outfh.WriteString("\n")
						}
					} else {
						outfh.Write(head)
						outfh.WriteString("\n")
					}
				}

				if printSeq {
					sequence = record.Seq
					if reverse {
						sequence = sequence.ReverseInplace()
					}
					if complement {
						if !config.Quiet && record.Seq.Alphabet == seq.Protein || record.Seq.Alphabet == seq.Unlimit {
							log.Warning("Complement does no take effect on protein/unlimit sequence")
						}
						sequence = sequence.ComplementInplace()
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

					if len(sequence.Seq) <= pageSize {
						outfh.Write(byteutil.WrapByteSlice(sequence.Seq, config.LineWidth))
					} else {
						if bufferedByteSliceWrapper == nil {
							bufferedByteSliceWrapper = byteutil.NewBufferedByteSliceWrapper2(1, len(sequence.Seq), config.LineWidth)
						}
						text, b = bufferedByteSliceWrapper.Wrap(sequence.Seq, config.LineWidth)
						outfh.Write(text)
						outfh.Flush()
						bufferedByteSliceWrapper.Recycle(b)
					}

					outfh.WriteString("\n")
				}

				if printQual {
					sequence = record.Seq
					if reverse {
						sequence = sequence.ReverseInplace()
					}
					if complement {
						if !config.Quiet && record.Seq.Alphabet == seq.Protein || record.Seq.Alphabet == seq.Unlimit {
							log.Warning("Complement does no take effect on protein/unlimit sequence")
						}
						sequence = sequence.ComplementInplace()
					}
					if removeGaps {
						sequence = sequence.RemoveGaps(gapLetters)
					}
					if !onlyQual {
						outfh.WriteString("+\n")
					}

					if len(sequence.Qual) <= pageSize {
						outfh.Write(byteutil.WrapByteSlice(sequence.Qual, config.LineWidth))
					} else {
						if bufferedByteSliceWrapper == nil {
							bufferedByteSliceWrapper = byteutil.NewBufferedByteSliceWrapper2(1, len(sequence.Qual), config.LineWidth)
						}
						text, b = bufferedByteSliceWrapper.Wrap(sequence.Qual, config.LineWidth)
						outfh.Write(text)
						outfh.Flush()
						bufferedByteSliceWrapper.Recycle(b)
					}

					outfh.WriteString("\n")
				}
			}

			config.LineWidth = lineWidth
		}

		outfh.Close()
	},
}

var pageSize = syscall.Getpagesize()

func init() {
	RootCmd.AddCommand(seqCmd)

	seqCmd.Flags().BoolP("reverse", "r", false, "reverse sequence")
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
	seqCmd.Flags().BoolP("validate-seq", "v", false, "validate bases according to the alphabet")
	seqCmd.Flags().IntP("validate-seq-length", "V", 10000, "length of sequence to validate (0 for whole seq)")

}
