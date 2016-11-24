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
	"bytes"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// commonCmd represents the common command
var commonCmd = &cobra.Command{
	Use:   "common",
	Short: "find common sequences of multiple files by id/name/sequence",
	Long: `find common sequences of multiple files by id/name/sequence

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		lineWidth := config.LineWidth
		outFile := config.OutFile
		quiet := config.Quiet
		seq.AlphabetGuessSeqLenghtThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		bySeq := getFlagBool(cmd, "by-seq")
		byName := getFlagBool(cmd, "by-name")
		ignoreCase := getFlagBool(cmd, "ignore-case")
		usingMD5 := getFlagBool(cmd, "md5")

		if bySeq && byName {
			checkError(fmt.Errorf("only one/none of the flags -s (--by-seq) and -n (--by-name) is allowed"))
		}
		if usingMD5 && !bySeq {
			checkError(fmt.Errorf("flag -m (--md5) must be used with flag -s (--by-seq)"))
		}

		files := getFileList(args)
		if len(files) < 2 {
			checkError(errors.New("at least 2 files needed"))
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		// target -> file -> struct{}
		counter := make(map[string]map[string]struct{})
		// target -> file -> seqname
		names := make(map[string]map[string][]string)

		var fastxReader *fastx.Reader

		// read all files
		var subject string
		var checkFirstFile = true
		var firstFile string
		for _, file := range files {
			if !quiet {
				log.Infof("read file: %s", file)
			}
			if checkFirstFile {
				firstFile = file
				checkFirstFile = false
			}

			fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)
			for {
				record, err := fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}

				if bySeq {
					if ignoreCase {
						if usingMD5 {
							subject = MD5(bytes.ToLower(record.Seq.Seq))
						} else {
							subject = string(bytes.ToLower(record.Seq.Seq))
						}
					} else {
						if usingMD5 {
							subject = MD5(record.Seq.Seq)
						} else {
							subject = string(record.Seq.Seq)
						}
					}
				} else if byName {
					if ignoreCase {
						subject = strings.ToLower(string(record.Name))
					} else {
						subject = string(record.Name)
					}
				} else { // byID
					if ignoreCase {
						subject = strings.ToLower(string(record.ID))
					} else {
						subject = string(record.ID)
					}
				}

				if _, ok := counter[subject]; !ok {
					counter[subject] = make(map[string]struct{})
				}
				counter[subject][file] = struct{}{}

				if _, ok := names[subject]; !ok {
					names[subject] = make(map[string][]string)
				}
				names[subject][file] = append(names[subject][file], string(record.Name))
			}
		}

		// find common seqs
		if !quiet {
			log.Info("find common seqs ...")
		}
		fileNum := len(files)
		namesOK := make(map[string]struct{})
		var n, n2 int
		var seqname string
		for subject, presence := range counter {
			if len(presence) != fileNum {
				continue
			}

			n++
			for _, seqname = range names[subject][firstFile] {
				n2++
				namesOK[seqname] = struct{}{}
			}
		}

		var t string
		if byName {
			t = "sequence headers"
		} else if bySeq {
			t = "sequences"
		} else {
			t = "sequence IDs"
		}
		if n == 0 {
			log.Infof("no common %s found", t)
			return
		}
		if !quiet {
			log.Infof("%d unique %s found in %d files, which belong to %d records in the first file: %s",
				n, t, fileNum, len(namesOK), firstFile)
		}

		if !quiet {
			log.Infof("extract seqs from the first file: %s", firstFile)
		}

		// extract
		fastxReader, err = fastx.NewReader(alphabet, firstFile, idRegexp)
		checkError(err)
		for {
			record, err := fastxReader.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				checkError(err)
				break
			}
			if fastxReader.IsFastq {
				config.LineWidth = 0
			}

			if _, ok := namesOK[string(record.Name)]; ok {
				record.FormatToWriter(outfh, lineWidth)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(commonCmd)

	commonCmd.Flags().BoolP("by-name", "n", false, "match by full name instead of just id")
	commonCmd.Flags().BoolP("by-seq", "s", false, "match by sequence")
	commonCmd.Flags().BoolP("md5", "m", false, "use MD5 instead of original seqs to reduce memory usage when comparing by seqs")
	commonCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
}
