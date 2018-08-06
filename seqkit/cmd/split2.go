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
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fai"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/util/pathutil"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// split2Cmd represents the split command
var split2Cmd = &cobra.Command{
	Use:   "split2",
	Short: "split sequences into files by size/parts (FASTA, PE/SE FASTQ)",
	Long: `split sequences into files by part size or number of parts

This command supports FASTA and paired- or single-end FASTQ with low memory
occupation and fast speed.

The file extensions of output are automatically detected and created
accorting to the input files.

`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			checkError(fmt.Errorf("no more than one file allowed, using -1/--read1 and -2/--read2 for paired-end reads"))
		}

		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		// lineWidth := config.LineWidth
		quiet := config.Quiet
		seq.AlphabetGuessSeqLenghtThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		fai.MapWholeFile = false
		runtime.GOMAXPROCS(config.Threads)

		files := getFileList(args)

		if len(files) > 1 {
			checkError(fmt.Errorf("no more than one file should be given"))
		}

		read1 := getFlagString(cmd, "read1")
		read2 := getFlagString(cmd, "read2")

		size := getFlagNonNegativeInt(cmd, "by-size")
		parts := getFlagNonNegativeInt(cmd, "by-part")

		outdir := getFlagString(cmd, "out-dir")
		force := getFlagBool(cmd, "force")

		if size == 0 && parts == 0 {
			checkError(fmt.Errorf(`one of flags should be given: -s/-p. type "seqkit split2 -h" for help`))
		}

		if parts >= 1000 {
			log.Warningf(`value of -p/--parts > 1000 may cause error of "too many open files"`)
		}

		var source string
		if read1 == "" {
			if read2 == "" {
				// single end from file or stdin
				if isStdin(files[0]) {
					outdir = "stdin.split"
					source = "stdin"
				} else {
					source = files[0]
				}
			} else {
				if !quiet {
					log.Infof("flag -2/--read2 given, ignore: %s", strings.Join(files, ", "))
				}
				files = []string{read2}
				source = read2
			}
		} else {
			if read2 == "" {
				if !quiet {
					log.Infof("flag -1/--read1 given, ignore: %s", strings.Join(files, ", "))
				}
				files = []string{read1}
				source = read1
			} else {
				if !quiet {
					log.Infof("flag -1/--read1 and -2/--read2 given, ignore: %s", strings.Join(files, ", "))
				}
				files = []string{read1, read2}
				source = read1 + " and " + read2
			}
		}

		if !quiet {
			log.Infof("split seqs from %s", source)
			if size > 0 {
				log.Infof("split into %d seqs per file", size)
			} else {
				log.Infof("split into %d parts", parts)
			}
		}

		var wg sync.WaitGroup
		for _, file := range files {
			isstdin := isStdin(file)
			var fileName, fileExt string
			if isstdin {
				fileName, fileExt = "stdin", ".fastx"
				outdir = "stdin.split"
			} else {
				fileName, fileExt = filepathTrimExtension(file)
				if outdir == "" {
					outdir = file + ".split"
				}
			}

			existed, err := pathutil.DirExists(outdir)
			checkError(err)
			if existed {
				empty, err := pathutil.IsEmpty(outdir)
				checkError(err)
				if !empty {
					if force {
						checkError(os.RemoveAll(outdir))
					} else {
						checkError(fmt.Errorf("outdir not empty: %s, use -f (--force) to overwrite", outdir))
					}
				} else {
					checkError(os.RemoveAll(outdir))
				}
			}
			checkError(os.MkdirAll(outdir, 0777))

			wg.Add(1)
			go func(file string) {
				defer wg.Done()

				renameFileExt := true
				var record *fastx.Record
				var fastxReader *fastx.Reader
				var err error

				var outfhs map[int]*xopen.Writer
				var counts map[int]int
				var outfiles map[int]string

				if size > 0 {
					outfhs = make(map[int]*xopen.Writer, 10)
					outfhs[0] = nil
					counts = make(map[int]int, 10)
					outfiles = make(map[int]string, 10)
				} else {
					outfhs = make(map[int]*xopen.Writer, parts)
					counts = make(map[int]int, parts)
					outfiles = make(map[int]string, parts)
					for i := 0; i < parts; i++ {
						outfhs[i] = nil
						counts[i] = 0
					}
				}

				var outfh *xopen.Writer
				fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
				checkError(err)
				i := 0 // nth part
				j := 0 // nth record
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

					if renameFileExt && isstdin {
						if len(record.Seq.Qual) > 0 {
							fileExt = suffixFQ
						} else {
							fileExt = suffixFA
						}
						renameFileExt = false
					}

					if size > 0 {
						if j == size {
							outfhs[i].Close()
							log.Infof("write %d sequences to file: %s\n", counts[i], outfiles[i])
							delete(outfhs, i)

							i++
							var outfh2 *xopen.Writer
							outfile := filepath.Join(outdir, fmt.Sprintf("%s.part_%03d%s", filepath.Base(fileName), i+1, fileExt))
							outfh2, err = xopen.Wopen(outfile)
							checkError(err)
							outfhs[i] = outfh2
							outfiles[i] = outfile

							j = 0
						}
					}

					if outfhs[i] == nil {
						var outfh2 *xopen.Writer
						outfile := filepath.Join(outdir, fmt.Sprintf("%s.part_%03d%s", filepath.Base(fileName), i+1, fileExt))
						outfh2, err = xopen.Wopen(outfile)
						checkError(err)
						outfhs[i] = outfh2
						outfiles[i] = outfile
					}
					outfh = outfhs[i]

					record.FormatToWriter(outfh, config.LineWidth)

					counts[i]++

					if size > 0 {
						j++
					} else {
						i++
						if i == parts {
							i = 0
						}
					}
				}

				sortedI := make([]int, len(outfhs))
				k := 0
				for i = range outfhs {
					sortedI[k] = i
					k++
				}
				sort.Ints(sortedI)

				for _, i := range sortedI {
					if outfhs[i] != nil {
						outfhs[i].Close()
					}

					log.Infof("write %d sequences to file: %s\n", counts[i], outfiles[i])
				}

			}(file)
		}

		wg.Wait()
	},
}

func init() {
	RootCmd.AddCommand(split2Cmd)

	split2Cmd.Flags().StringP("read1", "1", "", "read1 file")
	split2Cmd.Flags().StringP("read2", "2", "", "read2 file")
	split2Cmd.Flags().IntP("by-size", "s", 0, "split sequences into multi parts with N sequences")
	split2Cmd.Flags().IntP("by-part", "p", 0, "split sequences into N parts")
	split2Cmd.Flags().StringP("out-dir", "O", "", "output directory (default value is $infile.split)")
	split2Cmd.Flags().BoolP("force", "f", false, "overwrite output directory")
}
