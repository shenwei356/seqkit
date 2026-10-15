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
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fai"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// shuffleCmd represents the shuffle command
var shuffleCmd = &cobra.Command{
	GroupID: "order",

	Use:   "shuffle",
	Short: "shuffle sequences",
	Long: `shuffle sequences.

By default, all records will be read into memory.
For FASTA format, you can use flag -2 (--two-pass) to reduce memory usage.
While FASTQ is not supported. 
  1. Seqkit reads the sequence IDs. If the file is not a plain FASTA file,
     seqkit will write the sequences to temporary files, and create the FASTA index.
  2. Secondly, seqkit shuffles sequence IDs and extracts sequences by FASTA index.

Attention:
  1. For the two-pass mode (-2/--two-pass), The flag -U/--update-faidx is recommended to
     ensure the .fai file matches the FASTA file.
  2. By default, the output is deterministic; that is, given the same input and random seed,
     seqkit shuf will always generate identical results across different runs.
     For 'true randomness', please add '-r/--non-deterministic', which uses a time-based seed.

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		outFile := config.OutFile
		quiet := config.Quiet
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		fai.MapWholeFile = false
		runtime.GOMAXPROCS(config.Threads)

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", !config.SkipFileCheck)
		if !config.SkipFileCheck {
			for _, file := range files {
				checkIfFilesAreTheSame(file, outFile, "input", "output")
			}
		}

		seed := getFlagInt64(cmd, "rand-seed")
		nonDeterministic := getFlagBool(cmd, "non-deterministic")

		if nonDeterministic && cmd.Flags().Lookup("rand-seed").Changed {
			checkError(fmt.Errorf("the flags -s/--rand-seed and -r/--non-deterministic are incompatible"))
		}

		var r *rand.Rand
		if nonDeterministic {
			r = rand.New(rand.NewSource(time.Now().UnixNano()))
		} else {
			r = rand.New(rand.NewSource(seed))
		}

		twoPass := getFlagBool(cmd, "two-pass")
		updateFaidx := getFlagBool(cmd, "update-faidx")
		tmpDir := filepath.Clean(getFlagString(cmd, "tmp-dir"))
		keepTemp := getFlagBool(cmd, "keep-temp")
		if keepTemp && !twoPass {
			checkError(fmt.Errorf("flag -k (--keep-temp) must be used with flag -2 (--two-pass)"))
		}
		if updateFaidx && !twoPass {
			checkError(fmt.Errorf("flag -U (--update-faidx) must be used with flag -2 (--two-pass)"))
		}

		var record *fastx.Record
		var err error

		if !twoPass { // read all records into memory
			sequences := make([]*fastx.Record, 0, 1024)

			if !quiet {
				log.Infof("read sequences ...")
			}
			for _, file := range files {
				fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
				checkError(err)
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
						if fastxReader.IsFastq {
							if !config.LineWidthChanged {
								config.LineWidth = 0
							}
							fastx.ForcelyOutputFastq = true
						}
						checkAlphabet = false
					}

					sequences = append(sequences, record.Clone())
				}
				fastxReader.Close()
			}

			if !quiet {
				log.Infof("%d sequences loaded", len(sequences))
				log.Infof("shuffle ...")
			}
			r.Shuffle(len(sequences), func(i, j int) {
				sequences[i], sequences[j] = sequences[j], sequences[i]
			})

			if !quiet {
				log.Infof("output ...")
			}

			outfh, err := xopen.Wopen(outFile)
			checkError(err)
			defer closeOutput(outfh)

			for _, record := range sequences {
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
			var nseqs int
			newFile, nseqs, err = copySeqsToTemp(file, tmpDir)
			checkError(err)
			if !keepTemp {
				defer cleanupTempSeqs(newFile)
			}
			if !quiet {
				log.Infof("read and write sequences to temporary file: %s ...", newFile)
			}

			if !quiet {
				log.Infof("%d sequences saved", nseqs)
			}

			var isFastq bool
			var err error
			_, isFastq, err = fastx.GuessAlphabet(newFile)
			checkError(err)
			if isFastq {
				checkError(os.Remove(newFile))
				checkError(fmt.Errorf("Sorry, two-pass mode does not support FASTQ format"))
			}
		}

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

		ids, _, err := getSeqIDAndLengthFromFaidxFile(newFile + ".seqkit.fai")
		checkError(err)

		if !quiet {
			log.Infof("shuffle ...")
		}
		r.Shuffle(len(ids), func(i, j int) {
			ids[i], ids[j] = ids[j], ids[i]
		})

		if !quiet {
			log.Infof("output ...")
		}
		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer closeOutput(outfh)

		for _, chr := range ids {
			r, ok := faidx.Index[chr]
			if !ok {
				checkError(fmt.Errorf(`sequence (%s) not found in file: %s`, chr, newFile))
				continue
			}

			sequence := subseqByFaixNotCleaned(faidx, chr, r, 1, -1)
			outfh.Write(fmt.Appendf(nil, ">%s\n", chr))
			outfh.Write(sequence)
			if len(sequence) == 0 || sequence[len(sequence)-1] != '\n' {
				outfh.WriteString("\n")
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(shuffleCmd)
	shuffleCmd.Flags().Int64P("rand-seed", "s", 23, "rand seed for shuffle")
	shuffleCmd.Flags().BoolP("non-deterministic", "r", false, "use a time-based seed to generate non-deterministic (truly random) results")
	shuffleCmd.Flags().BoolP("two-pass", "2", false, "two-pass mode read files twice to lower memory usage. (only for FASTA format)")
	shuffleCmd.Flags().StringP("tmp-dir", "", "./", "tmp directory for saving temporary FASTA and .fai file when using 2-pass mode")
	shuffleCmd.Flags().BoolP("keep-temp", "k", false, "keep temporary FASTA and .fai file when using 2-pass mode")
	shuffleCmd.Flags().BoolP("update-faidx", "U", false, "update the fasta index file if it exists. Use this if you are not sure whether the fasta file changed")
}
