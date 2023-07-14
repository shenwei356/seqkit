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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
	"github.com/twotwotwo/sorts/sortutil"
	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
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
     - The same double-stranded genomes with different start positions or
       in reverse complement strand will not affect the result.
     - For single-stranded genomes like ssRNA genomes, use -s/--single-strand.
     - The message digest would change with different values of k-mer size.
  4. Multiple files are processed in parallel (-j/--threads).

Method:
  1. Converting the sequences to low cases, optionally removing gaps (-g).
  2. Computing the hash (xxhash) for all sequences or k-mers of a circular
     complete genome (-c/--circular).
  3. Sorting all hash values, for ignoring the order of sequences.
  4. Computing MD5 digest from the hash values, sequences length, and
     the number of sequences.

Following the seqhash in Poly (https://github.com/TimothyStiles/poly/),
We add meta information to the message digest, with the format of:

    seqkit.<version>_<seq type><seq structure><strand>_<kmer size>_<seq digest>

    <version>:       digest version
    <seq type>:      'D' for DNA, 'R' for RNA, 'P' for protein, 'N' for others
    <seq structure>: 'L' for linear sequence, 'C' for circular genome
    <strand>:        'D' for double-stranded, 'S' for single-stranded
    <kmer size>:     0 for linear sequence, other values for circular genome

Examples:

    seqkit.v0.1_DLS_k0_176250c8d1cde6c385397df525aa1a94    DNA.fq.gz
    seqkit.v0.1_PLS_k0_c244954e4960dd2a1409cd8ee53d92b9    Protein.fasta
    seqkit.v0.1_RLS_k0_0f1fb263f0c05a259ae179a61a80578d    single-stranded RNA.fasta

    seqkit.v0.1_DCD_k31_e59dad6d561f1f1f28ebf185c6f4c183   double-stranded-circular DNA.fasta
    seqkit.v0.1_DCS_k31_dd050490cd62ea5f94d73d4d636b7d60   single-stranded-circular DNA.fasta

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
		rna2dna := getFlagBool(cmd, "rna2dna")
		singleStrand := getFlagBool(cmd, "single-strand")

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		// process bar
		var pbs *mpb.Progress
		var bar *mpb.Bar
		var chDuration chan time.Duration
		var doneDuration chan int

		if !config.Quiet && len(files) > 1 {
			pbs = mpb.New(mpb.WithWidth(40), mpb.WithOutput(os.Stderr))
			bar = pbs.AddBar(int64(len(files)),
				mpb.BarStyle("[=>-]<+"),
				mpb.PrependDecorators(
					decor.Name("processed files: ", decor.WC{W: len("processed files: "), C: decor.DidentRight}),
					decor.Name("", decor.WCSyncSpaceR),
					decor.CountersNoUnit("%d / %d", decor.WCSyncWidth),
				),
				mpb.AppendDecorators(
					decor.Name("ETA: ", decor.WC{W: len("ETA: ")}),
					decor.EwmaETA(decor.ET_STYLE_GO, 5),
					decor.OnComplete(decor.Name(""), ". done"),
				),
			)

			chDuration = make(chan time.Duration, config.Threads)
			doneDuration = make(chan int)
			go func() {
				for t := range chDuration {
					bar.Increment()
					bar.DecoratorEwmaUpdate(t)
				}
				doneDuration <- 1
			}()
		}
		// process bar

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		tokens := make(chan int, config.Threads)
		done := make(chan int)
		threadsFloat := float64(config.Threads) // just avoid repeated type conversion
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
				startTime := time.Now()
				defer func() {
					<-tokens
					wg.Done()

					if !config.Quiet && len(files) > 1 {
						chDuration <- time.Duration(float64(time.Since(startTime)) / threadsFloat)
					}
				}()

				var n int    // number of sequences
				var lens int // lengths of all seqs
				var h uint64
				hashes := make([]uint64, 0, 1024)

				var record *fastx.Record
				var fastxReader *fastx.Reader
				var _seq *seq.Seq
				var ab *seq.Alphabet
				var ii int
				var bb byte

				// add tag to the hash
				// ref: https://github.com/TimothyStiles/poly/blob/prime/seqhash/seqhash.go
				var seqType string      // "D" for DNA, "R" for RNA, "P" for protein
				var seqStructure string // "L" for linear, "C" for circular
				var strand string       // "D" for double strands, "S" for single strand

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

				checkAlphabet := true

				if circular {
					seqStructure = "C"
					if singleStrand {
						strand = "S"
					} else {
						strand = "D"
					}
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

						if n >= 1 {
							// checkError(fmt.Errorf("only one sequence is allowed for circular genome"))
							ch <- &Aresult{
								id:     id,
								ok:     false,
								result: nil,
							}
							log.Warningf(fmt.Sprintf("skip file with more than 1 sequences: %s", file))
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
							log.Errorf(fmt.Sprintf("k (%d) is too big for sequence of %d bp: %s", k, len(record.Seq.Seq), file))
							return
						}

						if checkAlphabet {
							ab = fastxReader.Alphabet()
							if ab == seq.Protein {
								seqType = "P"

								ch <- &Aresult{
									id:     id,
									ok:     false,
									result: nil,
								}
								log.Errorf(fmt.Sprintf("the flag -c/--circular does not support protein sequences: %s", file))
								return

							} else if ab == seq.RNA || ab == seq.RNAredundant {
								seqType = "R"
							} else if ab == seq.DNA || ab == seq.DNAredundant {
								seqType = "D"
							} else {
								seqType = "N"
							}

							checkAlphabet = false
						}

						if removeGaps {
							_seq.RemoveGapsInplace(gapLetters)
						}

						_seq.Seq = bytes.ToLower(_seq.Seq)

						if rna2dna {
							if !(ab == seq.RNA || ab == seq.RNAredundant) {
								for ii, bb = range _seq.Seq {
									if bb == 'u' {
										_seq.Seq[ii] = 't'
									}
								}
							}
						}

						// // ntHash shoud be simpler and faster, but some thing wrong I didn't fingure out.
						// iter, err = sketches.NewHashIterator(_seq, k, true, true)
						// for {
						// 	h, ok = iter.NextHash()
						// 	if !ok {
						// 		break
						// 	}

						// 	hashes = append(hashes, h)
						// }

						if !singleStrand {
							rc = _seq.RevCom()
						}

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

								if !singleStrand {
									j = l - i
									src = rc.Seq[l-e:]
									src = append(src, rc.Seq[0:j]...)
								}
							} else {
								s = _seq.Seq[i : i+k]

								if !singleStrand {
									j = l - i
									src = rc.Seq[j-k : j]
								}
							}
							// fmt.Println(i, string(s), string(src))

							h = xxhash.Sum64(s)
							if singleStrand {
								hashes = append(hashes, h)
							} else {
								h2 = xxhash.Sum64(src)
								if h < h2 {
									hashes = append(hashes, h)
								} else {
									hashes = append(hashes, h2)
								}
							}

							i++
						}

						n++
						lens += len(record.Seq.Seq)
					}
				} else {
					seqStructure = "L"
					strand = "S"

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

						if checkAlphabet {
							ab = fastxReader.Alphabet()
							if ab == seq.Protein {
								seqType = "P"
								if !removeGaps {
									if !strings.Contains(gapLetters, "*") {
										gapLetters += "*"
									}
									removeGaps = true
									log.Infof(`the flag -g/--remove-gaps is switched on for removing the possible stop codon '*' character for protein sequences`)
								}
							} else if ab == seq.RNA || ab == seq.RNAredundant {
								seqType = "R"
							} else if ab == seq.DNA || ab == seq.DNAredundant {
								seqType = "D"
							} else {
								seqType = "N"
							}

							checkAlphabet = false
						}

						if removeGaps {
							_seq.RemoveGapsInplace(gapLetters)
						}

						_seq.Seq = bytes.ToLower(_seq.Seq)

						if rna2dna {
							if !(ab == seq.RNA || ab == seq.RNAredundant) {
								for ii, bb = range _seq.Seq {
									if bb == 'u' {
										_seq.Seq[ii] = 't'
									}
								}
							}
						}

						h = xxhash.Sum64(_seq.Seq)

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

				if !circular {
					k = 0
				}
				sum := fmt.Sprintf("seqkit.v%s_%s%s%s_k%d_%s",
					sumVersion,
					seqType,
					seqStructure,
					strand,
					k,
					hex.EncodeToString(digest[:]))

				ch <- &Aresult{
					id: id,
					ok: true,
					result: &SumResult{
						File:   file,
						SeqNum: n,
						SeqLen: lens,
						Digest: sum,
					},
				}
			}(file, id)
		}

		wg.Wait()

		if !config.Quiet && len(files) > 1 {
			close(chDuration)
			<-doneDuration
			pbs.Wait()
		}

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
	sumCmd.Flags().StringP("gap-letters", "G", "- 	.*", "gap letters")
	sumCmd.Flags().BoolP("all", "a", false, "show all information, including the sequences length and the number of sequences")
	sumCmd.Flags().BoolP("rna2dna", "", false, "convert RNA to DNA")
	sumCmd.Flags().BoolP("single-strand", "s", false, "only consider the positive strand of a circular genome, e.g., ssRNA virus genomes")
}

type SumResult struct {
	File   string
	SeqNum int
	SeqLen int
	Digest string
}

const sumVersion = "0.1"
