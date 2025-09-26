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
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	// "runtime/debug"

	"github.com/dsnet/compress/bzip2"
	gzip "github.com/klauspost/pgzip"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/spf13/cobra"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

// seqCmd represents the seq command
var seqCmd = &cobra.Command{
	GroupID: "basic",

	Use:   "seq",
	Short: "transform sequences (extract ID, filter by length, remove gaps, reverse complement...)",
	Long: `transform sequences (extract ID, filter by length, remove gaps, reverse complement...)

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		lineWidth := config.LineWidth
		outFile := config.OutFile
		quiet := config.Quiet
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ComplementSeqLenThreshold = 1000000
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
		minLen := getFlagInt(cmd, "min-len")
		maxLen := getFlagInt(cmd, "max-len")
		qBase := getFlagPositiveInt(cmd, "qual-ascii-base")
		minQual := getFlagFloat64(cmd, "min-qual")
		maxQual := getFlagFloat64(cmd, "max-qual")

		filterMinLen := minLen >= 0
		filterMaxLen := maxLen >= 0
		filterMinQual := minQual > 0
		filterMaxQual := maxQual > 0

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

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", !config.SkipFileCheck)
		for _, file := range files {
			checkIfFilesAreTheSame(file, outFile, "input", "output")
		}

		var seqCol *SeqColorizer
		if color {
			switch alphabet {
			case seq.DNA, seq.DNAredundant, seq.RNA, seq.RNAredundant:
				seqCol = NewSeqColorizer("nucleic")
			case seq.Protein:
				seqCol = NewSeqColorizer("amino")
			default:
				seqCol = NewSeqColorizer("nucleic")
			}

			if outFile != "-" {
				log.Warning("flag -k/--color only applies for stdout")
			}
		}
		var outfh *os.File
		var err error
		if outFile == "-" {
			outfh = os.Stdout
		} else {
			outfh, err = os.Create(outFile)
			checkError(err)
			color = false
		}

		outFileLower := strings.ToLower(outFile)
		gzippedOutfile := strings.HasSuffix(outFileLower, ".gz")
		xzOutfile := strings.HasSuffix(outFileLower, ".xz")
		zstdOutfile := strings.HasSuffix(outFileLower, ".zst")
		bzip2Outfile := strings.HasSuffix(outFileLower, ".bz2")

		var fh io.Writer
		var outbw *bufio.Writer
		var gw *gzip.Writer
		var xw *xz.Writer
		var zw *zstd.Encoder
		var bz2 *bzip2.Writer

		if color {
			fh = seqCol.WrapWriter(outfh)
			outbw = bufio.NewWriterSize(fh, bufSize)
		} else if gzippedOutfile {
			gw, err = gzip.NewWriterLevel(outfh, config.CompressionLevel)
			if err != nil {
				checkError(err)
			}
			outbw = bufio.NewWriterSize(gw, bufSize)
		} else if xzOutfile {
			xw, err = xz.NewWriter(outfh)
			if err != nil {
				checkError(err)
			}
			outbw = bufio.NewWriterSize(xw, bufSize)
		} else if zstdOutfile {
			zw, err = zstd.NewWriter(outfh, zstd.WithEncoderLevel(zstd.EncoderLevel(config.CompressionLevel)))
			if err != nil {
				checkError(err)
			}
			outbw = bufio.NewWriterSize(zw, bufSize)
		} else if bzip2Outfile {
			bz2, err = bzip2.NewWriter(outfh, &bzip2.WriterConfig{Level: config.CompressionLevel})
			if err != nil {
				checkError(err)
			}
			outbw = bufio.NewWriterSize(bz2, bufSize)
		} else {
			fh = outfh
			outbw = bufio.NewWriterSize(fh, bufSize)
		}

		defer func() {
			checkError(outbw.Flush())

			if gzippedOutfile {
				checkError(gw.Flush())
				checkError(gw.Close())
			}
			if xzOutfile {
				checkError(xw.Close())
			}

			if zstdOutfile {
				checkError(zw.Flush())
				checkError(zw.Close())
			}

			if bzip2Outfile {
				checkError(bz2.Close())
			}

			checkError(outfh.Close())
		}()

		var checkSeqType bool
		var isFastq bool
		var printName, printSeq, printQual bool
		var head []byte
		var sequence *seq.Seq
		var text []byte
		var buffer *bytes.Buffer
		var record *fastx.Record

		for _, file := range files {
			fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)

			checkSeqType = true
			printQual = false
			once := true
			if onlySeq || onlyQual {
				config.LineWidth = 0
			}
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

				if filterMinLen && len(record.Seq.Seq) < minLen {
					continue
				}

				if filterMaxLen && len(record.Seq.Seq) > maxLen {
					continue
				}

				if filterMinQual || filterMaxQual {
					avgQual := record.Seq.AvgQual(qBase)
					if filterMinQual && avgQual < minQual {
						continue
					}
					if filterMaxQual && avgQual >= maxQual {
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
							outbw.Write(_mark_fastq)
							outbw.Write(head)
							outbw.Write(_mark_newline)
						} else {
							outbw.Write(_mark_fasta)
							outbw.Write(head)
							outbw.Write(_mark_newline)
						}
					} else {
						outbw.Write(head)
						outbw.Write(_mark_newline)
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

					if isFastq {
						if color {
							if sequence.Qual != nil {
								outbw.Write(seqCol.ColorWithQuals(sequence.Seq, sequence.Qual))
							} else {
								outbw.Write(seqCol.Color(sequence.Seq))
							}
						} else {
							outbw.Write(sequence.Seq)
						}
					} else {
						text, buffer = wrapByteSlice(sequence.Seq, config.LineWidth, buffer)

						if color {
							if sequence.Qual != nil {
								text = seqCol.ColorWithQuals(text, sequence.Qual)
							} else {
								text = seqCol.Color(text)
							}
						}

						outbw.Write(text)
					}

					outbw.Write(_mark_newline)
				}

				if printQual {
					if !onlyQual {
						outbw.Write(_mark_plus_newline)
					}

					if color {
						outbw.Write(seqCol.ColorQuals(sequence.Qual))
					} else {
						outbw.Write(sequence.Qual)
					}

					outbw.Write(_mark_newline)
				}
			}
			fastxReader.Close()

			config.LineWidth = lineWidth
		}

	},
}

var bufSize = 65536

func init() {
	RootCmd.AddCommand(seqCmd)

	seqCmd.Flags().BoolP("reverse", "r", false, "reverse sequence")
	seqCmd.Flags().BoolP("complement", "p", false, "complement sequence, flag '-v' is recommended to switch on")
	seqCmd.Flags().BoolP("name", "n", false, "only print names/sequence headers")
	seqCmd.Flags().BoolP("seq", "s", false, "only print sequences")
	seqCmd.Flags().BoolP("qual", "q", false, "only print qualities")
	seqCmd.Flags().BoolP("only-id", "i", false, "print IDs instead of full headers")
	seqCmd.Flags().BoolP("remove-gaps", "g", false, `remove gaps letters set by -G/--gap-letters, e.g., spaces, tabs, and dashes (gaps "-" in aligned sequences)`)
	seqCmd.Flags().StringP("gap-letters", "G", "- 	.", `gap letters to be removed with -g/--remove-gaps`)
	seqCmd.Flags().BoolP("lower-case", "l", false, "print sequences in lower case")
	seqCmd.Flags().BoolP("upper-case", "u", false, "print sequences in upper case")
	seqCmd.Flags().BoolP("dna2rna", "", false, "DNA to RNA")
	seqCmd.Flags().BoolP("rna2dna", "", false, "RNA to DNA")
	seqCmd.Flags().BoolP("color", "k", false, "colorize sequences - to be piped into \"less -R\"")
	seqCmd.Flags().BoolP("validate-seq", "v", false, "validate bases according to the alphabet")
	seqCmd.Flags().IntP("min-len", "m", -1, "only print sequences longer than or equal to the minimum length (-1 for no limit)")
	seqCmd.Flags().IntP("max-len", "M", -1, "only print sequences shorter than or equal to the maximum length (-1 for no limit)")
	seqCmd.Flags().IntP("qual-ascii-base", "b", 33, "ASCII BASE, 33 for Phred+33")
	seqCmd.Flags().Float64P("min-qual", "Q", -1, "only print sequences with average quality greater or equal than this limit (-1 for no limit)")
	seqCmd.Flags().Float64P("max-qual", "R", -1, "only print sequences with average quality less than this limit (-1 for no limit)")
}

var _mark_fasta = []byte{'>'}
var _mark_fastq = []byte{'@'}
var _mark_plus_newline = []byte{'+', '\n'}
var _mark_newline = []byte{'\n'}
