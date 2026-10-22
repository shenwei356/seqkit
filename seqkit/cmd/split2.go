// Copyright © 2016-2026 Wei Shen <shenwei356@gmail.com>
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
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

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

Attention:
  1. Specify exactly one of -s, -p and -l.
  2. For paired inputs, parts contain corresponding records by position;
     unequal record counts are errors. With -l, read1 determines the boundaries
     and read2 follows.
  3. Lengths use K/M/G/T suffixes with base 1000; fractional bases are truncated.
  4. Existing matching outputs are overwritten; -f suppresses the directory
     warning. Other files, including old parts not produced this time, are retained.

The prefix of output files:
  1. For stdin: stdin
  2. Others: same to the input file
  3. Set via the options: -P/--out-prefix,
     or --by-length-prefix, --by-part-prefix, or --by-size-prefix (for keeping backwards compatibility)
  4. Use the ID of the first sequence in each subset.
     Output names must be unique; -N does not group records with the same ID.
     Conflicting output names are errors, including repeated names produced by -N.
  5. Prefix flags accept file names only, not directory paths.

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

		if !config.SkipFileCheck {
			for _, file := range files {
				checkIfFilesAreTheSame(file, outFile, "input", "output")
			}
		}

		read1 := getFlagString(cmd, "read1")
		read2 := getFlagString(cmd, "read2")

		size := getFlagNonNegativeInt(cmd, "by-size")
		parts := getFlagNonNegativeInt(cmd, "by-part")
		lengthS := getFlagString(cmd, "by-length")
		var length int64
		var err error
		if cmd.Flags().Changed("by-length") {
			length, err = parseSequenceLength(lengthS)
			if err != nil || length < 1 {
				checkError(fmt.Errorf("invalid sequence length: %q", lengthS))
			}
		}

		modes := 0
		for _, flag := range []string{"by-size", "by-part", "by-length"} {
			if cmd.Flags().Changed(flag) {
				modes++
			}
		}
		if modes != 1 {
			checkError(fmt.Errorf("exactly one of -s/-p/-l must be specified"))
		}

		outdir := getFlagString(cmd, "out-dir")
		force := getFlagBool(cmd, "force")

		extension := getFlagString(cmd, "extension")

		reRead := regexp.MustCompile(`\{read\}`)

		prefixAll := getFlagString(cmd, "out-prefix")
		prefixAllSet := cmd.Flags().Lookup("out-prefix").Changed

		prefixBySize := getFlagString(cmd, "by-size-prefix")
		prefixByPart := getFlagString(cmd, "by-part-prefix")
		prefixByLength := getFlagString(cmd, "by-length-prefix")

		prefixBySizeSet := cmd.Flags().Lookup("by-size-prefix").Changed || prefixAllSet
		prefixByPartSet := cmd.Flags().Lookup("by-part-prefix").Changed || prefixAllSet
		prefixByLengthSet := cmd.Flags().Lookup("by-length-prefix").Changed || prefixAllSet

		if prefixAllSet {
			prefixBySize = prefixAll
			prefixByPart = prefixAll
			prefixByLength = prefixAll
		}

		for _, prefix := range []string{prefixAll, prefixBySize, prefixByPart, prefixByLength} {
			validateOutputPrefix(prefix)
		}

		partNumberWidth := getFlagPositiveInt(cmd, "part-width")
		fileFmt := fmt.Sprintf("%%s%%0%dd%%s", partNumberWidth)

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
					checkError(fmt.Errorf(`-P/--out-prefix or --by-size-prefix should contains the placeholder "{read}" when paired-end files are given, such as "sample_{read}.fq.gz`))
				}
				if prefixByPartSet && !reRead.MatchString(prefixByPart) {
					checkError(fmt.Errorf(`-P/--out-prefix or --by-part-prefix should contains the placeholder "{read}" when paired-end files are given, such as "sample_{read}.fq.gz`))
				}
				if prefixByLengthSet && !reRead.MatchString(prefixByLength) {
					checkError(fmt.Errorf(`-P/--out-prefix or --by-size-prefix should contains the placeholder "{read}" when paired-end files are given, such as "sample_{read}.fq.gz`))
				}
			}
		}

		// Paired-read flags replace the shared input list and need the same check.
		checkError(validateStdinUsage(files, isStdin(getFlagString(cmd, "infile-list"))))

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

		if outdir == "" {
			if isStdin(files[0]) {
				outdir = "stdin.split"
			} else {
				outdir = files[0] + ".split"
			}
		}
		if force {
			checkOutputDir(files, outdir)
		}
		// Read mates together so all modes share one set of part boundaries.
		readers := make([]*fastx.Reader, len(files))
		records := make([]*fastx.Record, len(files))
		fastqs := make([]bool, len(files))
		widths := make([]int, len(files))
		stems := make([]string, len(files))
		exts := make([]string, len(files))
		for i, file := range files {
			readers[i], err = fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)
			defer readers[i].Close()
			records[i], err = readers[i].Read()
			if err != nil && err != io.EOF {
				checkError(fmt.Errorf("%s: %w", file, err))
			}
			if records[i] != nil {
				records[i] = records[i].Clone()
			}
			fastqs[i] = readers[i].IsFastq
			widths[i] = config.LineWidth
			if fastqs[i] && !config.LineWidthChanged {
				widths[i] = 0
			}
			if isStdin(file) {
				stems[i] = "stdin"
				if fastqs[i] {
					exts[i] = suffixFQ + extension
				} else {
					exts[i] = suffixFA + extension
				}
			} else {
				stem, ext, compression := filepathTrimExtension2(file, nil)
				stems[i], exts[i] = filepath.Base(stem), ext+compression
				if extension != "" {
					exts[i] = ext + extension
				}
			}
			if outFile != "-" {
				stems[i] = filepath.Base(outFile)
			}
		}
		fastx.ForcelyOutputFastq = false
		if pairedEnd && (records[0] == nil) != (records[1] == nil) {
			checkError(fmt.Errorf("record counts differ: %s and %s", files[0], files[1]))
		}
		if records[0] == nil {
			return
		}
		registry := splitOutputRegistry{inputs: files}
		filename := func(side, part int, record *fastx.Record) string {
			if seqIDAsFileName {
				return filepath.Join(outdir, pathutil.RemoveInvalidPathChars(string(record.ID), "__")+exts[side])
			}
			prefix := stems[side] + ".part_"
			if bySize && prefixBySizeSet {
				prefix = prefixBySize
			}
			if byParts && prefixByPartSet {
				prefix = prefixByPart
			}
			if byLength && prefixByLengthSet {
				prefix = prefixByLength
			}
			if pairedEnd {
				prefix = reRead.ReplaceAllString(prefix, strconv.Itoa(side+1))
			}
			return filepath.Join(outdir, fmt.Sprintf(fileFmt, prefix, part, exts[side]))
		}
		if pairedEnd {
			checkOutputFiles(files, []string{filename(0, 1, records[0]), filename(1, 1, records[1])})
		}
		prepareOutputDir(outdir, force)
		writers := make(map[int][]*xopen.Writer)
		counts := make(map[int]int)
		paths := make(map[int][]string)
		closePart := func(part int) {
			for side, w := range writers[part] {
				closeOutput(w)
				if !quiet {
					log.Infof("write %d sequences to file: %s", counts[part], paths[part][side])
				}
			}
			delete(writers, part)
		}
		closeAll := func() {
			for part := range writers {
				closePart(part)
			}
		}
		part := 1
		var bases int64
		total := 0
		for {
			if writers[part] == nil {
				for side, record := range records {
					path := filename(side, part, record)
					// Register every logical part once, including round-robin outputs.
					w, openErr := registry.open(path)
					if openErr != nil {
						closeAll()
						checkError(openErr)
					}
					writers[part] = append(writers[part], w)
					paths[part] = append(paths[part], path)
				}
			}
			for side, record := range records {
				writeFastxRecord(writers[part][side], record, fastqs[side], widths[side])
			}
			counts[part]++
			total++
			bases += int64(len(records[0].Seq.Seq))
			if byParts {
				part = (part % parts) + 1
			} else if bySize && counts[part] >= size || byLength && bases >= length {
				closePart(part)
				part++
				bases = 0
			}
			ended := 0
			for side, reader := range readers {
				record, readErr := reader.Read()
				if readErr == io.EOF {
					ended++
					records[side] = nil
					continue
				}
				if readErr != nil {
					closeAll()
					checkError(fmt.Errorf("%s: %w", files[side], readErr))
				}
				records[side] = record
			}
			if ended > 0 {
				closeAll()
				if ended != len(readers) {
					checkError(fmt.Errorf("record counts differ after %d pairs: %s and %s", total, files[0], files[1]))
				}
				break
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(split2Cmd)

	split2Cmd.Flags().StringP("read1", "1", "", "(gzipped) read1 file")
	split2Cmd.Flags().StringP("read2", "2", "", "(gzipped) read2 file")
	split2Cmd.Flags().IntP("by-size", "s", 0, "split sequences into multi parts with N sequences")
	split2Cmd.Flags().IntP("by-part", "p", 0, "split sequences into N parts with the round robin distribution")
	split2Cmd.Flags().StringP("by-length", "l", "", "split sequences into chunks of >=N bases, supports K/M/G suffix (base 1000)")
	split2Cmd.Flags().StringP("out-dir", "O", "", "output directory (default value is $infile.split)")
	split2Cmd.Flags().BoolP("force", "f", false, "overwrite matching output files without the nonempty-directory warning")

	split2Cmd.Flags().StringP("by-size-prefix", "", "", `file prefix for --by-size. The placeholder "{read}" is needed for paired-end files.`)
	split2Cmd.Flags().StringP("by-part-prefix", "", "", `file prefix for --by-part. The placeholder "{read}" is needed for paired-end files.`)
	split2Cmd.Flags().StringP("by-length-prefix", "", "", `file prefix for --by-length. The placeholder "{read}" is needed for paired-end files.`)
	split2Cmd.Flags().StringP("out-prefix", "P", "", `file prefix (it overrides --by-*-prefix). The placeholder "{read}" is needed for paired-end files.`)

	split2Cmd.Flags().BoolP("seqid-as-filename", "N", false, "use the first sequence ID as each part's file name; conflicting names are errors")

	split2Cmd.Flags().StringP("extension", "e", "", `set output file extension, e.g., ".gz", ".xz", or ".zst"`)

	split2Cmd.Flags().IntP("part-width", "W", 3, "number of digits used for output file part numbering (zero-padded), e.g., 001, 002")
}
