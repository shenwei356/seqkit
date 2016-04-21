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
	"github.com/shenwei356/util/stringutil"
	"github.com/spf13/cobra"
)

// sortCmd represents the seq command
var sortCmd = &cobra.Command{
	Use:   "sort",
	Short: "sort sequences by id/name/sequence/length",
	Long: `sort sequences by id/name/sequence/length

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

		files := getFileList(args)

		bySeq := getFlagBool(cmd, "by-seq")
		byName := getFlagBool(cmd, "by-name")
		byLength := getFlagBool(cmd, "by-length")
		reverse := getFlagBool(cmd, "reverse")
		ignoreCase := getFlagBool(cmd, "ignore-case")

		n := 0
		if bySeq {
			n++
		}
		if byName {
			n++
		}
		if byLength {
			n++
		}
		if n > 1 {
			checkError(fmt.Errorf("only one of the flags -l (--by-length), -n (--by-name) and -s (--by-seq) is allowed"))
		}

		byID := true
		if bySeq || byLength {
			byID = false
		}
		if !quiet {
			if byLength {
				if ignoreCase {
					log.Warning("flag -i (--ignore-case) is ignored when flag -l (--by-length) given")
				}
			}
		}
		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		sequences := make(map[string]*fastx.Record)
		name2sequence := []stringutil.String2ByteSlice{}
		name2length := []stringutil.StringCount{}

		if !quiet {
			log.Infof("read sequences ...")
		}
		var name string
		for _, file := range files {
			fastxReader, err := fastx.NewReader(alphabet, file, bufferSize, chunkSize, idRegexp)
			checkError(err)
			for chunk := range fastxReader.Ch {
				checkError(chunk.Err)

				for _, record := range chunk.Data {
					if byName {
						name = string(record.Name)
					} else if byID || bySeq || byLength {
						name = string(record.ID)
					}
					if ignoreCase {
						name = strings.ToLower(name)
					}

					sequences[name] = record
					if byLength {
						name2length = append(name2length, stringutil.StringCount{Key: name, Count: len(record.Seq.Seq)})
					} else if byID || byName || bySeq {
						if ignoreCase {
							name2sequence = append(name2sequence, stringutil.String2ByteSlice{Key: name, Value: bytes.ToLower(record.Seq.Seq)})
						} else {
							name2sequence = append(name2sequence, stringutil.String2ByteSlice{Key: name, Value: record.Seq.Seq})
						}
					}
				}
			}
		}

		if !quiet {
			log.Infof("%d sequences loaded", len(sequences))
			log.Infof("sorting ...")
		}

		if bySeq {
			if reverse {
				sort.Sort(stringutil.ReversedByValue{stringutil.String2ByteSliceList(name2sequence)})
			} else {
				sort.Sort(stringutil.ByValue{stringutil.String2ByteSliceList(name2sequence)})
			}
		} else if byLength {
			if reverse {
				sort.Sort(stringutil.ReversedStringCountList{stringutil.StringCountList(name2length)})
			} else {
				sort.Sort(stringutil.StringCountList(name2length))
			}
		} else if byName || byID { // by name/id
			if reverse {
				sort.Sort(stringutil.ReversedString2ByteSliceList{stringutil.String2ByteSliceList(name2sequence)})
			} else {
				sort.Sort(stringutil.String2ByteSliceList(name2sequence))
			}
		}

		if !quiet {
			log.Infof("output ...")
		}
		var record *fastx.Record
		if byName || byID || bySeq {
			for _, kv := range name2sequence {
				record = sequences[kv.Key]
				outfh.WriteString(record.Format(lineWidth))
			}
		} else if byLength {
			for _, kv := range name2length {
				record = sequences[kv.Key]
				outfh.WriteString(record.Format(lineWidth))
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(sortCmd)
	sortCmd.Flags().BoolP("by-name", "n", false, "by full name instead of just id")
	sortCmd.Flags().BoolP("by-seq", "s", false, "by sequence")
	sortCmd.Flags().BoolP("by-length", "l", false, "by sequence length")
	sortCmd.Flags().BoolP("reverse", "r", false, "reverse the result")
	sortCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")

}
