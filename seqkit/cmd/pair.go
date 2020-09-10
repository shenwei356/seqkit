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
	"os"
	"path/filepath"
	"runtime"

	"github.com/cespare/xxhash"
	"github.com/pkg/errors"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/util/pathutil"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// pairCmd represents the pair command
var pairCmd = &cobra.Command{
	Use:   "pair",
	Short: "match up paired-end reads from two fastq files",
	Long: `match up paired-end reads from two fastq files

Attension:
1. Orders of headers in the two files better be the same (sorted),
   Or lots of memory needed to cache reads in memory.

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		lineWidth := config.LineWidth
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		if len(args) > 0 {
			checkError(errors.New("no positional arugments should be given"))
		}

		read1 := getFlagString(cmd, "read1")
		read2 := getFlagString(cmd, "read2")
		if read1 == "" || read2 == "" {
			checkError(fmt.Errorf("flag -1/--read1 and -2/--read2 needed"))
		}

		outdir := getFlagString(cmd, "out-dir")
		force := getFlagBool(cmd, "force")

		if outdir == "" {
			outdir = fmt.Sprintf("%s.paired", read1)
		} else if filepath.Clean(filepath.Dir(read1)) == filepath.Clean(outdir) {
			checkError(fmt.Errorf("outdir (%s) should be different from inputdir (%s)", filepath.Clean(outdir), filepath.Clean(filepath.Dir(read1))))
		}

		if outdir != "./" && outdir != "." {
			existed, err := pathutil.DirExists(outdir)
			checkError(err)
			if existed {
				empty, err := pathutil.IsEmpty(outdir)
				checkError(err)
				if !empty {
					if force {
						checkError(os.RemoveAll(outdir))
						checkError(os.MkdirAll(outdir, 0755))
					} else {
						log.Warningf("outdir not empty: %s, you can use --force to overwrite", outdir)
					}
				}
			} else {
				checkError(os.MkdirAll(outdir, 0755))
			}
		}

		var reader1, reader2 *fastx.Reader
		var record1, record2 *fastx.Record

		var err error
		reader1, err = fastx.NewReader(alphabet, read1, idRegexp)
		checkError(errors.Wrap(err, read1))
		reader2, err = fastx.NewReader(alphabet, read2, idRegexp)
		checkError(errors.Wrap(err, read2))

		outFile1 := filepath.Join(outdir, filepath.Base(read1))
		outfh1, err := xopen.Wopen(outFile1)
		checkError(errors.Wrap(err, outFile1))
		defer outfh1.Close()

		outFile2 := filepath.Join(outdir, filepath.Base(read2))
		outfh2, err := xopen.Wopen(outFile2)
		checkError(errors.Wrap(err, outFile2))
		defer outfh2.Close()

		record1, err = reader1.Read()
		checkError(errors.Wrap(err, read1))
		record2, err = reader2.Read()
		checkError(errors.Wrap(err, read2))
		if !reader1.IsFastq || !reader2.IsFastq {
			checkError(fmt.Errorf("fastq files needed"))
		}

		m1 := make(map[uint64]*fastx.Record, 1024)
		m2 := make(map[uint64]*fastx.Record, 1024)
		var h1, h2 uint64
		var ok1, ok2 bool
		var r1, r2 *fastx.Record
		var eof1, eof2 bool
		var n uint64

		for {
			if eof1 && eof2 {
				break
			}

			if bytes.Compare(record1.Seq.Seq, record2.Seq.Seq) == 0 {
				record1.FormatToWriter(outfh1, lineWidth)
				record2.FormatToWriter(outfh2, lineWidth)
				n++

				// new read1
				if !eof1 {
					record1, err = reader1.Read()
					if err != nil {
						if err == io.EOF {
							eof1 = true
							// break
						} else {
							checkError(errors.Wrap(err, read1))
							break
						}
					}
				}

				// new read2
				if !eof2 {
					record2, err = reader2.Read()
					if err != nil {
						if err == io.EOF {
							eof2 = true
							// break
						} else {
							checkError(errors.Wrap(err, read2))
							break
						}
					}
				}
				continue
			}

			h1 = xxhash.Sum64(record1.ID)
			if r2, ok2 = m2[h1]; ok2 { // found pair of record1 in m2
				record1.FormatToWriter(outfh1, lineWidth)
				r2.FormatToWriter(outfh2, lineWidth)
				n++

				delete(m2, h1)
			} else {
				m1[h1] = record1.Clone()
			}

			// new read1

			if !eof1 {
				record1, err = reader1.Read()
				if err != nil {
					if err == io.EOF {
						eof1 = true
						// break
					} else {
						checkError(errors.Wrap(err, read1))
						break
					}
				}
			}

			// ---

			h2 = xxhash.Sum64(record2.ID)
			if r1, ok1 = m1[h2]; ok1 { // found pair of record2 in m1
				r1.FormatToWriter(outfh1, lineWidth)
				record2.FormatToWriter(outfh2, lineWidth)
				n++

				delete(m1, h2)
			} else {
				m2[h2] = record2.Clone()
			}

			// new read2

			if !eof2 {
				record2, err = reader2.Read()
				if err != nil {
					if err == io.EOF {
						eof2 = true
						// break
					} else {
						checkError(errors.Wrap(err, read2))
						break
					}
				}
			}
		}

		// left reads
		if len(m1) > 0 && len(m2) > 0 {
			for h1, r1 = range m1 {
				if r2, ok2 = m2[h1]; ok2 {
					record1.FormatToWriter(outfh1, lineWidth)
					record2.FormatToWriter(outfh2, lineWidth)
					n++
				}
			}
		}

		if !config.Quiet {
			log.Infof("%d paired-end reads saved", n)
		}

	},
}

func init() {
	RootCmd.AddCommand(pairCmd)

	pairCmd.Flags().StringP("read1", "1", "", "read1 file")
	pairCmd.Flags().StringP("read2", "2", "", "read2 file")
	pairCmd.Flags().StringP("out-dir", "O", "", "output directory (default value is $read1.paired)")
	pairCmd.Flags().BoolP("force", "f", false, "overwrite output directory")
}
