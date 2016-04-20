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
	"strings"
	"sync"

	"github.com/brentp/xopen"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/breader"
	"github.com/spf13/cobra"
)

// grepCmd represents the extract command
var grepCmd = &cobra.Command{
	Use:   "grep",
	Short: "search sequences by pattern(s) of name or sequence motifs",
	Long: `search sequences by pattern(s) of name or sequence motifs

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		chunkSize := config.ChunkSize
		threads := config.Threads
		lineWidth := config.LineWidth
		outFile := config.OutFile
		seq.AlphabetGuessSeqLenghtThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(threads)

		pattern := getFlagStringSlice(cmd, "pattern")
		patternFile := getFlagString(cmd, "pattern-file")
		useRegexp := getFlagBool(cmd, "use-regexp")
		deleteMatched := getFlagBool(cmd, "delete-matched")
		invertMatch := getFlagBool(cmd, "invert-match")
		bySeq := getFlagBool(cmd, "by-seq")
		byName := getFlagBool(cmd, "by-name")
		ignoreCase := getFlagBool(cmd, "ignore-case")
		degenerate := getFlagBool(cmd, "degenerate")

		if len(pattern) == 0 && patternFile == "" {
			checkError(fmt.Errorf("one of flags -p (--pattern) and -f (--pattern-file) needed"))
		}
		if useRegexp && degenerate {
			checkError(fmt.Errorf("could not give both flags -d (--degenerat) and -r (--use-regexp)"))
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
					p := data.(string)
					if degenerate || useRegexp {
						if degenerate {
							pattern2seq, err := seq.NewSeq(alphabet, []byte(p))
							if err != nil {
								checkError(fmt.Errorf("it seems that flag -d is given, "+
									"but you provide regular expression instead of available %s sequence", alphabet))
							}
							p = pattern2seq.Degenerate2Regexp()
						}
						if ignoreCase {
							p = "(?i)" + p
						}
						r, err := regexp.Compile(p)
						checkError(err)
						patterns[p] = r
					} else {
						if ignoreCase {
							patterns[strings.ToLower(p)] = nil
						} else {
							patterns[p] = nil
						}
					}
				}
			}
		} else {
			if degenerate || useRegexp {
				for _, p := range pattern {
					if degenerate {
						pattern2seq, err := seq.NewSeq(alphabet, []byte(p))
						if err != nil {
							checkError(fmt.Errorf("it seems that flag -d is given, "+
								"but you provide regular expression instead of available %s sequence", alphabet))
						}
						p = pattern2seq.Degenerate2Regexp()
					}
					if ignoreCase {
						p = "(?i)" + p
					}

					re, err := regexp.Compile(p)
					checkError(err)
					patterns[p] = re
				}
			} else {
				for _, p := range pattern {
					if ignoreCase {
						patterns[strings.ToLower(p)] = nil
					} else {
						patterns[p] = nil
					}
				}
			}
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		for _, file := range files {

			ch := make(chan fastx.RecordChunk, threads)
			done := make(chan int)

			// receiver
			go func() {
				var id uint64 = 0
				chunks := make(map[uint64]fastx.RecordChunk)
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
					sortedIDs := sortRecordChunkMapID(chunks)
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

			fastxReader, err := fastx.NewReader(alphabet, file, threads, chunkSize, idRegexp)
			checkError(err)
			for chunk := range fastxReader.Ch {
				checkError(chunk.Err)
				tokens <- 1
				wg.Add(1)

				go func(chunk fastx.RecordChunk) {
					defer func() {
						wg.Done()
						<-tokens
					}()

					var subject []byte
					var hit bool
					var chunkData []*fastx.Record
					for _, record := range chunk.Data {

						if byName {
							subject = record.Name
						} else if bySeq {
							subject = record.Seq.Seq
						} else {
							subject = record.ID
						}

						hit = false
						if degenerate || useRegexp {
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
							k := string(subject)
							if useRegexp {
								k = strings.ToLower(k)
							}
							if _, ok := patterns[k]; ok {
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
					ch <- fastx.RecordChunk{ID: chunk.ID, Data: chunkData, Err: nil}
				}(chunk)
			}
			wg.Wait()
			close(ch)
			<-done
		}
	},
}

func init() {
	RootCmd.AddCommand(grepCmd)

	grepCmd.Flags().StringSliceP("pattern", "p", []string{""}, "search pattern (multiple values supported)")
	grepCmd.Flags().StringP("pattern-file", "f", "", "pattern file")
	grepCmd.Flags().BoolP("use-regexp", "r", false, "patterns are regular expression")
	grepCmd.Flags().BoolP("delete-matched", "", false, "delete matched pattern to speedup")
	grepCmd.Flags().BoolP("invert-match", "v", false, "invert the sense of matching, to select non-matching records")
	grepCmd.Flags().BoolP("by-name", "n", false, "match by full name instead of just id")
	grepCmd.Flags().BoolP("by-seq", "s", false, "match by seq")
	grepCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
	grepCmd.Flags().BoolP("degenerate", "d", false, "pattern/motif contains degenerate base")
}
