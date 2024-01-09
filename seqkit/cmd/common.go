// Copyright © 2016-2023 Wei Shen <shenwei356@gmail.com>
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
	"io"
	"runtime"
	"sort"

	"github.com/cespare/xxhash/v2"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// commonCmd represents the common command
var commonCmd = &cobra.Command{
	GroupID: "set",

	Use:   "common",
	Short: "find common/shared sequences of multiple files by id/name/sequence",
	Long: `find common/shared sequences of multiple files by id/name/sequence

Note:
  1. 'seqkit common' is designed to support 2 and MORE files.
  2. When comparing by sequences,
     a) Both positive and negative strands are compared. You can switch on
        -P/--only-positive-strand for considering the positive strand only.
     b) You can switch on -e/--check-embedded-seqs to check embedded sequences.
          e.g, for file A and B, the reverse complement sequence of CCCC from file B
          is a part of TTGGGGTT from file A, we will extract and output GGGG from file A.
          If sequences CCC exist in other files except file A, we will skip it,
          as it is an embedded subsequence of GGGG.
        It is recommended to put the smallest file as the first file, for saving
        memory usage.
  3. For 2 files, 'seqkit grep' is much faster and consumes lesser memory:
       seqkit grep -f <(seqkit seq -n -i small.fq.gz) big.fq.gz # by seq ID
     But note that searching by sequence would be much slower, as it's partly
     string matching.
       seqkit grep -s -f <(seqkit seq -s small.fq.gz) big.fq.gz # much slower!!!!
  4. Some records in one file may have same sequences/IDs. They will ALL be
     retrieved if the sequence/ID was shared in multiple files.
     So the records number may be larger than that of the smallest file.

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		// lineWidth := config.LineWidth
		outFile := config.OutFile
		quiet := config.Quiet
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		bySeq := getFlagBool(cmd, "by-seq")
		byName := getFlagBool(cmd, "by-name")
		ignoreCase := getFlagBool(cmd, "ignore-case")
		checkembeddedSeqs := getFlagBool(cmd, "check-embedded-seqs")

		if bySeq && byName {
			checkError(fmt.Errorf("only one/none of the flags -s (--by-seq) and -n (--by-name) is allowed"))
		}

		// revcom := getFlagBool(cmd, "consider-revcom")
		revcom := !getFlagBool(cmd, "only-positive-strand")

		if !revcom && !bySeq {
			checkError(fmt.Errorf("flag -s (--by-seq) needed when using -P (--only-positive-strand)"))
		}

		if checkembeddedSeqs && !bySeq {
			checkError(fmt.Errorf("flag -s (--by-seq) needed when using -e (--check-embedded-seqs)"))
		}

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		if len(files) < 2 {
			checkError(errors.New("at least 2 files needed"))
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var fastxReader *fastx.Reader
		var record *fastx.Record
		var rc *seq.Seq

		var _seq0, _seq, _seqRC []byte
		var subject uint64
		var subjectRC uint64 // hash value of reverse complement sequence
		var checkFirstFile = true
		var isFirstFile = true
		var firstFile string
		var ok bool

		// ------------ for bySeq && checkembeddedSeqs ---------------

		type loc2hash struct {
			id    string
			begin int
			end   int
			len   int
		}

		// relation structures
		var seqs map[string]*fastx.Record                  // seqid -> record, sequences in the first file
		var hashes map[uint64]*[]*loc2hash                 // hash -> [(id,begin,end,len)]
		var seqid2Hashes map[string]map[uint64]interface{} // seqid -> hashes of subseqs

		var hitSeqs map[string]interface{}
		var hitHashes map[uint64]interface{}
		rmSeqs := make([]string, 0, 1024)
		rmHashes := make([]uint64, 0, 1024)
		blackHashes := make(map[uint64]interface{}, 1024)

		var seqid string
		var _record *fastx.Record
		var loc2hashes *[]*loc2hash
		var _loc2hash *loc2hash
		var j, begin, end, _begin, _end int
		var hash uint64
		var foundSameSeq bool
		var foundSubseq bool // found a subsequence for the previous seq
		var foundSubseqID string

		// debug := false

		if bySeq && checkembeddedSeqs {
			seqs = make(map[string]*fastx.Record, 1024)
			hashes = make(map[uint64]*[]*loc2hash, 1024)
			seqid2Hashes = make(map[string]map[uint64]interface{}, 1024)

			for i, file := range files {
				if !quiet {
					log.Infof("read file %d/%d: %s", i+1, len(files), file)
				}
				if checkFirstFile && !isStdin(file) {
					firstFile = file
					checkFirstFile = false
				}

				if !isFirstFile { // compare
					hitSeqs = make(map[string]interface{}, 1024)
					hitHashes = make(map[uint64]interface{}, 1024)
				}

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

					_seq = record.Seq.Seq
					if ignoreCase {
						_seq = bytes.ToLower(_seq)
						record.Seq.Seq = _seq
					}
					subject = xxhash.Sum64(_seq)

					if revcom {
						rc = record.Seq.RevCom()
						_seqRC = rc.Seq
						// if ignoreCase {
						// 	_seqRC = bytes.ToLower(_seqRC)
						// }
						subjectRC = xxhash.Sum64(_seqRC)

						if subjectRC < subject { // only save the smaller one, like keeping canonical k-mers
							subject = subjectRC
						}
					}

					seqid = string(record.ID)
					if isFirstFile { // just save data
						if _, ok = seqs[seqid]; ok {
							checkError(fmt.Errorf("duplicated ID not allowed: %s", record.ID))
						} else {
							seqs[seqid] = record.Clone()
						}

						if loc2hashes, ok = hashes[subject]; !ok {
							loc2hashes = &[]*loc2hash{}
							hashes[subject] = loc2hashes
						}
						*loc2hashes = append(*loc2hashes, &loc2hash{
							id:    seqid,
							begin: 1,
							end:   -1,
							len:   len(record.Seq.Seq),
						})

						if _, ok = seqid2Hashes[seqid]; !ok {
							seqid2Hashes[seqid] = make(map[uint64]interface{})
						}
						seqid2Hashes[seqid][subject] = struct{}{}

						continue
					}

					// 2+ files
					// if debug {
					// 	fmt.Printf("check seq: %s\n", seqid)
					// }
					if loc2hashes, ok = hashes[subject]; ok { // existed
						foundSameSeq = false
						for _, _loc2hash = range *loc2hashes {
							if _loc2hash.len == len(record.Seq.Seq) { // hash and seqlen both matched
								foundSameSeq = true
								foundSubseqID = _loc2hash.id
								break
							}
						}
						if foundSameSeq {
							hitHashes[subject] = struct{}{}
							hitSeqs[foundSubseqID] = struct{}{}

							// if debug {
							// 	fmt.Printf("    0) exactly the same seq (%s) vs (%s)\n", record.ID, _loc2hash.id)
							// }
						}
					} else {
						for _, _record = range seqs {
							foundSubseq = false

							seqid = string(_record.ID)

							// if debug {
							// 	fmt.Printf("  - compare to %s\n", seqid)
							// }

						THESEQ:
							for hash = range seqid2Hashes[seqid] { // this seqs has many subseqs
								// if debug {
								// 	fmt.Printf("  - check hash %d\n", hash)
								// }
								for _, _loc2hash = range *(hashes[hash]) { // for each record
									if _loc2hash.id != seqid { // subseq of other seqs
										continue
									}
									_begin, _end, _ = seq.SubLocation(len(_record.Seq.Seq), _loc2hash.begin, _loc2hash.end)
									_seq0 = _record.Seq.Seq[_begin-1 : _end]

									if j = bytes.Index(_seq0, _seq); j >= 0 { // current seq is part of one previous seq
										begin, end = _begin+j, _begin+j+len(_seq)-1
										foundSubseq = true

										// if debug {
										// 	fmt.Printf("    1) Previous seq (%s):%d-%d contains this one (%s)\n", seqid, begin, end, record.ID)
										// }
									} else if j = bytes.Index(_seq, _seq0); j >= 0 { // one previous seq is part of current seq
										hitSeqs[seqid] = struct{}{}
										hitHashes[hash] = struct{}{} // mark hashes with matches

										// if debug {
										// 	begin, end = _begin+j, _begin+j+len(_seq0)-1 // just for debug
										// 	fmt.Printf("    3) This seq (%s) countains previous one (%s):%d-%d\n", record.ID, seqid, begin, end)
										// }
									} else if revcom {
										if j = bytes.Index(_seq0, _seqRC); j >= 0 { // current seq is part of one previous seq
											begin, end = _begin+j, _begin+j+len(_seq)-1
											foundSubseq = true

											// if debug {
											// 	fmt.Printf("    2) Previous seq (%s):%d-%d contains this one (%s) (RC)\n", seqid, begin, end, record.ID)
											// }
										} else if j = bytes.Index(_seqRC, _seq0); j >= 0 { // one previous seq is part of current seq
											hitSeqs[seqid] = struct{}{}
											hitHashes[hash] = struct{}{} // mark hashes with matches

											// if debug {
											// 	begin, end = _begin+j, _begin+j+len(_seq0)-1 // just for debug
											// 	fmt.Printf("    4) This seq (%s) RC countains previous one (%s):%d-%d\n", record.ID, seqid, begin, end)
											// }
										}
									}

									if !foundSubseq {
										continue
									}

									// if debug {
									// 	fmt.Printf("  > add new record %d for %s\n", subject, seqid)
									// }

									hitSeqs[seqid] = struct{}{}               // mark the sequence with matches
									hitHashes[subject] = struct{}{}           // mark hashes with matches
									seqid2Hashes[seqid][subject] = struct{}{} // add new record

									// add a new record for subsequence
									if loc2hashes, ok = hashes[subject]; !ok {
										loc2hashes = &[]*loc2hash{}
										hashes[subject] = loc2hashes
									}
									*loc2hashes = append(*loc2hashes, &loc2hash{
										id:    seqid,
										begin: begin,
										end:   end,
										len:   end - begin + 1,
									})

									break THESEQ // just need one hit in the same sequence
								}
							} // all subsequences of a target seq
						} // target seqs to compare
					} // compare seqs by bytes.index
				} // all input seqs
				fastxReader.Close()

				if isFirstFile {
					if !quiet {
						log.Infof("  %d seqs loaded", len(seqs))
					}

					isFirstFile = false
				} else {
					rmHashes = rmHashes[:0]
					rmSeqs = rmSeqs[:0]

					// clean hashes
					for hash, loc2hashes = range hashes {
						if _, ok = blackHashes[hash]; ok {
							rmHashes = append(rmHashes, hash)
						}
						if _, ok = hitHashes[hash]; !ok {
							rmHashes = append(rmHashes, hash)
						}

						for _, _loc2hash = range *loc2hashes { // some seqs do not exist anymore
							if _, ok = hitSeqs[_loc2hash.id]; !ok {
								rmSeqs = append(rmSeqs, seqid)
							}
						}
					}
					for _, hash = range rmHashes {
						blackHashes[hash] = struct{}{}
						delete(hashes, hash)
					}

					// clean seqs and seqid2Hashes

					for seqid = range seqs {
						if _, ok = hitSeqs[seqid]; !ok {
							rmSeqs = append(rmSeqs, seqid)
						}

						for hash = range seqid2Hashes[seqid] {
							if _, ok = blackHashes[hash]; ok {
								delete(seqid2Hashes[seqid], hash)
							}
						}
					}

					for _, seqid = range rmSeqs {
						delete(seqs, seqid)
						delete(seqid2Hashes, seqid)
					}

					if !quiet {
						log.Infof("  %d seqs left", len(seqs))

						// if debug {
						// 	fmt.Printf("   %d seqs left\n", len(seqs))
						// 	for seqid = range seqs {
						// 		fmt.Printf("    %s\n", seqid)
						// 	}
						// }
					}
				}

				if len(seqs) == 0 {
					log.Info("no common sequences found")
					return
				}

				// list hashes
				// if debug {
				// 	fmt.Println("-----------------------------------")
				// 	var _hashes map[uint64]interface{}
				// 	for seqid, _hashes = range seqid2Hashes {
				// 		fmt.Printf("%s\n", seqid)
				// 		for hash = range _hashes {
				// 			fmt.Printf("  %d\n", hash)
				// 		}
				// 	}
				// 	for subject, loc2hashes = range hashes {
				// 		fmt.Println(subject)
				// 		for _, _loc2hash = range *loc2hashes {
				// 			fmt.Printf("  %s:%d-%d=%d %s\n", _loc2hash.id, _loc2hash.begin, _loc2hash.end, _loc2hash.len, seqs[_loc2hash.id].Seq.SubSeq(_loc2hash.begin, _loc2hash.end).Seq)
				// 		}
				// 	}
				// }
			}

			// retrieve
			var nHashes, nOriginalRecords, nOutput int
			var loc [3]int
			var locs *[][3]int
			seq2loc := make(map[string]*[][3]int, len(hashes))

			for subject, loc2hashes = range hashes {
				nHashes++ // number of hash values

				for _, _loc2hash = range *loc2hashes {
					if locs, ok = seq2loc[_loc2hash.id]; !ok {
						seq2loc[_loc2hash.id] = &[][3]int{
							{_loc2hash.begin, _loc2hash.end, _loc2hash.len},
						}
					} else {
						*locs = append(*locs, [3]int{_loc2hash.begin, _loc2hash.end, _loc2hash.len})
					}
				}
			}

			var loc1 [3]int
			var text []byte
			var buffer *bytes.Buffer

			fastxReader, err := fastx.NewReader(alphabet, firstFile, idRegexp)
			checkError(err)
			checkFormat := true
			var isFastq bool
			var embedded bool
			for {
				record, err = fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}
				if checkFormat {
					checkFormat = false
					if fastxReader.IsFastq {
						config.LineWidth = 0
						fastx.ForcelyOutputFastq = true
					}
					isFastq = fastxReader.IsFastq
				}

				if locs, ok = seq2loc[string(record.ID)]; !ok {
					continue
				}

				nOriginalRecords++ // number of records

				sort.Slice(*locs, func(i, j int) bool {
					return (*locs)[i][2] > (*locs)[j][2]
				})

				for j, loc = range *locs {
					if j > 0 {
						embedded = false
						for _, loc1 = range (*locs)[0:j] {
							if (loc1[0] == 1 && loc1[1] == -1) || // the full seqs
								(loc[0] >= loc1[0] && loc[1] <= loc1[1]) { // subseq
								embedded = true
								break
							}
						}
						if embedded {
							continue
						}
					}

					nOutput++ // output sequences

					if !(loc[0] == 1 && loc[1] == -1) {
						if len(record.Desc) > 0 {
							record.Name = []byte(fmt.Sprintf("%s:%d-%d %s", record.ID, loc[0], loc[1], record.Desc))
						} else {
							record.Name = []byte(fmt.Sprintf("%s:%d-%d", record.ID, loc[0], loc[1]))
						}
					}

					if isFastq {
						record.FormatToWriter(outfh, 0)
					} else {
						begin, end, _ = seq.SubLocation(len(record.Seq.Seq), loc[0], loc[1])
						text, buffer = wrapByteSlice(record.Seq.Seq[begin-1:end], config.LineWidth, buffer)

						outfh.Write(_mark_fasta)
						outfh.Write(record.Name)
						outfh.Write(_mark_newline)
						outfh.Write(text)
						outfh.Write(_mark_newline)
					}
				}
			}
			fastxReader.Close()

			fileNum := len(files)
			t := "sequences"
			if !quiet {
				log.Infof("%d unique %s found in %d files, which belong to %d records in the first file: %s",
					nHashes, t, fileNum, nOriginalRecords, firstFile)
				log.Infof("%d common/shared sequences saved to: %s", nOutput, outFile)
				log.Infof("note that embedded subsequences of a sequence are not outputted")
				if nOutput-nOriginalRecords > 0 {
					log.Infof("more than one subsequences are outputted in some (%d) records", nOutput-nOriginalRecords)
				}
			}

			return
		}

		// ------------------------- for other situations  --------------------------------

		// target -> file idx -> struct{}
		counter := make(map[uint64]map[int]struct{}, 1000)
		var _counter map[int]struct{} // file idx -> struct{}

		// target -> seqnames in firstFile
		// note that it's []string, i.e., records may have same sequences
		names := make(map[uint64][]string, 1024)

		for i, file := range files {
			if !quiet {
				log.Infof("read file %d/%d: %s", i+1, len(files), file)
			}
			if checkFirstFile && !isStdin(file) {
				firstFile = file
				checkFirstFile = false
			}

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

				if bySeq {
					_seq = record.Seq.Seq
					if ignoreCase {
						_seq = bytes.ToLower(_seq)
					}
					subject = xxhash.Sum64(_seq)

					if revcom {
						rc = record.Seq.RevCom()
						_seqRC = rc.Seq
						if ignoreCase {
							_seqRC = bytes.ToLower(_seqRC)
						}
						subjectRC = xxhash.Sum64(_seqRC)

						if subjectRC < subject { // only save the smaller one, like keeping canonical k-mers
							subject = subjectRC
						}
					}
				} else if byName {
					if ignoreCase {
						subject = xxhash.Sum64(bytes.ToLower(record.Name))
					} else {
						subject = xxhash.Sum64(record.Name)
					}
				} else { // byID
					if ignoreCase {
						subject = xxhash.Sum64(bytes.ToLower(record.ID))
					} else {
						subject = xxhash.Sum64(record.ID)
					}
				}

				if _counter, ok = counter[subject]; !ok {
					_counter = make(map[int]struct{})
					counter[subject] = _counter
				}
				_counter[i] = struct{}{}

				if isFirstFile {
					if _, ok = names[subject]; !ok {
						names[subject] = make([]string, 0, 1)
					}
					names[subject] = append(names[subject], string(record.Name))
				}
			}
			fastxReader.Close()

			if isFirstFile {
				isFirstFile = false
			}
		}

		// find common seqs
		if !quiet {
			log.Info("find common seqs ...")
		}
		fileNum := len(files)
		namesOK := make(map[string]struct{})

		var nHashes, nOriginalRecords, nOutput int
		var seqname string
		for subject, presence := range counter {
			if len(presence) != fileNum {
				continue
			}

			nHashes++                               // number of hash values
			nOriginalRecords += len(names[subject]) // number of records

			for _, seqname = range names[subject] {
				namesOK[seqname] = struct{}{}
			}
		}

		var t string
		if byName {
			t = "sequence headers"
		} else if bySeq {
			t = "sequences"
		} else {
			t = "sequence IDs"
		}
		if nHashes == 0 {
			log.Infof("no common %s found", t)
			return
		}
		if !quiet {
			log.Infof("%d unique %s found in %d files, which belong to %d records in the first file: %s",
				nHashes, t, fileNum, nOriginalRecords, firstFile)
		}

		// retrieve
		fastxReader, err = fastx.NewReader(alphabet, firstFile, idRegexp)
		checkError(err)
		checkFormat := true
		for {
			record, err = fastxReader.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				checkError(err)
				break
			}
			if checkFormat {
				checkFormat = false
				if fastxReader.IsFastq {
					config.LineWidth = 0
					fastx.ForcelyOutputFastq = true
				}
			}

			if _, ok = namesOK[string(record.Name)]; ok {
				nOutput++
				record.FormatToWriter(outfh, config.LineWidth)
			}
		}
		fastxReader.Close()

		if !quiet {
			log.Infof("%d common/shared sequences saved to: %s", nOutput, outFile)
		}
	},
}

func init() {
	RootCmd.AddCommand(commonCmd)

	commonCmd.Flags().BoolP("by-name", "n", false, "match by full name instead of just id")
	commonCmd.Flags().BoolP("by-seq", "s", false, "match by sequence")
	commonCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
	// commonCmd.Flags().BoolP("consider-revcom", "r", false, "considering the reverse compelment sequence")
	commonCmd.Flags().BoolP("only-positive-strand", "P", false, "only considering the positive strand when comparing by sequence")
	commonCmd.Flags().BoolP("check-embedded-seqs", "e", false, "check embedded sequences, e.g., if a sequence is part of another one, we'll keep the shorter one")
}
