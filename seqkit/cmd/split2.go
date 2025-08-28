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
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
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
	GroupID: "set",

	Use:   "split2",
	Short: "split sequences into files by size/parts (FASTA, PE/SE FASTQ)",
	Long: `split sequences into files by part size or number of parts

This command supports FASTA and paired- or single-end FASTQ with low memory
occupation and fast speed.

The prefix of output files:
  1. For stdin: stdin
  2. Others: same to the input file
  3. Set via the options: --by-length-prefix, --by-part-prefix, or --by-size-prefix
  4. Use the ID of the first sequence in each subset.
     E.g, 'seqkit split2 --by-size 1 --seqid-as-filename' is equal to
     'seqkit split --by-id', but it's much faster and uses less memory.

The extension of output files:
  1. For stdin: .fast[aq]
  2. Others: same to the input file
  3. Additional extension via the option -e/--extension, e.g.， outputting
     gzipped files for plain text input:
         seqkit split2 -p 2 -O test tests/hairpin.fa -e .gz


If you want to cut a sequence into multiple segments.
  1. For cutting into even chunks, please use 'kmcp utils split-genomes'
     (https://bioinf.shenwei.me/kmcp/usage/#split-genomes).
     E.g., cutting into 4 segments of equal size, with no overlap between adjacent segments:
        kmcp utils split-genomes -m 1 -k 1 --split-number 4 --split-overlap 0 input.fasta -O out
  2. For cutting into multiple chunks of fixed size, please using 'seqkit sliding'.
     E.g., cutting into segments of 40 bp and keeping the last segment which can be shorter than 40 bp.
        seqkit sliding -g -s 40 -W 40 input.fasta -o out.fasta

`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			checkError(fmt.Errorf("no more than one file allowed, using -1/--read1 and -2/--read2 for paired-end reads"))
		}

		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		// lineWidth := config.LineWidth
		outFile := config.OutFile
		quiet := config.Quiet
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		fai.MapWholeFile = false
		runtime.GOMAXPROCS(config.Threads)

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", !config.SkipFileCheck)

		if len(files) > 1 {
			checkError(fmt.Errorf("no more than one file should be given"))
		}

		read1 := getFlagString(cmd, "read1")
		read2 := getFlagString(cmd, "read2")

		size := getFlagNonNegativeInt(cmd, "by-size")
		parts := getFlagNonNegativeInt(cmd, "by-part")
		lengthS := getFlagString(cmd, "by-length")
		var length int64
		var err error
		if lengthS != "" {
			length, err = ParseByteSize(lengthS)
			if err != nil {
				checkError(fmt.Errorf("parsing sequence length: %s", lengthS))
			}
		}

		outdir := getFlagString(cmd, "out-dir")
		force := getFlagBool(cmd, "force")

		extension := getFlagString(cmd, "extension")

		reRead := regexp.MustCompile(`\{read\}`)

		prefixBySize := getFlagString(cmd, "by-size-prefix")
		prefixByPart := getFlagString(cmd, "by-part-prefix")
		prefixByLength := getFlagString(cmd, "by-length-prefix")

		prefixBySizeSet := cmd.Flags().Lookup("by-size-prefix").Changed
		prefixByPartSet := cmd.Flags().Lookup("by-part-prefix").Changed
		prefixByLengthSet := cmd.Flags().Lookup("by-length-prefix").Changed

		seqIDAsFileName := getFlagBool(cmd, "seqid-as-filename")

		if size == 0 && parts == 0 && length == 0 {
			checkError(fmt.Errorf(`one of flags should be given: -s/-p/-l. type "seqkit split2 -h" for help`))
		}

		bySize := size > 0
		byParts := parts > 0
		byLength := length > 0

		if parts >= 1000 {
			log.Warningf(`value of -p/--parts > 1000 may cause error of "too many open files"`)
		}

		var source string
		var pairedEnd bool
		if read1 == "" {
			if read2 == "" {
				// single end from file or stdin
				if isStdin(files[0]) {
					if outdir == "" {
						outdir = "stdin.split"
					}
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

				pairedEnd = true
				if prefixBySizeSet && !reRead.MatchString(prefixBySize) {
					checkError(fmt.Errorf(`--by-size-prefix should contains the placeholder "{read}" when paired-end files are given, such as "sample_{read}.fq.gz`))
				}
				if prefixByPartSet && !reRead.MatchString(prefixByPart) {
					checkError(fmt.Errorf(`--by-part-prefix should contains the placeholder "{read}" when paired-end files are given, such as "sample_{read}.fq.gz`))
				}
				if prefixByLengthSet && !reRead.MatchString(prefixByLength) {
					checkError(fmt.Errorf(`--by-size-prefix should contains the placeholder "{read}" when paired-end files are given, such as "sample_{read}.fq.gz`))
				}
			}
		}

		if pairedEnd && seqIDAsFileName {
			checkError(fmt.Errorf("the flag -N/--seqid-as-filename is not applicable for paired-end reads"))
		}

		if !quiet {
			log.Infof("split seqs from %s", source)
			if bySize {
				log.Infof("split into %d seqs per file", size)
			} else if byParts {
				log.Infof("split into %d parts", parts)
			} else {
				log.Infof("split by sequence length: %s", lengthS)
			}
		}

		var wg sync.WaitGroup
		for i, file := range files {
			isstdin := isStdin(file)
			var fileName, fileExt, fileExt2 string
			if isstdin {
				fileName, fileExt = "stdin", extension
				if outdir == "" {
					outdir = "stdin.split"
				}
			} else {
				fileName, fileExt, fileExt2 = filepathTrimExtension2(file, nil)
				if extension != "" {
					fileExt += extension
				} else {
					fileExt += fileExt2
				}

				if outdir == "" {
					outdir = file + ".split"
				}
			}

			if outFile != "-" {
				fileName = filepath.Base(outFile)
			}

			pwd, _ := os.Getwd()
			if outdir != "./" && outdir != "." && pwd != filepath.Clean(outdir) {
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

			wg.Add(1)
			go func(file string, pairedEnd bool, r int) {
				defer wg.Done()

				renameFileExt := true
				var record *fastx.Record
				var err error

				var outfhs []*xopen.Writer
				var counts []int
				var outfiles []string

				// by size or by length
				var outfhPre *xopen.Writer
				var prefix string
				var outfilePre string

				var flag bool

				if bySize || byLength { // by size or by length
				} else if byParts { // by part
					outfhs = make([]*xopen.Writer, 0, parts)
					counts = make([]int, 0, parts)
					outfiles = make([]string, 0, parts)
				} else {
					checkError(fmt.Errorf(`one of flags should be given: -s/-p/-l. type "seqkit split2 -h" for help`))
				}

				fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
				checkError(err)
				i := 0 // nth part
				j := 0
				var n int64 // length sum
				once := true
				for {
					record, err = fastxReader.Read()
					if err != nil {
						if err == io.EOF {
							break
						}
						checkError(err)
						break
					}

					if once {
						if fastxReader.IsFastq {
							config.LineWidth = 0
							fastx.ForcelyOutputFastq = true
						}

						if renameFileExt && isstdin {
							if len(record.Seq.Qual) > 0 {
								fileExt = suffixFQ + extension
							} else {
								fileExt = suffixFA + extension
							}
							renameFileExt = false
						}
						once = false
					}

					n += int64(len(record.Seq.Seq))

					if bySize {
						if j == size {
							outfhPre.Close()
							if !quiet {
								log.Infof("write %d sequences to file: %s\n", j, outfilePre)
							}

							i++

							if !seqIDAsFileName {
								if prefixBySizeSet {
									prefix = prefixBySize
									if pairedEnd {
										prefix = reRead.ReplaceAllString(prefix, strconv.Itoa(r))
									}
								} else {
									prefix = fmt.Sprintf("%s.part_", filepath.Base(fileName))
								}
								outfilePre = filepath.Join(outdir, fmt.Sprintf("%s%03d%s", prefix, i+1, fileExt))
							} else {
								outfilePre = filepath.Join(outdir, fmt.Sprintf("%s%s", pathutil.RemoveInvalidPathChars(string(record.ID), "__"), fileExt))
							}
							outfhPre, err = xopen.Wopen(outfilePre)
							checkError(err)

							j = 0
						}
					} else if byLength {
						flag = false

						if outfhPre == nil { // first record
							var outfh2 *xopen.Writer
							var outfile string
							if !seqIDAsFileName {
								if prefixByLengthSet {
									prefix = prefixByLength
									if pairedEnd {
										prefix = reRead.ReplaceAllString(prefix, strconv.Itoa(r))
									}
								} else {
									prefix = fmt.Sprintf("%s.part_", filepath.Base(fileName))
								}
								outfile = filepath.Join(outdir, fmt.Sprintf("%s%03d%s", prefix, i+1, fileExt))
							} else {
								outfile = filepath.Join(outdir, fmt.Sprintf("%s%s", pathutil.RemoveInvalidPathChars(string(record.ID), "__"), fileExt))
							}
							outfh2, err = xopen.Wopen(outfile)
							checkError(err)

							outfhPre = outfh2
							outfilePre = outfile

							j = 0
						}

						if n >= length {
							record.FormatToWriter(outfhPre, config.LineWidth)
							j++

							outfhPre.Close()
							if !quiet {
								log.Infof("write %d sequences to file: %s\n", j, outfilePre)
							}
							i++

							var outfh2 *xopen.Writer
							var outfile string
							if !seqIDAsFileName {
								if prefixByLengthSet {
									prefix = prefixByLength
									if pairedEnd {
										prefix = reRead.ReplaceAllString(prefix, strconv.Itoa(r))
									}
								} else {
									prefix = fmt.Sprintf("%s.part_", filepath.Base(fileName))
								}
								outfile = filepath.Join(outdir, fmt.Sprintf("%s%03d%s", prefix, i+1, fileExt))
							} else {
								outfile = filepath.Join(outdir, fmt.Sprintf("%s%s", pathutil.RemoveInvalidPathChars(string(record.ID), "__"), fileExt))
							}
							outfh2, err = xopen.Wopen(outfile)
							checkError(err)

							outfhPre = outfh2
							outfilePre = outfile

							j = 0
							n = 0
							flag = false
						} else { // write this record later
							flag = true
						}
					}

					if bySize {
						// first record, for bySize
						if outfhPre == nil {
							if !seqIDAsFileName {
								if prefixBySizeSet {
									prefix = prefixBySize
									if pairedEnd {
										prefix = reRead.ReplaceAllString(prefix, strconv.Itoa(r))
									}
								} else {
									prefix = fmt.Sprintf("%s.part_", filepath.Base(fileName))
								}
								outfilePre = filepath.Join(outdir, fmt.Sprintf("%s%03d%s", prefix, i+1, fileExt))
							} else {
								outfilePre = filepath.Join(outdir, fmt.Sprintf("%s%s", pathutil.RemoveInvalidPathChars(string(record.ID), "__"), fileExt))
							}
							outfhPre, err = xopen.Wopen(outfilePre)
							checkError(err)

							j = 0
						}

						record.FormatToWriter(outfhPre, config.LineWidth)

						j++ // increase size
					} else if byLength {
						if flag {
							record.FormatToWriter(outfhPre, config.LineWidth)

							j++
						}
					} else {
						// first record, for byParts
						if i+1 > len(outfhs) {
							var outfh2 *xopen.Writer
							var outfile string
							if !seqIDAsFileName {
								if prefixByLengthSet {
									prefix = prefixByLength
									if pairedEnd {
										prefix = reRead.ReplaceAllString(prefix, strconv.Itoa(r))
									}
								} else {
									prefix = fmt.Sprintf("%s.part_", filepath.Base(fileName))
								}
								outfile = filepath.Join(outdir, fmt.Sprintf("%s%03d%s", prefix, i+1, fileExt))
							} else {
								outfile = filepath.Join(outdir, fmt.Sprintf("%s%s", pathutil.RemoveInvalidPathChars(string(record.ID), "__"), fileExt))
							}
							outfh2, err = xopen.Wopen(outfile)
							checkError(err)

							outfhs = append(outfhs, outfh2)
							counts = append(counts, 0)
							outfiles = append(outfiles, outfile)
						}

						record.FormatToWriter(outfhs[i], config.LineWidth)
						counts[i]++

						i++
						if i == parts { // reset index
							i = 0
						}
					}
				}
				fastxReader.Close()

				if byParts {
					for i, outfh := range outfhs {
						outfh.Close()

						if !quiet {
							if counts[i] == 0 {

							} else {
								log.Infof("write %d sequences to file: %s\n", counts[i], outfiles[i])
							}
						}
					}
				} else {
					if outfhPre != nil {
						outfhPre.Close()
					}

					if j == 0 {
						os.Remove(outfilePre)
					} else {
						log.Infof("write %d sequences to file: %s\n", j, outfilePre)
					}
				}

			}(file, pairedEnd, i+1)
		}

		wg.Wait()
	},
}

