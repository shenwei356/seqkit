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
	"runtime"
	"strings"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
	"github.com/twotwotwo/sorts/sortutil"
)

// concateCmd represents the concatenate command
var concateCmd = &cobra.Command{
	Use:     "concat",
	Aliases: []string{"concate"},
	Short:   "concatenate sequences with same ID from multiple files",
	Long: `concatenate sequences with same ID from multiple files

Attentions:
   1. By default, only sequences with IDs that appear in all files are outputted.
      use -f/--full to output all sequences.
   2. If there are more than one sequences of the same ID, we output the Cartesian
      product of sequences.
   3. Description are also concatenated with a separator (-s/--separator).
   4. Order of sequences with different IDs are random.

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		lineWidth := config.LineWidth
		outFile := config.OutFile
		quiet := config.Quiet
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		full := getFlagBool(cmd, "full")
		separator := getFlagString(cmd, "separator")

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		if len(files) < 2 {
			checkError(errors.New("at least 2 files needed"))
		}

		// id -> seqs
		seqs0 := make([]map[string]*[]*fastx.Record, 0, 1024)

		// id -> file_idx -> #seqs
		ids0 := make(map[string]*map[int]int, 1024)

		// var record *fastx.Record

		var isFastq bool
		first := true

		var idxFile int
		for _, file := range files {
			if !quiet {
				log.Infof("read file: %s", file)
			}

			seqs, err := fastx.GetSeqs(file, alphabet, config.Threads, 1024, idRegexp)
			checkError(err)

			if len(seqs) == 0 {
				log.Warningf("no seqs found in file: %s", file)
				continue
			}

			if !quiet {
				log.Infof("%d records loaded from file: %s", len(seqs), file)
			}

			if first {
				isFastq = len(seqs[0].Seq.Qual) > 0
				if isFastq {
					fastx.ForcelyOutputFastq = true
				}
				first = false
			} else {
				if len(seqs[0].Seq.Qual) > 0 != isFastq {
					checkError(fmt.Errorf("concatenating FASTA and FASTQ is not allowed"))
				}
			}

			seqsM := make(map[string]*[]*fastx.Record, len(seqs))
			var ok bool
			var id string
			var list *[]*fastx.Record
			var m *map[int]int
			for _, _seq := range seqs {
				id = string(_seq.ID)

				if list, ok = seqsM[id]; !ok {
					tmp := make([]*fastx.Record, 0, 1)
					list = &tmp
					seqsM[id] = list
				}
				*list = append(*list, _seq)

				if m, ok = ids0[id]; !ok {
					tmp2 := make(map[int]int, len(files))
					m = &tmp2
					ids0[id] = m
				}
				(*m)[idxFile]++
			}

			seqs0 = append(seqs0, seqsM)

			idxFile++
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		n := len(files)

		var bufName, bufSeq, bufQual bytes.Buffer
		var buffer *bytes.Buffer
		var _r *fastx.Record
		var text []byte
		descs := make([]string, len(files))
		var hasDesc bool
		for id, m := range ids0 {
			if !full && len(*m) != n { // ignore these ids not appeared in all files
				continue
			}

			if len(*m) == 1 { // only appear in one file, just print it
				for idxFile = range *m {
					for _, r := range *seqs0[idxFile][id] {
						(*r).FormatToWriter(outfh, lineWidth)
					}
				}
				continue
			}

			// sort file indexes
			idxes := make([]int, 0, len(*m))
			for idx := range *m {
				idxes = append(idxes, idx)
			}
			sortutil.Ints(idxes)

			lists := make([][]int, 0, len(*m))
			for _, idx := range idxes {
				vs := make([]int, 0, (*m)[idx])
				for i := 0; i < (*m)[idx]; i++ {
					vs = append(vs, i)
				}
				lists = append(lists, vs)
			}

			prod := product(lists...)

			for _, locs := range prod {
				bufName.Reset()
				bufSeq.Reset()
				bufQual.Reset()
				descs = descs[:0]
				hasDesc = false

				bufName.WriteString(id)

				for _i, _j := range locs {
					_r = (*seqs0[idxes[_i]][id])[_j]

					if len(_r.Desc) > 0 {
						hasDesc = true
					}
					descs = append(descs, string(_r.Desc))

					bufSeq.Write(_r.Seq.Seq)
					bufQual.Write(_r.Seq.Qual)
				}

				if isFastq {
					outfh.Write(_mark_fastq)
					outfh.Write(bufName.Bytes())
					if hasDesc {
						outfh.WriteString(" ")
						outfh.WriteString(strings.Join(descs, separator))
					}
					outfh.Write(_mark_newline)

					outfh.Write(bufSeq.Bytes())
					outfh.Write(_mark_newline)

					outfh.Write(_mark_plus_newline)

					outfh.Write(bufQual.Bytes())
					outfh.Write(_mark_newline)
				} else {
					outfh.Write(_mark_fasta)
					outfh.Write(bufName.Bytes())
					if hasDesc {
						outfh.WriteString(" ")
						outfh.WriteString(strings.Join(descs, separator))
					}
					outfh.Write(_mark_newline)

					text, buffer = wrapByteSlice(bufSeq.Bytes(), config.LineWidth, buffer)
					outfh.Write(text)
					outfh.Write(_mark_newline)
				}
			}

		}
	},
}

func init() {
	RootCmd.AddCommand(concateCmd)

	concateCmd.Flags().BoolP("full", "f", false, "keep all sequences, like full/outter join")
	concateCmd.Flags().StringP("separator", "s", "|", "separator for descriptions of records with the same ID")
}

func product(lists ...[]int) (results [][]int) {
	if len(lists) == 0 {
		return nil
	}
	if len(lists) == 1 {
		results = [][]int{lists[0]}
		return results
	}

	results = make([][]int, 0, 8)
	for _, list := range lists {
		if len(results) == 0 {
			for _, e := range list {
				results = append(results, []int{e})
			}
		} else {
			results2 := make([][]int, 0, len(results)*len(list))
			for _, l := range results {
				for _, e := range list {
					l2 := make([]int, len(l)+1)
					copy(l2[:len(l)], l)
					l2[len(l2)-1] = e
					results2 = append(results2, l2)
				}
			}
			results = results2
		}
	}

	return results
}
