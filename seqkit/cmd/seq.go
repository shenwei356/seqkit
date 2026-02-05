// Copyright © 2016-2026 Wei Shen <shenwei356@gmail.com>
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
	"regexp"
	"runtime"
	"strings"

	// "runtime/debug"

	"github.com/cespare/xxhash/v2"
	"github.com/dsnet/compress/bzip2"
	gzip "github.com/klauspost/pgzip"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/breader"
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

Filtering records to edit:
  You can use flags similar to those in "seqkit grep" to choose partly records to edit.

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

		// -----------------------------------------------------------------------------
		// filter for records to edit

		fpattern := getFlagStringSlice(cmd, "f-pattern")
		fpatternFile := getFlagString(cmd, "f-pattern-file")
		fuseRegexp := getFlagBool(cmd, "f-use-regexp")
		finvertMatch := getFlagBool(cmd, "f-invert-match")
		fbySeq := getFlagBool(cmd, "f-by-seq")
		fbyName := getFlagBool(cmd, "f-by-name")
		fignoreCase := getFlagBool(cmd, "f-ignore-case")
		fonlyPositiveStrand := getFlagBool(cmd, "f-only-positive-strand")

		// check pattern with unquoted comma
		hasUnquotedComma := false
		for _, _pattern := range fpattern {
			if reUnquotedComma.MatchString(_pattern) {
				hasUnquotedComma = true
				break
			}
		}
		if hasUnquotedComma {
			if outFile == "-" {
				defer log.Warningf(helpUnquotedComma)
			} else {
				log.Warningf(helpUnquotedComma)
			}
		}

		// prepare pattern
		usingDefaultIDRegexp := config.IDRegexp == fastx.DefaultIDRegexp

		patternsR := make(map[uint64]*regexp.Regexp, 1<<10)
		patternsN := make(map[uint64]interface{}, 1<<20)
		// patternsS := make(map[string]interface{}, 1<<10)
		patternsS := make([][]byte, 0, 16)

		var pbyte []byte
		if fpatternFile != "" {
			var reader *breader.BufferedReader
			var err error
			reader, err = breader.NewDefaultBufferedReader(fpatternFile)
			checkError(err)
			for chunk := range reader.Ch {
				checkError(chunk.Err)
				for _, data := range chunk.Data {
					p := data.(string)
					if p == "" {
						continue
					}

					if !quiet {
						if p[0] == '>' {
							log.Warningf(`symbol ">" detected, it should not be a part of the sequence ID/name: %s`, p)
						} else if p[0] == '@' {
							log.Warningf(`symbol "@" detected, it should not be a part of the sequence ID/name. %s`, p)
						} else if !fbyName && usingDefaultIDRegexp && strings.ContainsAny(p, "\t ") {
							log.Warningf("space found in pattern, you may need use --f-by-name: %s", p)
						}
					}

					if fuseRegexp {
						if fignoreCase {
							p = "(?i)" + p
						}
						r, err := regexp.Compile(p)
						checkError(err)
						patternsR[xxhash.Sum64String(p)] = r
					} else if fbySeq {
						pbyte = []byte(p)
						if seq.DNAredundant.IsValid(pbyte) == nil ||
							seq.RNAredundant.IsValid(pbyte) == nil ||
							seq.Protein.IsValid(pbyte) == nil { // legal sequence
							if fignoreCase {
								// patternsS[strings.ToLower(p)] = struct{}{}
								patternsS = append(patternsS, bytes.ToLower(pbyte))
							} else {
								// patternsS[p] = struct{}{}
								patternsS = append(patternsS, pbyte)
							}
						} else {
							checkError(fmt.Errorf("illegal DNA/RNA/Protein sequence: %s", p))
						}
					} else {
						if fignoreCase {
							patternsN[xxhash.Sum64String(strings.ToLower(p))] = struct{}{}
						} else {
							patternsN[xxhash.Sum64String(p)] = struct{}{}
						}
					}
				}
			}
			if !quiet {
				if len(patternsR)+len(patternsN)+len(patternsS) == 0 {
					log.Warningf("%d patterns loaded from file", 0)
				} else {
					log.Infof("%d patterns loaded from file", len(patternsR)+len(patternsN)+len(patternsS))
				}
			}
		} else if len(fpattern) > 0 {
			for _, p := range fpattern {
				if !quiet {
					if p[0] == '>' {
						log.Warningf(`symbol ">" detected, it should not be a part of the sequence ID/name: %s`, p)
					} else if p[0] == '@' {
						log.Warningf(`symbol "@" detected, it should not be a part of the sequence ID/name. %s`, p)
					} else if !fbyName && usingDefaultIDRegexp && strings.ContainsAny(p, "\t ") {
						log.Warningf("space found in pattern, you may need use --f-by-name: %s", p)
					}
				}

				if fuseRegexp {
					if fignoreCase {
						p = "(?i)" + p
					}
					r, err := regexp.Compile(p)
					checkError(err)
					patternsR[xxhash.Sum64String(p)] = r
				} else if fbySeq {
					pbyte = []byte(p)
					if seq.DNAredundant.IsValid(pbyte) == nil ||
						seq.RNAredundant.IsValid(pbyte) == nil ||
						seq.Protein.IsValid(pbyte) == nil { // legal sequence
						if fignoreCase {
							// patternsS[strings.ToLower(p)] = struct{}{}
							patternsS = append(patternsS, bytes.ToLower(pbyte))
						} else {
							// patternsS[p] = struct{}{}
							patternsS = append(patternsS, pbyte)
						}
					} else {
						checkError(fmt.Errorf("illegal DNA/RNA/Protein sequence: %s", p))
					}
				} else {
					if fignoreCase {
						patternsN[xxhash.Sum64String(strings.ToLower(p))] = struct{}{}
					} else {
						patternsN[xxhash.Sum64String(p)] = struct{}{}
					}
				}
			}
		}

		useFilter := len(fpattern) > 0 || fpatternFile != ""

		// -----------------------------------------------------------------------------

		// ---------------------------------------------------------------------------------------------

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", !config.SkipFileCheck)
		if !config.SkipFileCheck {
			for _, file := range files {
				checkIfFilesAreTheSame(file, outFile, "input", "output")
			}
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

		// for filtering
		var target []byte
		var matched bool
		var k2 []byte
		var re *regexp.Regexp
		var h uint64
		var ok bool
		matched = true // does not filter

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

				// -----------------------------------------------------------------------------
				// filter

				if useFilter {
					if fbyName {
						target = record.Name
					} else if fbySeq {
						target = record.Seq.Seq
					} else {
						target = record.ID
					}

					matched = false

					if fuseRegexp {
						for h, re = range patternsR {
							if re.Match(target) {
								matched = true
								break
							}
						}
					} else if fbySeq {
						if fignoreCase {
							target = bytes.ToLower(target)
						}
						for _, k2 = range patternsS {
							if bytes.Contains(target, k2) {
								matched = true
								break
							}
						}
						// search the reverse complement seq
						if !matched && !fonlyPositiveStrand {
							target = record.Seq.RevCom().Seq
							if fignoreCase {
								target = bytes.ToLower(target)
							}
							for _, k2 = range patternsS {
								if bytes.Contains(target, k2) {
									matched = true
									break
								}
							}
						}
					} else {
						h = xxhash.Sum64(target)
						if fignoreCase {
							h = xxhash.Sum64(bytes.ToLower(target))
						}
						if _, ok = patternsN[h]; ok {
							matched = true
						}
					}

					if finvertMatch {
						matched = !matched
					}
				}

				// -----------------------------------------------------------------------------

				if matched && removeGaps {
					record.Seq.RemoveGapsInplace(gapLetters)
				}

				if matched && filterMinLen && len(record.Seq.Seq) < minLen {
					continue
				}

				if matched && filterMaxLen && len(record.Seq.Seq) > maxLen {
					continue
				}

				if matched && (filterMinQual || filterMaxQual) {
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
				if matched && reverse {
					sequence = sequence.ReverseInplace()
				}
				if matched && complement {
					if !config.Quiet && record.Seq.Alphabet == seq.Protein || record.Seq.Alphabet == seq.Unlimit {
						log.Warning("complement does no take effect on protein/unlimit sequence")
					}
					sequence = sequence.ComplementInplace()
				}

				if printSeq {
					if matched && dna2rna {
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
					if matched && rna2dna {
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
					if matched {
						if lowerCase {
							sequence.Seq = bytes.ToLower(sequence.Seq)
						} else if upperCase {
							sequence.Seq = bytes.ToUpper(sequence.Seq)
						}
					}

					if isFastq {
						if matched && color {
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

						if matched && color {
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

					if matched && color {
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
	seqCmd.Flags().BoolP("remove-gaps", "g", false, `remove gaps letters seft by -G/--gap-letters, e.g., spaces, tabs, and dashes (gaps "-" in aligned sequences)`)
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

	// flags to choose which sequence to edit and output
	seqCmd.Flags().StringSliceP("f-pattern", "", []string{""}, `[target filter] search pattern (multiple values supported. Attention: use double quotation marks for patterns containing comma, e.g., -p '"A{2,}"')`)
	seqCmd.Flags().StringP("f-pattern-file", "", "", "[target filter] pattern file (one record per line)")
	seqCmd.Flags().BoolP("f-use-regexp", "", false, "[target filter] patterns are regular expression")
	seqCmd.Flags().BoolP("f-invert-match", "", false, "[target filter] invert the sense of matching, to select non-matching records")
	seqCmd.Flags().BoolP("f-by-name", "", false, "[target filter] match by full name instead of just ID")
	seqCmd.Flags().BoolP("f-by-seq", "", false, "[target filter] search subseq on seq, both positive and negative strand are searched")
	seqCmd.Flags().BoolP("f-ignore-case", "", false, "[target filter] ignore case")
	seqCmd.Flags().BoolP("f-only-positive-strand", "", false, "[target filter] only search on positive strand")
}

var _mark_fasta = []byte{'>'}
var _mark_fastq = []byte{'@'}
var _mark_plus_newline = []byte{'+', '\n'}
var _mark_newline = []byte{'\n'}
