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
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
	"github.com/twotwotwo/sorts/sortutil"
)

// sumCmd represents the sum command
var sumCmd = &cobra.Command{
	Use:   "sum",
	Short: "compute message digest for all sequences in FASTA/Q files",
	Long: `compute message digest for all sequences in FASTA/Q files

Attentions:
  1. Sequence headers and qualities are skipped, only sequences matter.
  2. The order of sequences records does not matter.
  3. Circular complete genomes are supported with the flag -c/--circular.
     - The same genomes with different start positions or in reverse
       complement strand will not affect the result.
     - The message digest would change with different values of k-mer size.
  4. Multiple files are processed in parallel (-j/--threads).

Method:
  1. Converting the sequences to low cases, optionally removing gaps (-g).
  2. Computing the hash (xxhash) for all sequences or k-mers of a circular
     complete genome (-c/--circular).
  3. Sorting all hash values, for ignoring the order of sequences.
  4. Computing MD5 digest from the hash values, sequences length, and
     the number of sequences.

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

		basename := getFlagBool(cmd, "basename")
		circular := getFlagBool(cmd, "circular")
		k := getFlagPositiveInt(cmd, "kmer-size")
		removeGaps := getFlagBool(cmd, "remove-gaps")
		gapLetters := getFlagString(cmd, "gap-letters")
		all := getFlagBool(cmd, "all")

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		tokens := make(chan int, config.Threads)
		done := make(chan int)
		var wg sync.WaitGroup

		type Aresult struct {
			ok     bool
			id     uint64
			result *SumResult
		}
		ch := make(chan *Aresult, config.Threads)

		go func() {
			m := make(map[uint64]*Aresult, config.Threads)
			var id, _id uint64
			var ok bool
			var _r *Aresult

			id = 1
			for r := range ch {
				_id = r.id

				if _id == id { // right there
					if r.ok {
						if all {
							fmt.Fprintf(outfh, "%s\t%s\t%d\t%d\n", r.result.Digest, r.result.File, r.result.SeqNum, r.result.SeqLen)
						} else {
							fmt.Fprintf(outfh, "%s\t%s\n", r.result.Digest, r.result.File)
						}

						outfh.Flush()
					}
					id++
					continue
				}

				m[_id] = r // save for later check

				if _r, ok = m[id]; ok { // check buffered
					if _r.ok {
						if all {
							fmt.Fprintf(outfh, "%s\t%s\t%d\t%d\n", _r.result.Digest, _r.result.File, _r.result.SeqNum, _r.result.SeqLen)
						} else {
							fmt.Fprintf(outfh, "%s\t%s\n", _r.result.Digest, _r.result.File)
						}
						outfh.Flush()
					}
					delete(m, id)
					id++
				}
			}

			if len(m) > 0 {
				ids := make([]uint64, len(m))
				i := 0
				for _id = range m {
					ids[i] = _id
					i++
				}
				sortutil.Uint64s(ids)
				for _, _id = range ids {
					_r = m[_id]

					if _r.ok {
						if all {
							fmt.Fprintf(outfh, "%s\t%s\t%d\t%d\n", _r.result.Digest, _r.result.File, _r.result.SeqNum, _r.result.SeqLen)
						} else {
							fmt.Fprintf(outfh, "%s\t%s\n", _r.result.Digest, _r.result.File)
						}
						outfh.Flush()
					}
				}
			}
			done <- 1
		}()

		var id uint64
		for _, file := range files {
			tokens <- 1
			wg.Add(1)
			id++

			go func(file string, id uint64) {
				defer func() {
					<-tokens
					wg.Done()
				}()

				var n int    // number of sequences
				var lens int // lengths of all seqs
				var h uint64
				hashes := make([]uint64, 0, 1024)

				var record *fastx.Record
				var fastxReader *fastx.Reader
				var _seq *seq.Seq

				fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
				// checkError(err)
				if err != nil {
					ch <- &Aresult{
						id:     id,
						ok:     false,
						result: nil,
					}
					log.Warningf(fmt.Sprintf("skip file: %s: %s", file, err))
					return
				}

				if circular {
					// var ok bool

					// var iter *sketches.Iterator
					var rc *seq.Seq
					var s, src []byte
					var h2 uint64
					var i, j, e, l, end, originalLen int
					for {
						record, err = fastxReader.Read()
						if err != nil {
							if err == io.EOF {
								break
							}
							// checkError(err)
							// break
							ch <- &Aresult{
								id:     id,
								ok:     false,
								result: nil,
							}
							log.Warningf(fmt.Sprintf("skip file: %s: %s", file, err))
							return
						}

						_seq = record.Seq

						if k > len(_seq.Seq) {
							// checkError(fmt.Errorf("k is too big for sequence of %d bp: %s", len(record.Seq.Seq), file))
							ch <- &Aresult{
								id:     id,
								ok:     false,
								result: nil,
							}
							log.Errorf(fmt.Sprintf("k is too big for sequence of %d bp: %s", len(record.Seq.Seq), file))
							return
						}

						if n >= 1 {
							// checkError(fmt.Errorf("only one sequence is allowed for circular genome"))
							ch <- &Aresult{
								id:     id,
								ok:     false,
								result: nil,
							}
							log.Warningf(fmt.Sprintf("skip file with > 1 sequences: %s", file))
							return
						}

						if removeGaps {
							_seq.RemoveGapsInplace(gapLetters)
						}

						_seq.Seq = bytes.ToLower(_seq.Seq)

						// // ntHash shoud be simpler and faster, but some thing wrong I didn't fingure out.
						// iter, err = sketches.NewHashIterator(_seq, k, true, true)
						// for {
						// 	h, ok = iter.NextHash()
						// 	if !ok {
						// 		break
						// 	}

						// 	hashes = append(hashes, h)
						// }

						rc = _seq.RevCom()

						l = len(_seq.Seq)
						originalLen = l
						end = l - 1

						i = 0
						for {
							if i > end {
								break
							}
							e = i + k

							if e > originalLen {
								e = e - originalLen
								s = _seq.Seq[i:]
								s = append(s, _seq.Seq[0:e]...)

								j = l - i
								src = rc.Seq[l-e:]
								src = append(src, rc.Seq[0:j]...)
							} else {
								s = _seq.Seq[i : i+k]

								j = l - i
								src = rc.Seq[j-k : j]
							}
							// fmt.Println(i, string(s), string(src))

							h = xxhash.Sum64(s)
							h2 = xxhash.Sum64(src)
							if h < h2 {
								hashes = append(hashes, h)
							} else {
								hashes = append(hashes, h2)
							}

							i++
						}

						n++
						lens += len(record.Seq.Seq)
					}
				} else {
					for {
						record, err = fastxReader.Read()
						if err != nil {
							if err == io.EOF {
								break
							}
							// checkError(err)
							// break
							ch <- &Aresult{
								id:     id,
								ok:     false,
								result: nil,
							}
							log.Warningf(fmt.Sprintf("%s: %s", file, err))
							return
						}

						_seq = record.Seq

						if removeGaps {
							_seq.RemoveGapsInplace(gapLetters)
						}

						h = xxhash.Sum64(bytes.ToLower(_seq.Seq))
						hashes = append(hashes, h)

						n++
						lens += len(_seq.Seq)
					}
				}

				// sequences
				sortutil.Uint64s(hashes)

				var le = binary.LittleEndian
				buf := make([]byte, 8)
				di := xxhash.New()

				for _, h = range hashes {
					// fmt.Println(h)
					le.PutUint64(buf, h)
					di.Write(buf)
				}

				// sequence length
				le.PutUint64(buf, uint64(lens))
				di.Write(buf)

				// sequence number
				le.PutUint64(buf, uint64(n))
				di.Write(buf)

				// sum up
				le.PutUint64(buf, di.Sum64())
				digest := md5.Sum(buf)

				// return result
				if basename {
					file = filepath.Base(file)
				}
				ch <- &Aresult{
					id: id,
					ok: true,
					result: &SumResult{
						File:   file,
						SeqNum: n,
						SeqLen: lens,
						Digest: hex.EncodeToString(digest[:]),
					},
				}
			}(file, id)
		}

		wg.Wait()
		close(ch)
		<-done
	},
}

func init() {
	RootCmd.AddCommand(sumCmd)

	sumCmd.Flags().BoolP("circular", "c", false, "the file contains a single cicular genome sequence")
	sumCmd.Flags().IntP("kmer-size", "k", 1000, "k-mer size for processing circular genomes")
	sumCmd.Flags().BoolP("basename", "b", false, "only output basename of files")
	sumCmd.Flags().BoolP("remove-gaps", "g", false, "remove gaps")
	sumCmd.Flags().StringP("gap-letters", "G", "- 	.", "gap letters")
	sumCmd.Flags().BoolP("all", "a", false, "all information, including the sequences length and the number of sequences")
}

type SumResult struct {
	File   string
	SeqNum int
	SeqLen int
	Digest string
}
