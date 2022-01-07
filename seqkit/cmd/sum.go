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
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
	"github.com/twotwotwo/sorts/sortutil"
)

// sumCmd represents the fq2fa command
var sumCmd = &cobra.Command{
	Use:   "sum",
	Short: "compute message digest for FASTA/Q files",
	Long: `compute message digest for FASTA/Q files

Methods:
  1. computing the hash (xxhash) for every sequence (in lower case, gap removed).
  2. sorting all hash values, for ignoring the order or sequences.
  3. computing MD5 digest from the hash values from 2),
     along with sequences length and number.

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		// lineWidth := config.LineWidth
		outFile := config.OutFile
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		basename := getFlagBool(cmd, "basename")
		circular := getFlagBool(cmd, "circular")
		// removeGaps := getFlagBool(cmd, "remove-gaps")
		gapLetters := getFlagString(cmd, "gap-letters")

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		tokens := make(chan int, config.Threads)
		ch := make(chan SumResult, config.Threads)
		done := make(chan int)
		var wg sync.WaitGroup

		go func() {
			var file string
			for r := range ch {
				file = r.File
				if basename {
					file = filepath.Base(file)
				}
				fmt.Fprintf(outfh, "%s\t%s\n", r.Digest, file)
			}
			done <- 1
		}()

		for _, file := range files {
			tokens <- 1
			wg.Add(1)

			go func(file string) {
				defer func() {
					<-tokens
					wg.Done()
				}()

				var n int // number of sequences
				var l int // lengths of all seqs
				var h uint64
				hashes := make([]uint64, 0, 1024)

				var record *fastx.Record
				var fastxReader *fastx.Reader

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

					if circular && n >= 1 {
						checkError(fmt.Errorf("only one sequence is allowed for circular genome"))
					}

					h = xxhash.Sum64(bytes.ToLower(record.Seq.RemoveGapsInplace(gapLetters).Seq))

					hashes = append(hashes, h)

					n++
					l += len(record.Seq.Seq)
				}

				// sequences
				sortutil.Uint64s(hashes)

				var le = binary.LittleEndian
				buf := make([]byte, 8)
				di := xxhash.New()

				for _, h = range hashes {
					le.PutUint64(buf, h)
					di.Write(buf)
				}

				// sequence length
				le.PutUint64(buf, uint64(l))
				di.Write(buf)

				// sequence number
				le.PutUint64(buf, uint64(n))
				di.Write(buf)

				// sum up
				le.PutUint64(buf, di.Sum64())
				digest := md5.Sum(buf)

				// return result
				ch <- SumResult{
					File:   file,
					SeqNum: n,
					SeqLen: l,
					Digest: hex.EncodeToString(digest[:]),
				}
			}(file)
		}

		wg.Wait()
		close(ch)
		<-done
	},
}

func init() {
	RootCmd.AddCommand(sumCmd)

	sumCmd.Flags().BoolP("circular", "c", false, "the file contains a single cicular genome sequence")
	sumCmd.Flags().BoolP("basename", "b", false, "only output basename of files")
	// seqCmd.Flags().BoolP("remove-gaps", "g", false, "remove gaps")
	sumCmd.Flags().StringP("gap-letters", "G", "- 	.", "gap letters")
}

type SumResult struct {
	File   string
	SeqNum int
	SeqLen int
	Digest string
}
