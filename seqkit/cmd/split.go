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

// splitCmd represents the split command
var splitCmd = &cobra.Command{
	GroupID: "set",

	Use:   "split",
	Short: "split sequences into files by id/seq region/size/parts (mainly for FASTA)",
	Long: fmt.Sprintf(`split sequences into files by name ID, subsequence of given region,
part size or number of parts.

If you just want to split by parts or sizes, please use "seqkit split2",
which can apply to paired- and single-end FASTQ.

If you want to cut a sequence into multiple segments.
  1. For cutting into even chunks, please use 'kmcp utils split-genomes'
     (https://bioinf.shenwei.me/kmcp/usage/#split-genomes).
     E.g., cutting into 4 segments of equal size, with no overlap between adjacent segments:
        kmcp utils split-genomes -m 1 -k 1 --split-number 4 --split-overlap 0 input.fasta -O out
  2. For cutting into multiple chunks of fixed size, please using 'seqkit sliding'.
     E.g., cutting into segments of 40 bp and keeping the last segment which can be shorter than 40 bp.
        seqkit sliding -g -s 40 -W 40 input.fasta -o out.fasta

Attention:
  1. For the two-pass mode (-2/--two-pass), The flag -U/--update-faidx is recommended to
     ensure the .fai file matches the FASTA file.
  2. For splitting by sequence IDs in Windows/MacOS, where the file systems might be case-insensitive,
     output files might be overwritten if they are only different in cases, like Abc and ABC.

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
		// lineWidth := config.LineWidth
		quiet := config.Quiet
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		fai.MapWholeFile = false
		runtime.GOMAXPROCS(config.Threads)

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		if len(files) > 1 {
			checkError(fmt.Errorf("no more than one file should be given"))
		}

		size := getFlagNonNegativeInt(cmd, "by-size")
		part := getFlagNonNegativeInt(cmd, "by-part")

		byID := getFlagBool(cmd, "by-id")
		ignoreCase := getFlagBool(cmd, "ignore-case")
		region := getFlagString(cmd, "by-region")
		twoPass := getFlagBool(cmd, "two-pass")
		updateFaidx := getFlagBool(cmd, "update-faidx")
		keepTemp := getFlagBool(cmd, "keep-temp")
		if keepTemp && !twoPass {
			checkError(fmt.Errorf("flag -k (--keep-temp) must be used with flag -2 (--two-pass)"))
		}
		if updateFaidx && !twoPass {
			checkError(fmt.Errorf("flag -U (--update-faidx) must be used with flag -2 (--two-pass)"))
		}

		dryRun := getFlagBool(cmd, "dry-run")

		outdir := getFlagString(cmd, "out-dir")
		force := getFlagBool(cmd, "force")

		extension := getFlagString(cmd, "extension")

		prefixBySize := getFlagString(cmd, "by-size-prefix")
		prefixByPart := getFlagString(cmd, "by-part-prefix")
		prefixByID := getFlagString(cmd, "by-id-prefix")
		prefixByRegion := getFlagString(cmd, "by-region-prefix")

		prefixBySizeSet := cmd.Flags().Lookup("by-size-prefix").Changed
		prefixByPartSet := cmd.Flags().Lookup("by-part-prefix").Changed
		prefixByIDSet := cmd.Flags().Lookup("by-id-prefix").Changed
		prefixByRegionSet := cmd.Flags().Lookup("by-region-prefix").Changed

		file := files[0]
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

		renameFileExt := true
		var outfile string
		var prefix string
		var record *fastx.Record

		if !dryRun {
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
		}

		var outfh *xopen.Writer
		var err error

		if size > 0 {
			if !twoPass {
				if !quiet {
					log.Infof("split into %d seqs per file", size)
				}

				i := 1
				records := []*fastx.Record{}

				fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
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
					records = append(records, record.Clone())
					if len(records) == size {
						if prefixBySizeSet {
							prefix = prefixBySize
						} else {
							prefix = fmt.Sprintf("%s.part_", filepath.Base(fileName))
						}
						outfile = filepath.Join(outdir, fmt.Sprintf("%s%03d%s", prefix, i, fileExt))
						writeSeqs(records, outfile, config.LineWidth, quiet, dryRun)
						i++
						records = []*fastx.Record{}
					}
				}
				fastxReader.Close()
				if len(records) > 0 {
					// outfile = filepath.Join(outdir, fmt.Sprintf("%s.part_%03d%s", filepath.Base(fileName), i, fileExt))
					if prefixBySizeSet {
						prefix = prefixBySize
					} else {
						prefix = fmt.Sprintf("%s.part_", filepath.Base(fileName))
					}
					outfile = filepath.Join(outdir, fmt.Sprintf("%s%03d%s", prefix, i, fileExt))
					writeSeqs(records, outfile, config.LineWidth, quiet, dryRun)
				}

				return
			}

			var alphabet2 *seq.Alphabet

			newFile := file

			if isstdin || !isPlainFile(file) {
				if isstdin {
					newFile = "stdin" + ".fastx"
				} else {
					newFile = file + ".fastx"
				}
				if !quiet {
					log.Infof("read and write sequences to temporary file: %s ...", newFile)
				}

				var nseqs int
				nseqs, err = copySeqs(file, newFile)
				checkError(err)
				if !quiet {
					log.Infof("%d sequences saved", nseqs)
				}

				var isFastq bool
				var err error
				alphabet2, isFastq, err = fastx.GuessAlphabet(newFile)
				checkError(err)
				if renameFileExt && isstdin {
					if isFastq {
						fileExt = suffixFQ + extension
					} else {
						fileExt = suffixFA + extension
					}
					renameFileExt = false
				}
				if isFastq {
					checkError(os.Remove(newFile))
					checkError(fmt.Errorf("Sorry, two-pass mode does not support FASTQ format"))
				}
			}
			fileExt = suffixFA + extension

			fileFai := newFile + ".seqkit.fai"

			if FileExists(fileFai) && updateFaidx {
				checkError(os.RemoveAll(fileFai))
				if !quiet {
					log.Infof("delete the old FASTA index file: %s", fileFai)
				}
			}

			if !quiet {
				log.Infof("create or read FASTA index ...")
			}
			faidx := getFaidx(newFile, `^(.+)$`, quiet)
			defer func() {
				checkError(faidx.Close())
			}()

			if len(faidx.Index) == 0 {
				log.Warningf("  0 records loaded from %s, please check if it matches the fasta file, or switch on the flag -U/--update-faidx", fileFai)
				return
			} else if !quiet {
				log.Infof("  %d records loaded from %s", len(faidx.Index), fileFai)
			}

			IDs, _, err := getSeqIDAndLengthFromFaidxFile(fileFai)
			checkError(err)

			n := 1
			if len(IDs) > 0 {
				// outfile = filepath.Join(outdir, fmt.Sprintf("%s.part_%03d%s", filepath.Base(fileName), n, fileExt))
				if prefixBySizeSet {
					prefix = prefixBySize
				} else {
					prefix = fmt.Sprintf("%s.part_", filepath.Base(fileName))
				}
				outfile = filepath.Join(outdir, fmt.Sprintf("%s%03d%s", prefix, n, fileExt))
				if !dryRun {
					outfh, err = xopen.Wopen(outfile)
					checkError(err)
				}
			}
			j := 0
			var record *fastx.Record
			for _, chr := range IDs {
				if !dryRun {
					r, ok := faidx.Index[chr]
					if !ok {
						checkError(fmt.Errorf(`sequence (%s) not found in file: %s`, chr, newFile))
					}

					sequence := subseqByFaix(faidx, chr, r, 1, -1)
					record, err = fastx.NewRecord(alphabet2, []byte(chr), []byte(chr), []byte{}, sequence)
					checkError(err)

					record.FormatToWriter(outfh, config.LineWidth)
				}
				j++
				if j == size {
					if !quiet {
						log.Infof("write %d sequences to file: %s\n", j, outfile)
					}
					n++
					// outfile = filepath.Join(outdir, fmt.Sprintf("%s.part_%03d%s", filepath.Base(fileName), n, fileExt))
					if prefixBySizeSet {
						prefix = prefixBySize
					} else {
						prefix = fmt.Sprintf("%s.part_", filepath.Base(fileName))
					}
					outfile = filepath.Join(outdir, fmt.Sprintf("%s%03d%s", prefix, n, fileExt))
					if !dryRun {
						outfh.Close()
						outfh, err = xopen.Wopen(outfile)
						checkError(err)
					}
					j = 0
				}
			}
			if j > 0 && !quiet {
				log.Infof("write %d sequences to file: %s\n", j, outfile)
			}
			if j == 0 {
				os.Remove(outfile)
			}
			if !dryRun {
				outfh.Close()
			}

			if (isstdin || !isPlainFile(file)) && !keepTemp {
				checkError(os.Remove(newFile))
				checkError(os.Remove(newFile + ".seqkit.fai"))
			}
			return
		}

		if part > 0 {
			if !quiet {
				log.Infof("split into %d parts", part)
			}
			if !twoPass {
				i := 1
				records := []*fastx.Record{}

				if !quiet {
					log.Info("read sequences ...")
				}
				allRecords, err := fastx.GetSeqs(file, alphabet, config.Threads, 10, idRegexp)
				checkError(err)
				if !quiet {
					log.Infof("read %d sequences", len(allRecords))
				}

				if len(allRecords) > 0 && len(allRecords[0].Seq.Qual) > 0 {
					config.LineWidth = 0
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
					if renameFileExt && isstdin {
						if len(record.Seq.Qual) > 0 {
							fileExt = suffixFQ + extension
						} else {
							fileExt = suffixFA + extension
						}
						renameFileExt = false
					}
					records = append(records, record)
					if len(records) == size {
						// outfile = filepath.Join(outdir, fmt.Sprintf("%s.part_%03d%s", filepath.Base(fileName), i, fileExt))
						if prefixByPartSet {
							prefix = prefixByPart
						} else {
							prefix = fmt.Sprintf("%s.part_", filepath.Base(fileName))
						}
						outfile = filepath.Join(outdir, fmt.Sprintf("%s%03d%s", prefix, i, fileExt))
						writeSeqs(records, outfile, config.LineWidth, quiet, dryRun)
						i++
						records = []*fastx.Record{}
					}
				}
				if len(records) > 0 {
					// outfile = filepath.Join(outdir, fmt.Sprintf("%s.part_%03d%s", filepath.Base(fileName), i, fileExt))
					if prefixByPartSet {
						prefix = prefixByPart
					} else {
						prefix = fmt.Sprintf("%s.part_", filepath.Base(fileName))
					}
					outfile = filepath.Join(outdir, fmt.Sprintf("%s%03d%s", prefix, i, fileExt))
					writeSeqs(records, outfile, config.LineWidth, quiet, dryRun)
				}
				return
			}

			var alphabet2 *seq.Alphabet

			newFile := file

			if isstdin || !isPlainFile(file) {
				if isstdin {
					newFile = "stdin" + ".fastx"
				} else {
					newFile = file + ".fastx"
				}
				if !quiet {
					log.Infof("read and write sequences to temporary file: %s ...", newFile)
				}

				var nseqs int
				nseqs, err = copySeqs(file, newFile)
				checkError(err)
				if !quiet {
					log.Infof("%d sequences saved", nseqs)
				}

				var isFastq bool
				var err error
				alphabet2, isFastq, err = fastx.GuessAlphabet(newFile)
				checkError(err)
				if renameFileExt && isstdin {
					if isFastq {
						fileExt = suffixFQ + extension
					} else {
						fileExt = suffixFA + extension
					}
					renameFileExt = false
				}
				if isFastq {
					checkError(os.Remove(newFile))
					checkError(fmt.Errorf("Sorry, two-pass mode does not support FASTQ format"))
				}
			}
			fileExt = suffixFA + extension

			fileFai := newFile + ".seqkit.fai"

			if FileExists(fileFai) && updateFaidx {
				checkError(os.RemoveAll(fileFai))
				if !quiet {
					log.Infof("delete the old FASTA index file: %s", fileFai)
				}
			}

			if !quiet {
				log.Infof("create or read FASTA index ...")
			}
			faidx := getFaidx(newFile, `^(.+)$`, quiet)
			defer func() {
				checkError(faidx.Close())
			}()

			if len(faidx.Index) == 0 {
				log.Warningf("  0 records loaded from %s, please check if it matches the fasta file, or switch on the flag -U/--update-faidx", fileFai)
				return
			} else if !quiet {
				log.Infof("  %d records loaded from %s", len(faidx.Index), fileFai)
			}

			IDs, _, err := getSeqIDAndLengthFromFaidxFile(fileFai)
			checkError(err)

			N := len(IDs)
			if N%part > 0 {
				size = int(N/part) + 1
				if N%size == 0 {
					if !quiet {
						log.Infof("corrected: split into %d parts", N/size)
					}
				}
			} else {
				size = int(N / part)
			}

			n := 1
			if len(IDs) > 0 {
				// outfile = filepath.Join(outdir, fmt.Sprintf("%s.part_%03d%s", filepath.Base(fileName), n, fileExt))
				if prefixByPartSet {
					prefix = prefixByPart
				} else {
					prefix = fmt.Sprintf("%s.part_", filepath.Base(fileName))
				}
				outfile = filepath.Join(outdir, fmt.Sprintf("%s%03d%s", prefix, n, fileExt))
				if !dryRun {
					outfh, err = xopen.Wopen(outfile)
					checkError(err)
				}
			}
			j := 0
			var record *fastx.Record
			for _, chr := range IDs {
				if !dryRun {
					r, ok := faidx.Index[chr]
					if !ok {
						checkError(fmt.Errorf(`sequence (%s) not found in file: %s`, chr, newFile))
					}

					sequence := subseqByFaix(faidx, chr, r, 1, -1)
					record, err = fastx.NewRecord(alphabet2, []byte(chr), []byte(chr), []byte{}, sequence)
					checkError(err)

					record.FormatToWriter(outfh, config.LineWidth)
				}
				j++
				if j == size {
					if !quiet {
						log.Infof("write %d sequences to file: %s\n", j, outfile)
					}
					n++
					// outfile = filepath.Join(outdir, fmt.Sprintf("%s.part_%03d%s", filepath.Base(fileName), n, fileExt))
					if prefixByPartSet {
						prefix = prefixByPart
					} else {
						prefix = fmt.Sprintf("%s.part_", filepath.Base(fileName))
					}
					outfile = filepath.Join(outdir, fmt.Sprintf("%s%03d%s", prefix, n, fileExt))
					if !dryRun {
						outfh.Close()
						outfh, err = xopen.Wopen(outfile)
						checkError(err)
					}
					j = 0
				}
			}
			if j > 0 {
				if quiet {
					log.Infof("write %d sequences to file: %s\n", j, outfile)
				}
				if !dryRun {
					outfh.Close()
				}
			} else {
				if !dryRun {
					checkError(os.Remove(outfile))
				}
			}

			if (isstdin || !isPlainFile(file)) && !keepTemp {
				checkError(os.Remove(newFile))
				checkError(os.Remove(newFile + ".seqkit.fai"))
			}
			return
		}

		if byID {
			if !quiet {
				log.Infof("split by ID. idRegexp: %s", idRegexp)
			}

			if !twoPass {
				if !quiet {
					log.Info("read sequences ...")
				}
				allRecords, err := fastx.GetSeqs(file, alphabet, config.Threads, 10, idRegexp)
				checkError(err)
				if !quiet {
					log.Infof("read %d sequences", len(allRecords))
				}

				if len(allRecords) > 0 && len(allRecords[0].Seq.Qual) > 0 {
					config.LineWidth = 0
				}

				recordsByID := make(map[string][]*fastx.Record)

				var id string
				for _, record := range allRecords {
					if renameFileExt && isstdin {
						if len(record.Seq.Qual) > 0 {
							fileExt = suffixFQ + extension
						} else {
							fileExt = suffixFA + extension
						}
						renameFileExt = false
					}
					id = string(record.ID)
					if ignoreCase {
						id = strings.ToLower(id)
					}
					if _, ok := recordsByID[id]; !ok {
						recordsByID[id] = []*fastx.Record{}
					}
					recordsByID[id] = append(recordsByID[id], record)
				}

				var outfile string
				for id, records := range recordsByID {
					// outfile = filepath.Join(outdir, fmt.Sprintf("%s.id_%s%s",
					// 	filepath.Base(fileName),
					// 	pathutil.RemoveInvalidPathChars(id, "__"), fileExt))
					if prefixByIDSet {
						prefix = prefixByID
					} else {
						prefix = fmt.Sprintf("%s.part_", filepath.Base(fileName))
					}
					outfile = filepath.Join(outdir, fmt.Sprintf("%s%s%s",
						prefix,
						pathutil.RemoveInvalidPathChars(id, "__"), fileExt))
					writeSeqs(records, outfile, config.LineWidth, quiet, dryRun)
				}
				return
			}

			var alphabet2 *seq.Alphabet

			newFile := file

			if isstdin || !isPlainFile(file) {
				if isstdin {
					newFile = "stdin" + ".fastx"
				} else {
					newFile = file + ".fastx"
				}
				if !quiet {
					log.Infof("read and write sequences to temporary file: %s ...", newFile)
				}

				var nseqs int
				nseqs, err = copySeqs(file, newFile)
				checkError(err)
				if !quiet {
					log.Infof("%d sequences saved", nseqs)
				}

				var isFastq bool
				var err error
				alphabet2, isFastq, err = fastx.GuessAlphabet(newFile)
				checkError(err)
				if renameFileExt && isstdin {
					if isFastq {
						fileExt = suffixFQ + extension
					} else {
						fileExt = suffixFA + extension
					}
					renameFileExt = false
				}
				if isFastq {
					checkError(os.Remove(newFile))
					checkError(fmt.Errorf("sorry, two-pass mode does not support FASTQ format"))
				}
			}
			fileExt = suffixFA + extension

			fileFai := newFile + ".seqkit.fai"

			if FileExists(fileFai) && updateFaidx {
				checkError(os.RemoveAll(fileFai))
				if !quiet {
					log.Infof("delete the old FASTA index file: %s", fileFai)
				}
			}

			if !quiet {
				log.Infof("create or read FASTA index ...")
			}
			faidx := getFaidx(newFile, `^(.+)$`, quiet)
			defer func() {
				checkError(faidx.Close())
			}()

			if len(faidx.Index) == 0 {
				log.Warningf("  0 records loaded from %s, please check if it matches the fasta file, or switch on the flag -U/--update-faidx", fileFai)
				return
			} else if !quiet {
				log.Infof("  %d records loaded from %s", len(faidx.Index), fileFai)
			}

			IDs, _, err := getSeqIDAndLengthFromFaidxFile(fileFai)
			checkError(err)

			idRe, err := regexp.Compile(idRegexp)
			if err != nil {
				checkError(fmt.Errorf("fail to compile regexp: %s", idRegexp))
			}

			idsMap := make(map[string][]string)
			for _, ID := range IDs {
				id := string(fastx.ParseHeadID(idRe, []byte(ID)))
				if ignoreCase {
					id = strings.ToLower(id)
				}
				if _, ok := idsMap[id]; !ok {
					idsMap[id] = []string{}
				}
				idsMap[id] = append(idsMap[id], ID)
			}

			var wg sync.WaitGroup
			tokens := make(chan int, config.Threads)

			var record *fastx.Record
			for id, _IDs := range idsMap {
				wg.Add(1)
				tokens <- 1

				func(id string, _IDs []string) {
					defer func() {
						wg.Done()
						<-tokens
					}()
					var outfh *xopen.Writer
					var err error
					// outfile := filepath.Join(outdir, fmt.Sprintf("%s.id_%s%s",
					// 	filepath.Base(fileName),
					// 	pathutil.RemoveInvalidPathChars(id, "__"), fileExt))
					if prefixByIDSet {
						prefix = prefixByID
					} else {
						prefix = fmt.Sprintf("%s.part_", filepath.Base(fileName))
					}
					outfile = filepath.Join(outdir, fmt.Sprintf("%s%s%s",
						prefix,
						pathutil.RemoveInvalidPathChars(id, "__"), fileExt))

					if !dryRun {
						outfh, err = xopen.Wopen(outfile)
						checkError(err)
						for _, chr := range _IDs {
							r, ok := faidx.Index[chr]
							if !ok {
								checkError(fmt.Errorf(`sequence (%s) not found in file: %s`, chr, newFile))
							}

							sequence := subseqByFaix(faidx, chr, r, 1, -1)
							record, err = fastx.NewRecord(alphabet2, []byte(chr), []byte(chr), []byte{}, sequence)
							checkError(err)

							record.FormatToWriter(outfh, config.LineWidth)
						}
					}

					if !quiet {
						log.Infof("write %d sequences to file: %s\n", len(_IDs), outfile)
					}
					if !dryRun {
						outfh.Close()
					}
				}(id, _IDs)
			}
			wg.Wait()

			if (isstdin || !isPlainFile(file)) && !keepTemp {
				checkError(os.Remove(newFile))
				checkError(os.Remove(newFile + ".seqkit.fai"))
			}
			return
		}

		if region != "" {
			if !reRegion.MatchString(region) {
				checkError(fmt.Errorf(`invalid region: %s. type "seqkit subseq -h" for more examples`, region))
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

			if !twoPass {
				if !quiet {
					log.Info("read sequences ...")
				}
				allRecords, err := fastx.GetSeqs(file, alphabet, config.Threads, 10, idRegexp)
				checkError(err)
				if !quiet {
					log.Infof("read %d sequences", len(allRecords))
				}

				if len(allRecords) > 0 && len(allRecords[0].Seq.Qual) > 0 {
					config.LineWidth = 0
				}

				recordsBySeqs := make(map[string][]*fastx.Record)

				var subseq string
				var s, e int
				var ok bool
				for _, record := range allRecords {
					if renameFileExt && isstdin {
						if len(record.Seq.Qual) > 0 {
							fileExt = suffixFQ + extension
						} else {
							fileExt = suffixFA + extension
						}
						renameFileExt = false
					}
					s, e, ok = seq.SubLocation(len(record.Seq.Seq), start, end)
					if !ok {
						checkError(fmt.Errorf("region (%s) not match sequence (%s) with length of %d", region, record.Name, len(record.Seq.Seq)))
					}
					subseq = string(record.Seq.SubSeq(s, e).Seq)
					if _, ok := recordsBySeqs[subseq]; !ok {
						recordsBySeqs[subseq] = []*fastx.Record{}
					}
					recordsBySeqs[subseq] = append(recordsBySeqs[subseq], record)
				}

				var outfile string
				for subseq, records := range recordsBySeqs {
					// outfile = filepath.Join(outdir, fmt.Sprintf("%s.region_%d:%d_%s%s", filepath.Base(fileName), start, end, subseq, fileExt))
					if prefixByRegionSet {
						prefix = prefixByRegion
					} else {
						prefix = fmt.Sprintf("%s.part_", filepath.Base(fileName))
					}
					outfile = filepath.Join(outdir, fmt.Sprintf("%s%d:%d_%s%s", prefix, start, end, subseq, fileExt))
					writeSeqs(records, outfile, config.LineWidth, quiet, dryRun)
				}
				return
			}

			var alphabet2 *seq.Alphabet

			newFile := file

			if isstdin || !isPlainFile(file) {
				if isstdin {
					newFile = "stdin" + ".fastx"
				} else {
					newFile = file + ".fastx"
				}
				if !quiet {
					log.Infof("read and write sequences to temporary file: %s ...", newFile)
				}

				var nseqs int
				nseqs, err = copySeqs(file, newFile)
				checkError(err)
				if !quiet {
					log.Infof("%d sequences saved", nseqs)
				}

				var isFastq bool
				var err error
				alphabet2, isFastq, err = fastx.GuessAlphabet(newFile)
				if renameFileExt && isstdin {
					if isFastq {
						fileExt = suffixFQ + extension
					} else {
						fileExt = suffixFA + extension
					}
					renameFileExt = false
				}
				checkError(err)
				if isFastq {
					checkError(os.Remove(newFile))
					checkError(fmt.Errorf("sorry, two-pass mode does not support FASTQ format"))
				}
			}
			fileExt = suffixFA + extension

			if !quiet {
				log.Infof("read sequence IDs and sequence region from FASTA file ...")
			}
			region2name := make(map[string][]string)

			fastxReader, err := fastx.NewReader(alphabet2, newFile, idRegexp)
			checkError(err)
			var name string
			var subseq string
			var s, e int
			var ok bool
			for {
				record, err := fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}

				s, e, ok = seq.SubLocation(len(record.Seq.Seq), start, end)
				if !ok {
					checkError(fmt.Errorf("region (%s) not match sequence (%s) with length of %d", region, record.Name, len(record.Seq.Seq)))
				}
				subseq = string(record.Seq.SubSeq(s, e).Seq)
				name = string(record.Name)
				if _, ok := region2name[subseq]; !ok {
					region2name[subseq] = []string{}
				}
				region2name[subseq] = append(region2name[subseq], name)
			}
			fastxReader.Close()

			fileFai := newFile + ".seqkit.fai"

			if FileExists(fileFai) && updateFaidx {
				checkError(os.RemoveAll(fileFai))
				if !quiet {
					log.Infof("delete the old FASTA index file: %s", fileFai)
				}
			}

			if !quiet {
				log.Infof("create or read FASTA index ...")
			}
			faidx := getFaidx(newFile, `^(.+)$`, quiet)
			defer func() {
				checkError(faidx.Close())
			}()

			if len(faidx.Index) == 0 {
				log.Warningf("  0 records loaded from %s, please check if it matches the fasta file, or switch on the flag -U/--update-faidx", fileFai)
				return
			} else if !quiet {
				log.Infof("  %d records loaded from %s", len(faidx.Index), fileFai)
			}

			var wg sync.WaitGroup
			tokens := make(chan int, config.Threads)

			var record *fastx.Record
			for subseq, chrs := range region2name {
				wg.Add(1)
				tokens <- 1
				go func(subseq string, chrs []string) {
					defer func() {
						wg.Done()
						<-tokens
					}()

					var outfh *xopen.Writer
					var err error

					// outfile := filepath.Join(outdir, fmt.Sprintf("%s.region_%d:%d_%s%s", filepath.Base(fileName), start, end, subseq, fileExt))
					if prefixByRegionSet {
						prefix = prefixByRegion
					} else {
						prefix = fmt.Sprintf("%s.part_", filepath.Base(fileName))
					}
					outfile = filepath.Join(outdir, fmt.Sprintf("%s%d:%d_%s%s", prefix, start, end, subseq, fileExt))

					if !dryRun {
						outfh, err = xopen.Wopen(outfile)
						checkError(err)

						for _, chr := range chrs {
							r, ok := faidx.Index[chr]
							if !ok {
								checkError(fmt.Errorf(`sequence (%s) not found in file: %s`, chr, newFile))
							}

							sequence := subseqByFaix(faidx, chr, r, 1, -1)
							record, err = fastx.NewRecord(alphabet2, []byte(chr), []byte(chr), []byte{}, sequence)
							checkError(err)

							record.FormatToWriter(outfh, config.LineWidth)
						}
					}
					if !quiet {
						log.Infof("write %d sequences to file: %s\n", len(chrs), outfile)
					}
					if !dryRun {
						outfh.Close()
					}
				}(subseq, chrs)
			}
			wg.Wait()

			return
		}

		checkError(fmt.Errorf(`one of flags should be given: -s/-p/-i/-r. type "seqkit split -h" for help`))
	},
}

func init() {
	RootCmd.AddCommand(splitCmd)

	splitCmd.Flags().IntP("by-size", "s", 0, "split sequences into multi parts with N sequences")
	splitCmd.Flags().IntP("by-part", "p", 0, "split sequences into N parts")
	splitCmd.Flags().BoolP("by-id", "i", false, "split squences according to sequence ID")
	splitCmd.Flags().BoolP("ignore-case", "", false, "ignore case when using -i/--by-id")
	splitCmd.Flags().StringP("by-region", "r", "", "split squences according to subsequence of given region. "+
		`e.g 1:12 for first 12 bases, -12:-1 for last 12 bases. type "seqkit split -h" for more examples`)
	splitCmd.Flags().BoolP("two-pass", "2", false, "two-pass mode read files twice to lower memory usage. (only for FASTA format)")
	splitCmd.Flags().BoolP("update-faidx", "U", false, "update the fasta index file if it exists. Use this if you are not sure whether the fasta file changed")
	splitCmd.Flags().BoolP("dry-run", "d", false, "dry run, just print message and no files will be created.")
	splitCmd.Flags().BoolP("keep-temp", "k", false, "keep temporary FASTA and .fai file when using 2-pass mode")
	splitCmd.Flags().StringP("out-dir", "O", "", "output directory (default value is $infile.split)")
	splitCmd.Flags().BoolP("force", "f", false, "overwrite output directory")

	splitCmd.Flags().StringP("by-size-prefix", "", "", "file prefix for --by-size")
	splitCmd.Flags().StringP("by-part-prefix", "", "", "file prefix for --by-part")
	splitCmd.Flags().StringP("by-id-prefix", "", "", "file prefix for --by-id")
	splitCmd.Flags().StringP("by-region-prefix", "", "", "file prefix for --by-region")

	splitCmd.Flags().StringP("extension", "e", "", `set output file extension, e.g., ".gz", ".xz", or ".zst"`)
}

var suffixFA = ".fasta"
var suffixFQ = ".fastq"
