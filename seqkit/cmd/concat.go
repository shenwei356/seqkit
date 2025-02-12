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
)

// concateCmd represents the concatenate command
var concateCmd = &cobra.Command{
	GroupID: "edit",

	Use:     "concat",
	Aliases: []string{"concate"},
	Short:   "concatenate sequences with the same ID from multiple files",
	Long: `concatenate sequences with same ID from multiple files

Attention:
   1. By default, only sequences with IDs that appear in all files are outputted.
      use -f/--full to output all sequences.
      * If you are processing multiple-sequence-alignment results, you can use
        -F/--fill to fill with N bases/residues for IDs missing in some files when
        using -f/--full.
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
		fillBase := getFlagString(cmd, "fill")
		fill := fillBase != ""
		separator := getFlagString(cmd, "separator")

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		if len(files) < 2 {
			checkError(errors.New("at least 2 files needed"))
		}

		// id -> seqs
		seqs0 := make([]map[string]*[]*fastx.Record, 0, 1024)

		// id -> file_idx -> #seqs
		ids0 := make(map[string]*map[int]int, 1024)

		// lengths of sequence in all files
		seqLens := make([]int, len(files))

		// var record *fastx.Record

		var isFastq bool
		first := true

		var idxFile int
		for i, file := range files {
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
			for j, _seq := range seqs {
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

				if !fill {
					continue
				}
				if j == 0 {
					seqLens[i] = len(_seq.Seq.Seq)
				} else if seqLens[i] != len(_seq.Seq.Seq) {
					log.Warningf("records with different lengths (%d, %d) detected in file '%s' , -F/--fill is disabled",
						seqLens[i], len(_seq.Seq.Seq), file)
					fill = false
				}
			}

			seqs0 = append(seqs0, seqsM)

			idxFile++
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		n := len(files)

		lists := make([][]int, 0, 1024)
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

			lists = lists[:0]
			for idx := 0; idx < n; idx++ {
				if !full && (*m)[idx] == 0 {
					continue
				}

				// use -1 to mark no exist
				if (*m)[idx] == 0 {
					lists = append(lists, []int{-1})
					continue
				}

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
					if _j < 0 { // no exist in this file
						if !fill {
							continue
						}
						bufSeq.WriteString(strings.Repeat(fillBase, seqLens[_i]))
						if isFastq {
							bufQual.WriteString(strings.Repeat("B", seqLens[_i]))
						}
						continue
					}

					_r = (*seqs0[_i][id])[_j]

					if len(_r.Desc) > 0 {
						hasDesc = true
					}
					descs = append(descs, string(_r.Desc))

					bufSeq.Write(_r.Seq.Seq)
					if isFastq {
						bufQual.Write(_r.Seq.Qual)
					}
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

					text, buffer = wrapByteSlice(bufSeq.Bytes(), lineWidth, buffer)
					outfh.Write(text)
					outfh.Write(_mark_newline)
				}
			}

		}
	},
}

func init() {
	RootCmd.AddCommand(concateCmd)

	concateCmd.Flags().BoolP("full", "f", false, "keep all sequences, like full/outer join")
	concateCmd.Flags().StringP("fill", "F", "", "fill with N bases/residues for IDs missing in some files when using -f/--full")
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
