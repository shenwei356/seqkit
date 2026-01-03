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
	"math"
	"math/rand"
	"runtime"
	"time"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// sampleCmd represents the sample command
var sample2Cmd = &cobra.Command{
	GroupID: "set",

	Use:   "sample",
	Short: "sample sequences by number or proportion",
	Long: `sample sequences by number or proportion.

Attention:
1. Do not use '-n' on large FASTQ files, it loads all seqs into memory!
   use 'seqkit sample -p 0.1 seqs.fq.gz | seqkit head -n N' instead!
2. By default, the output is deterministic; that is, given the same input and random seed,
   seqkit shuf will always generate identical results across different runs.
   For 'true randomness', please add '-r/--non-deterministic', which uses a time-based seed.

`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			checkError(fmt.Errorf("no more than one file needed (%d)", len(args)))
		}

		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		// lineWidth := config.LineWidth
		outFile := config.OutFile
		quiet := config.Quiet
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
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

		twoPass := getFlagBool(cmd, "two-pass")
		number := getFlagInt64(cmd, "number")
		proportion := getFlagFloat64(cmd, "proportion")

		file := files[0]

		if twoPass && isStdin(file) {
			checkError(fmt.Errorf("two-pass mode (-2) will failed when reading from stdin. please disable flag: -2"))
		}

		if number == 0 && proportion == 0 {
			checkError(fmt.Errorf("one of flags -n (--number) and -p (--proportion) needed"))
		}

		if number > 0 && proportion > 0 {
			checkError(fmt.Errorf("flags -n (--number) and -p (--proportion) should not be set at the same time"))
		}

		if number < 0 {
			checkError(fmt.Errorf("value of -n (--number) and should be greater than 0"))
		}
		if proportion < 0 || proportion > 1 {
			checkError(fmt.Errorf("value of -p (--proportion) (%f) should be in range of (0, 1]", proportion))
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var _rand *rand.Rand
		if nonDeterministic {
			_rand = rand.New(rand.NewSource(time.Now().UnixNano()))
		} else {
			_rand = rand.New(rand.NewSource(seed))
		}

		// 4 branches:
		// by number + memory: reservior
		// by prop           : stream + prob
		// by prop   + 2 pass: get number, reservior
		// by number + 2 pass: reservior
		n := int64(0)

		outputRecord := func(rec *fastx.Record) {
			rec.FormatToWriter(outfh, config.LineWidth)
			n++
		}
		outputRecords := func(recs []*fastx.Record) {
			for _, rec := range recs {
				outputRecord(rec)
			}
		}

		// Branch A
		if number > 0 && !twoPass {
			if !quiet {
				log.Info("loading all sequences into memory...")
			}
			records, err := fastx.GetSeqs(file, alphabet, config.Threads, 10, idRegexp)
			checkError(err)

			totalSeqs := int64(len(records))
			if totalSeqs > 0 && len(records[0].Seq.Qual) > 0 {
				config.LineWidth = 0
			}

			if number >= totalSeqs {
				// if required number >= total, output directly
				outputRecords(records)
			} else {
				// again, sampling using reservoir
				reservoir := make([]*fastx.Record, number)

				// load first k elements into reservoir
				for i := int64(0); i < number; i++ {
					reservoir[i] = records[i]
				}

				// i-th element replace by prob
				for i := number; i < totalSeqs; i++ {
					// current is (i+1)-th element, prob should be k/(i+1)
					j := _rand.Int63n(i + 1)
					if j < number {
						reservoir[j] = records[i]
					}
				}

				// output sequences
				outputRecords(reservoir)
			}
			if !quiet {
				log.Infof("%d sequences outputted", n)
			}
			return
		}

		fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
		checkError(err)
		defer fastxReader.Close()

		if fastxReader.IsFastq {
			config.LineWidth = 0
			fastx.ForcelyOutputFastq = true
		}
		var record *fastx.Record

		// Branch B
		if proportion > 0.0 && !twoPass {
			for {
				record, err = fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}

				// if <-randg <= proportion {
				if _rand.Float64() <= proportion {
					outputRecord(record)
				}
			}
			if !quiet {
				log.Infof("%d sequences outputted", n)
			}
			return
		}

		// Two Pass Mode, first pass
		if !quiet {
			log.Info("first pass: counting seq number")
		}
		seqNum, err := fastx.GetSeqNumber(file)
		checkError(err)
		if !quiet {
			log.Infof("seq number: %d", seqNum)
		}

		// if by prop, get exact number
		if proportion > 0.0 && twoPass {
			number = int64(math.Floor(float64(seqNum) * proportion))
			if !quiet {
				log.Infof("sample %d/%d by proportion (%f)", number, seqNum, proportion)
			}
		}
		// second pass
		if !quiet {
			log.Info("second pass: reading and sampling")
		}
		// if number >= total, output all and return
		if number >= int64(seqNum) {
			for {
				record, err = fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}
				outputRecord(record)
			}
			if !quiet {
				log.Infof("%d sequences outputted", n)
			}
			return
		}

		reservoir := make([]*fastx.Record, 0, number)
		var count int64 = 0
		// number < total, sampling using reservoir
		// first load k samples to reservoir (k=desired sampling number)
		// iterate, for i-th sample (i>k), replacing random one of reservoir with prob of k/i
		for {
			record, err = fastxReader.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				checkError(err)
				break
			}

			count++
			if count <= number {
				// first k elements: add directly to reservoir
				reservoir = append(reservoir, record.Clone())
			} else {
				// for element i (i > k), replace with probability k/i
				// random int [0, count)
				j := _rand.Int63n(count)
				if j < number {
					reservoir[j] = record.Clone()
				}
			}
		}
		// output sequences
		outputRecords(reservoir)

		if !quiet {
			log.Infof("%d sequences outputted", n)
		}
	},
}

func init() {
	RootCmd.AddCommand(sampleCmd)

	sampleCmd.Flags().Int64P("rand-seed", "s", 11, "random seed. For paired-end data, use the same seed across fastq files to sample the same read pairs")
	sampleCmd.Flags().BoolP("non-deterministic", "r", false, "use a time-based seed to generate non-deterministic (truly random) results")
	sampleCmd.Flags().Int64P("number", "n", 0, "sample by number (result may not exactly match), DO NOT use on large FASTQ files.")
	sampleCmd.Flags().Float64P("proportion", "p", 0, "sample by proportion")
	sampleCmd.Flags().BoolP("two-pass", "2", false, "2-pass mode read files twice to lower memory usage. Not allowed when reading from stdin")
}
