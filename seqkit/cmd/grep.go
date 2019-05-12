// Copyright © 2016 Wei Shen <shenwei356@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, target to the following conditions:
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
	"strconv"
	"strings"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/breader"
	"github.com/shenwei356/bwt/fmi"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// grepCmd represents the grep command
var grepCmd = &cobra.Command{
	Use:   "grep",
	Short: "search sequences by ID/name/sequence/sequence motifs, mismatch allowed",
	Long: fmt.Sprintf(`search sequences by ID/name/sequence/sequence motifs, mismatch allowed

Attentions:
    1. Unlike POSIX/GNU grep, we compare the pattern to the whole target
       (ID/full header) by default. Please switch "-r/--use-regexp" on
       for partly matching.
    2. While when searching by sequences, it's partly matching. And mismatch
       is allowed using flag "-m/--max-mismatch".
    3. The order of sequences in result is consistent with that in original
       file, not the order of the query patterns.

You can specify the sequence region for searching with flag -R (--region).
The definition of region is 1-based and with some custom design.

Examples:
%s
`, regionExample),
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		outFile := config.OutFile
		lineWidth := config.LineWidth
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		pattern := getFlagStringSlice(cmd, "pattern")
		patternFile := getFlagString(cmd, "pattern-file")
		useRegexp := getFlagBool(cmd, "use-regexp")
		deleteMatched := getFlagBool(cmd, "delete-matched")
		invertMatch := getFlagBool(cmd, "invert-match")
		bySeq := getFlagBool(cmd, "by-seq")
		mismatches := getFlagNonNegativeInt(cmd, "max-mismatch")
		byName := getFlagBool(cmd, "by-name")
		ignoreCase := getFlagBool(cmd, "ignore-case")
		degenerate := getFlagBool(cmd, "degenerate")
		region := getFlagString(cmd, "region")

		if len(pattern) == 0 && patternFile == "" {
			checkError(fmt.Errorf("one of flags -p (--pattern) and -f (--pattern-file) needed"))
		}

		if degenerate && !bySeq {
			log.Infof("when flag -d (--degenerate) given, flag -s (--by-seq) is automatically on")
			bySeq = true
		}

		var sfmi *fmi.FMIndex
		if mismatches > 0 {
			if useRegexp || degenerate {
				checkError(fmt.Errorf("flag -r (--use-regexp) or -d (--degenerate) not allowed when giving flag -m (--max-mismatch)"))
			}
			if !bySeq {
				log.Infof("when value of flag -m (--max-mismatch) > 0, flag -s (--by-seq) is automatically on")
				bySeq = true
			}
			sfmi = fmi.NewFMIndex()
			if mismatches > 4 {
				log.Warningf("large value flag -m/--max-mismatch will slow down the search")
			}
		}

		if useRegexp && degenerate {
			checkError(fmt.Errorf("could not give both flags -d (--degenerate) and -r (--use-regexp)"))
		}

		var start, end int
		var err error
		var limitRegion bool
		if region != "" {
			limitRegion = true
			if !bySeq {
				log.Infof("when flag -R (--region) given, flag -s (--by-seq) is automatically on")
				bySeq = true
			}
			if !reRegion.MatchString(region) {
				checkError(fmt.Errorf(`invalid region: %s. type "seqkit grep -h" for more examples`, region))
			}
			r := strings.Split(region, ":")
			start, err = strconv.Atoi(r[0])
			checkError(err)
			end, err = strconv.Atoi(r[1])
			checkError(err)
			if start == 0 || end == 0 {
				checkError(fmt.Errorf("both start and end should not be 0"))
			}
			if start < 0 && end > 0 {
				checkError(fmt.Errorf("when start < 0, end should not > 0"))
			}
		}

		files := getFileList(args)

		// prepare pattern
		patterns := make(map[string]*regexp.Regexp)
		var pattern2seq *seq.Seq
		var pbyte []byte
		if patternFile != "" {
			var reader *breader.BufferedReader
			reader, err = breader.NewDefaultBufferedReader(patternFile)
			checkError(err)
			for chunk := range reader.Ch {
				checkError(chunk.Err)
				for _, data := range chunk.Data {
					p := data.(string)
					if p == "" {
						continue
					}
					if degenerate || useRegexp {
						if degenerate {
							pattern2seq, err = seq.NewSeq(alphabet, []byte(p))
							if err != nil {
								checkError(fmt.Errorf("it seems that flag -d is given, but you provide regular expression instead of available %s sequence", alphabet.String()))
							}
							p = pattern2seq.Degenerate2Regexp()
						}
						if ignoreCase {
							p = "(?i)" + p
						}
						r, err := regexp.Compile(p)
						checkError(err)
						patterns[p] = r
					} else if bySeq {
						pbyte = []byte(p)
						if mismatches > 0 && mismatches > len(p) {
							checkError(fmt.Errorf("mismatch should be <= length of sequence: %s", p))
						}
						if seq.DNAredundant.IsValid(pbyte) == nil ||
							seq.RNAredundant.IsValid(pbyte) == nil ||
							seq.Protein.IsValid(pbyte) == nil { // legal sequence
							if ignoreCase {
								patterns[strings.ToLower(p)] = nil
							} else {
								patterns[p] = nil
							}
						} else {
							checkError(fmt.Errorf("illegal DNA/RNA/Protein sequence: %s", p))
						}
					} else {
						if ignoreCase {
							patterns[strings.ToLower(p)] = nil
						} else {
							patterns[p] = nil
						}
					}
				}
			}
		} else {
			for _, p := range pattern {
				if degenerate || useRegexp {
					if degenerate {
						pattern2seq, err = seq.NewSeq(alphabet, []byte(p))
						if err != nil {
							checkError(fmt.Errorf("it seems that flag -d is given, but you provide regular expression instead of available %s sequence", alphabet.String()))
						}
						p = pattern2seq.Degenerate2Regexp()
					}
					if ignoreCase {
						p = "(?i)" + p
					}
					r, err := regexp.Compile(p)
					checkError(err)
					patterns[p] = r
				} else if bySeq {
					pbyte = []byte(p)
					if mismatches > 0 && mismatches > len(p) {
						checkError(fmt.Errorf("mismatch should be <= length of sequence: %s", p))
					}
					if seq.DNAredundant.IsValid(pbyte) == nil ||
						seq.RNAredundant.IsValid(pbyte) == nil ||
						seq.Protein.IsValid(pbyte) == nil { // legal sequence
						if ignoreCase {
							patterns[strings.ToLower(p)] = nil
						} else {
							patterns[p] = nil
						}
					} else {
						checkError(fmt.Errorf("illegal DNA/RNA/Protein sequence: %s", p))
					}
				} else {
					if ignoreCase {
						patterns[strings.ToLower(p)] = nil
					} else {
						patterns[p] = nil
					}
				}
			}
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var target []byte
		var ok, hit bool
		var record *fastx.Record
		var fastxReader *fastx.Reader
		var k string
		var locs []int
		var re *regexp.Regexp
		var p string
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
				if fastxReader.IsFastq {
					config.LineWidth = 0
					fastx.ForcelyOutputFastq = true
				}

				if byName {
					target = record.Name
				} else if bySeq {
					if limitRegion {
						target = record.Seq.SubSeq(start, end).Seq
					} else {
						target = record.Seq.Seq
					}
				} else {
					target = record.ID
				}

				hit = false

				if degenerate || useRegexp {
					for p, re = range patterns {
						if re.Match(target) {
							hit = true
							if deleteMatched {
								delete(patterns, p)
							}
							break
						}
					}
				} else if bySeq {
					if ignoreCase {
						target = bytes.ToLower(target)
					}
					if mismatches == 0 {
						for k = range patterns {
							if bytes.Contains(target, []byte(k)) {
								hit = true
								break
							}
						}
					} else {
						_, err = sfmi.TransformForLocate(target)
						if err != nil {
							checkError(fmt.Errorf("fail to build FMIndex for sequence: %s", record.Name))
						}
						for k = range patterns {
							locs, err = sfmi.Locate([]byte(k), mismatches)
							if err != nil {
								checkError(fmt.Errorf("fail to search pattern '%s' on seq '%s': %s", k, record.Name, err))
							}
							if len(locs) > 0 {
								hit = true
								break
							}
						}
					}
				} else {
					k = string(target)
					if ignoreCase {
						k = strings.ToLower(k)
					}
					if _, ok = patterns[k]; ok {
						hit = true
					}
				}

				if invertMatch {
					if hit {
						continue
					}
				} else {
					if !hit {
						continue
					}
				}

				record.FormatToWriter(outfh, config.LineWidth)
			}

			config.LineWidth = lineWidth
		}
	},
}

