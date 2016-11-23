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
	"io"
	"math/rand"
	"os"
	"runtime"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fai"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/util/randutil"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// shuffleCmd represents the shuffle command
var shuffleCmd = &cobra.Command{
	Use:   "shuffle",
	Short: "shuffle sequences",
	Long: `shuffle sequences.

By default, all records will be readed into memory.
For FASTA format, use flag -2 (--two-pass) to reduce memory usage. FASTQ not
supported.

Firstly, seqkit reads the sequence IDs. If the file is not plain FASTA file,
seqkit will write the sequences to tempory files, and create FASTA index.

Secondly, seqkit shuffles sequence IDs and extract sequences by FASTA index.

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
		fai.MapWholeFile = false
		runtime.GOMAXPROCS(config.Threads)

		files := getFileList(args)
		seed := getFlagInt64(cmd, "rand-seed")
		twoPass := getFlagBool(cmd, "two-pass")
		keepTemp := getFlagBool(cmd, "keep-temp")
		if keepTemp && !twoPass {
			checkError(fmt.Errorf("flag -k (--keep-temp) must be used with flag -2 (--two-pass)"))
		}

		index2name := make(map[int]string)

		if !twoPass { // read all records into memory
			sequences := make(map[string]*fastx.Record)

			if !quiet {
				log.Infof("read sequences ...")
			}
			i := 0
			for _, file := range files {
				fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
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

					sequences[string(record.Name)] = record.Clone()
					index2name[i] = string(record.Name)
					i++
				}
			}

			if !quiet {
				log.Infof("%d sequences loaded", len(sequences))
				log.Infof("shuffle ...")
			}
			rand.Seed(seed)
			indices := make([]int, len(index2name))
			for i := 0; i < len(index2name); i++ {
				indices[i] = i
			}
			randutil.Shuffle(indices)

			if !quiet {
				log.Infof("output ...")
			}

			outfh, err := xopen.Wopen(outFile)
			checkError(err)
			defer outfh.Close()

			var record *fastx.Record
			for _, i := range indices {
				record = sequences[index2name[i]]
				record.FormatToWriter(outfh, config.LineWidth)
			}
			return
		}

		// two-pass
		if len(files) > 1 {
			checkError(fmt.Errorf("no more than one file should be given"))
		}

		file := files[0]

		newFile := file
		if isStdin(file) || !isPlainFile(file) {
			if isStdin(file) {
				newFile = "stdin" + ".fastx"
			} else {
				newFile = file + ".fastx"
			}
			if !quiet {
				log.Infof("read and write sequences to tempory file: %s ...", newFile)
			}

			copySeqs(file, newFile)

			var isFastq bool
			var err error
			_, isFastq, err = fastx.GuessAlphabet(newFile)
			checkError(err)
			if isFastq {
				checkError(os.Remove(newFile))
				checkError(fmt.Errorf("Sorry, two-pass mode does not support FASTQ format"))
			}
		}

		if !quiet {
			log.Infof("create and read FASTA index ...")
		}
		faidx := getFaidx(newFile, `^(.+)$`)
		defer func() {
			checkError(faidx.Close())
		}()

		if !quiet {
			log.Infof("read sequence IDs from FASTA index ...")
		}
		ids, _, err := getSeqIDAndLengthFromFaidxFile(newFile + ".seqkit.fai")
		checkError(err)
		for i, id := range ids {
			index2name[i] = id
		}

		if !quiet {
			log.Infof("%d sequences loaded", len(ids))
			log.Infof("shuffle ...")
		}
		rand.Seed(seed)
		indices := make([]int, len(index2name))
		for i := 0; i < len(index2name); i++ {
			indices[i] = i
		}
		randutil.Shuffle(indices)

		if !quiet {
			log.Infof("output ...")
		}
		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var chr string
		for _, i := range indices {
			chr = index2name[i]
			r, ok := faidx.Index[chr]
			if !ok {
				checkError(fmt.Errorf(`sequence (%s) not found in file: %s`, chr, newFile))
				continue
			}

			sequence := subseqByFaixNotCleaned(faidx, chr, r, 1, -1)
			outfh.Write([]byte(fmt.Sprintf(">%s\n", chr)))
			outfh.Write(sequence)
			outfh.WriteString("\n")
		}

		if (isStdin(file) || !isPlainFile(file)) && !keepTemp {
			checkError(os.Remove(newFile))
			checkError(os.Remove(newFile + ".seqkit.fai"))
		}

	},
}

func init() {
	RootCmd.AddCommand(shuffleCmd)
	shuffleCmd.Flags().Int64P("rand-seed", "s", 23, "rand seed for shuffle")
	shuffleCmd.Flags().BoolP("two-pass", "2", false, "two-pass mode read files twice to lower memory usage. (only for FASTA format)")
	shuffleCmd.Flags().BoolP("keep-temp", "k", false, "keep tempory FASTA and .fai file when using 2-pass mode")
}
