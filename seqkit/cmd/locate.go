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

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// locateCmd represents the locate command
var locateCmd = &cobra.Command{
	Use:   "locate",
	Short: "locate subsequences/motifs",
	Long: `locate subsequences/motifs

Motifs could be EITHER plain sequence containing "ACTGN" OR regular
expression like "A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)" for ORFs.
Degenerate bases like "RYMM.." are also supported by flag -d.

By default, motifs are treated as regular expression.
When flag -d given, regular expression may be wrong.
For example: "\w" will be wrongly converted to "\[AT]".
`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		outFile := config.OutFile
		seq.AlphabetGuessSeqLenghtThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = true
		seq.ValidateWholeSeq = false
		seq.ValidSeqLengthThreshold = getFlagValidateSeqLength(cmd, "validate-seq-length")
		seq.ValidSeqThreads = config.Threads
		seq.ComplementThreads = config.Threads
		runtime.GOMAXPROCS(config.Threads)

		pattern := getFlagStringSlice(cmd, "pattern")
		patternFile := getFlagString(cmd, "pattern-file")
		degenerate := getFlagBool(cmd, "degenerate")
		ignoreCase := getFlagBool(cmd, "ignore-case")
		onlyPositiveStrand := getFlagBool(cmd, "only-positive-strand")

		if len(pattern) == 0 && patternFile == "" {
			checkError(fmt.Errorf("one of flags --pattern and --pattern-file needed"))
		}

		files := getFileList(args)

		// prepare pattern
		regexps := make(map[string]*regexp.Regexp)
		patterns := make(map[string][]byte)
		var s string
		if patternFile != "" {
			records, err := fastx.GetSeqsMap(patternFile, nil, config.Threads, 10, "")
			checkError(err)
			for name, record := range records {
				patterns[name] = record.Seq.Seq

				if degenerate {
					s = record.Seq.Degenerate2Regexp()
				} else {
					s = string(record.Seq.Seq)
				}

				if ignoreCase {
					s = "(?i)" + s
				}
				re, err := regexp.Compile(s)
				checkError(err)
				regexps[name] = re
			}
		} else {
			for _, p := range pattern {
				patterns[p] = []byte(p)

				if degenerate {
					pattern2seq, err := seq.NewSeq(alphabet, []byte(p))
					if err != nil {
						checkError(fmt.Errorf("it seems that flag -d is given, "+
							"but you provide regular expression instead of available %s sequence", alphabet))
					}
					s = pattern2seq.Degenerate2Regexp()
				} else {
					s = p
				}

				if ignoreCase {
					s = "(?i)" + s
				}
				re, err := regexp.Compile(s)
				checkError(err)
				regexps[p] = re
			}
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		outfh.WriteString("seqID\tpatternName\tpattern\tstrand\tstart\tend\tmatched\n")
		var seqRP *seq.Seq
		var offset, l int
		var loc []int
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

				l = len(record.Seq.Seq)
				if !onlyPositiveStrand {
					seqRP = record.Seq.RevCom()
				}
				for pName, re := range regexps {
					offset = 0
					for {
						loc = re.FindSubmatchIndex(record.Seq.Seq[offset:])
						if loc == nil {
							break
						}
						outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
							record.ID,
							pName,
							patterns[pName],
							"+",
							offset+loc[0]+1,
							offset+loc[1],
							record.Seq.Seq[offset+loc[0]:offset+loc[1]]))

						offset = offset + loc[0] + 1
						if offset >= len(record.Seq.Seq) {
							break
						}
					}

					if onlyPositiveStrand {
						continue
					}

					offset = 0
					for {
						loc = re.FindSubmatchIndex(seqRP.Seq[offset:])
						if loc == nil {
							break
						}
						if len(loc) > 0 {
							outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
								record.ID,
								pName,
								patterns[pName],
								"-",
								l-offset-loc[1]+1,
								l-offset-loc[0],
								record.Seq.SubSeq(l-offset-loc[1]+1, l-offset-loc[0]).RevCom().Seq))
						}

						offset = offset + loc[0] + 1
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

	locateCmd.Flags().StringSliceP("pattern", "p", []string{""}, "search pattern/motif (multiple values supported)")
	locateCmd.Flags().StringP("pattern-file", "f", "", "pattern/motif file (FASTA format)")
	locateCmd.Flags().BoolP("degenerate", "d", false, "pattern/motif contains degenerate base")
	locateCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
	locateCmd.Flags().BoolP("only-positive-strand", "P", false, "only search at positive strand")
	locateCmd.Flags().IntP("validate-seq-length", "V", 10000, "length of sequence to validate (0 for whole seq)")

}
