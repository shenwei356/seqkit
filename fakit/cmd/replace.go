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
	"runtime"
	"sync"

	"github.com/brentp/xopen"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fasta"
	"github.com/spf13/cobra"
)

// replaceCmd represents the extract command
var replaceCmd = &cobra.Command{
	Use:   "replace",
	Short: "replace name/sequence/by regular expression",
	Long: `replace name/sequence/by regular expression.

Note that the replacement supports capture variables.
e.g. $1 represents the text of the first submatch.
ATTENTION: use SINGLE quote NOT double quotes in *nix OS.

Examples: Adding space to all bases.

    fakit replace -p "(.)" -r '$1 ' -s

Or use the \ escape character.

    fakit replace -p "(.)" -r "\$1 " -s

more on: http://shenwei356.github.io/fakit/usage/#replace

`,
	Run: func(cmd *cobra.Command, args []string) {
		alphabet := getAlphabet(cmd, "seq-type")
		idRegexp := getIDRegexp(cmd, "id-regexp")
		chunkSize := getFlagPositiveInt(cmd, "chunk-size")
		threads := getFlagPositiveInt(cmd, "threads")
		lineWidth := getFlagNonNegativeInt(cmd, "line-width")
		outFile := getFlagString(cmd, "out-file")
		seq.AlphabetGuessSeqLenghtThreshold = getFlagalphabetGuessSeqLength(cmd, "alphabet-guess-seq-length")
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(threads)

		pattern := getFlagString(cmd, "pattern")
		replacement := []byte(getFlagString(cmd, "replacement"))
		bySeq := getFlagBool(cmd, "by-seq")
		// byName := getFlagBool(cmd, "by-name")
		ignoreCase := getFlagBool(cmd, "ignore-case")

		if pattern == "" {
			checkError(fmt.Errorf("flags -p (--pattern) needed"))
		}

		p := pattern
		if ignoreCase {
			p = "(?i)" + p
		}
		patternRegexp, err := regexp.Compile(p)
		checkError(err)

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		for _, file := range files {

			ch := make(chan fasta.FastaRecordChunk, threads)
			done := make(chan int)

			// receiver
			go func() {
				var id uint64 = 0
				chunks := make(map[uint64]fasta.FastaRecordChunk)
				for chunk := range ch {
					checkError(chunk.Err)

					if chunk.ID == id {
						for _, record := range chunk.Data {
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
					sortedIDs := sortFastaRecordChunkMapID(chunks)
					for _, id := range sortedIDs {
						chunk := chunks[id]
						for _, record := range chunk.Data {
							outfh.WriteString(fmt.Sprintf(">%s\n%s\n", record.Name, record.FormatSeq(lineWidth)))
						}
					}
				}

				done <- 1
			}()

			// producer and worker
			var wg sync.WaitGroup
			tokens := make(chan int, threads)

			fastaReader, err := fasta.NewFastaReader(alphabet, file, threads, chunkSize, idRegexp)
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

					var chunkData []*fasta.FastaRecord
					for _, record := range chunk.Data {
						if bySeq {
							record.Seq.Seq = patternRegexp.ReplaceAll(record.Seq.Seq, replacement)
						} else {
							record.Name = patternRegexp.ReplaceAll(record.Name, replacement)
						}
						chunkData = append(chunkData, record)
					}
					ch <- fasta.FastaRecordChunk{ID: chunk.ID, Data: chunkData, Err: nil}
				}(chunk)
			}
			wg.Wait()
			close(ch)
			<-done
		}
	},
}

func init() {
	RootCmd.AddCommand(replaceCmd)
	replaceCmd.Flags().StringP("pattern", "p", "", "search regular expression")
	replaceCmd.Flags().StringP("replacement", "r", "",
		"replacement. supporting capture variables. "+
			" e.g. $1 represents the text of the first submatch. "+
			"ATTENTION: use SINGLE quote NOT double quotes in *nix OS or "+
			"use the \\ escape character.")
	// replaceCmd.Flags().BoolP("by-name", "n", false, "replace full name instead of just id")
	replaceCmd.Flags().BoolP("by-seq", "s", false, "replace seq")
	replaceCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
}
