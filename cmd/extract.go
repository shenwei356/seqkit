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
	"github.com/shenwei356/bio/seqio/fasta"
	"github.com/shenwei356/breader"
	"github.com/spf13/cobra"
)

// extractCmd represents the extract command
var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "extract sequences by patterns/motifs",
	Long: `extract sequence by patterns/motifs

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
		useRegexp := getFlagBool(cmd, "use-regexp")
		deleteMatched := getFlagBool(cmd, "delete-matched")
		invertMatch := getFlagBool(cmd, "invert-match")
		bySeq := getFlagBool(cmd, "by-seq")
		byName := getFlagBool(cmd, "by-name")

		if len(pattern) == 0 && patternFile == "" {
			checkError(fmt.Errorf("one of flags -p (--pattern) and -f (--pattern-file) needed"))
		}

		files := getFileList(args)

		// prepare pattern
		patterns := make(map[string]*regexp.Regexp)
		if patternFile != "" {
			reader, err := breader.NewDefaultBufferedReader(patternFile)
			checkError(err)
			for chunk := range reader.Ch {
				checkError(chunk.Err)
				for _, data := range chunk.Data {
					pattern := data.(string)
					if useRegexp {
						r, err := regexp.Compile(pattern)
						checkError(err)
						patterns[pattern] = r
					} else {
						patterns[pattern] = nil
					}
				}
			}
		} else {
			if useRegexp {
				for _, p := range pattern {
					re, err := regexp.Compile(p)
					checkError(err)
					patterns[p] = re
				}
			} else {
				for _, p := range pattern {
					patterns[p] = nil
				}
			}
		}

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

					var subject []byte
					var hit bool
					var chunkData []*fasta.FastaRecord
					for _, record := range chunk.Data {

						if byName {
							subject = record.Name
						} else if bySeq {
							subject = record.Seq.Seq
						} else {
							subject = record.ID
						}

						hit = false
						if useRegexp {
							for pattern, re := range patterns {
								if re.Match(subject) {
									hit = true
									if deleteMatched {
										delete(patterns, pattern)
									}
									break
								}
							}
						} else {
							if _, ok := patterns[string(subject)]; ok {
								hit = true
							}
						}

						if invertMatch {
							if hit {
								continue
							}
						} else {
							if !hit {
								continue
							}
						}

						chunkData = append(chunkData, record)
					}
					ch <- fasta.FastaRecordChunk{chunk.ID, chunkData, nil}
				}(chunk)
			}
			wg.Wait()
			close(ch)
			<-done
		}
	},
}

func init() {
	RootCmd.AddCommand(extractCmd)

	extractCmd.Flags().StringSliceP("pattern", "p", []string{""}, "search pattern (multiple values supported)")
	extractCmd.Flags().StringP("pattern-file", "f", "", "pattern file")
	extractCmd.Flags().BoolP("use-regexp", "r", false, "patterns are regular expression")
	extractCmd.Flags().BoolP("delete-matched", "d", false, "delete matched pattern to speedup")
	extractCmd.Flags().BoolP("invert-match", "v", false, "invert the sense of matching, to select non-matching records")
	extractCmd.Flags().BoolP("by-name", "n", false, "match by full name instead of just id")
	extractCmd.Flags().BoolP("by-seq", "s", false, "match by seq")
}
