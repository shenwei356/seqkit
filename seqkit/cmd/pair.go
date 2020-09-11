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

Attensions:
1. Orders of headers in the two files better be the same (not shuffled),
   otherwise it consumes huge number of memory for buffering reads in memory.
2. Unpaired reads are optional outputted with flag -u/--save-unpaired.
3. If flag -O/--out-dir not given, output will be saved in the same directory
   of input, with suffix "paired", e.g., read_1.paired.fq.gz.
   Otherwise names are kept untouched in the given output directory.

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		lineWidth := 0
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		if len(args) > 0 {
			checkError(errors.New("no positional arugments are allowed"))
		}

		read1 := getFlagString(cmd, "read1")
		read2 := getFlagString(cmd, "read2")
		if read1 == "" || read2 == "" {
			checkError(fmt.Errorf("flag -1/--read1 and -2/--read2 needed"))
		}
		if read1 == read2 {
			checkError(fmt.Errorf("values of flag -1/--read1 and -2/--read2 can not be the same"))
		}

		outdir := getFlagString(cmd, "out-dir")
		force := getFlagBool(cmd, "force")
		saveUnpaired := getFlagBool(cmd, "save-unpaired")

		if outdir == "" {
			outdir = filepath.Dir(read1)
		}

		var addSuffix bool
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
		} else if filepath.Clean(filepath.Dir(read1)) == filepath.Clean(outdir) {
			addSuffix = true
		}

		var reader1, reader2 *fastx.Reader
		var record1, record2 *fastx.Record

		// readers
		var err error
		reader1, err = fastx.NewReader(alphabet, read1, idRegexp)
		checkError(errors.Wrap(err, read1))
		reader2, err = fastx.NewReader(alphabet, read2, idRegexp)
		checkError(errors.Wrap(err, read2))

		// out file 1
		var outFile1, base1, suffix1 string
		if addSuffix {
			base1, suffix1 = filepathTrimExtension(filepath.Base(read1))
			outFile1 = filepath.Join(outdir, base1+".paired"+suffix1)
		} else {
			outFile1 = filepath.Join(outdir, filepath.Base(read1))
		}
		outfh1, err := xopen.Wopen(outFile1)
		checkError(errors.Wrap(err, outFile1))
		defer outfh1.Close()

		// out file 2
		var outFile2, base2, suffix2 string
		if addSuffix {
			base2, suffix2 = filepathTrimExtension(filepath.Base(read2))
			outFile2 = filepath.Join(outdir, base2+".paired"+suffix2)
		} else {
			outFile2 = filepath.Join(outdir, filepath.Base(read2))
		}
		outfh2, err := xopen.Wopen(outFile2)
		checkError(errors.Wrap(err, outFile2))
		defer outfh2.Close()

		// load first records
		record1, err = reader1.Read()
		checkError(errors.Wrap(err, read1))
		record2, err = reader2.Read()
		checkError(errors.Wrap(err, read2))

		// require fastq
		if !reader1.IsFastq || !reader2.IsFastq {
			checkError(fmt.Errorf("fastq files needed"))
		}

		// buffer for saving unpaired reads
		m1 := make(map[uint64]*fastx.Record, 1024)
		m2 := make(map[uint64]*fastx.Record, 1024)

		var h1, h2 uint64
		var ok1, ok2 bool
		var r1, r2 *fastx.Record
		var eof1, eof2 bool
		var n uint64

		for {
			// break when finishing reading both files.
			if eof1 && eof2 {
				break
			}

			// paired
			if !eof1 && !eof2 && bytes.Compare(record1.ID, record2.ID) == 0 { // same ID
				// output paired reads
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

			if !eof1 {
				h1 = xxhash.Sum64(record1.ID)
				if r2, ok2 = m2[h1]; ok2 { // found pair of record1 in m2
					// output paired reads
					record1.FormatToWriter(outfh1, lineWidth)
					r2.FormatToWriter(outfh2, lineWidth)
					n++

					delete(m2, h1)
				} else {
					m1[h1] = record1.Clone()
				}

				// new read1
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

			if !eof2 {
				h2 = xxhash.Sum64(record2.ID)
				if r1, ok1 = m1[h2]; ok1 { // found pair of record2 in m1
					// output paired reads
					r1.FormatToWriter(outfh1, lineWidth)
					record2.FormatToWriter(outfh2, lineWidth)
					n++

					delete(m1, h2)
				} else {
					m2[h2] = record2.Clone()
				}

				// new read2
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

		var outFile1U, outFile2U string
		var outfh1U, outfh2U *xopen.Writer
		var n1U, n2U uint64
		if saveUnpaired {
			if !addSuffix {
				base1, suffix1 = filepathTrimExtension(filepath.Base(read1))
				base2, suffix2 = filepathTrimExtension(filepath.Base(read2))
			}
		}

		// left reads
		if len(m1) > 0 && len(m2) > 0 {
			for h1, r1 = range m1 {

				if string(r1.ID) == "A00582:209:HWWJCDSXX:1:1101:8314:2581" {
					fmt.Println("shit----")
				}

				if r2, ok2 = m2[h1]; ok2 {
					// output paired reads
					r1.FormatToWriter(outfh1, lineWidth)
					r2.FormatToWriter(outfh2, lineWidth)
					n++

					if saveUnpaired { // delete paired reads in m2
						// delete(m1, h1) // no need
						delete(m2, h2)
					}
				} else if saveUnpaired { // unpaired reads in m1
					if outfh1U == nil {
						outFile1U = filepath.Join(outdir, base1+".unpaired"+suffix1)
						outfh1U, err = xopen.Wopen(outFile1U)
						checkError(errors.Wrap(err, outFile1U))
						defer outfh1U.Close()
					}
					r1.FormatToWriter(outfh1U, lineWidth)
					n1U++
				}
			}
			if saveUnpaired {
				for _, r2 = range m2 { // left unpaired reads in m2
					if outfh2U == nil {
						outFile2U = filepath.Join(outdir, base2+".unpaired"+suffix2)
						outfh2U, err = xopen.Wopen(outFile2U)
						checkError(errors.Wrap(err, outFile2U))
						defer outfh2U.Close()
					}

					r2.FormatToWriter(outfh2U, lineWidth)
					n2U++
				}
			}
		} else if len(m1) > 0 {
			if saveUnpaired {
				for _, r1 = range m1 { // all reads in m1 are unpaired
					if outfh1U == nil {
						outFile1U = filepath.Join(outdir, base1+".unpaired"+suffix1)
						outfh1U, err = xopen.Wopen(outFile1U)
						checkError(errors.Wrap(err, outFile1U))
						defer outfh1U.Close()
					}

					r1.FormatToWriter(outfh1U, lineWidth)
					n1U++
				}
			}
		} else if saveUnpaired { // len(m2) > 0
			for _, r2 = range m2 { // all reads in m2 are unpaired
				if outfh2U == nil {
					outFile2U = filepath.Join(outdir, base2+".unpaired"+suffix2)
					outfh2U, err = xopen.Wopen(outFile2U)
					checkError(errors.Wrap(err, outFile2U))
					defer outfh2U.Close()
				}

				r2.FormatToWriter(outfh2U, lineWidth)
				n2U++
			}
		}

		if !config.Quiet {
			log.Infof("%d paired-end reads saved to %s and %s", n, outFile1, outFile2)
			if saveUnpaired {
				if n1U > 0 {
					log.Infof("%d unpaired reads saved to %s", n1U, outFile1U)
				} else {
					log.Infof("no unpaired reads in %s", read1)
				}

				if n2U > 0 {
					log.Infof("%d unpaired reads saved to %s", n2U, outFile2U)
				} else {
					log.Infof("no unpaired reads in %s", read2)
				}
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(pairCmd)

	pairCmd.Flags().StringP("read1", "1", "", "(gzipped) read1 file")
	pairCmd.Flags().StringP("read2", "2", "", "(gzipped) read2 file")
	pairCmd.Flags().StringP("out-dir", "O", "", "output directory")
	pairCmd.Flags().BoolP("force", "f", false, "overwrite output directory")
	pairCmd.Flags().BoolP("save-unpaired", "u", false, "save unpaired reads if there are")
}
