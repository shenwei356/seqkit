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
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// mutateCmd represents the mutate command
var mutateCmd = &cobra.Command{
	Use:   "mutate",
	Short: "edit sequence (point mutation, insertion, deletion)",
	Long: `edit sequence (point mutation, insertion, deletion)

Attentions:

1. Mutiple point mutations (-p/--point) are allowed, but only single 
   insertion (-i/--insertion) OR single deletion (-d/--deletion) is allowed.
2. Point mutation takes place before insertion/deletion.

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		lineWidth := config.LineWidth
		outFile := config.OutFile
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		files := getFileList(args)

		mPoints := []_mutatePoint{}
		for _, val := range getFlagStringSlice(cmd, "point") {
			if !reMutationPoint.MatchString(val) {
				checkError(fmt.Errorf("invalid value of flag -p/--point : %s", val))
			}
			items := strings.Split(val, ":")
			pos, _ := strconv.Atoi(items[0])
			if pos < 1 {
				checkError(fmt.Errorf("position should be positive integer, %d given", pos))
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
			items := strings.Split(valDel, "-")
			start, _ := strconv.Atoi(items[0])
			end, _ := strconv.Atoi(items[1])
			if start < 1 {
				checkError(fmt.Errorf("start position should be positive integer, %d given", start))
			}
			if end < 1 {
				checkError(fmt.Errorf("end position should be positive integer, %d given", end))
			}
			if start > end {
				checkError(fmt.Errorf("start position (%d) should be smaller than end position (%d)", start, end))
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
			if pos < 1 {
				checkError(fmt.Errorf("position should be positive integer, %d given", pos))
			}
			mIns = &_mutateIns{pos: pos, seq: []byte(items[1])}
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var record *fastx.Record
		var fastxReader *fastx.Reader
		var checkFQ = true
		var mp _mutatePoint
		var seqLen int
		// var newSeq []byte
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

				if checkFQ && fastxReader.IsFastq {
					checkError(fmt.Errorf("FASTQ not supported"))
					checkFQ = false
				}

				seqLen = len(record.Seq.Seq)

				for _, mp = range mPoints {
					if mp.pos > seqLen {
						log.Warningf("[%s]: point mutation: position (%d) out of sequence length (%d)", record.ID, mp.pos, seqLen)
						continue
					}
					record.Seq.Seq[mp.pos-1] = mp.base
				}

				if mDel != nil {
					if mDel.start > seqLen || mDel.end > seqLen {
						log.Warningf("[%s]: deletion mutation: range (%d-%d) out of sequence length (%d)", record.ID, mDel.start, mDel.end, seqLen)
					} else {
						copy(record.Seq.Seq[mDel.start-1:seqLen-(mDel.end-mDel.start+1)], record.Seq.Seq[mDel.end:])
						record.Seq.Seq = record.Seq.Seq[0 : seqLen-(mDel.end-mDel.start+1)]
					}
				}

				if mIns != nil {
					if mIns.pos > seqLen+1 {
						log.Warningf("[%s]: point mutation: position (%d) out of sequence length + 1 (%d)", record.ID, mp.pos, seqLen+1)
					} else {
						record.Seq.Seq = append(record.Seq.Seq, mIns.seq...)
						copy(record.Seq.Seq[mIns.pos+len(mIns.seq)-1:], record.Seq.Seq[mIns.pos-1:mIns.pos+seqLen])
						copy(record.Seq.Seq[mIns.pos-1:mIns.pos+len(mIns.seq)-1], mIns.seq)
					}
				}
				record.FormatToWriter(outfh, lineWidth)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(mutateCmd)

	mutateCmd.Flags().StringSliceP("point", "p", []string{}, `point mutation: change base at sepcific postion. e.g., -p 2:C for setting 2nd base as C`)
	mutateCmd.Flags().StringP("deletion", "d", "", `deletion mutation: delete subsequence in a range. e.g., -d 1:2 for deleting leading two bases`)
	mutateCmd.Flags().StringP("insertion", "i", "", `insertion mutation: insert at front of position, e.g., -i 1:ACGT for inserting ACGT at the beginning`)
}

var reMutationPoint = regexp.MustCompile(`^(\d+)\:(\w)$`)
var reMutationDel = regexp.MustCompile(`^(\d+)\-(\d+)$`)
var reMutationIns = regexp.MustCompile(`^(\d+)\:(\w+)$`)

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
