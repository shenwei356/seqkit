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
	"runtime"

	"github.com/cespare/xxhash/v2"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/util/pathutil"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// renameCmd represents the rename command
var renameCmd = &cobra.Command{
	Use:   "rename",
	Short: "rename duplicated IDs",
	Long: `rename duplicated IDs

Attention:
  1. This command only appends "_N" to duplicated sequence IDs to make them unique.
  2. Use "seqkit replace" for editing sequence IDs/headers using regular expression.

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

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		byName := getFlagBool(cmd, "by-name")
		mOutputs := getFlagBool(cmd, "multiple-outfiles")
		outdir := getFlagString(cmd, "out-dir")
		force := getFlagBool(cmd, "force")

		separator := getFlagString(cmd, "separator")
		startNum := getFlagPositiveInt(cmd, "start-num")
		rename1st := getFlagBool(cmd, "rename-1st-rec")

		var outfh *xopen.Writer
		var err error

		if !mOutputs {
			outfh, err = xopen.Wopen(outFile)
			checkError(err)
			defer outfh.Close()
		} else {
			if outdir == "" {
				checkError(fmt.Errorf("out dir (flag -O/--out-dir) should not be empty"))
			}
			for _, file := range files {
				if isStdin(file) {
					checkError(fmt.Errorf("stdin detected, should not use -m/--mutliple-outfiles"))
				}
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
		}

		var record *fastx.Record
		var fastxReader *fastx.Reader
		var newID string
		var k uint64
		var ok bool
		numbers := make(map[uint64]int, 1<<20)
		for _, file := range files {
			func(file string) {
				fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
				checkError(err)

				if mOutputs {
					outfh, err = xopen.Wopen(filepath.Join(outdir, filepath.Base(file)))
					checkError(err)
					defer outfh.Close()
				}

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

					if byName {
						k = xxhash.Sum64(record.Name)
					} else {
						k = xxhash.Sum64(record.ID)
					}

					if _, ok = numbers[k]; ok {
						numbers[k]++
						newID = fmt.Sprintf("%s%s%d", record.ID, separator, numbers[k]-2+startNum)
						if len(record.Desc) > 0 {
							record.Name = []byte(fmt.Sprintf("%s %s", newID, record.Desc))
						} else {
							record.Name = []byte(newID)
						}
					} else {
						numbers[k] = 1
						if rename1st {
							newID = fmt.Sprintf("%s%s%d", record.ID, separator, numbers[k]-2+startNum)
							if len(record.Desc) > 0 {
								record.Name = []byte(fmt.Sprintf("%s %s", newID, record.Desc))
							} else {
								record.Name = []byte(newID)
							}
						}
					}

					record.FormatToWriter(outfh, config.LineWidth)
				}
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
	renameCmd.Flags().BoolP("force", "f", false, "overwrite output directory")
}
