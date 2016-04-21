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
	"runtime"
	"strconv"
	"strings"

	"github.com/brentp/xopen"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/util/byteutil"
	"github.com/spf13/cobra"
)

// splitCmd represents the split command
var splitCmd = &cobra.Command{
	Use:   "split",
	Short: "split sequences into files by id/seq region/size/parts",
	Long: fmt.Sprintf(`split sequences into files by name ID, subsequence of given region,
part size or number of parts.

The definition of region is 1-based and with some custom design.

Examples:
%s
`, regionExample),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			checkError(fmt.Errorf("no more than one file needed (%d)", len(args)))
		}

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

		size := getFlagInt(cmd, "by-size")
		if size < 0 {
			checkError(fmt.Errorf("value of flag -s (--size) should be greater than 0: %d ", size))
		}
		part := getFlagInt(cmd, "by-part")
		if part < 0 {
			checkError(fmt.Errorf("value of flag -p (--part) should be greater than 0: %d ", part))
		}
		byID := getFlagBool(cmd, "by-id")
		region := getFlagString(cmd, "by-region")
		twoPass := getFlagBool(cmd, "two-pass")
		usingMD5 := getFlagBool(cmd, "md5")
		if usingMD5 && region == "" {
			checkError(fmt.Errorf("flag -m (--md5) must be used with flag -r (--region)"))
		}
		dryRun := getFlagBool(cmd, "dry-run")

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		file := files[0]
		var fileName, fileExt string
		if file == "-" {
			fileName, fileExt = "stdin", ".fastx"
		} else {
			fileName, fileExt = filepathTrimExtension(file)
		}

		var outfile string

		if size > 0 {
			if !quiet {
				log.Infof("split into %d seqs per file", size)
			}
			if twoPass {
				log.Warning("no need for two-pass, ignored")
			}

			i := 1
			records := []*fastx.Record{}

			fastxReader, err := fastx.NewReader(alphabet, file, bufferSize, chunkSize, idRegexp)
			checkError(err)

			for chunk := range fastxReader.Ch {
				checkError(chunk.Err)
				for _, record := range chunk.Data {
					records = append(records, record)
					if len(records) == size {
						outfile = fmt.Sprintf("%s.part_%03d%s", fileName, i, fileExt)
						writeSeqs(records, outfile, lineWidth, quiet, dryRun)
						i++
						records = []*fastx.Record{}
					}
				}
			}
			if len(records) > 0 {
				outfile = fmt.Sprintf("%s.part_%03d%s", fileName, i, fileExt)
				writeSeqs(records, outfile, lineWidth, quiet, dryRun)
			}
			return
		}

		if part > 0 {
			if !quiet {
				log.Infof("split into %d parts", part)
			}
			if twoPass {
				if xopen.IsStdin() {
					checkError(fmt.Errorf("2-pass mode (-2) will failed when reading from stdin. please disable flag: -2"))
				}
				// first pass, get seq number
				if !quiet {
					log.Info("first pass: get seq number")
				}
				names, err := fastx.GetSeqNames(file)
				checkError(err)

				if !quiet {
					log.Infof("seq number: %d", len(names))
				}

				n := len(names)
				if n%part > 0 {
					size = int(n/part) + 1
					if n%size == 0 {
						if !quiet {
							log.Infof("corrected: split into %d parts", n/size)
						}
					}
				} else {
					size = int(n / part)
				}

				if !quiet {
					log.Info("second pass: read and split")
				}
				i := 1
				records := []*fastx.Record{}
				fastxReader, err := fastx.NewReader(alphabet, file, bufferSize, chunkSize, idRegexp)
				checkError(err)
				for chunk := range fastxReader.Ch {
					checkError(chunk.Err)
					for _, record := range chunk.Data {
						records = append(records, record)
						if len(records) == size {
							outfile = fmt.Sprintf("%s.part_%03d%s", fileName, i, fileExt)
							writeSeqs(records, outfile, lineWidth, quiet, dryRun)
							i++
							records = []*fastx.Record{}
						}
					}
				}
				if len(records) > 0 {
					outfile = fmt.Sprintf("%s.part_%03d%s", fileName, i, fileExt)
					writeSeqs(records, outfile, lineWidth, quiet, dryRun)
				}
			} else {
				i := 1
				records := []*fastx.Record{}

				if !quiet {
					log.Info("read sequences ...")
				}
				allRecords, err := fastx.GetSeqs(file, alphabet, bufferSize, chunkSize, idRegexp)
				checkError(err)
				if !quiet {
					log.Infof("read %d sequences", len(allRecords))
				}

				n := len(allRecords)
				if n%part > 0 {
					size = int(n/part) + 1
					if n%size == 0 {
						if !quiet {
							log.Infof("corrected: split into %d parts", n/size)
						}
					}
				} else {
					size = int(n / part)
				}

				for _, record := range allRecords {
					records = append(records, record)
					if len(records) == size {
						outfile = fmt.Sprintf("%s.part_%03d%s", fileName, i, fileExt)
						writeSeqs(records, outfile, lineWidth, quiet, dryRun)
						i++
						records = []*fastx.Record{}
					}
				}
				if len(records) > 0 {
					outfile = fmt.Sprintf("%s.part_%03d%s", fileName, i, fileExt)
					writeSeqs(records, outfile, lineWidth, quiet, dryRun)
				}
			}
			return
		}

		if byID {
			if !quiet {
				log.Infof("split by ID. idRegexp: %s", idRegexp)
			}
			if twoPass {
				log.Warning("no need for two-pass, ignored")
			}
			if !quiet {
				log.Info("read sequences ...")
			}
			allRecords, err := fastx.GetSeqs(file, alphabet, bufferSize, chunkSize, idRegexp)
			checkError(err)
			if !quiet {
				log.Infof("read %d sequences", len(allRecords))
			}

			recordsByID := make(map[string][]*fastx.Record)

			var id string
			for _, record := range allRecords {
				id = string(record.ID)
				if _, ok := recordsByID[id]; !ok {
					recordsByID[id] = []*fastx.Record{}
				}
				recordsByID[id] = append(recordsByID[id], record)
			}

			var outfile string
			for id, records := range recordsByID {
				outfile = fmt.Sprintf("%s.id_%s%s", fileName, id, fileExt)
				writeSeqs(records, outfile, lineWidth, quiet, dryRun)
			}
			return
		}

		if region != "" {
			if !reRegion.MatchString(region) {
				checkError(fmt.Errorf(`invalid region: %s. type "fakit subseq -h" for more examples`, region))
			}
			r := strings.Split(region, ":")
			start, err := strconv.Atoi(r[0])
			checkError(err)
			end, err := strconv.Atoi(r[1])
			checkError(err)
			if start == 0 || end == 0 {
				checkError(fmt.Errorf("both start and end should not be 0"))
			}
			if start < 0 && end > 0 {
				checkError(fmt.Errorf("when start < 0, end should not > 0"))
			}

			if !quiet {
				log.Infof("split by region: %s", region)
			}
			if twoPass {
				log.Warning("no need for two-pass, ignored")
			}
			if !quiet {
				log.Info("read sequences ...")
			}
			allRecords, err := fastx.GetSeqs(file, alphabet, bufferSize, chunkSize, idRegexp)
			checkError(err)
			if !quiet {
				log.Infof("read %d sequences", len(allRecords))
			}

			recordsBySeqs := make(map[string][]*fastx.Record)

			var subseq string
			var s, e int
			for _, record := range allRecords {
				s, e = start, end
				if s > 0 {
					s--
				}
				if e < 0 {
					e++
				}

				if usingMD5 {
					subseq = string(MD5(byteutil.SubSlice(record.Seq.Seq, s, e)))
				} else {
					subseq = string(byteutil.SubSlice(record.Seq.Seq, s, e))
				}
				if _, ok := recordsBySeqs[subseq]; !ok {
					recordsBySeqs[subseq] = []*fastx.Record{}
				}
				recordsBySeqs[subseq] = append(recordsBySeqs[subseq], record)
			}

			var outfile string
			for subseq, records := range recordsBySeqs {
				outfile = fmt.Sprintf("%s.region_%d:%d_%s%s", fileName, start, end, subseq, fileExt)
				writeSeqs(records, outfile, lineWidth, quiet, dryRun)
			}
			return
		}

		checkError(fmt.Errorf(`one of flags should be given: -s/-p/-i/-r. type "fakit split -h" for help`))
	},
}

func init() {
	RootCmd.AddCommand(splitCmd)

	splitCmd.Flags().IntP("by-size", "s", 0, "split squences into multi parts with N sequences")
	splitCmd.Flags().IntP("by-part", "p", 0, "split squences into N parts")
	splitCmd.Flags().BoolP("by-id", "i", false, "split squences according to sequence ID")
	splitCmd.Flags().StringP("by-region", "r", "", "split squences according to subsequence of given region. "+
		`e.g 1:12 for first 12 bases, -12:-1 for last 12 bases. type "fakit split -h" for more examples`)
	splitCmd.Flags().BoolP("md5", "m", false, "use MD5 instead of region sequence in output file when using flag -r (--by-region)")
	splitCmd.Flags().BoolP("two-pass", "2", false, "2-pass mode read files twice to lower memory usage. Not allowed when reading from stdin")
	splitCmd.Flags().BoolP("dry-run", "d", false, "dry run, just print message and no files will be created.")
}
