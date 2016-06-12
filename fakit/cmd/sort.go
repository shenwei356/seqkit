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
	"bytes"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/brentp/xopen"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fai"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/util/stringutil"
	"github.com/spf13/cobra"
)

// sortCmd represents the sort command
var sortCmd = &cobra.Command{
	Use:   "sort",
	Short: "sort sequences by id/name/sequence/length",
	Long: `sort sequences by id/name/sequence/length.

By default, all records will be readed into memory.
For FASTA format, use flag -2 (--two-pass) to reduce memory usage. FASTQ not
supported.

Firstly, fakit reads the sequence head and length information.
If the file is not plain FASTA file,
fakit will write the sequences to tempory files, and create FASTA index.

Secondly, fakit sort sequence by head and length information
and extract sequences by FASTA index.

ATTENTION: the .fai file created by fakit is a little different from .fai file
created by samtools. Fakit use full sequence head instead of just ID as key.
So please delete .fai file created by samtools.

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		chunkSize := config.ChunkSize
		bufferSize := config.BufferSize
		lineWidth := config.LineWidth
		outFile := config.OutFile
		quiet := config.Quiet
		seq.AlphabetGuessSeqLenghtThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		fai.MapWholeFile = false
		runtime.GOMAXPROCS(config.Threads)

		files := getFileList(args)

		bySeq := getFlagBool(cmd, "by-seq")
		byName := getFlagBool(cmd, "by-name")
		byLength := getFlagBool(cmd, "by-length")
		reverse := getFlagBool(cmd, "reverse")
		ignoreCase := getFlagBool(cmd, "ignore-case")
		twoPass := getFlagBool(cmd, "two-pass")
		seqPrefixLength := getFlagNonNegativeInt(cmd, "seq-prefix-length")
		keepTemp := getFlagBool(cmd, "keep-temp")
		if keepTemp && !twoPass {
			checkError(fmt.Errorf("flag -k (--keep-temp) must be used with flag -2 (--two-pass)"))
		}

		n := 0
		if bySeq {
			n++
		}
		if byName {
			n++
		}
		if byLength {
			n++
		}
		if n > 1 {
			checkError(fmt.Errorf("only one of the flags -l (--by-length), -n (--by-name) and -s (--by-seq) is allowed"))
		}

		byID := true
		if bySeq || byLength {
			byID = false
		}
		if !quiet {
			if byLength {
				if ignoreCase {
					log.Warning("flag -i (--ignore-case) is ignored when flag -l (--by-length) given")
				}
			}
		}

		name2sequence := []stringutil.String2ByteSlice{}
		name2length := []stringutil.StringCount{}

		// for indexing when output and duplicated sequences checking
		id2name := make(map[string][]byte)

		if !twoPass { // read all records into memory
			sequences := make(map[string]*fastx.Record)

			if !quiet {
				log.Infof("read sequences ...")
			}
			var name string
			for _, file := range files {
				fastxReader, err := fastx.NewReader(alphabet, file, bufferSize, chunkSize, idRegexp)
				checkError(err)
				for chunk := range fastxReader.Ch {
					checkError(chunk.Err)

					for _, record := range chunk.Data {
						if byName {
							name = string(record.Name)
						} else if byID || bySeq || byLength {
							name = string(record.ID)
						}

						if _, ok := id2name[name]; ok {
							checkError(fmt.Errorf(`duplicated sequences found: %s. use "fakit rename" to rename duplicated IDs`, name))
						}
						id2name[name] = []byte(string(record.Name))

						if ignoreCase {
							name = strings.ToLower(name)
						}

						record2 := record.Clone()
						sequences[name] = record2
						if byLength {
							name2length = append(name2length, stringutil.StringCount{Key: name, Count: len(record2.Seq.Seq)})
						} else if byID || byName || bySeq {
							if ignoreCase {
								name2sequence = append(name2sequence, stringutil.String2ByteSlice{Key: name, Value: bytes.ToLower(record2.Seq.Seq)})
							} else {
								name2sequence = append(name2sequence, stringutil.String2ByteSlice{Key: name, Value: record2.Seq.Seq})
							}
						}
					}
				}
			}

			if !quiet {
				log.Infof("%d sequences loaded", len(sequences))
				log.Infof("sorting ...")
			}

			if bySeq {
				if reverse {
					sort.Sort(stringutil.ReversedByValue{stringutil.String2ByteSliceList(name2sequence)})
				} else {
					sort.Sort(stringutil.ByValue{stringutil.String2ByteSliceList(name2sequence)})
				}
			} else if byLength {
				if reverse {
					sort.Sort(stringutil.ReversedStringCountList{stringutil.StringCountList(name2length)})
				} else {
					sort.Sort(stringutil.StringCountList(name2length))
				}
			} else if byName || byID { // by name/id
				if reverse {
					sort.Sort(stringutil.ReversedString2ByteSliceList{stringutil.String2ByteSliceList(name2sequence)})
				} else {
					sort.Sort(stringutil.String2ByteSliceList(name2sequence))
				}
			}

			if !quiet {
				log.Infof("output ...")
			}
			outfh, err := xopen.Wopen(outFile)
			checkError(err)
			defer outfh.Close()

			var record *fastx.Record
			if byName || byID || bySeq {
				for _, kv := range name2sequence {
					record = sequences[kv.Key]
					record.FormatToWriter(outfh, lineWidth)
				}
			} else if byLength {
				for _, kv := range name2length {
					record = sequences[kv.Key]
					record.FormatToWriter(outfh, lineWidth)
				}
			}

			return
		}

		// two-pass
		if len(files) > 1 {
			checkError(fmt.Errorf("no more than one file should be given"))
		}

		file := files[0]

		var alphabet2 *seq.Alphabet

		newFile := file
		if isStdin(file) || !isPlainFile(file) {
			if isStdin(file) {
				newFile = "stdin" + ".fastx"
			} else {
				newFile = file + ".fastx"
			}
			if !quiet {
				log.Infof("read and write sequences to tempory file: %s ...", newFile)
			}

			copySeqs(file, newFile)

			var isFastq bool
			var err error
			alphabet2, isFastq, err = fastx.GuessAlphabet(newFile)
			checkError(err)
			if isFastq {
				checkError(os.Remove(newFile))
				checkError(fmt.Errorf("Sorry, two-pass mode does not support FASTQ format"))
			}
		}

		if !quiet {
			log.Infof("create and read FASTA index ...")
		}

		faidx := getFaidx(newFile, `^(.+)$`)
		defer func() {
			checkError(faidx.Close())
		}()

		if !bySeq { // if not by seq, just read faidx
			if !quiet {
				log.Infof("read sequence IDs and lengths from FASTA index ...")
			}

			idRe, err := regexp.Compile(idRegexp)
			if err != nil {
				checkError(fmt.Errorf("fail to compile regexp: %s", idRegexp))
			}

			ids, lengths, err := getSeqIDAndLengthFromFaidxFile(newFile + ".fakit.fai")
			checkError(err)
			var name string
			for i, head := range ids {
				if byName {
					name = head
				} else if byID || bySeq || byLength {
					name = string(fastx.ParseHeadID(idRe, []byte(head)))
				}

				if _, ok := id2name[name]; ok {
					checkError(fmt.Errorf(`duplicated sequences found: %s. use "fakit rename" to rename duplicated IDs`, name))
				}
				id2name[name] = []byte(head)

				if ignoreCase {
					name = strings.ToLower(name)
				}

				name2sequence = append(name2sequence,
					stringutil.String2ByteSlice{Key: name, Value: []byte{}})
				name2length = append(name2length,
					stringutil.StringCount{Key: name, Count: lengths[i]})
			}
		} else { // have to read the sequences
			if !quiet {
				log.Infof("read sequence IDs and sequence prefix from FASTA file ...")
			}
			fastxReader, err := fastx.NewReader(alphabet2, newFile, bufferSize, chunkSize, idRegexp)
			checkError(err)
			var name string
			var prefix []byte
			for chunk := range fastxReader.Ch {
				checkError(chunk.Err)
				for _, record := range chunk.Data {
					if byName {
						name = string(record.Name)
					} else if byID || bySeq || byLength {
						name = string(record.ID)
					}

					if _, ok := id2name[name]; ok {
						checkError(fmt.Errorf(`duplicated sequences found: %s. use "fakit rename" to rename duplicated IDs`, name))
					}
					id2name[name] = []byte(string(record.Name))

					if ignoreCase {
						name = strings.ToLower(name)
					}

					if seqPrefixLength == 0 || len(record.Seq.Seq) <= seqPrefixLength {
						prefix = record.Seq.Seq
					} else {
						prefix = record.Seq.Seq[0:seqPrefixLength]
					}
					name2sequence = append(name2sequence,
						stringutil.String2ByteSlice{Key: name, Value: []byte(string(prefix))})
					name2length = append(name2length,
						stringutil.StringCount{Key: name, Count: len(record.Seq.Seq)})
				}
			}
		}

		if !quiet {
			log.Infof("%d sequences loaded", len(id2name))
			log.Infof("sorting ...")
		}

		if bySeq {
			if reverse {
				sort.Sort(stringutil.ReversedByValue{stringutil.String2ByteSliceList(name2sequence)})
			} else {
				sort.Sort(stringutil.ByValue{stringutil.String2ByteSliceList(name2sequence)})
			}
		} else if byLength {
			if reverse {
				sort.Sort(stringutil.ReversedStringCountList{stringutil.StringCountList(name2length)})
			} else {
				sort.Sort(stringutil.StringCountList(name2length))
			}
		} else if byName || byID { // by name/id
			if reverse {
				sort.Sort(stringutil.ReversedString2ByteSliceList{stringutil.String2ByteSliceList(name2sequence)})
			} else {
				sort.Sort(stringutil.String2ByteSliceList(name2sequence))
			}
		}

		if !quiet {
			log.Infof("output ...")
		}
		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		// var record *fastx.Record
		var chr string
		if byName || byID || bySeq {
			for _, kv := range name2sequence {
				chr = string(id2name[kv.Key])
				r, ok := faidx.Index[chr]
				if !ok {
					checkError(fmt.Errorf(`sequence (%s) not found in file: %s`, chr, newFile))
					continue
				}

				sequence := subseqByFaixNotCleaned(faidx, chr, r, 1, -1)
				outfh.Write([]byte(fmt.Sprintf(">%s\n", chr)))
				outfh.Write(sequence)
				if len(sequence) > 0 && sequence[len(sequence)-1] == '\n' {
				} else {
					outfh.WriteString("\n")
				}
			}
		} else if byLength {
			for _, kv := range name2length {
				chr = string(id2name[kv.Key])
				r, ok := faidx.Index[chr]
				if !ok {
					checkError(fmt.Errorf(`sequence (%s) not found in file: %s`, chr, newFile))
					continue
				}

				sequence := subseqByFaixNotCleaned(faidx, chr, r, 1, -1)
				outfh.Write([]byte(fmt.Sprintf(">%s\n", chr)))
				outfh.Write(sequence)
				if len(sequence) > 0 && sequence[len(sequence)-1] == '\n' {
				} else {
					outfh.WriteString("\n")
				}
			}
		}

		if (isStdin(file) || !isPlainFile(file)) && !keepTemp {
			checkError(os.Remove(newFile))
			checkError(os.Remove(newFile + ".fakit.fai"))
		}
	},
}

func init() {
	RootCmd.AddCommand(sortCmd)
	sortCmd.Flags().BoolP("by-name", "n", false, "by full name instead of just id")
	sortCmd.Flags().BoolP("by-seq", "s", false, "by sequence")
	sortCmd.Flags().BoolP("by-length", "l", false, "by sequence length")
	sortCmd.Flags().BoolP("reverse", "r", false, "reverse the result")
	sortCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")

	sortCmd.Flags().BoolP("two-pass", "2", false, "two-pass mode read files twice to lower memory usage. (only for FASTA format)")
	sortCmd.Flags().BoolP("keep-temp", "k", false, "keep tempory FASTA and .fai file when using 2-pass mode")
	sortCmd.Flags().IntP("seq-prefix-length", "L", 10000, "length of sequence prefix on which fakit sorts by sequences (0 for whole sequence)")
}
