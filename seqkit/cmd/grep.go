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
	"io"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/breader"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// grepCmd represents the grep command
var grepCmd = &cobra.Command{
	Use:   "grep",
	Short: "search sequences by pattern(s) of name or sequence motifs",
	Long: fmt.Sprintf(`search sequences by pattern(s) of name or sequence motifs

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
		seq.AlphabetGuessSeqLenghtThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		pattern := getFlagStringSlice(cmd, "pattern")
		patternFile := getFlagString(cmd, "pattern-file")
		useRegexp := getFlagBool(cmd, "use-regexp")
		deleteMatched := getFlagBool(cmd, "delete-matched")
		invertMatch := getFlagBool(cmd, "invert-match")
		bySeq := getFlagBool(cmd, "by-seq")
		byName := getFlagBool(cmd, "by-name")
		ignoreCase := getFlagBool(cmd, "ignore-case")
		degenerate := getFlagBool(cmd, "degenerate")
		region := getFlagString(cmd, "region")

		if len(pattern) == 0 && patternFile == "" {
			checkError(fmt.Errorf("one of flags -p (--pattern) and -f (--pattern-file) needed"))
		}
		if useRegexp && degenerate {
			checkError(fmt.Errorf("could not give both flags -d (--degenerat) and -r (--use-regexp)"))
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
		if patternFile != "" {
			reader, err := breader.NewDefaultBufferedReader(patternFile)
			checkError(err)
			for chunk := range reader.Ch {
				checkError(chunk.Err)
				for _, data := range chunk.Data {
					p := data.(string)
					if degenerate || useRegexp {
						if degenerate {
							pattern2seq, err := seq.NewSeq(alphabet, []byte(p))
							if err != nil {
								checkError(fmt.Errorf("it seems that flag -d is given, "+
									"but you provide regular expression instead of available %s sequence", alphabet))
							}
							p = pattern2seq.Degenerate2Regexp()
						}
						if ignoreCase {
							p = "(?i)" + p
						}
						r, err := regexp.Compile(p)
						checkError(err)
						patterns[p] = r
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
			if degenerate || useRegexp {
				for _, p := range pattern {
					if degenerate {
						pattern2seq, err := seq.NewSeq(alphabet, []byte(p))
						if err != nil {
							checkError(fmt.Errorf("it seems that flag -d is given, "+
								"but you provide regular expression instead of available %s sequence", alphabet))
						}
						p = pattern2seq.Degenerate2Regexp()
					}
					if ignoreCase {
						p = "(?i)" + p
					}

					re, err := regexp.Compile(p)
					checkError(err)
					patterns[p] = re
				}
			} else {
				for _, p := range pattern {
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

		var subject []byte
		var hit bool
		for _, file := range files {
			fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)

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

				if byName {
					subject = record.Name
				} else if bySeq {
					if limitRegion {
						subject = record.Seq.SubSeq(start, end).Seq
					} else {
						subject = record.Seq.Seq
					}
				} else {
					subject = record.ID
				}

				hit = false
				if degenerate || useRegexp {
					for pattern, re := range patterns {
						if re.Match(subject) {
							hit = true
							if deleteMatched {
								delete(patterns, pattern)
							}
							break
						}
					}
				} else {
					k := string(subject)
					if ignoreCase {
						k = strings.ToLower(k)
					}
					if _, ok := patterns[k]; ok {
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

	grepCmd.Flags().StringSliceP("pattern", "p", []string{""}, "search pattern (multiple values supported)")
	grepCmd.Flags().StringP("pattern-file", "f", "", "pattern file (one record per line)")
	grepCmd.Flags().BoolP("use-regexp", "r", false, "patterns are regular expression")
	grepCmd.Flags().BoolP("delete-matched", "", false, "delete matched pattern to speedup")
	grepCmd.Flags().BoolP("invert-match", "v", false, "invert the sense of matching, to select non-matching records")
	grepCmd.Flags().BoolP("by-name", "n", false, "match by full name instead of just id")
	grepCmd.Flags().BoolP("by-seq", "s", false, "match by seq")
	grepCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
	grepCmd.Flags().BoolP("degenerate", "d", false, "pattern/motif contains degenerate base")
	grepCmd.Flags().StringP("region", "R", "", "specify sequence region for searching. "+
		"e.g 1:12 for first 12 bases, -12:-1 for last 12 bases")

}
