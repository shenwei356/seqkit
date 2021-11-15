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
	"sync"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/bwt"
	"github.com/shenwei356/bwt/fmi"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
	"github.com/twotwotwo/sorts/sortutil"
)

// locateCmd represents the locate command
var locateCmd = &cobra.Command{
	Use:   "locate",
	Short: "locate subsequences/motifs, mismatch allowed",
	Long: `locate subsequences/motifs, mismatch allowed

Attentions:

  1. Motifs could be EITHER plain sequence containing "ACTGN" OR regular
     expression like "A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)" for ORFs.     
  2. Degenerate bases/residues like "RYMM.." are also supported by flag -d.
     But do not use degenerate bases/residues in regular expression, you need
     convert them to regular expression, e.g., change "N" or "X"  to ".".
  3. When providing search patterns (motifs) via flag '-p',
     please use double quotation marks for patterns containing comma, 
     e.g., -p '"A{2,}"' or -p "\"A{2,}\"". Because the command line argument
     parser accepts comma-separated-values (CSV) for multiple values (motifs).
     Patterns in file do not follow this rule.     
  4. Mismatch is allowed using flag "-m/--max-mismatch",
     you can increase the value of "-j/--threads" to accelerate processing.
  5. When using flag --circular, end position of matched subsequence that 
     crossing genome sequence end would be greater than sequence length.

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

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		pattern := getFlagStringSlice(cmd, "pattern")
		patternFile := getFlagString(cmd, "pattern-file")
		degenerate := getFlagBool(cmd, "degenerate")
		useRegexp := getFlagBool(cmd, "use-regexp")
		useFMI := getFlagBool(cmd, "use-fmi")
		ignoreCase := getFlagBool(cmd, "ignore-case")
		onlyPositiveStrand := getFlagBool(cmd, "only-positive-strand")
		nonGreedy := getFlagBool(cmd, "non-greedy")
		outFmtGTF := getFlagBool(cmd, "gtf")
		outFmtBED := getFlagBool(cmd, "bed")
		mismatches := getFlagNonNegativeInt(cmd, "max-mismatch")
		hideMatched := getFlagBool(cmd, "hide-matched")
		circular := getFlagBool(cmd, "circular")

		immediateOutput := getFlagBool(cmd, "immediate-output")

		if config.Alphabet == seq.Protein {
			onlyPositiveStrand = true
		}

		if len(pattern) == 0 && patternFile == "" {
			checkError(fmt.Errorf("one of flags -p (--pattern) and -f (--pattern-file) needed"))
		}

		// check pattern with unquoted comma
		hasUnquotedComma := false
		for _, _pattern := range pattern {
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

		if mismatches > 0 {
			if degenerate {
				checkError(fmt.Errorf("flag -d (--degenerate) not allowed when giving flag -m (--max-mismatch)"))
			}
			if useRegexp {
				checkError(fmt.Errorf("flag -r (--use-regexp) not allowed when giving flag -m (--use-regexp)"))
			}
			if nonGreedy && !quiet {
				log.Infof("flag -G (--non-greedy) ignored when giving flag -m (--max-mismatch)")
			}

		}
		if useFMI {
			if degenerate {
				checkError(fmt.Errorf("flag -d (--degenerate) ignored when giving flag -F (--use-fmi)"))
			}
			if useRegexp {
				checkError(fmt.Errorf("flag -r (--use-regexp) ignored when giving flag -F (--use-fmi)"))
			}
		}

		// prepare pattern
		regexps := make(map[string]*regexp.Regexp)
		patterns := make(map[string][]byte)
		var s string
		if patternFile != "" {
			records, err := fastx.GetSeqsMap(patternFile, seq.Unlimit, config.Threads, 10, "")
			checkError(err)
			if len(records) == 0 {
				checkError(fmt.Errorf("no FASTA sequences found in pattern file: %s", patternFile))
			}
			for name, record := range records {
				patterns[name] = record.Seq.Seq
				if !quiet && bytes.IndexAny(record.Seq.Seq, "\t ") >= 0 {
					log.Warningf("space found in sequence: %s", name)
				}

				if degenerate {
					s = record.Seq.Degenerate2Regexp()
				} else if useRegexp {
					s = string(record.Seq.Seq)
				} else {
					if ignoreCase {
						patterns[name] = bytes.ToLower(record.Seq.Seq)
					}
				}

				// check pattern
				if mismatches > 0 {
					if mismatches > len(record.Seq.Seq) {
						checkError(fmt.Errorf("mismatch should be <= length of sequence: %s", record.Seq.Seq))
					}
					if seq.DNAredundant.IsValid(record.Seq.Seq) == nil ||
						seq.RNAredundant.IsValid(record.Seq.Seq) == nil ||
						seq.Protein.IsValid(record.Seq.Seq) == nil { // legal sequence
					} else {
						checkError(fmt.Errorf("illegal DNA/RNA/Protein sequence: %s", record.Name))
					}
				} else {
					if degenerate || useRegexp {
						if ignoreCase {
							s = "(?i)" + s
						}
						re, err := regexp.Compile(s)
						checkError(err)
						regexps[name] = re
					} else if bytes.Index(record.Seq.Seq, []byte(".")) >= 0 ||
						!(seq.DNAredundant.IsValid(record.Seq.Seq) == nil ||
							seq.RNAredundant.IsValid(record.Seq.Seq) == nil ||
							seq.Protein.IsValid(record.Seq.Seq) == nil) {
						checkError(fmt.Errorf("illegal DNA/RNA/Protein sequence: %s, you may switch on -d/--degenerate or -r/--use-regexp", record.Name))
					}
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
				} else if useRegexp {
					s = p
				} else {
					if ignoreCase {
						patterns[p] = bytes.ToLower(patterns[p])
					}
				}

				// check pattern
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
					if degenerate || useRegexp {
						if ignoreCase {
							s = "(?i)" + s
						}
						re, err := regexp.Compile(s)
						checkError(err)
						regexps[p] = re
					} else if bytes.Index(patterns[p], []byte(".")) >= 0 ||
						!(seq.DNAredundant.IsValid(patterns[p]) == nil ||
							seq.RNAredundant.IsValid(patterns[p]) == nil ||
							seq.Protein.IsValid(patterns[p]) == nil) {
						checkError(fmt.Errorf("illegal DNA/RNA/Protein sequence: %s, you may switch on -d/--degenerate or -r/--use-regexp", p))
					}
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

		// -------------------------------------------------------------------
		// only for m > 0, where FMI is slow

		var record *fastx.Record
		var fastxReader *fastx.Reader
		_onlyPositiveStrand := onlyPositiveStrand

		if mismatches > 0 || useFMI {
			type Arecord struct {
				id     uint64
				ok     bool
				record []string
			}

			var wg sync.WaitGroup
			ch := make(chan *Arecord, config.Threads)
			tokens := make(chan int, config.Threads)

			done := make(chan int)
			go func() {
				m := make(map[uint64]*Arecord, config.Threads)
				var id, _id uint64
				var ok bool
				var _r *Arecord
				var row string

				id = 1
				for r := range ch {
					_id = r.id

					if _id == id { // right there
						if r.ok {
							for _, row = range r.record {
								outfh.WriteString(row)
							}

							if immediateOutput {
								outfh.Flush()
							}
						}
						id++
						continue
					}

					m[_id] = r // save for later check

					if _r, ok = m[id]; ok { // check buffered
						if _r.ok {
							for _, row = range _r.record {
								outfh.WriteString(row)
							}

							if immediateOutput {
								outfh.Flush()
							}
						}
						delete(m, id)
						id++
					}
				}

				if len(m) > 0 {
					ids := make([]uint64, len(m))
					i := 0
					for _id = range m {
						ids[i] = _id
						i++
					}
					sortutil.Uint64s(ids)
					for _, _id = range ids {
						_r = m[_id]

						if _r.ok {
							for _, row = range _r.record {
								outfh.WriteString(row)
							}

							if immediateOutput {
								outfh.Flush()
							}
						}
					}
				}
				done <- 1
			}()

			var id uint64
			for _, file := range files {
				fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
				checkError(err)

				checkAlphabet := true
				for {
					record, err = fastxReader.Read()
					if err != nil {
						if err == io.EOF {
							break
						}
						checkError(err)
						break
					}

					if checkAlphabet {
						if fastxReader.Alphabet() == seq.Unlimit || fastxReader.Alphabet() == seq.Protein {
							_onlyPositiveStrand = true
						}
						checkAlphabet = false
					}

					tokens <- 1
					wg.Add(1)
					id++
					go func(record *fastx.Record, id uint64) {
						defer func() {
							wg.Done()
							<-tokens
						}()

						var seqRP *seq.Seq
						var l int
						var loc []int
						var i, begin, end int
						var pSeq []byte
						var pName string
						var sfmi *fmi.FMIndex
						sfmi = fmi.NewFMIndex()
						results := make([]string, 0, 2)

						if !(degenerate || useRegexp) && ignoreCase {
							record.Seq.Seq = bytes.ToLower(record.Seq.Seq)
						}

						l = len(record.Seq.Seq)

						if circular { // concat two copies of sequence
							record.Seq.Seq = append(record.Seq.Seq, record.Seq.Seq...)
						}

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
								if circular && i+1 > l { // 2nd clone of original part
									continue
								}

								begin = i + 1

								end = i + len(pSeq)
								if i+len(pSeq) > len(record.Seq.Seq) {
									continue
								}
								if outFmtGTF {
									results = append(results, fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%s\t%s\tgene_id \"%s\"; \n",
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
									results = append(results, fmt.Sprintf("%s\t%d\t%d\t%s\t%d\t%s\n",
										record.ID,
										begin-1,
										end,
										pName,
										0,
										"+"))
								} else {
									if hideMatched {
										results = append(results, fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\n",
											record.ID,
											pName,
											patterns[pName],
											"+",
											begin,
											end))
									} else {
										results = append(results, fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
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

						if _onlyPositiveStrand {
							ch <- &Arecord{record: results, id: id, ok: len(results) > 0}
							return
						}

						seqRP = record.Seq.RevCom()

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
								if circular && i+1 > l { // 2nd clone of original part
									continue
								}

								begin = l - i - len(pSeq) + 1
								end = l - i
								if i+len(pSeq) > len(record.Seq.Seq) {
									continue
								}
								if outFmtGTF {
									results = append(results, fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%s\t%s\tgene_id \"%s\"; \n",
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
									results = append(results, fmt.Sprintf("%s\t%d\t%d\t%s\t%d\t%s\n",
										record.ID,
										begin-1,
										end,
										pName,
										0,
										"-"))
								} else {
									if hideMatched {
										results = append(results, fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\n",
											record.ID,
											pName,
											patterns[pName],
											"-",
											begin,
											end))
									} else {
										results = append(results, fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
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

						ch <- &Arecord{record: results, id: id, ok: len(results) > 0}
					}(record.Clone(), id)
				}
			}

			wg.Wait()
			close(ch)
			<-done

			return
		}

		// -------------------------------------------------------------------

		var seqRP *seq.Seq
		var offset, l, lpatten int
		var loc []int
		var locs, locsNeg [][2]int
		var i, begin, end int
		var flag bool
		var pSeq, p []byte
		var pName string
		var re *regexp.Regexp
		var sfmi *fmi.FMIndex
		if mismatches > 0 || useFMI {
			sfmi = fmi.NewFMIndex()
		}

		for _, file := range files {
			fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)

			checkAlphabet := true
			for {
				record, err = fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}

				if checkAlphabet {
					if fastxReader.Alphabet() == seq.Unlimit || fastxReader.Alphabet() == seq.Protein {
						_onlyPositiveStrand = true
					}
					checkAlphabet = false
				}

				if !(degenerate || useRegexp) && ignoreCase {
					record.Seq.Seq = bytes.ToLower(record.Seq.Seq)
				}

				l = len(record.Seq.Seq)

				if circular { // concat two copies of sequence
					record.Seq.Seq = append(record.Seq.Seq, record.Seq.Seq...)
				}

				if mismatches > 0 || useFMI {
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
							if circular && i+1 > l { // 2nd clone of original part
								continue
							}

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

					if _onlyPositiveStrand {
						continue
					}

					seqRP = record.Seq.RevCom()

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
							if circular && i+1 > l { // 2nd clone of original part
								continue
							}

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

					if immediateOutput {
						outfh.Flush()
					}

					continue
				}

				for pName = range patterns {
					locs = make([][2]int, 0, 1000)

					offset = 0
					if !(useRegexp || degenerate) {
						p = patterns[pName]
						lpatten = len(p)
					}
					for {
						if useRegexp || degenerate {
							re = regexps[pName]
							loc = re.FindSubmatchIndex(record.Seq.Seq[offset:])
							if loc == nil {
								break
							}

						} else {
							i = bytes.Index(record.Seq.Seq[offset:], p)
							if i < 0 {
								break
							}
							loc = []int{i, i + lpatten}
						}
						begin = offset + loc[0] + 1

						if circular && begin > l { // 2nd clone of original part
							break
						}

						end = offset + loc[1]

						flag = true // check "duplicated" region
						if useRegexp || degenerate {
							for i = len(locs) - 1; i >= 0; i-- {
								if locs[i][0] <= begin && locs[i][1] >= end {
									flag = false
									break
								}
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
										record.Seq.Seq[begin-1:end]))
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

					seqRP = record.Seq.RevCom()

					locsNeg = make([][2]int, 0, 1000)

					offset = 0

					for {
						if useRegexp || degenerate {
							re = regexps[pName]
							loc = re.FindSubmatchIndex(seqRP.Seq[offset:])
							if loc == nil {
								break
							}
						} else {
							i = bytes.Index(seqRP.Seq[offset:], p)
							if i < 0 {
								break
							}
							loc = []int{i, i + lpatten}
						}

						if circular && offset+loc[0]+1 > l { // 2nd clone of original part
							break
						}

						begin = l - offset - loc[1] + 1
						end = l - offset - loc[0]
						if offset+loc[1] > l {
							begin += l
							end += l
						}

						flag = true
						if useRegexp || degenerate {
							for i = len(locsNeg) - 1; i >= 0; i-- {
								if locsNeg[i][0] <= begin && locsNeg[i][1] >= end {
									flag = false
									break
								}
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
										seqRP.Seq[offset+loc[0]:offset+loc[1]]))
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

				if immediateOutput {
					outfh.Flush()
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
	locateCmd.Flags().BoolP("use-regexp", "r", false, "patterns/motifs are regular expression")
	locateCmd.Flags().BoolP("use-fmi", "F", false, "use FM-index for much faster search of lots of sequence patterns")
	locateCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
	locateCmd.Flags().BoolP("only-positive-strand", "P", false, "only search on positive strand")
	locateCmd.Flags().IntP("validate-seq-length", "V", 10000, "length of sequence to validate (0 for whole seq)")
	locateCmd.Flags().BoolP("non-greedy", "G", false, "non-greedy mode, faster but may miss motifs overlapping with others")
	locateCmd.Flags().BoolP("gtf", "", false, "output in GTF format")
	locateCmd.Flags().BoolP("bed", "", false, "output in BED6 format")
	locateCmd.Flags().IntP("max-mismatch", "m", 0, "max mismatch when matching by seq. For large genomes like human genome, using mapping/alignment tools would be faster")
	locateCmd.Flags().BoolP("hide-matched", "M", false, "do not show matched sequences")
	locateCmd.Flags().BoolP("circular", "c", false, `circular genome. type "seqkit locate -h" for details`)
	locateCmd.Flags().BoolP("immediate-output", "I", false, "print output immediately, do not use write buffer")
}
