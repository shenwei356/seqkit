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
	"sort"
	"sync"

	"github.com/brentp/xopen"
	"github.com/cznic/sortutil"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/spf13/cobra"
)

// locateCmd represents the locate command
var locateCmd = &cobra.Command{
	Use:   "locate",
	Short: "locate subsequences/motifs",
	Long: `locate subsequences/motifs

Motifs could be EITHER plain sequence containing "ACTGN" OR regular
expression like "A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)" for ORFs.
Degenerate bases like "RYMM.." are also supported by flag -d.

By default, motifs are treated as regular expression.
When flag -d given, regular expression may be wrong.
For example: "\w" will be wrongly converted to "\[AT]".
`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		chunkSize := config.ChunkSize
		bufferSize := config.BufferSize
		outFile := config.OutFile
		seq.AlphabetGuessSeqLenghtThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = true
		seq.ValidateWholeSeq = false
		seq.ValidSeqLengthThreshold = getFlagValidateSeqLength(cmd, "validate-seq-length")
		seq.ValidSeqThreads = config.Threads
		seq.ComplementThreads = config.Threads
		runtime.GOMAXPROCS(config.Threads)

		pattern := getFlagStringSlice(cmd, "pattern")
		patternFile := getFlagString(cmd, "pattern-file")
		degenerate := getFlagBool(cmd, "degenerate")
		ignoreCase := getFlagBool(cmd, "ignore-case")
		onlyPositiveStrand := getFlagBool(cmd, "only-positive-strand")

		if len(pattern) == 0 && patternFile == "" {
			checkError(fmt.Errorf("one of flags --pattern and --pattern-file needed"))
		}

		files := getFileList(args)

		// prepare pattern
		regexps := make(map[string]*regexp.Regexp)
		patterns := make(map[string][]byte)
		var s string
		if patternFile != "" {
			records, err := fastx.GetSeqsMap(patternFile, nil, 10, config.Threads, "")
			checkError(err)
			for name, record := range records {
				patterns[name] = record.Seq.Seq

				if degenerate {
					s = record.Seq.Degenerate2Regexp()
				} else {
					s = string(record.Seq.Seq)
				}

				if ignoreCase {
					s = "(?i)" + s
				}
				re, err := regexp.Compile(s)
				checkError(err)
				regexps[name] = re
			}
		} else {
			for _, p := range pattern {
				patterns[p] = []byte(p)

				if degenerate {
					pattern2seq, err := seq.NewSeq(alphabet, []byte(p))
					if err != nil {
						checkError(fmt.Errorf("it seems that flag -d is given, "+
							"but you provide regular expression instead of available %s sequence", alphabet))
					}
					s = pattern2seq.Degenerate2Regexp()
				} else {
					s = p
				}

				if ignoreCase {
					s = "(?i)" + s
				}
				re, err := regexp.Compile(s)
				checkError(err)
				regexps[p] = re
			}
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		outfh.WriteString("seqID\tpatternName\tpattern\tstrand\tstart\tend\tmatched\n")
		for _, file := range files {

			ch := make(chan LocationChunk, config.Threads)
			done := make(chan int)

			// receiver
			go func() {
				var id uint64 = 0
				chunks := make(map[uint64]LocationChunk)
				for chunk := range ch {
					if chunk.ID == id {
						for _, locationInfo := range chunk.Data {
							var s []byte
							for _, loc := range locationInfo.Locations {
								if locationInfo.Strand == "+" {
									s = locationInfo.Record.Seq.Seq[loc[0]:loc[1]]
								} else {
									s = locationInfo.Record.Seq.SubSeq(loc[0]+1, loc[1]).RevCom().Seq
								}
								outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
									locationInfo.Record.ID,
									locationInfo.PatternName,
									patterns[locationInfo.PatternName],
									locationInfo.Strand,
									loc[0]+1,
									loc[1],
									s))
							}
						}
						id++
					} else { // check bufferd result
						for true {
							if chunk, ok := chunks[id]; ok {
								for _, locationInfo := range chunk.Data {
									var s []byte
									for _, loc := range locationInfo.Locations {
										if locationInfo.Strand == "+" {
											s = locationInfo.Record.Seq.Seq[loc[0]:loc[1]]
										} else {
											s = locationInfo.Record.Seq.SubSeq(loc[0]+1, loc[1]).RevCom().Seq
										}
										outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
											locationInfo.Record.ID,
											locationInfo.PatternName,
											patterns[locationInfo.PatternName],
											locationInfo.Strand,
											loc[0]+1,
											loc[1],
											s))
									}
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
					sortedIDs := sortLocationChunkMapID(chunks)
					for _, id := range sortedIDs {
						chunk := chunks[id]
						for _, locationInfo := range chunk.Data {
							var s []byte
							for _, loc := range locationInfo.Locations {
								if locationInfo.Strand == "+" {
									s = locationInfo.Record.Seq.Seq[loc[0]:loc[1]]
								} else {
									s = locationInfo.Record.Seq.SubSeq(loc[0]+1, loc[1]).RevCom().Seq
								}
								outfh.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
									locationInfo.Record.ID,
									locationInfo.PatternName,
									patterns[locationInfo.PatternName],
									locationInfo.Strand,
									loc[0]+1,
									loc[1],
									s))
							}
						}
					}
				}

				done <- 1
			}()

			// producer and worker
			var wg sync.WaitGroup
			tokens := make(chan int, config.Threads)

			fastxReader, err := fastx.NewReader(alphabet, file, bufferSize, chunkSize, idRegexp)
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

					var locations []LocationInfo
					for _, record := range chunk.Data {
						for pName, re := range regexps {
							found := re.FindAllSubmatchIndex(record.Seq.Seq, -1)
							if len(found) > 0 {
								locations = append(locations, LocationInfo{record, pName, "+", found})
							}

							if onlyPositiveStrand {
								continue
							}
							seqRP := record.Seq.RevCom()
							found = re.FindAllSubmatchIndex(seqRP.Seq, -1)
							if len(found) > 0 {
								l := len(seqRP.Seq)
								tlocs := make([][]int, len(found))
								for i, loc := range found {
									tlocs[i] = []int{l - loc[1], l - loc[0]}
								}
								locations = append(locations, LocationInfo{record, pName, "-", tlocs})
							}
						}

					}
					ch <- LocationChunk{chunk.ID, locations}
				}(chunk)
			}
			wg.Wait()
			close(ch)
			<-done
		}
	},
}

// LocationChunk is LocationChunk
type LocationChunk struct {
	ID   uint64
	Data []LocationInfo
}

// LocationInfo is LocationInfo
type LocationInfo struct {
	Record      *fastx.Record
	PatternName string
	Strand      string
	Locations   [][]int
}

func sortLocationChunkMapID(chunks map[uint64]LocationChunk) sortutil.Uint64Slice {
	ids := make(sortutil.Uint64Slice, len(chunks))
	i := 0
	for id := range chunks {
		ids[i] = id
		i++
	}
	sort.Sort(ids)
	return ids
}

func init() {
	RootCmd.AddCommand(locateCmd)

	locateCmd.Flags().StringSliceP("pattern", "p", []string{""}, "search pattern/motif (multiple values supported)")
	locateCmd.Flags().StringP("pattern-file", "f", "", "pattern/motif file (FASTA format)")
	locateCmd.Flags().BoolP("degenerate", "d", false, "pattern/motif contains degenerate base")
	locateCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
	locateCmd.Flags().BoolP("only-positive-strand", "P", false, "only search at positive strand")
	locateCmd.Flags().IntP("validate-seq-length", "V", 10000, "length of sequence to validate (0 for whole seq)")

}