func init() {
	RootCmd.AddCommand(grepCmd)

	grepCmd.Flags().StringSliceP("pattern", "p", []string{""}, `search pattern (multiple values supported. Attention: use double quotation marks for patterns containing comma, e.g., -p '"A{2,}"'))`)
	grepCmd.Flags().StringP("pattern-file", "f", "", "pattern file (one record per line)")
	grepCmd.Flags().BoolP("use-regexp", "r", false, "patterns are regular expression")
	grepCmd.Flags().BoolP("delete-matched", "", false, "delete matched patterns, brings speed improvement when using regular expressions")
	grepCmd.Flags().BoolP("invert-match", "v", false, "invert the sense of matching, to select non-matching records")
	grepCmd.Flags().BoolP("by-name", "n", false, "match by full name instead of just id")
	grepCmd.Flags().BoolP("by-seq", "s", false, "search subseq on seq, mismach allowed using flag -m/--max-mismatch")
	grepCmd.Flags().IntP("max-mismatch", "m", 0, "max mismatch when matching by seq (experimental, costs too much RAM for large genome, 8G for 50Kb sequence)")
	grepCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
	grepCmd.Flags().BoolP("degenerate", "d", false, "pattern/motif contains degenerate base")
	grepCmd.Flags().StringP("region", "R", "", "specify sequence region for searching. "+
		"e.g 1:12 for first 12 bases, -12:-1 for last 12 bases")

}
