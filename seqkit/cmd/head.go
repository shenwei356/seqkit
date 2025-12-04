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
	"runtime"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// headCmd represents the head command
var headCmd = &cobra.Command{
	GroupID: "set",

	Use:   "head",
	Short: "print the first N FASTA/Q records, or leading records whose total length >= L",
	Long: `print the first N FASTA/Q records, or leading records whose total length >= L

For returning the last N records, use:
    seqkit range -r -N:-1 seqs.fasta

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		outFile := config.OutFile
		lineWidth := config.LineWidth
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		number := getFlagPositiveInt(cmd, "number")

		setLength := cmd.Flags().Lookup("length").Changed
		lengthS := getFlagString(cmd, "length")

		var length int64
		var err error
		if setLength && lengthS != "" {
			length, err = ParseByteSize(lengthS)
			if err != nil {
				checkError(fmt.Errorf("parsing length: %s", lengthS))
			}

			if length > 0 {
				number = 0
			}
		}

		if number > 0 && length > 0 {
			checkError(fmt.Errorf("-n/--number is incompatible with -l/--length"))
		}
		byRecords := number > 0

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", !config.SkipFileCheck)
		if !config.SkipFileCheck {
			for _, file := range files {
				checkIfFilesAreTheSame(file, outFile, "input", "output")
			}
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var record *fastx.Record
		i := 0
		var l int64 = 0
		for _, file := range files {
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
				i++
				l += int64(record.Seq.Length())
				record.FormatToWriter(outfh, config.LineWidth)

				if byRecords {
					if number == i {
						return
					}
				} else {
					if l >= length {
						return
					}
				}
			}
			fastxReader.Close()

			config.LineWidth = lineWidth
		}
	},
}

func init() {
	RootCmd.AddCommand(headCmd)
	headCmd.Flags().IntP("number", "n", 10, "print the first N FASTA/Q records")
	headCmd.Flags().StringP("length", "l", "", "print leading FASTA/Q records whose total sequence length >= L (supports K/M/G suffix). This flag overrides -n/--number")
}
