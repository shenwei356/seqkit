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
	"runtime"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/util/stringutil"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// headGenomeCmd represents the head command
var headGenomeCmd = &cobra.Command{
	Use:   "head-genome",
	Short: "print sequences of the first genome with common prefixes in name",
	Long: `print sequences of the first genome with common prefixes in name

For a FASTA file containing multiple contigs of strains (see example below),
these's no list of IDs available for retrieving sequences of a certain strain,
while descriptions of each strain share the same prefix.

This command is used to restrieve sequences of the first strain,
i.e., "Vibrio cholerae strain M29".

>NZ_JFGR01000001.1 Vibrio cholerae strain M29 Contig_1, whole genome shotgun sequence
>NZ_JFGR01000002.1 Vibrio cholerae strain M29 Contig_2, whole genome shotgun sequence
>NZ_JFGR01000003.1 Vibrio cholerae strain M29 Contig_3, whole genome shotgun sequence
>NZ_JSTP01000001.1 Vibrio cholerae strain 2012HC-12 NODE_79, whole genome shotgun sequence
>NZ_JSTP01000002.1 Vibrio cholerae strain 2012HC-12 NODE_78, whole genome shotgun sequence

Attention:

  1. Sequences in file should be well organized.

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

		minWords := getFlagPositiveInt(cmd, "mini-common-words")

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var record *fastx.Record
		var prefixes, words []string
		var i, N int
		var nSharedWords, pNSharedWords int

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

				if len(record.Desc) == 0 {
					checkError(fmt.Errorf("no description: %s", record.ID))
				}

				if prefixes == nil { // first record
					record.FormatToWriter(outfh, config.LineWidth)

					prefixes = stringutil.Split(string(record.Desc), "\t ")
					continue
				}

				words = stringutil.Split(string(record.Desc), "\t ")
				if len(words) < len(prefixes) {
					N = len(words)
				} else {
					N = len(prefixes)
				}

				nSharedWords = 0
				for i = 0; i < N; i++ {
					if words[i] != prefixes[i] {
						break
					}
					nSharedWords++
				}

				if nSharedWords < minWords {
					return
				}

				if pNSharedWords == 0 { // 2nd sequence
					pNSharedWords = nSharedWords
				} else if nSharedWords != pNSharedWords { // number of shared words changed
					return
				}

				record.FormatToWriter(outfh, config.LineWidth)
			}
			fastxReader.Close()

			config.LineWidth = lineWidth
		}
	},
}

func init() {
	RootCmd.AddCommand(headGenomeCmd)

	headGenomeCmd.Flags().IntP("mini-common-words", "m", 1, "minimal shared prefix words")
}
