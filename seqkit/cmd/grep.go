// Copyright © 2016-2019 Wei Shen <shenwei356@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, target to the following conditions:
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
	"io"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/shenwei356/bwt"
	"github.com/twotwotwo/sorts/sortutil"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/breader"
	"github.com/shenwei356/bwt/fmi"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// grepCmd represents the grep command
var grepCmd = &cobra.Command{
	GroupID: "search",

	Use:   "grep",
	Short: "search sequences by ID/name/sequence/sequence motifs, mismatch allowed",
	Long: fmt.Sprintf(`search sequences by ID/name/sequence/sequence motifs, mismatch allowed

Attention:

  0. By default, we match sequence ID with patterns, use "-n/--by-name"
     for matching full name instead of just ID.
  1. Unlike POSIX/GNU grep, we compare the pattern to the whole target
     (ID/full header) by default. Please switch "-r/--use-regexp" on
     for partly matching.
  2. When searching by sequences, it's partly matching, and both positive
     and negative strands are searched.
     Please switch on "-P/--only-positive-strand" if you would like to
     search only on the positive strand.
     Mismatch is allowed using flag "-m/--max-mismatch", you can increase
     the value of "-j/--threads" to accelerate processing.
  3. Degenerate bases/residues like "RYMM.." are also supported by flag -d.
     But do not use degenerate bases/residues in regular expression, you need
     convert them to regular expression, e.g., change "N" or "X"  to ".".
  4. When providing search patterns (motifs) via flag '-p',
     please use double quotation marks for patterns containing comma, 
     e.g., -p '"A{2,}"' or -p "\"A{2,}\"". Because the command line argument
     parser accepts comma-separated-values (CSV) for multiple values (motifs).
     Patterns in file do not follow this rule.
  5. The order of sequences in result is consistent with that in original
     file, not the order of the query patterns. 
     But for FASTA file, you can use:
        seqkit faidx seqs.fasta --infile-list IDs.txt
  6. For multiple patterns, you can either set "-p" multiple times, i.e.,
     -p pattern1 -p pattern2, or give a file of patterns via "-f/--pattern-file".

You can specify the sequence region for searching with the flag -R (--region).
The definition of region is 1-based and with some custom design.

Examples:
%s
`, regionExample),
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		outFile := config.OutFile
		lineWidth := config.LineWidth
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		usingDefaultIDRegexp := config.IDRegexp == fastx.DefaultIDRegexp
		quiet := config.Quiet
		runtime.GOMAXPROCS(config.Threads)

		bwt.CheckEndSymbol = false

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", !config.SkipFileCheck)
		for _, file := range files {
			checkIfFilesAreTheSame(file, outFile, "input", "output")
		}

		justCount := getFlagBool(cmd, "count")
		pattern := getFlagStringSlice(cmd, "pattern")
		patternFile := getFlagString(cmd, "pattern-file")
		useRegexp := getFlagBool(cmd, "use-regexp")
		deleteMatched := getFlagBool(cmd, "delete-matched")
		invertMatch := getFlagBool(cmd, "invert-match")
		bySeq := getFlagBool(cmd, "by-seq")
		onlyPositiveStrand := getFlagBool(cmd, "only-positive-strand")
		mismatches := getFlagNonNegativeInt(cmd, "max-mismatch")
		byName := getFlagBool(cmd, "by-name")
		ignoreCase := getFlagBool(cmd, "ignore-case")
		degenerate := getFlagBool(cmd, "degenerate")
		region := getFlagString(cmd, "region")
		circular := getFlagBool(cmd, "circular")
		allowDups := getFlagBool(cmd, "allow-duplicated-patterns")

		immediateOutput := getFlagBool(cmd, "immediate-output")

		if len(pattern) == 0 && patternFile == "" {
			checkError(fmt.Errorf("one of flags -p (--pattern) and -f (--pattern-file) needed"))
		}

		// check pattern with unquoted comma
		hasUnquotedComma := false
		for _, _pattern := range pattern {
			if reUnquotedComma.MatchString(_pattern) {
				hasUnquotedComma = true
				break
			}
		}
		if hasUnquotedComma {
			if outFile == "-" {
				defer log.Warningf(helpUnquotedComma)
			} else {
				log.Warningf(helpUnquotedComma)
			}
		}

		if degenerate && !bySeq {
			log.Infof("when flag -d (--degenerate) given, flag -s (--by-seq) is automatically on")
			bySeq = true
		}

		var sfmi *fmi.FMIndex
		if mismatches > 0 {
			if useRegexp || degenerate {
				checkError(fmt.Errorf("flag -r (--use-regexp) or -d (--degenerate) not allowed when giving flag -m (--max-mismatch)"))
			}
			if !bySeq {
				log.Infof("when value of flag -m (--max-mismatch) > 0, flag -s (--by-seq) is automatically on")
				bySeq = true
			}
			sfmi = fmi.NewFMIndex()
			if mismatches > 4 {
				log.Warningf("large value flag -m/--max-mismatch will slow down the search")
			}
		}

		if useRegexp && degenerate {
			checkError(fmt.Errorf("could not give both flags -d (--degenerate) and -r (--use-regexp)"))
		}

		var start, end int
		var err error
		var limitRegion bool
		if region != "" {
			limitRegion = true
			if !bySeq {
				log.Infof("when flag -R (--region) given, flag -s (--by-seq) is automatically on")
				bySeq = true
			}
			if !reRegion.MatchString(region) {
				checkError(fmt.Errorf(`invalid region: %s. type "seqkit grep -h" for more examples`, region))
			}
			r := strings.Split(region, ":")
			start, err = strconv.Atoi(r[0])
			checkError(err)
			end, err = strconv.Atoi(r[1])
			checkError(err)
			if start == 0 || end == 0 {
				checkError(fmt.Errorf("both start and end should not be 0"))
			}
			if start < 0 && end > 0 {
				checkError(fmt.Errorf("when start < 0, end should not > 0"))
			}
		}

		// prepare pattern
		patternsR := make(map[uint64]*regexp.Regexp, 1<<10)
		patternsN := make(map[uint64]int, 1<<20)
		// patternsS := make(map[string]interface{}, 1<<10)
		patternsS := make([][]byte, 0, 16)

		var pattern2seq *seq.Seq
		var pbyte []byte
		if patternFile != "" {
			var reader *breader.BufferedReader
			reader, err = breader.NewDefaultBufferedReader(patternFile)
			checkError(err)
			for chunk := range reader.Ch {
				checkError(chunk.Err)
				for _, data := range chunk.Data {
					p := data.(string)
					if p == "" {
						continue
					}

					if !quiet {
						if p[0] == '>' {
							log.Warningf(`symbol ">" detected, it should not be a part of the sequence ID/name: %s`, p)
						} else if p[0] == '@' {
							log.Warningf(`symbol "@" detected, it should not be a part of the sequence ID/name. %s`, p)
						} else if !byName && usingDefaultIDRegexp && strings.ContainsAny(p, "\t ") {
							log.Warningf("space found in pattern, you may need use -n/--by-name: %s", p)
						}
					}

					if degenerate || useRegexp {
						if degenerate {
							pattern2seq, err = seq.NewSeq(alphabet, []byte(p))
							if err != nil {
								checkError(fmt.Errorf("it seems that flag -d is given, but you provide regular expression instead of available %s sequence", alphabet.String()))
							}
							p = pattern2seq.Degenerate2Regexp()
						}
						if ignoreCase {
							p = "(?i)" + p
						}
						r, err := regexp.Compile(p)
						checkError(err)
						patternsR[xxhash.Sum64String(p)] = r
					} else if bySeq {
						pbyte = []byte(p)
						if mismatches > 0 && mismatches > len(p) {
							checkError(fmt.Errorf("mismatch should be <= length of sequence: %s", p))
						}
						if seq.DNAredundant.IsValid(pbyte) == nil ||
							seq.RNAredundant.IsValid(pbyte) == nil ||
							seq.Protein.IsValid(pbyte) == nil { // legal sequence
							if ignoreCase {
								// patternsS[strings.ToLower(p)] = struct{}{}
								patternsS = append(patternsS, bytes.ToLower(pbyte))
							} else {
								// patternsS[p] = struct{}{}
								patternsS = append(patternsS, pbyte)
							}
						} else {
							checkError(fmt.Errorf("illegal DNA/RNA/Protein sequence: %s", p))
						}
					} else {
						if ignoreCase {
							patternsN[xxhash.Sum64String(strings.ToLower(p))]++
						} else {
							patternsN[xxhash.Sum64String(p)]++
						}
					}
				}
			}
			if !quiet {
				if len(patternsR)+len(patternsN)+len(patternsS) == 0 {
					log.Warningf("%d patterns loaded from file", 0)
				} else {
					log.Infof("%d patterns loaded from file", len(patternsR)+len(patternsN)+len(patternsS))
				}
			}
		} else {
			for _, p := range pattern {
				if !quiet {
					if p[0] == '>' {
						log.Warningf(`symbol ">" detected, it should not be a part of the sequence ID/name: %s`, p)
					} else if p[0] == '@' {
						log.Warningf(`symbol "@" detected, it should not be a part of the sequence ID/name. %s`, p)
					} else if !byName && usingDefaultIDRegexp && strings.ContainsAny(p, "\t ") {
						log.Warningf("space found in pattern, you may need use -n/--by-name: %s", p)
					}
				}

				if degenerate || useRegexp {
					if degenerate {
						pattern2seq, err = seq.NewSeq(alphabet, []byte(p))
						if err != nil {
							checkError(fmt.Errorf("it seems that flag -d is given, but you provide regular expression instead of available %s sequence", alphabet.String()))
						}
						p = pattern2seq.Degenerate2Regexp()
					}
					if ignoreCase {
						p = "(?i)" + p
					}
					r, err := regexp.Compile(p)
					checkError(err)
					patternsR[xxhash.Sum64String(p)] = r
				} else if bySeq {
					pbyte = []byte(p)
					if mismatches > 0 && mismatches > len(p) {
						checkError(fmt.Errorf("mismatch should be <= length of sequence: %s", p))
					}
					if seq.DNAredundant.IsValid(pbyte) == nil ||
						seq.RNAredundant.IsValid(pbyte) == nil ||
						seq.Protein.IsValid(pbyte) == nil { // legal sequence
						if ignoreCase {
							// patternsS[strings.ToLower(p)] = struct{}{}
							patternsS = append(patternsS, bytes.ToLower(pbyte))
						} else {
							// patternsS[p] = struct{}{}
							patternsS = append(patternsS, pbyte)
						}
					} else {
						checkError(fmt.Errorf("illegal DNA/RNA/Protein sequence: %s", p))
					}
				} else {
					if ignoreCase {
						patternsN[xxhash.Sum64String(strings.ToLower(p))]++
					} else {
						patternsN[xxhash.Sum64String(p)]++
					}
				}
			}
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var record *fastx.Record
		strands := []byte{'+', '-'}

		var count int

		// -------------------------------------------------------------------
		// only for searching with sequences and mismatch > 0, were FMI is very slow

		if bySeq && mismatches > 0 {
			type Arecord struct {
				id     uint64
				ok     bool
				record *fastx.Record
			}

			var wg sync.WaitGroup
			ch := make(chan *Arecord, config.Threads)
			tokens := make(chan int, config.Threads)

			done := make(chan int)
			go func() {
				m := make(map[uint64]*Arecord, config.Threads)
				var id, _id uint64
				var ok bool
				var _r *Arecord

				id = 1
				for r := range ch {
					if justCount {
						if r.ok {
							count++
						}
						continue
					}

					_id = r.id

					if _id == id { // right there
						if r.ok {
							r.record.FormatToWriter(outfh, config.LineWidth)
							if immediateOutput {
								outfh.Flush()
							}
						}
						id++
						continue
					}

					m[_id] = r // save for later check

					if _r, ok = m[id]; ok { // check buffered
						if _r.ok {
							_r.record.FormatToWriter(outfh, config.LineWidth)
							if immediateOutput {
								outfh.Flush()
							}
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
							_r.record.FormatToWriter(outfh, config.LineWidth)
							if immediateOutput {
								outfh.Flush()
							}
						}
					}
				}
				done <- 1
			}()

			var id uint64
			for _, file := range files {
				fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
				checkError(err)

				checkAlphabet := true
				for {
					record, err = fastxReader.Read()
					if err != nil {
						if err == io.EOF {
							break
						}
						checkError(err)
						break
					}

					if len(record.Seq.Seq) == 0 {
						continue
					}

					if checkAlphabet {
						if fastxReader.Alphabet() == seq.Unlimit || fastxReader.Alphabet() == seq.Protein {
							onlyPositiveStrand = true
						}
						checkAlphabet = false
					}

					if fastxReader.IsFastq {
						config.LineWidth = 0
						fastx.ForcelyOutputFastq = true
					}

					tokens <- 1
					wg.Add(1)
					id++
					go func(record *fastx.Record, id uint64) {
						defer func() {
							wg.Done()
							<-tokens
						}()

						var sequence *seq.Seq
						var target []byte
						var hit bool
						// var k string
						var k []byte

						sfmi := fmi.NewFMIndex()

						for _, strand := range strands {
							if hit {
								break
							}

							if strand == '-' && onlyPositiveStrand {
								break
							}

							sequence = record.Seq
							if strand == '-' {
								sequence = record.Seq.RevCom()
							}
							if limitRegion {
								target = sequence.SubSeq(start, end).Seq
								if len(target) == 0 {
									continue
								}
							} else if circular {
								// concat two copies of sequence, and do not change orginal sequence
								target = make([]byte, len(sequence.Seq)*2)
								copy(target[0:len(sequence.Seq)], sequence.Seq)
								copy(target[len(sequence.Seq):], sequence.Seq)
							} else {
								target = sequence.Seq
							}

							if ignoreCase {
								target = bytes.ToLower(target)
							}

							_, err = sfmi.Transform(target)
							if err != nil {
								checkError(fmt.Errorf("fail to build FMIndex for sequence: %s", record.Name))
							}
							// for k = range patternsS {
							for _, k = range patternsS {
								// hit, err = sfmi.Match([]byte(k), mismatches)
								hit, err = sfmi.Match(k, mismatches)
								if err != nil {
									checkError(fmt.Errorf("fail to search pattern '%s' on seq '%s': %s", k, record.Name, err))
								}
								if hit {
									break
								}
							}

						}

						if invertMatch {
							if hit {
								ch <- &Arecord{record: nil, ok: false, id: id}
								return
							}
						} else {
							if !hit {
								ch <- &Arecord{record: nil, ok: false, id: id}
								return
							}
						}

						ch <- &Arecord{record: record, ok: true, id: id}

					}(record.Clone(), id)
				}
				fastxReader.Close()
			}

			wg.Wait()
			close(ch)
			<-done

			if justCount {
				fmt.Fprintf(outfh, "%d\n", count)
			}
			return
		}

		// -------------------------------------------------------------------

		var sequence *seq.Seq
		var target []byte
		var ok, hit bool
		// var k string
		var k []byte
		var re *regexp.Regexp
		var h uint64
		var strand byte
		var i, n int // for output records multiple times when duplicated patterns are given.
		for _, file := range files {
			fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)

			checkAlphabet := true
			for {
				if deleteMatched && len(patternsR)+len(patternsN)+len(patternsS) == 0 {
					break
				}

				record, err = fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}

				if len(record.Seq.Seq) == 0 {
					continue
				}

				if checkAlphabet {
					if fastxReader.Alphabet() == seq.Unlimit || fastxReader.Alphabet() == seq.Protein {
						onlyPositiveStrand = true
					}
					checkAlphabet = false
				}

				if fastxReader.IsFastq {
					config.LineWidth = 0
					fastx.ForcelyOutputFastq = true
				}

				if byName {
					target = record.Name
				} else if bySeq {

				} else {
					target = record.ID
				}

				hit = false

				n = 1

				for _, strand = range strands {
					if hit {
						break
					}

					if strand == '-' {
						if bySeq {
							if onlyPositiveStrand {
								break
							}
						} else {
							break
						}
					}

					if bySeq {
						sequence = record.Seq
						if strand == '-' {
							sequence = record.Seq.RevCom()
						}
						if limitRegion {
							target = sequence.SubSeq(start, end).Seq
							if len(target) == 0 {
								continue
							}
						} else if circular {
							// concat two copies of sequence, and do not change orginal sequence
							target = make([]byte, len(sequence.Seq)*2)
							copy(target[0:len(sequence.Seq)], sequence.Seq)
							copy(target[len(sequence.Seq):], sequence.Seq)
						} else {
							target = sequence.Seq
						}
					}

					if degenerate || useRegexp {
						for h, re = range patternsR {
							if re.Match(target) {
								hit = true
								if deleteMatched && !invertMatch {
									delete(patternsR, h)
								}
								break
							}
						}
					} else if bySeq {
						if ignoreCase {
							target = bytes.ToLower(target)
						}
						if mismatches == 0 {
							// for k = range patternsS {
							for _, k = range patternsS {
								// if bytes.Contains(target, []byte(k)) {
								if bytes.Contains(target, k) {
									hit = true
									// if deleteMatched && !invertMatch {
									// 	delete(patternsS, k)
									// }
									break
								}
							}
						} else {
							_, err = sfmi.Transform(target)
							if err != nil {
								checkError(fmt.Errorf("fail to build FMIndex for sequence: %s", record.Name))
							}
							// for k = range patternsS {
							for _, k = range patternsS {
								// hit, err = sfmi.Match([]byte(k), mismatches)
								hit, err = sfmi.Match(k, mismatches)
								if err != nil {
									checkError(fmt.Errorf("fail to search pattern '%s' on seq '%s': %s", k, record.Name, err))
								}
								if hit {
									break
								}
							}
						}
					} else {
						h = xxhash.Sum64(target)
						if ignoreCase {
							h = xxhash.Sum64(bytes.ToLower(target))
						}
						if n, ok = patternsN[h]; ok {
							hit = true
							if deleteMatched && !invertMatch {
								delete(patternsN, h)
							}
						}
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

				if justCount {
					count++
					if allowDups && n > 1 {
						count += n - 1
					}
				} else {
					record.FormatToWriter(outfh, config.LineWidth)
					if allowDups && n > 1 {
						for i = 0; i < n-1; i++ {
							record.FormatToWriter(outfh, config.LineWidth)
						}
					}
				}

				if immediateOutput {
					outfh.Flush()
				}
			}
			fastxReader.Close()

			config.LineWidth = lineWidth
		}

		if justCount {
			fmt.Fprintf(outfh, "%d\n", count)
		}
	},
}

