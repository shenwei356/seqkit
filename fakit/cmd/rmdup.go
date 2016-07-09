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
	"bytes"
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/brentp/xopen"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/spf13/cobra"
)

// rmdupCmd represents the rmdup command
var rmdupCmd = &cobra.Command{
	Use:   "rmdup",
	Short: "remove duplicated sequences by id/name/sequence",
	Long: `remove duplicated sequences by id/name/sequence

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		chunkSize := config.ChunkSize
		bufferSize := config.BufferSize
		lineWidth := config.LineWidth
		outFile := config.OutFile
		quiet := config.Quiet
		seq.AlphabetGuessSeqLenghtThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		bySeq := getFlagBool(cmd, "by-seq")
		byName := getFlagBool(cmd, "by-name")
		ignoreCase := getFlagBool(cmd, "ignore-case")
		dupFile := getFlagString(cmd, "dup-seqs-file")
		numFile := getFlagString(cmd, "dup-num-file")
		usingMD5 := getFlagBool(cmd, "md5")

		if bySeq && byName {
			checkError(fmt.Errorf("only one/none of the flags -s (--by-seq) and -n (--by-name) is allowed"))
		}
		if usingMD5 && !bySeq {
			checkError(fmt.Errorf("flag -m (--md5) must be used with flag -s (--by-seq)"))
		}

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var outfhDup *xopen.Writer
		if len(dupFile) > 0 {
			outfhDup, err = xopen.Wopen(dupFile)
			checkError(err)
			defer outfhDup.Close()
		}

		counter := make(map[string]int)
		names := make(map[string][]string)

		var subject string
		removed := 0
		for _, file := range files {
			fastxReader, err := fastx.NewReader(alphabet, file, bufferSize, chunkSize, idRegexp)
			checkError(err)
			for chunk := range fastxReader.Ch {
				checkError(chunk.Err)

				for _, record := range chunk.Data {
					if bySeq {
						if ignoreCase {
							if usingMD5 {
								subject = MD5(bytes.ToLower(record.Seq.Seq))
							} else {
								subject = string(bytes.ToLower(record.Seq.Seq))
							}
						} else {
							if usingMD5 {
								subject = MD5(record.Seq.Seq)
							} else {
								subject = string(record.Seq.Seq)
							}
						}
					} else if byName {
						subject = string(record.Name)
					} else { // byID
						subject = string(record.ID)
					}

					if _, ok := counter[subject]; ok { // duplicated
						counter[subject]++
						removed++
						if len(dupFile) > 0 {
							outfhDup.Write(record.Format(lineWidth))
						}
						if len(numFile) > 0 {
							names[subject] = append(names[subject], string(record.ID))
						}
					} else { // new one
						record.FormatToWriter(outfh, lineWidth)
						counter[subject]++

						if len(numFile) > 0 {
							names[subject] = []string{}
							names[subject] = append(names[subject], string(record.ID))
						}
					}

					record.Recycle()
				}
			}
		}
		if removed > 0 && len(numFile) > 0 {
			outfhNum, err := xopen.Wopen(numFile)
			checkError(err)
			defer outfhNum.Close()

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
	rmdupCmd.Flags().BoolP("md5", "m", false, "use MD5 instead of original seqs to reduce memory usage when comparing by seqs")
	rmdupCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
	rmdupCmd.Flags().StringP("dup-seqs-file", "d", "", "file to save duplicated seqs")
	rmdupCmd.Flags().StringP("dup-num-file", "D", "", "file to save number and list of duplicated seqs")
}

type listOfStringSlice struct {
	data [][]string
}

func (l listOfStringSlice) Len() int           { return len(l.data) }
func (l listOfStringSlice) Less(i, j int) bool { return len(l.data[i]) > len(l.data[j]) }
func (l listOfStringSlice) Swap(i, j int)      { l.data[i], l.data[j] = l.data[j], l.data[i] }
