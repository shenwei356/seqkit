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
	"regexp"
	"sync"

	"github.com/brentp/xopen"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fasta"
	"github.com/spf13/cobra"
)

// locateCmd represents the extract command
var locateCmd = &cobra.Command{
	Use:   "locate",
	Short: "locate subseq/motif",
	Long: `locate subseq/motif

motifs could be EITHER plain sequence containing "ACTGN" OR regular
expression like "A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)" for ORFs.
Degenerate bases like "RYMM.." are also supported by flag -d.

In default, motifs are treated as regular expression.
When flag -d given, regular expression may be wrong.
For example: "\w" -> "\[AT]".
`,
	Run: func(cmd *cobra.Command, args []string) {
		alphabet := getAlphabet(cmd, "seq-type")
		idRegexp := getFlagString(cmd, "id-regexp")
		chunkSize := getFlagInt(cmd, "chunk-size")
		threads := getFlagInt(cmd, "threads")
		lineWidth := getFlagInt(cmd, "line-width")
		outFile := getFlagString(cmd, "out-file")

		pattern := getFlagStringSlice(cmd, "pattern")
		patternFile := getFlagString(cmd, "pattern-file")
		degenerate := getFlagBool(cmd, "degenerate")

		if len(pattern) == 0 && patternFile == "" {
			checkError(fmt.Errorf("one of flags --pattern and --pattern-file needed"))
		}

		files := getFileList(args)

		// prepare pattern
		regexps := make(map[string]*regexp.Regexp)
		sequences := make(map[string]*seq.Seq)
		var s string
		if patternFile != "" {
			sequences := getSeqsAsMap(seq.Unlimit, patternFile)
			for name, sequence := range sequences {
				if degenerate {
					s = sequence.Degenerate2Regexp()
				} else {
					s = string(sequence.Seq)
				}

				re, err := regexp.Compile(s)
				checkError(err)
				regexps[name] = re
			}
		} else {
			for _, p := range pattern {
				sequence, err := seq.NewSeq(alphabet, []byte(p))
				checkError(err)

				sequences[p] = sequence

				if degenerate {
					s = sequence.Degenerate2Regexp()
				} else {
					s = string(sequence.Seq)
				}

				re, err := regexp.Compile(s)
				checkError(err)
				regexps[p] = re
			}
		}
		fmt.Println(sequences, regexps)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		for _, file := range files {

			ch := make(chan fasta.FastaRecordChunk, threads)
			done := make(chan int)

			// receiver
			var j int
			go func() {
				var id uint64 = 0
				chunks := make(map[uint64]fasta.FastaRecordChunk)
				for chunk := range ch {
					checkError(chunk.Err)

					if chunk.ID == id {
						for _, record := range chunk.Data {
							j++
							outfh.WriteString(fmt.Sprintf(">%s\n%s\n", record.Name, record.FormatSeq(lineWidth)))
						}
						id++
					} else { // check bufferd result
						for true {
							if chunk, ok := chunks[id]; ok {
								for _, record := range chunk.Data {
									outfh.WriteString(fmt.Sprintf(">%s\n%s\n", record.Name, record.FormatSeq(lineWidth)))
								}
								id++
								delete(chunks, chunk.ID)
							} else {
								break
							}
						}
						chunks[chunk.ID] = chunk
					}
				}

				if len(chunks) > 0 {
					sortedIDs := sortChunksID(chunks)
					for _, id := range sortedIDs {
						chunk := chunks[id]
						for _, record := range chunk.Data {
							j++
							outfh.WriteString(fmt.Sprintf(">%s\n%s\n", record.Name, record.FormatSeq(lineWidth)))
						}
					}
				}

				done <- 1
			}()

			// producer and worker
			var wg sync.WaitGroup
			tokens := make(chan int, threads)

			fastaReader, err := fasta.NewFastaReader(alphabet, file, chunkSize, threads, idRegexp)
			checkError(err)
			for chunk := range fastaReader.Ch {
				checkError(chunk.Err)
				tokens <- 1
				wg.Add(1)

				go func(chunk fasta.FastaRecordChunk) {
					defer func() {
						wg.Done()
						<-tokens
					}()

					// for _, record := range chunk.Data {
					//
					// }
				}(chunk)
			}
			wg.Wait()
			close(ch)
			<-done
		}
	},
}

func init() {
	RootCmd.AddCommand(locateCmd)

	locateCmd.Flags().StringSliceP("pattern", "p", []string{""}, "search pattern/motif (multiple values supported)")
	locateCmd.Flags().StringP("pattern-file", "f", "", "pattern/motif file (FASTA format)")
	locateCmd.Flags().BoolP("degenerate", "d", false, "pattern/motif contains degenerate base")
}
