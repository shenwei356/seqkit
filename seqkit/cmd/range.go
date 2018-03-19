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
	"runtime"
	"strconv"
	"strings"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// rangeCmd represents the range command
var rangeCmd = &cobra.Command{
	Use:   "range",
	Short: "print FASTA/Q records in a range (start:end)",
	Long: `print FASTA/Q records in a range (start:end)

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		lineWidth := config.LineWidth
		outFile := config.OutFile
		seq.AlphabetGuessSeqLenghtThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		rangeStr := getFlagString(cmd, "range")

		if rangeStr == "" {
			checkError(fmt.Errorf("flag -r (--range) needed"))
		}
		if !reRegion.MatchString(rangeStr) {
			checkError(fmt.Errorf(`invalid range: %s. type "seqkit range -h" for more examples`, rangeStr))
		}
		var start, end int
		var err error
		r := strings.Split(rangeStr, ":")
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
		if start < 0 && end < 0 && start > end {
			checkError(fmt.Errorf("when start < 0 and end < 0, start should be < end"))
		}
		if start > 0 && end < 0 {
			checkError(fmt.Errorf("not supported range: %d:%d", start, end))
		}

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var n int
		var bufSize int
		var buf *RecordLoopBuffer
		if start < 0 && end < 0 {
			bufSize = -start
		}

		var record *fastx.Record
		var fastxReader *fastx.Reader
		for _, file := range files {
			fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)

			if start < 0 && end < 0 {
				buf, err = NewRecordLoopBuffer(bufSize)
				checkError(err)
			}

			n = 0
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

				n++

				if start > 0 && end > 0 {
					if n < start {
						continue
					}
					if n > end {
						break
					}
					record.FormatToWriter(outfh, config.LineWidth)
					continue
				}

				if start < 0 && end < 0 {
					buf.Add(record.Clone())
				}
			}

			if start < 0 && end < 0 {
				current0 := buf.Current
				buf.Backward(-end - 1)
				tail := buf.Current
				// fmt.Println(current0, tail, buf.Size, buf.Capacity)
				buf.Current = current0
				var nextNode *RecordNode
				for true {
					nextNode = buf.Next()
					if nextNode == nil {
						break
					}

					nextNode.Value.FormatToWriter(outfh, config.LineWidth)

					if nextNode == tail {
						break
					}
				}
			}

			config.LineWidth = lineWidth
		}
	},
}

func init() {
	RootCmd.AddCommand(rangeCmd)

	rangeCmd.Flags().StringP("range", "r", "", `range. e.g., 1:12 for first 12 records (head -n 12), -12:-1 for last 12 records (tail -n 12)`)
}

// RecordNode is the node for double-linked loop list
type RecordNode struct {
	Value      *fastx.Record
	prev, next *RecordNode
}

func (node RecordNode) String() string {
	return fmt.Sprintf("prev: %s, value: %s, next: %s.", node.prev.Value.ID, node.Value.ID, node.next.Value.ID)
}

// RecordLoopBuffer is a loop buffer for FASTA/Q records
type RecordLoopBuffer struct {
	Size, Capacity int
	Current        *RecordNode
}

// NewRecordLoopBuffer creats new RecordLoopBuffer object with certern capacity
func NewRecordLoopBuffer(capacity int) (*RecordLoopBuffer, error) {
	if capacity < 1 {
		return nil, fmt.Errorf("RecordLoopBuffer: capacity should be > 0")
	}
	return &RecordLoopBuffer{Size: 0, Capacity: capacity, Current: nil}, nil
}

// Add add new RecordNode
func (buf *RecordLoopBuffer) Add(value *fastx.Record) {
	node := &RecordNode{Value: value, prev: nil, next: nil}

	if buf.Size == 0 {
		node.prev = node
		node.next = node
		buf.Current = node
		buf.Size++
		return
	}

	// full
	if buf.Size >= buf.Capacity {
		buf.Current = buf.Current.next
		buf.Current.Value = value
		buf.Size = buf.Capacity
		return
	}

	// not full
	node.prev = buf.Current
	node.next = buf.Current.next
	buf.Current.next.prev = node
	buf.Current.next = node
	buf.Current = node
	buf.Size++
}

// Next returns next node
func (buf *RecordLoopBuffer) Next() *RecordNode {
	if buf.Size == 0 {
		return nil
	}
	buf.Current = buf.Current.next
	return buf.Current
}

// Prev returns previous node
func (buf *RecordLoopBuffer) Prev() *RecordNode {
	if buf.Size == 0 {
		return nil
	}
	buf.Current = buf.Current.prev
	return buf.Current
}

// Backward moves the current pointer backward N nodes
func (buf *RecordLoopBuffer) Backward(n int) {
	for i := 0; i < n; i++ {
		buf.Prev()
	}
}
