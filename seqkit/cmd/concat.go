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
	"bytes"
	"errors"
	"fmt"
	"io"
	"runtime"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// concateCmd represents the concatenate command
var concateCmd = &cobra.Command{
	Use:     "concat",
	Aliases: []string{"concate"},
	Short:   "concatenate sequences with same ID from multiple files",
	Long: `concatenate sequences with same ID from multiple files

Example: concatenating leading 2 bases and last 2 bases

    $ cat t.fa
    >test
    ACCTGATGT
    >test2
    TGATAGCTACTAGGGTGTCTATCG

    $ seqkit concate <(seqkit subseq -r 1:2 t.fa) <(seqkit subseq -r -2:-1 t.fa)
    >test
    ACGT
    >test2
    TGCG

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		// lineWidth := config.LineWidth
		outFile := config.OutFile
		quiet := config.Quiet
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		infileList := getFlagString(cmd, "infile-list")
		files := getFileList(args, true)
		if infileList != "" {
			_files, err := getListFromFile(infileList, true)
			checkError(err)
			files = append(files, _files...)
		}

		if len(files) < 2 {
			checkError(errors.New("at least 2 files needed"))
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		seqs := make(map[string][][]byte, 1000)
		quals := make(map[string][][]byte, 1000)

		var fastxReader *fastx.Reader
		var record *fastx.Record
		var clone *seq.Seq
		var ok bool
		var id string
		ids := make([]string, 0, 1000)
		var n int
		var isFastq bool
		for i, file := range files {
			if !quiet {
				log.Infof("read file: %s", file)
			}

			fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)

			n = 0
			for {
				record, err = fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}

				if i == 0 {
					isFastq = fastxReader.IsFastq
					if isFastq {
						fastx.ForcelyOutputFastq = true
					}
				} else {
					if isFastq != fastxReader.IsFastq {
						checkError(fmt.Errorf("concatenating FASTA and FASTQ is not allowed"))
					}
				}

				n++
				id = string(record.ID)
				if _, ok = seqs[id]; !ok {
					seqs[id] = make([][]byte, 0, len(files))
				}
				clone = record.Seq.Clone()
				seqs[id] = append(seqs[id], clone.Seq)

				if isFastq {
					if _, ok = quals[id]; !ok {
						quals[id] = make([][]byte, 0, len(files))
					}
					quals[id] = append(quals[id], clone.Qual)
				}

				if i == 0 { // first file
					ids = append(ids, id)
				}
			}

			if !quiet {
				log.Infof("%d records loaded", n)
			}
		}

		var buf, bufQ bytes.Buffer
		var s, q []byte
		for _, id := range ids {
			buf.Reset()
			for _, s = range seqs[id] {
				buf.Write(s)
			}

			if isFastq {
				bufQ.Reset()
				for _, q = range quals[id] {
					bufQ.Write(q)
				}
				record, err = fastx.NewRecordWithQualWithoutValidation(seq.Unlimit,
					[]byte(id), []byte(id), []byte{}, []byte(buf.String()), []byte(bufQ.String()))

			} else {
				record, err = fastx.NewRecordWithoutValidation(seq.Unlimit,
					[]byte(id), []byte(id), []byte{}, []byte(buf.String()))
			}
			checkError(err)

			record.FormatToWriter(outfh, config.LineWidth)
		}
	},
}

func init() {
	RootCmd.AddCommand(concateCmd)

}
