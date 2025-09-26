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
	"sort"
	"strings"

	"github.com/cespare/xxhash/v2"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// rmdupCmd represents the rmdup command
var rmdupCmd = &cobra.Command{
	GroupID: "set",

	Use:   "rmdup",
	Short: "remove duplicated sequences by ID/name/sequence",
	Long: `remove duplicated sequences by ID/name/sequence

Attention:
  1. When comparing by sequences, both positive and negative strands are
     compared. Switch on -P/--only-positive-strand for considering the
     positive strand only.
  2. Only the first record is saved for duplicates.
     
`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		lineWidth := config.LineWidth
		outFile := config.OutFile
		quiet := config.Quiet
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		bySeq := getFlagBool(cmd, "by-seq")
		byName := getFlagBool(cmd, "by-name")
		ignoreCase := getFlagBool(cmd, "ignore-case")
		dupFile := getFlagString(cmd, "dup-seqs-file")
		numFile := getFlagString(cmd, "dup-num-file")

		saveDupFile := dupFile != ""
		saveNumFile := numFile != ""

		// revcom := getFlagBool(cmd, "consider-revcom")
		revcom := !getFlagBool(cmd, "only-positive-strand")

		if bySeq && byName {
			checkError(fmt.Errorf("only one/none of the flags -s (--by-seq) and -n (--by-name) is allowed"))
		}

		if !revcom && !bySeq {
			checkError(fmt.Errorf("flag -s (--by-seq) needed when using -P (--only-positive-strand)"))
		}

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", !config.SkipFileCheck)
		if !config.SkipFileCheck {
			for _, file := range files {
				checkIfFilesAreTheSame(file, outFile, "input", "output")
			}
		}
		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var outfhDup *xopen.Writer
		if saveDupFile {
			outfhDup, err = xopen.Wopen(dupFile)
			checkError(err)
			defer outfhDup.Close()
		}

		counter := make(map[uint64]int)
		names := make(map[uint64][]string)

		var subject uint64
		var removed int
		var record *fastx.Record
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
				if fastxReader.IsFastq {
					config.LineWidth = 0
					fastx.ForcelyOutputFastq = true
				}

				if bySeq {
					if ignoreCase {
						subject = xxhash.Sum64(bytes.ToLower(record.Seq.Seq))
					} else {
						subject = xxhash.Sum64(record.Seq.Seq)
					}
				} else if byName {
					if ignoreCase {
						subject = xxhash.Sum64(bytes.ToLower(record.Name))
					} else {
						subject = xxhash.Sum64(record.Name)
					}
				} else { // byID
					if ignoreCase {
						subject = xxhash.Sum64(bytes.ToLower(record.ID))
					} else {
						subject = xxhash.Sum64(record.ID)
					}
				}

				if _, ok := counter[subject]; ok { // duplicated
					counter[subject]++
					removed++
					if saveDupFile {
						outfhDup.Write(record.Format(config.LineWidth))
					}
					if saveNumFile {
						names[subject] = append(names[subject], string(record.ID))
					}

					continue
				}

				if bySeq && revcom {
					if ignoreCase {
						subject = xxhash.Sum64(bytes.ToLower(record.Seq.RevCom().Seq))
					} else {
						subject = xxhash.Sum64(record.Seq.RevCom().Seq)
					}

					if _, ok := counter[subject]; ok { // duplicated
						counter[subject]++
						removed++
						if saveDupFile {
							outfhDup.Write(record.Format(config.LineWidth))
						}
						if saveNumFile {
							names[subject] = append(names[subject], string(record.ID))
						}
						continue
					}
				}

				record.FormatToWriter(outfh, config.LineWidth)
				counter[subject]++

				if saveNumFile {
					names[subject] = []string{string(record.ID)}
				}
			}
			fastxReader.Close()

			config.LineWidth = lineWidth
		}

		var outfhNum *xopen.Writer
		if saveNumFile {
			outfhNum, err = xopen.Wopen(numFile)
			checkError(err)
			defer outfhNum.Close()
		}
		if removed > 0 {
			list := new(listOfStringSlice)
			for _, l := range names {
				if len(l) > 1 {
					list.data = append(list.data, l)
				}
			}
			sort.Sort(list)
			for _, l := range list.data {
				outfhNum.WriteString(fmt.Sprintf("%d\t%s\n", len(l), strings.Join(l, ", ")))
			}
		}

		if !quiet {
			log.Infof("%d duplicated records removed", removed)
		}
	},
}

func init() {
	RootCmd.AddCommand(rmdupCmd)

	rmdupCmd.Flags().BoolP("by-name", "n", false, "by full name instead of just id")
	rmdupCmd.Flags().BoolP("by-seq", "s", false, "by seq")
	rmdupCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
	rmdupCmd.Flags().StringP("dup-seqs-file", "d", "", "file to save duplicated seqs")
	rmdupCmd.Flags().StringP("dup-num-file", "D", "", "file to save numbers and ID lists of duplicated seqs")
	// rmdupCmd.Flags().BoolP("consider-revcom", "r", false, "considering the reverse compelment sequence")
	rmdupCmd.Flags().BoolP("only-positive-strand", "P", false, "only considering positive strand when comparing by sequence")
}

type listOfStringSlice struct {
	data [][]string
}

func (l listOfStringSlice) Len() int           { return len(l.data) }
func (l listOfStringSlice) Less(i, j int) bool { return len(l.data[i]) > len(l.data[j]) }
func (l listOfStringSlice) Swap(i, j int)      { l.data[i], l.data[j] = l.data[j], l.data[i] }
