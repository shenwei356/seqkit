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

		var outfh *xopen.Writer
		var err error

		if !mOutputs {
			outfh, err = xopen.Wopen(outFile)
			checkError(err)
			defer outfh.Close()
		} else {
			if outdir == "" {
				checkError(fmt.Errorf("out dir should not be empty"))
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
						} else {
							checkError(fmt.Errorf("outdir not empty: %s, use -f (--force) to overwrite", outdir))
						}
					} else {
						checkError(os.RemoveAll(outdir))
					}
				}
				checkError(os.MkdirAll(outdir, 0777))
			}
		}

		var record *fastx.Record
		var fastxReader *fastx.Reader
		var newID string
		var k string
		var ok bool
		var numbers map[string]int
		numbers = make(map[string]int)
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
						k = string(record.Name)
					} else {
						k = string(record.ID)
					}

					if _, ok = numbers[k]; ok {
						numbers[k]++
						newID = fmt.Sprintf("%s_%d", record.ID, numbers[k])
						record.Name = []byte(fmt.Sprintf("%s %s", newID, record.Name))
					} else {
						numbers[k] = 1
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

	renameCmd.Flags().BoolP("by-name", "n", false, "check duplication by full name instead of just id")
	renameCmd.Flags().BoolP("multiple-outfiles", "m", false, "write results into separated files for multiple input files")
	renameCmd.Flags().StringP("out-dir", "O", "renamed", "output directory")
	renameCmd.Flags().BoolP("force", "f", false, "overwrite output directory")
}
