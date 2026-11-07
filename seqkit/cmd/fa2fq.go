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
	"bytes"
	"fmt"
	"io"
	"runtime"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// fa2fqCmd represents the fq2fa command
var fa2fqCmd = &cobra.Command{
	GroupID: "format",

	Use:   "fa2fq",
	Short: "retrieve corresponding FASTQ records by a FASTA file",
	Long: `retrieve corresponding FASTQ records by a FASTA file

Attention:
  1. We assume the FASTA file comes from the FASTQ file,
     so they share sequence names (not just IDs), and sequences in FASTA
     should be subseq of sequences in FASTQ file.
  2. The complete FASTA reference is checked before output is written.
     The FASTA reference and FASTQ input cannot both be stdin.
  3. Protein and unrestricted alphabets are searched only in their original orientation.

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		// lineWidth := config.LineWidth
		outFile := config.OutFile
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		fileFasta := getFlagString(cmd, "fasta-file")
		if fileFasta == "" {
			checkError(fmt.Errorf("flag -f (--fasta-file) needed"))
		}
		onlyPositiveStrand := getFlagBool(cmd, "only-positive-strand")

		files := getFileListFromArgsAndFile(cmd, args, !config.SkipFileCheck, "infile-list", !config.SkipFileCheck, getFlagString(cmd, "fasta-file"))
		checkOutputFiles(files, []string{outFile}, !config.SkipFileCheck)
		checkOutputFiles([]string{fileFasta}, []string{outFile}, true)

		// Read the complete reference before opening output; partial maps are unsafe.
		referenceRecords, err := readRecordsChecked(fileFasta, seq.Unlimit, "")
		checkError(err)
		records := make(map[string]*fastx.Record, len(referenceRecords))
		for _, record := range referenceRecords {
			records[string(record.Name)] = record
		}
		referenceRecords = nil
		if len(records) == 0 {
			checkError(fmt.Errorf("no sequences found in fasta file: %s", fileFasta))
		} else {
			log.Infof("%d sequences loaded", len(records))
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer closeOutput(outfh)

		var record *fastx.Record
		var ok bool
		var fa *fastx.Record
		var i, j int
		checkingFastq := true
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

				if checkingFastq && !fastxReader.IsFastq {
					checkError(fmt.Errorf("this command only works for FASTQ format"))
				}

				fa, ok = records[string(record.Name)]
				if !ok {
					continue
				}

				i = bytes.Index(record.Seq.Seq, fa.Seq.Seq)
				if i >= 0 {
					j = i + len(fa.Seq.Seq)
					outfh.Write(_mark_fastq)
					outfh.Write(record.ID)
					outfh.Write(_mark_newline)
					outfh.Write(record.Seq.Seq[i:j])
					outfh.Write(_mark_newline)
					outfh.Write(_mark_plus_newline)
					outfh.Write(record.Seq.Qual[i:j])
					outfh.Write(_mark_newline)
					continue
				}

				// Only nucleotide alphabets have a complementary strand.
				if onlyPositiveStrand || record.Seq.Alphabet == seq.Protein || record.Seq.Alphabet == seq.Unlimit {
					continue
				}

				record.Seq.RevComInplace()

				i = bytes.Index(record.Seq.Seq, fa.Seq.Seq)
				if i >= 0 {
					j = i + len(fa.Seq.Seq)
					outfh.Write(_mark_fastq)
					outfh.Write(record.ID)
					outfh.Write(_mark_newline)
					outfh.Write(record.Seq.Seq[i:j])
					outfh.Write(_mark_newline)
					outfh.Write(_mark_plus_newline)
					outfh.Write(record.Seq.Qual[i:j])
					outfh.Write(_mark_newline)
				}
			}
			fastxReader.Close()
		}
	},
}

func init() {
	RootCmd.AddCommand(fa2fqCmd)

	fa2fqCmd.Flags().StringP("fasta-file", "f", "", "FASTA file")
	fa2fqCmd.Flags().BoolP("only-positive-strand", "P", false, "only search on positive strand")
}
