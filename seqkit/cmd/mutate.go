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

// mutateCmd represents the mutate command
var mutateCmd = &cobra.Command{
	GroupID: "edit",

	Use:   "mutate",
	Short: "edit sequence (point mutation, insertion, deletion)",
	Long: fmt.Sprintf(`edit sequence (point mutation, insertion, deletion)

Attention:

  1. Multiple point mutations (-p/--point) are allowed, but only single
     insertion (-i/--insertion) OR single deletion (-d/--deletion) is allowed.
  2. Point mutation takes place before insertion/deletion.

Notes:

  1. You can choose certain sequences to edit using similar flags in
     'seqkit grep'.

The definition of position is 1-based and with some custom design.

Examples:
%s
`, regionExample),
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		lineWidth := config.LineWidth
		outFile := config.OutFile
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)
		quiet := config.Quiet

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)
		var err error

		mPoints := []_mutatePoint{}
		for _, val := range getFlagStringSlice(cmd, "point") {
			if !reMutationPoint.MatchString(val) {
				checkError(fmt.Errorf("invalid value of flag -p/--point : %s", val))
			}
			items := strings.Split(val, ":")
			pos, _ := strconv.Atoi(items[0])
			if pos == 0 {
				checkError(fmt.Errorf("position should be non-zero integer"))
			}
			mPoints = append(mPoints, _mutatePoint{pos: pos, base: items[1][0]})
		}

		valDel := getFlagString(cmd, "deletion")
		valIns := getFlagString(cmd, "insertion")

		if valDel != "" && valIns != "" {
			checkError(fmt.Errorf("flag -i/--insertion and -d/--deletion can't be used at the same time"))
		}

		var mDel *_mutateDel
		if valDel != "" {
			if !reMutationDel.MatchString(valDel) {
				checkError(fmt.Errorf("invalid value of flag -id/--deletion : %s", valDel))
			}
			items := strings.Split(valDel, ":")
			start, _ := strconv.Atoi(items[0])
			end, _ := strconv.Atoi(items[1])

			if start == 0 || end == 0 {
				checkError(fmt.Errorf("both start and end should not be 0"))
			}
			if start < 0 && end > 0 {
				checkError(fmt.Errorf("when start < 0, end should not > 0"))
			}

			mDel = &_mutateDel{start: start, end: end}
		}

		var mIns *_mutateIns
		if valIns != "" {
			if !reMutationIns.MatchString(valIns) {
				checkError(fmt.Errorf("invalid value of flag -i/--insertion : %s", valIns))
			}
			items := strings.Split(valIns, ":")
			pos, _ := strconv.Atoi(items[0])

			mIns = &_mutateIns{pos: pos, seq: []byte(items[1])}
		}

		// flags for choose which sequences to mutate/edit

		pattern := getFlagStringSlice(cmd, "pattern")
		patternFile := getFlagString(cmd, "pattern-file")
		useRegexp := getFlagBool(cmd, "use-regexp")
		invertMatch := getFlagBool(cmd, "invert-match")
		byName := getFlagBool(cmd, "by-name")
		ignoreCase := getFlagBool(cmd, "ignore-case")

		editAll := len(pattern) == 0 && patternFile == ""
		// if len(pattern) == 0 && patternFile == "" {
		// 	checkError(fmt.Errorf("one of flags -p (--pattern) and -f (--pattern-file) needed"))
		// }

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

		// prepare pattern
		patterns := make(map[string]*regexp.Regexp)
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
					if useRegexp {
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
			for _, p := range pattern {
				if useRegexp {
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

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var record *fastx.Record
		var checkFQ = true
		var mp _mutatePoint
		var seqLen int
		var s, e int
		var ok bool
		var target []byte
		var hit bool
		var k string
		var re *regexp.Regexp
		for _, file := range files {
			fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
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

				if checkFQ && fastxReader.IsFastq {
					checkError(fmt.Errorf("FASTQ not supported"))
					checkFQ = false
				}

				if !editAll {

					// ----------- only mutate some matched sequence -------------

					if byName {
						target = record.Name
					} else {
						target = record.ID
					}

					hit = false

					if useRegexp {
						for _, re = range patterns {
							if re.Match(target) {
								hit = true
								break
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
							// do not mutate
							record.FormatToWriter(outfh, lineWidth)
							continue
						}
					} else {
						if !hit {
							// do not mutate
							record.FormatToWriter(outfh, lineWidth)
							continue
						}
					}

					// need to mutate
				}

				// -----------------------------------------------------------
				if !quiet {
					log.Infof("edit seq: %s", record.Name)
				}

				seqLen = len(record.Seq.Seq)

				for _, mp = range mPoints {
					s, _, ok = seq.SubLocation(seqLen, mp.pos, mp.pos)
					if !ok {
						log.Warningf("[%s]: point mutation: position (%d) out of sequence length (%d)", record.ID, mp.pos, seqLen)
						continue
					}
					record.Seq.Seq[s-1] = mp.base
				}

				if mDel != nil {
					s, e, ok = seq.SubLocation(seqLen, mDel.start, mDel.end)

					if !ok {
						log.Warningf("[%s]: deletion mutation: range (%d-%d) out of sequence length (%d)", record.ID, mDel.start, mDel.end, seqLen)
					} else {
						copy(record.Seq.Seq[s-1:seqLen-(e-s+1)], record.Seq.Seq[e:])
						record.Seq.Seq = record.Seq.Seq[0 : seqLen-(e-s+1)]
					}
				}

				if mIns != nil {
					if mIns.pos == 0 {
						s = 0
						record.Seq.Seq = append(record.Seq.Seq, mIns.seq...)
						copy(record.Seq.Seq[s+len(mIns.seq):], record.Seq.Seq[s:seqLen])
						copy(record.Seq.Seq[s:s+len(mIns.seq)], mIns.seq)
					} else {
						s, _, ok = seq.SubLocation(seqLen, mIns.pos, mIns.pos)
						if !ok {
							log.Warningf("[%s]: insertion mutation: position (%d) out of sequence length (%d)", record.ID, mIns.pos, seqLen)
						} else {
							record.Seq.Seq = append(record.Seq.Seq, mIns.seq...)             // append insert sequence to the end
							copy(record.Seq.Seq[s+len(mIns.seq):], record.Seq.Seq[s:seqLen]) // move sequence behind the insert location at the end, so leave space for IS.
							copy(record.Seq.Seq[s:s+len(mIns.seq)], mIns.seq)                // copy IS into desired region.
						}
					}

				}
				record.FormatToWriter(outfh, lineWidth)
			}
			fastxReader.Close()
		}
	},
}

func init() {
	RootCmd.AddCommand(mutateCmd)

	mutateCmd.Flags().StringSliceP("point", "p", []string{}, `point mutation: changing base at given position. e.g., -p 2:C for setting 2nd base as C, -p -1:A for change last base as A`)
	mutateCmd.Flags().StringP("deletion", "d", "", `deletion mutation: deleting subsequence in a range. e.g., -d 1:2 for deleting leading two bases, -d -3:-1 for removing last 3 bases`)
	mutateCmd.Flags().StringP("insertion", "i", "", `insertion mutation: inserting bases behind of given position, e.g., -i 0:ACGT for inserting ACGT at the beginning, -1:* for add * to the end`)

	mutateCmd.Flags().StringSliceP("pattern", "s", []string{""}, `[match seqs to mutate] search pattern (multiple values supported. Attention: use double quotation marks for patterns containing comma, e.g., -p '"A{2,}"'))`)
	mutateCmd.Flags().StringP("pattern-file", "f", "", "[match seqs to mutate] pattern file (one record per line)")
	mutateCmd.Flags().BoolP("use-regexp", "r", false, "[match seqs to mutate] search patterns are regular expression")
	mutateCmd.Flags().BoolP("invert-match", "v", false, "[match seqs to mutate] invert the sense of matching, to select non-matching records")
	mutateCmd.Flags().BoolP("by-name", "n", false, "[match seqs to mutate] match by full name instead of just id")
	mutateCmd.Flags().BoolP("ignore-case", "I", false, "[match seqs to mutate] ignore case of search pattern")

}

var reMutationPoint = regexp.MustCompile(`^(\-?\d+)\:(.)$`)
var reMutationDel = regexp.MustCompile(`^(\-?\d+):(\-?\d+)$`)
var reMutationIns = regexp.MustCompile(`^(\-?\d+)\:(.+)$`)

type _mutatePoint struct {
	pos  int
	base byte
}

type _mutateDel struct {
	start, end int
}

type _mutateIns struct {
	pos int
	seq []byte
}
