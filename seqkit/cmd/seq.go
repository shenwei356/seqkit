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
		quiet := config.Quiet
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		runtime.GOMAXPROCS(config.Threads)

		reverse := getFlagBool(cmd, "reverse")
		complement := getFlagBool(cmd, "complement")
		onlyName := getFlagBool(cmd, "name")
		onlySeq := getFlagBool(cmd, "seq")
		onlyQual := getFlagBool(cmd, "qual")
		onlyID := getFlagBool(cmd, "only-id")
		removeGaps := getFlagBool(cmd, "remove-gaps")
		gapLetters := getFlagString(cmd, "gap-letters")
		lowerCase := getFlagBool(cmd, "lower-case")
		upperCase := getFlagBool(cmd, "upper-case")
		dna2rna := getFlagBool(cmd, "dna2rna")
		rna2dna := getFlagBool(cmd, "rna2dna")
		color := getFlagBool(cmd, "color")
		validateSeq := getFlagBool(cmd, "validate-seq")
		validateSeqLength := getFlagValidateSeqLength(cmd, "validate-seq-length")
		minLen := getFlagInt(cmd, "min-len")
		maxLen := getFlagInt(cmd, "max-len")
		qBase := getFlagPositiveInt(cmd, "qual-ascii-base")
		minQual := getFlagFloat64(cmd, "min-qual")
		maxQual := getFlagFloat64(cmd, "max-qual")

		if gapLetters == "" {
			checkError(fmt.Errorf("value of flag -G (--gap-letters) should not be empty"))
		}
		for _, c := range gapLetters {
			if c > 127 {
				checkError(fmt.Errorf("value of -G (--gap-letters) contains non-ASCII characters"))
			}
		}

		if minLen >= 0 && maxLen >= 0 && minLen > maxLen {
			checkError(fmt.Errorf("value of flag -m (--min-len) should be >= value of flag -M (--max-len)"))
		}
		if minQual >= 0 && maxQual >= 0 && minQual > maxQual {
			checkError(fmt.Errorf("value of flag -Q (--min-qual) should be <= value of flag -R (--max-qual)"))
		}
		// if minLen >= 0 || maxLen >= 0 {
		// 	removeGaps = true
		// 	if !quiet {
		// 		log.Infof("flag -g (--remove-gaps) is switched on when using -m (--min-len) or -M (--max-len)")
		// 	}
		// }
		if (minLen >= 0 || maxLen >= 0) && !removeGaps {
			log.Warning("you may switch on flag -g/--remove-gaps to remove spaces")
		}

		seq.ValidateSeq = validateSeq
		seq.ValidateWholeSeq = false
		seq.ValidSeqLengthThreshold = validateSeqLength
		seq.ValidSeqThreads = config.Threads
		seq.ComplementThreads = config.Threads

		if complement && (alphabet == nil || alphabet == seq.Protein) {
			log.Warningf("flag -t (--seq-type) (DNA/RNA) is recommended for computing complement sequences")
		}

		if !validateSeq && !(alphabet == nil || alphabet == seq.Unlimit) {
			if !quiet {
				log.Info("when flag -t (--seq-type) given, flag -v (--validate-seq) is automatically switched on")
			}
			validateSeq = true
			seq.ValidateSeq = true
		}

		if lowerCase && upperCase {
			checkError(fmt.Errorf("could not give both flags -l (--lower-case) and -u (--upper-case)"))
		}

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var checkSeqType bool
		var isFastq bool
		var printName, printSeq, printQual bool
		var head []byte
		var sequence *seq.Seq
		var text []byte
		var b *bytes.Buffer
		var record *fastx.Record
		var fastxReader *fastx.Reader
		var seqCol *SeqColorizer
		if color {
			switch alphabet {
			case seq.DNA, seq.DNAredundant:
				seqCol = NewSeqColorizer("nucleic")
			case seq.Protein:
				seqCol = NewSeqColorizer("amino")
			default:
				seqCol = NewSeqColorizer("nucleic")
			}
		}
		for _, file := range files {
			fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)

			checkSeqType = true
			printQual = false
			once := true
			for {
				record, err = fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}

				if checkSeqType {
					isFastq = fastxReader.IsFastq
					if isFastq {
						config.LineWidth = 0
						printQual = true
					}
					checkSeqType = false
				}

				if removeGaps {
					record.Seq.RemoveGapsInplace(gapLetters)
				}

				if minLen >= 0 && len(record.Seq.Seq) < minLen {
					continue
				}

				if maxLen >= 0 && len(record.Seq.Seq) > maxLen {
					continue
				}

				if minQual > 0 || maxQual > 0 {
					avgQual := record.Seq.AvgQual(qBase)
					if minQual > 0 && avgQual < minQual {
						continue
					}
					if maxQual > 0 && avgQual >= maxQual {
						continue
					}
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

				sequence = record.Seq
				if reverse {
					sequence = sequence.ReverseInplace()
				}
				if complement {
					if !config.Quiet && record.Seq.Alphabet == seq.Protein || record.Seq.Alphabet == seq.Unlimit {
						log.Warning("complement does no take effect on protein/unlimit sequence")
					}
					sequence = sequence.ComplementInplace()
				}

				if printSeq {
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
						text := byteutil.WrapByteSlice(sequence.Seq, config.LineWidth)
						if color {
							text = seqCol.Color(text)
						}
						outfh.Write(text)
					} else {
						if bufferedByteSliceWrapper == nil {
							bufferedByteSliceWrapper = byteutil.NewBufferedByteSliceWrapper2(1, len(sequence.Seq), config.LineWidth)
						}
						text, b = bufferedByteSliceWrapper.Wrap(sequence.Seq, config.LineWidth)
						if color {
							text = seqCol.Color(text)
						}
						outfh.Write(text)
						outfh.Flush()
						bufferedByteSliceWrapper.Recycle(b)
					}

					outfh.WriteString("\n")
				}

				if printQual {
					if !onlyQual {
						outfh.WriteString("+\n")
					}

					if len(sequence.Qual) <= pageSize {
						if color {
							outfh.Write(byteutil.WrapByteSlice(seqCol.ColorQuals(sequence.Qual), config.LineWidth))
						} else {
							outfh.Write(byteutil.WrapByteSlice(sequence.Qual, config.LineWidth))
						}
					} else {
						if bufferedByteSliceWrapper == nil {
							bufferedByteSliceWrapper = byteutil.NewBufferedByteSliceWrapper2(1, len(sequence.Qual), config.LineWidth)
						}
						text, b = bufferedByteSliceWrapper.Wrap(sequence.Qual, config.LineWidth)
						if color {
							text = seqCol.ColorQuals(text)
						}
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
	seqCmd.Flags().BoolP("complement", "p", false, "complement sequence, flag '-v' is recommended to switch on")
	seqCmd.Flags().BoolP("name", "n", false, "only print names")
	seqCmd.Flags().BoolP("seq", "s", false, "only print sequences")
	seqCmd.Flags().BoolP("qual", "q", false, "only print qualities")
	seqCmd.Flags().BoolP("only-id", "i", false, "print ID instead of full head")
	seqCmd.Flags().BoolP("remove-gaps", "g", false, "remove gaps")
	seqCmd.Flags().StringP("gap-letters", "G", "- 	.", "gap letters")
	seqCmd.Flags().BoolP("lower-case", "l", false, "print sequences in lower case")
	seqCmd.Flags().BoolP("upper-case", "u", false, "print sequences in upper case")
	seqCmd.Flags().BoolP("dna2rna", "", false, "DNA to RNA")
	seqCmd.Flags().BoolP("rna2dna", "", false, "RNA to DNA")
	seqCmd.Flags().BoolP("color", "k", false, "colorize sequences - to be piped into \"less -R\"")
	seqCmd.Flags().BoolP("validate-seq", "v", false, "validate bases according to the alphabet")
	seqCmd.Flags().IntP("validate-seq-length", "V", 10000, "length of sequence to validate (0 for whole seq)")
	seqCmd.Flags().IntP("min-len", "m", -1, "only print sequences longer than the minimum length (-1 for no limit)")
	seqCmd.Flags().IntP("max-len", "M", -1, "only print sequences shorter than the maximum length (-1 for no limit)")
	seqCmd.Flags().IntP("qual-ascii-base", "b", 33, "ASCII BASE, 33 for Phred+33")
	seqCmd.Flags().Float64P("min-qual", "Q", -1, "only print sequences with average quality qreater or equal than this limit (-1 for no limit)")
	seqCmd.Flags().Float64P("max-qual", "R", -1, "only print sequences with average quality less than this limit (-1 for no limit)")
}
