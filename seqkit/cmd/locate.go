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
	"regexp"
	"runtime"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/bwt"
	"github.com/shenwei356/bwt/fmi"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// locateCmd represents the locate command
var locateCmd = &cobra.Command{
	Use:   "locate",
	Short: "locate subsequences/motifs, mismatch allowed",
	Long: `locate subsequences/motifs, mismatch allowed

Motifs could be EITHER plain sequence containing "ACTGN" OR regular
expression like "A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)" for ORFs.
Degenerate bases like "RYMM.." are also supported by flag -d.

By default, motifs are treated as regular expression.
When flag -d given, regular expression may be wrong.
For example: "\w" will be wrongly converted to "\[AT]".

Mismatch is allowed using flag "-m/--max-mismatch",
but it's not fast enough for large genome like human genome.
Though, it's fast enough for microbial genomes.

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		outFile := config.OutFile
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = true
		seq.ValidateWholeSeq = false
		seq.ValidSeqLengthThreshold = getFlagValidateSeqLength(cmd, "validate-seq-length")
		seq.ValidSeqThreads = config.Threads
		seq.ComplementThreads = config.Threads
		quiet := config.Quiet
		runtime.GOMAXPROCS(config.Threads)

		bwt.CheckEndSymbol = false

		pattern := getFlagStringSlice(cmd, "pattern")
		patternFile := getFlagString(cmd, "pattern-file")
		degenerate := getFlagBool(cmd, "degenerate")
		ignoreCase := getFlagBool(cmd, "ignore-case")
		onlyPositiveStrand := getFlagBool(cmd, "only-positive-strand")
		nonGreedy := getFlagBool(cmd, "non-greedy")
		outFmtGTF := getFlagBool(cmd, "gtf")
		outFmtBED := getFlagBool(cmd, "bed")
		mismatches := getFlagNonNegativeInt(cmd, "max-mismatch")
		hideMatched := getFlagBool(cmd, "hide-matched")

		if len(pattern) == 0 && patternFile == "" {
			checkError(fmt.Errorf("one of flags -p (--pattern) and -f (--pattern-file) needed"))
		}

		var sfmi *fmi.FMIndex
		if mismatches > 0 {
			if degenerate {
				checkError(fmt.Errorf("flag -d (--degenerate) not allowed when giving flag -m (--max-mismatch)"))
			}
			if nonGreedy && !quiet {
				log.Infof("flag -G (--non-greedy) ignored when giving flag -m (--max-mismatch)")
			}
			sfmi = fmi.NewFMIndex()
			if mismatches > 4 {
				log.Warningf("large value flag -m/--max-mismatch will slow down the search")
			}
		}

		files := getFileList(args, true)

		// prepare pattern
		regexps := make(map[string]*regexp.Regexp)
		patterns := make(map[string][]byte)
		var s string
		if patternFile != "" {
			records, err := fastx.GetSeqsMap(patternFile, seq.Unlimit, config.Threads, 10, "")
			checkError(err)
			for name, record := range records {
				patterns[name] = record.Seq.Seq
				if !quiet && bytes.IndexAny(record.Seq.Seq, "\t ") >= 0 {
					log.Warningf("space found in sequence: %s", name)
				}

				if degenerate {
					s = record.Seq.Degenerate2Regexp()
				} else {
					s = string(record.Seq.Seq)
				}

				if mismatches > 0 {
					if seq.DNAredundant.IsValid(record.Seq.Seq) == nil ||
						seq.RNAredundant.IsValid(record.Seq.Seq) == nil ||
						seq.Protein.IsValid(record.Seq.Seq) == nil { // legal sequence
					} else {
						checkError(fmt.Errorf("illegal DNA/RNA/Protein sequence: %s", record.Name))
					}
				} else {
					if ignoreCase {
						s = "(?i)" + s
					}
					re, err := regexp.Compile(s)
					checkError(err)
					regexps[name] = re
				}
			}
		} else {
			for _, p := range pattern {
				patterns[p] = []byte(p)

				if !quiet && bytes.IndexAny(patterns[p], " \t") >= 0 {
					log.Warningf("space found in sequence: '%s'", p)
				}

				if degenerate {
					pattern2seq, err := seq.NewSeq(alphabet, []byte(p))
					if err != nil {
						checkError(fmt.Errorf("it seems that flag -d is given, but you provide regular expression instead of available %s sequence", alphabet.String()))
					}
					s = pattern2seq.Degenerate2Regexp()
				} else {
					s = p
				}

				if mismatches > 0 {
					if mismatches > len(patterns[p]) {
						checkError(fmt.Errorf("mismatch should be <= length of sequence: %s", p))
					}
					if seq.DNAredundant.IsValid(patterns[p]) == nil ||
						seq.RNAredundant.IsValid(patterns[p]) == nil ||
						seq.Protein.IsValid(patterns[p]) == nil { // legal sequence
					} else {
						checkError(fmt.Errorf("illegal DNA/RNA/Protein sequence: %s", p))
					}
				} else {
					if ignoreCase {
						s = "(?i)" + s
					}
					re, err := regexp.Compile(s)
					checkError(err)
					regexps[p] = re
				}
			}
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		if !(outFmtGTF || outFmtBED) {
			if hideMatched {
				outfh.WriteString("seqID\tpatternName\tpattern\tstrand\tstart\tend\n")
			} else {
				outfh.WriteString("seqID\tpatternName\tpattern\tstrand\tstart\tend\tmatched\n")
			}
		}
		var seqRP *seq.Seq
		var offset, l int
		var loc []int
		var locs, locsNeg [][2]int
		var i, begin, end int
		var flag bool
		var record *fastx.Record
		var fastxReader *fastx.Reader
		var pSeq []byte
		var pName string
		for _, file := range files {
			fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)
			for {
				record, err = fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}

				if mismatches > 0 && ignoreCase {
					record.Seq.Seq = bytes.ToLower(record.Seq.Seq)
				}

				l = len(record.Seq.Seq)
				if !onlyPositiveStrand {
					seqRP = record.Seq.RevCom()
				}

				if mismatches > 0 {
					_, err = sfmi.Transform(record.Seq.Seq)
					if err != nil {
						checkError(fmt.Errorf("fail to build FMIndex for sequence: %s", record.Name))
					}

					for pName, pSeq = range patterns {
						loc, err = sfmi.Locate(pSeq, mismatches)
						if err != nil {
							checkError(fmt.Errorf("fail to search pattern '%s' on seq '%s': %s", pName, record.Name, err))
						}
						for _, i = range loc {
							begin = i + 1
							end = i + len(pSeq)
							if i+len(pSeq) > len(record.Seq.Seq) {
								continue
							}
							if outFmtGTF {
								outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%s\t%s\tgene_id \"%s\"; \n",
									record.ID,
									"SeqKit",
									"location",
									begin,
									end,
									0,
									"+",
									".",
									pName))
							} else if outFmtBED {
								outfh.WriteString(fmt.Sprintf("%s\t%d\t%d\t%s\t%d\t%s\n",
									record.ID,
									begin-1,
									end,
									pName,
									0,
									"+"))
							} else {
								if hideMatched {
									outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\n",
										record.ID,
										pName,
										patterns[pName],
										"+",
										begin,
										end))
								} else {
									outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
										record.ID,
										pName,
										patterns[pName],
										"+",
										begin,
										end,
										record.Seq.Seq[i:i+len(pSeq)]))
								}
							}
						}
					}

					if onlyPositiveStrand {
						continue
					}

					_, err = sfmi.Transform(seqRP.Seq)
					if err != nil {
						checkError(fmt.Errorf("fail to build FMIndex for reverse complement sequence: %s", record.Name))
					}
					for pName, pSeq = range patterns {
						loc, err = sfmi.Locate(pSeq, mismatches)
						if err != nil {
							checkError(fmt.Errorf("fail to search pattern '%s' on seq '%s': %s", pName, record.Name, err))
						}
						for _, i = range loc {
							begin = l - i - len(pSeq) + 1
							end = l - i
							if i+len(pSeq) > len(record.Seq.Seq) {
								continue
							}
							if outFmtGTF {
								outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%s\t%s\tgene_id \"%s\"; \n",
									record.ID,
									"SeqKit",
									"location",
									begin,
									end,
									0,
									"-",
									".",
									pName))
							} else if outFmtBED {
								outfh.WriteString(fmt.Sprintf("%s\t%d\t%d\t%s\t%d\t%s\n",
									record.ID,
									begin-1,
									end,
									pName,
									0,
									"-"))
							} else {
								if hideMatched {
									outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\n",
										record.ID,
										pName,
										patterns[pName],
										"-",
										begin,
										end))
								} else {
									outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
										record.ID,
										pName,
										patterns[pName],
										"-",
										begin,
										end,
										seqRP.Seq[i:i+len(pSeq)]))
								}
							}
						}
					}

					continue
				}

				for pName, re := range regexps {
					locs = make([][2]int, 0, 1000)

					offset = 0
					for {
						loc = re.FindSubmatchIndex(record.Seq.Seq[offset:])
						if loc == nil {
							break
						}
						begin = offset + loc[0] + 1
						end = offset + loc[1]

						flag = true
						for i = len(locs) - 1; i >= 0; i-- {
							if locs[i][0] <= begin && locs[i][1] >= end {
								flag = false
								break
							}
						}

						if flag {
							if outFmtGTF {
								outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%s\t%s\tgene_id \"%s\"; \n",
									record.ID,
									"SeqKit",
									"location",
									begin,
									end,
									0,
									"+",
									".",
									pName))
							} else if outFmtBED {
								outfh.WriteString(fmt.Sprintf("%s\t%d\t%d\t%s\t%d\t%s\n",
									record.ID,
									begin-1,
									end,
									pName,
									0,
									"+"))
							} else {
								if hideMatched {
									outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\n",
										record.ID,
										pName,
										patterns[pName],
										"+",
										begin,
										end))
								} else {
									outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
										record.ID,
										pName,
										patterns[pName],
										"+",
										begin,
										end,
										record.Seq.Seq[offset+loc[0]:offset+loc[1]]))
								}
							}
							locs = append(locs, [2]int{begin, end})
						}

						if nonGreedy {
							offset = offset + loc[1] + 1
						} else {
							offset = offset + loc[0] + 1
						}
						if offset >= len(record.Seq.Seq) {
							break
						}
					}

					if onlyPositiveStrand {
						continue
					}

					locsNeg = make([][2]int, 0, 1000)

					offset = 0

					for {
						loc = re.FindSubmatchIndex(seqRP.Seq[offset:])
						if loc == nil {
							break
						}
						begin = l - offset - loc[1] + 1
						end = l - offset - loc[0]

						flag = true
						for i = len(locsNeg) - 1; i >= 0; i-- {
							if locsNeg[i][0] <= begin && locsNeg[i][1] >= end {
								flag = false
								break
							}
						}

						if flag {
							if outFmtGTF {
								outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%s\t%s\tgene_id \"%s\"; \n",
									record.ID,
									"SeqKit",
									"location",
									begin,
									end,
									0,
									"-",
									".",
									pName))
							} else if outFmtBED {
								outfh.WriteString(fmt.Sprintf("%s\t%d\t%d\t%s\t%d\t%s\n",
									record.ID,
									begin-1,
									end,
									pName,
									0,
									"-"))
							} else {
								if hideMatched {
									outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\n",
										record.ID,
										pName,
										patterns[pName],
										"-",
										begin,
										end))
								} else {
									outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
										record.ID,
										pName,
										patterns[pName],
										"-",
										begin,
										end,
										record.Seq.SubSeq(l-offset-loc[1]+1, l-offset-loc[0]).RevCom().Seq))
								}
							}
							locsNeg = append(locsNeg, [2]int{begin, end})
						}

						if nonGreedy {
							offset = offset + loc[1] + 1
						} else {
							offset = offset + loc[0] + 1
						}
						if offset >= len(record.Seq.Seq) {
							break
						}
					}
				}
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(locateCmd)

	locateCmd.Flags().StringSliceP("pattern", "p", []string{""}, `pattern/motif (multiple values supported. Attention: use double quotation marks for patterns containing comma, e.g., -p '"A{2,}"')`)
	locateCmd.Flags().StringP("pattern-file", "f", "", "pattern/motif file (FASTA format)")
	locateCmd.Flags().BoolP("degenerate", "d", false, "pattern/motif contains degenerate base")
	locateCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
	locateCmd.Flags().BoolP("only-positive-strand", "P", false, "only search on positive strand")
	locateCmd.Flags().IntP("validate-seq-length", "V", 10000, "length of sequence to validate (0 for whole seq)")
	locateCmd.Flags().BoolP("non-greedy", "G", false, "non-greedy mode, faster but may miss motifs overlapping with others")
	locateCmd.Flags().BoolP("gtf", "", false, "output in GTF format")
	locateCmd.Flags().BoolP("bed", "", false, "output in BED6 format")
	locateCmd.Flags().IntP("max-mismatch", "m", 0, "max mismatch when matching by seq. For large genomes like human genome, using mapping/alignment tools would be faster")
	locateCmd.Flags().BoolP("hide-matched", "M", false, "do not show matched sequences")
}