func init() {
	RootCmd.AddCommand(split2Cmd)

	split2Cmd.Flags().StringP("read1", "1", "", "(gzipped) read1 file")
	split2Cmd.Flags().StringP("read2", "2", "", "(gzipped) read2 file")
	split2Cmd.Flags().IntP("by-size", "s", 0, "split sequences into multi parts with N sequences")
	split2Cmd.Flags().IntP("by-part", "p", 0, "split sequences into N parts with the round robin distribution")
	split2Cmd.Flags().StringP("by-length", "l", "", "split sequences into chunks of >=N bases, supports K/M/G suffix")
	split2Cmd.Flags().StringP("out-dir", "O", "", "output directory (default value is $infile.split)")
	split2Cmd.Flags().BoolP("force", "f", false, "overwrite output directory")

	split2Cmd.Flags().StringP("by-size-prefix", "", "", `file prefix for --by-size. The placeholder "{read}" is needed for paired-end files.`)
	split2Cmd.Flags().StringP("by-part-prefix", "", "", `file prefix for --by-part. The placeholder "{read}" is needed for paired-end files.`)
	split2Cmd.Flags().StringP("by-length-prefix", "", "", `file prefix for --by-length. The placeholder "{read}" is needed for paired-end files.`)

	split2Cmd.Flags().BoolP("seqid-as-filename", "N", false, "use the first sequence ID as the file name. E.g., using '-N -s 1' is equal to 'seqkit split --by-id' but much faster and uses less memory.")

	split2Cmd.Flags().StringP("extension", "e", "", `set output file extension, e.g., ".gz", ".xz", or ".zst"`)
}
