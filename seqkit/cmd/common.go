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

Note:
    1. 'seqkit common' is designed to support 2 and MORE files.
    2. For 2 files, 'seqkit grep' is much faster and consumes lesser memory:
         seqkit grep -f <(seqkit seq -n -i small.fq.gz) big.fq.gz # by seq ID
         seqkit grep -s -f <(seqkit seq -s small.fq.gz) big.fq.gz # by seq
    3. Some records in one file may have same sequences/IDs. They will ALL be
       retrieved if the sequence/ID was shared in multiple files.
       So the records number may be larger than that of the smallest file.

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		// lineWidth := config.LineWidth
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
		counter := make(map[string]map[string]struct{}, 1000)

		// target -> seqnames in firstFile
		// note that it's []string, i.e., records may have same sequences
		names := make(map[string][]string, 1000)

		var fastxReader *fastx.Reader
		var record *fastx.Record

		// read all files
		var subject string
		var checkFirstFile = true
		var isFirstFile = true
		var firstFile string
		var filenames = make(map[string]int)
		var ok bool
		for _, file := range files {
			if !quiet {
				log.Infof("read file: %s", file)
			}
			if checkFirstFile && !isStdin(file) {
				firstFile = file
				checkFirstFile = false
			}

			fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)

			// alowing finding common records in ONE file
			if _, ok = filenames[file]; !ok {
				filenames[file] = 1
			} else {
				filenames[file]++
				file = fmt.Sprintf("%s_%d", file, filenames[file])
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

				if _, ok = counter[subject]; !ok {
					counter[subject] = make(map[string]struct{})
				}
				counter[subject][file] = struct{}{}

				if isFirstFile {
					if _, ok = names[subject]; !ok {
						names[subject] = make([]string, 0, 1)
					}
					names[subject] = append(names[subject], string(record.Name))
				}
			}
			if isFirstFile {
				isFirstFile = false
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
			for _, seqname = range names[subject] {
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
				n, t, fileNum, n2, firstFile)
		}

		if !quiet {
			log.Infof("retrieve seqs from the first file: %s", firstFile)
		}

		// retrieve
		fastxReader, err = fastx.NewReader(alphabet, firstFile, idRegexp)
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
			}

			if _, ok := namesOK[string(record.Name)]; ok {
				record.FormatToWriter(outfh, config.LineWidth)
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
