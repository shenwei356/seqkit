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
	"runtime"

	"github.com/cespare/xxhash/v2"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// renameCmd represents the rename command
var renameCmd = &cobra.Command{
	GroupID: "edit",

	Use:   "rename",
	Short: "rename duplicated IDs",
	Long: `rename duplicated IDs

Attention:
  1. This command only appends "_N" to duplicated sequence IDs to make them unique.
  2. Use "seqkit replace" for editing sequence IDs/headers using regular expression.
  3. With -m, unrelated files in the output directory are preserved, even with --force.

Example:

    $ seqkit seq seqs.fasta 
    >id comment
    actg
    >id description
    ACTG

    $ seqkit rename seqs.fasta
    >id comment
    actg
    >id_2 description
    ACTG

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

		files := getFileListFromArgsAndFile(cmd, args, !config.SkipFileCheck, "infile-list", !config.SkipFileCheck)
		checkOutputFiles(files, []string{outFile}, !config.SkipFileCheck)
		byName := getFlagBool(cmd, "by-name")
		mOutputs := getFlagBool(cmd, "multiple-outfiles")
		outdir := getFlagString(cmd, "out-dir")
		force := getFlagBool(cmd, "force")

		separator := getFlagString(cmd, "separator")
		startNum := uint64(getFlagPositiveInt(cmd, "start-num"))
		rename1st := getFlagBool(cmd, "rename-1st-rec")

		var outfh *xopen.Writer
		var err error

		if !mOutputs {
			outfh, err = xopen.Wopen(outFile)
			checkError(err)
			defer closeOutput(outfh)
		} else {
			if outdir == "" {
				checkError(fmt.Errorf("out dir (flag -O/--out-dir) should not be empty"))
			}
			for _, file := range files {
				if isStdin(file) {
					checkError(fmt.Errorf("stdin detected, should not use -m/--mutliple-outfiles"))
				}
			}
			outputs := make([]string, len(files))
			for i, file := range files {
				outputs[i] = filepath.Join(outdir, filepath.Base(file))
			}
			checkOutputFiles(files, outputs, !config.SkipFileCheck)
			if infileList := getFlagString(cmd, "infile-list"); infileList != "" {
				checkOutputFiles([]string{infileList}, outputs, true)
			}
			prepareOutputDir(outdir, force)
		}

		var record *fastx.Record
		var newID string
		var k uint64
		var ok bool
		// Suffixes may exceed MaxInt when a valid starting number is near the limit.
		numbers := make(map[uint64]uint64)
		emitted := make(map[uint64]struct{}) // Reserve actual output IDs/names, including generated suffixes.
		for _, file := range files {
			func(file string) {
				fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
				checkError(err)

				if mOutputs {
					outfh, err = xopen.Wopen(filepath.Join(outdir, filepath.Base(file)))
					checkError(err)
					defer closeOutput(outfh)
				}

				checkAlphabet := true
				for {
					record, err = fastxReader.Read()
					if err != nil {
						if err == io.EOF {
							break
						}
						checkError(err)
						break
					}

					if checkAlphabet {
						fastx.ForcelyOutputFastq = fastxReader.IsFastq
						if fastxReader.IsFastq {
							if !config.LineWidthChanged {
								config.LineWidth = 0
							}
						}
						checkAlphabet = false
					}

					if idRegexp == fastx.DefaultIDRegexp || idRegexp == "" {
						restoreDescription(record)
					}

					if byName {
						k = xxhash.Sum64(record.Name)
					} else {
						k = xxhash.Sum64(record.ID)
					}

					numbers[k]++
					_, ok = emitted[k]
					if numbers[k] > 1 || rename1st || ok {
						count := numbers[k] - 1 + (startNum - 1)
						if numbers[k] == 1 && !rename1st {
							count++
						}
						for {
							newID = fmt.Sprintf("%s%s%d", record.ID, separator, count)
							name := newID
							if len(record.Desc) > 0 {
								name += " " + string(record.Desc)
							}
							outputKey := xxhash.Sum64String(newID)
							if byName {
								outputKey = xxhash.Sum64String(name)
							}
							if _, exists := emitted[outputKey]; !exists {
								numbers[k] = count - (startNum - 1) + 1
								record.Name = []byte(name)
								emitted[outputKey] = struct{}{}
								break
							}
							count++
						}
					} else {
						emitted[k] = struct{}{}
					}

					record.FormatToWriter(outfh, config.LineWidth)
				}
				fastxReader.Close()
				config.LineWidth = lineWidth
			}(file)
		}
	},
}

func init() {
	RootCmd.AddCommand(renameCmd)

	renameCmd.Flags().StringP("separator", "s", "_", "separator between original ID/name and the counter")
	renameCmd.Flags().IntP("start-num", "N", 2, `starting count number for *duplicated* IDs/names, should be greater than zero`)
	renameCmd.Flags().BoolP("rename-1st-rec", "1", false, "rename the first record as well")

	renameCmd.Flags().BoolP("by-name", "n", false, "check duplication by full name instead of just id")
	renameCmd.Flags().BoolP("multiple-outfiles", "m", false, "write results into separated files for multiple input files")
	renameCmd.Flags().StringP("out-dir", "O", "renamed", "output directory")
	renameCmd.Flags().BoolP("force", "f", false, "overwrite matching output files without the nonempty-directory warning")
}