func init() {
	RootCmd.AddCommand(grepCmd)

	grepCmd.Flags().StringSliceP("pattern", "p", []string{""}, `search pattern. `+helpMultipleValues)
	grepCmd.Flags().BoolP("allow-duplicated-patterns", "D", false, "output records multiple times when duplicated patterns are given")
	grepCmd.Flags().StringP("pattern-file", "f", "", "pattern file (one record per line)")
	grepCmd.Flags().BoolP("use-regexp", "r", false, "patterns are regular expression")
	grepCmd.Flags().BoolP("delete-matched", "", false, "delete a pattern right after being matched, this keeps the firstly matched data and speedups when using regular expressions")
	grepCmd.Flags().BoolP("invert-match", "v", false, "invert the sense of matching, to select non-matching records")
	grepCmd.Flags().BoolP("by-name", "n", false, "match by full name instead of just ID")
	grepCmd.Flags().BoolP("by-seq", "s", false, "search subseq on seq. Both positive and negative strand are searched by default, you might use -P/--only-positive-strand. Mismatch allowed using flag -m/--max-mismatch")
	grepCmd.Flags().BoolP("only-positive-strand", "P", false, "only search on the positive strand")
	grepCmd.Flags().IntP("max-mismatch", "m", 0, "max mismatch when matching by seq. For large genomes like human genome, using mapping/alignment tools would be faster")
	grepCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
	grepCmd.Flags().BoolP("degenerate", "d", false, "pattern/motif contains degenerate base")
	grepCmd.Flags().StringP("region", "R", "", "specify sequence region for searching. "+
		"e.g 1:12 for first 12 bases, -12:-1 for last 12 bases")
	grepCmd.Flags().BoolP("circular", "c", false, "circular genome")
	grepCmd.Flags().BoolP("immediate-output", "I", false, "print output immediately, do not use write buffer")
	grepCmd.Flags().BoolP("count", "C", false, "just print a count of matching records. with the -v/--invert-match flag, count non-matching records")
}

var reUnquotedComma = regexp.MustCompile(`\{[^\}]*$|^[^\{]*\}`)
var helpUnquotedComma = `possible unquoted comma detected, please use double quotation marks for patterns containing comma, e.g., -p '"A{2,}"' or -p "\"A{2,}\""`

var helpMultipleValues = `Multiple values supported: comma-separated (e.g., -p "p1,p2") OR use -p multiple times (e.g., -p p1 -p p2). Make sure to quote literal commas, e.g. in regex patterns '"A{2,}"'`
