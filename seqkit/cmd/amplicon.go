// Copyright © 2016-2026 Wei Shen <shenwei356@gmail.com>
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
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/shenwei356/bio/featio/gtf"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fai"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/bwt"
	"github.com/shenwei356/bwt/fmi"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
	"github.com/twotwotwo/sorts/sortutil"
)

// ampliconCmd represents the amplicon command
var ampliconCmd = &cobra.Command{
	GroupID: "search",

	Use:   "amplicon",
	Short: "extract amplicon (or specific region around it) via primer(s)",
	Long: `extract amplicon (or specific region around it) via primer(s).

Attention:
  1. Only one (the longest) matching location is returned for every primer pair.
  2. Mismatch is allowed, but the mismatch location (5' or 3') is not controlled.
     You can increase the value of "-j/--threads" to accelerate processing.
     You can switch "-M/--output-mismatches" to append total mismatches and
     mismatches of 5' end and 3' end.
  3. Degenerate bases/residues like "RYMM.." are also supported.
     But do not use degenerate bases/residues in regular expression, you need
     convert them to regular expression, e.g., change "N" or "X"  to ".".

Examples:
  0. no region given.
  
                    F
        -----===============-----
             F             R
        -----=====-----=====-----
             
             ===============         amplicon

  1. inner region (-r x:y).

                    F
        -----===============-----
             1 3 5                    x/y
                      -5-3-1          x/y
             F             R
        -----=====-----=====-----     x:y
        
             ===============          1:-1
             =======                  1:7
               =====                  3:7
                  =====               6:10
                  =====             -10:-6
                     =====           -7:-3
                                     -x:y (invalid)
                    
  2. flanking region (-r x:y -f)
        
                    F
        -----===============-----
         -3-1                        x/y
                            1 3 5    x/y
             F             R
        -----=====-----=====-----
        
        =====                        -5:-1
        ===                          -5:-3
                            =====     1:5
                              ===     3:5
            =================        -1:1
        =========================    -5:5
                                      x:-y (invalid)

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		lineWidth := config.LineWidth
		outFile := config.OutFile
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		gtf.Threads = config.Threads
		fai.MapWholeFile = false
		Threads = config.Threads
		runtime.GOMAXPROCS(config.Threads)
		bwt.CheckEndSymbol = false

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", !config.SkipFileCheck)
		if !config.SkipFileCheck {
			for _, file := range files {
				checkIfFilesAreTheSame(file, outFile, "input", "output")
			}
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		region := getFlagString(cmd, "region")
		fregion := getFlagBool(cmd, "flanking-region")

		forward0 := getFlagString(cmd, "forward")
		reverse0 := getFlagString(cmd, "reverse")
		primerFile := getFlagString(cmd, "primer-file")
		maxMismatch := getFlagNonNegativeInt(cmd, "max-mismatch")
		outputMismatches := getFlagBool(cmd, "output-mismatches")
		strict := getFlagBool(cmd, "strict-mode")
		onlyPositiveStrand := getFlagBool(cmd, "only-positive-strand")
		outFmtBED := getFlagBool(cmd, "bed")
		saveUnmatched := getFlagBool(cmd, "save-unmatched")

		immediateOutput := getFlagBool(cmd, "immediate-output")

		var list [][3]string
		var primers [][3][]byte

		if primerFile != "" {
			list, err = loadPrimers(primerFile)
			checkError(err)
		} else {
			list = [][3]string{{".", forward0, reverse0}}
		}

		primers, err = parsePrimers(list)
		checkError(err)

		if !config.Quiet {
			log.Infof("%d primer pair loaded", len(primers))
		}

		var begin, end int

		var usingRegion bool
		if region != "" {
			if !reRegion.MatchString(region) {
				checkError(fmt.Errorf(`invalid region: %s. type "seqkit amplicon -h" for more examples`, region))
			}
			r := strings.Split(region, ":")
			begin, err = strconv.Atoi(r[0])
			checkError(err)
			end, err = strconv.Atoi(r[1])
			checkError(err)

			if begin == 0 || end == 0 {
				checkError(fmt.Errorf("both begin and end in region should not be 0"))
			}

			if fregion {
				if begin > 0 && end < 0 {
					checkError(fmt.Errorf("invalid flanking region (x:-y): %d:%d", begin, end))
				}
				if begin > end {
					checkError(fmt.Errorf("invalid flanking region (x should be smaller than y): %d:%d", begin, end))
				}
			} else {
				if begin < 0 && end > 0 {
					checkError(fmt.Errorf("invalid inner region (-x:y): %d:%d", begin, end))
				}
			}
			usingRegion = true
		}

		strands := []string{"+", "-"}

		// -----------
		// finder pools
		finderPools := make([]*sync.Pool, len(primers))
		for i, primer := range primers {
			f := make([]byte, len(primer[1])) // copy the primer sequence
			copy(f, primer[1])

			r := make([]byte, len(primer[2])) // copy the primer sequence
			copy(r, primer[2])

			finderPools[i] = &sync.Pool{New: func() interface{} {
				finder, _ := NewAmpliconFinder([]byte{'A'}, f, r, maxMismatch)
				return finder
			}}
		}

		// -------------------------------------------------------------------
		// only for m > 0, where FMI is slow

		if maxMismatch > 0 {
			type Arecord struct {
				id     uint64
				ok     bool
				record []string
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
				var row string

				id = 1
				for r := range ch {
					_id = r.id

					if _id == id { // right there
						if r.ok {
							for _, row = range r.record {
								outfh.WriteString(row)
							}

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
							for _, row = range _r.record {
								outfh.WriteString(row)
							}

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
							for _, row = range _r.record {
								outfh.WriteString(row)
							}

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
				var record *fastx.Record

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

						var j int
						var finder *AmpliconFinder
						var loc []int
						var mis []int
						var strand string
						var tmpSeq *seq.Seq
						var primer [3][]byte
						var start1, end1 int

						results := make([]string, 0, 2)
						var s []byte
						name0 := string(record.Name)

						for _, strand = range strands {
							if strand == "-" {
								if onlyPositiveStrand {
									continue
								}
								record.Seq.RevComInplace()
							}

							for j, primer = range primers {
								// finder, err = NewAmpliconFinder(record.Seq.Seq, primer[1], primer[2], maxMismatch)
								// checkError(err)
								finder = finderPools[j].Get().(*AmpliconFinder)
								finder.Reset(record.Seq.Seq, maxMismatch)

								if usingRegion {
									loc, mis, err = finder.LocateRange(begin, end, fregion, strict)
								} else {
									loc, mis, err = finder.Locate()
								}
								checkError(err)

								if loc == nil {
									continue
								}

								if outFmtBED {
									start1, end1 = loc[0]-1, loc[1]
									if strand == "-" {
										start1, end1 = len(record.Seq.Seq)-loc[1], len(record.Seq.Seq)-loc[0]+1
									}
									if outputMismatches {
										s = record.Seq.SubSeq(loc[0], loc[1]).Seq
										results = append(results, fmt.Sprintf("%s\t%d\t%d\t%s\t%d\t%s\t%s\t%d\t%d\t%d\n",
											record.ID,
											start1,
											end1,
											primer[0],
											0,
											strand,
											s,
											mis[0]+mis[1],
											mis[0],
											mis[1],
										))
									} else {
										results = append(results, fmt.Sprintf("%s\t%d\t%d\t%s\t%d\t%s\t%s\n",
											record.ID,
											start1,
											end1,
											primer[0],
											0,
											strand,
											record.Seq.SubSeq(loc[0], loc[1]).Seq))
									}

									continue
								}
								tmpSeq = record.Seq

								record.Seq = record.Seq.SubSeq(loc[0], loc[1])
								if outputMismatches {
									record.Name = []byte(fmt.Sprintf("%s mismatches=%d(%d+%d)", name0, mis[0]+mis[1], mis[0], mis[1]))
								}
								results = append(results, string(record.Format(config.LineWidth)))

								record.Seq = tmpSeq

								finderPools[j].Put(finder)
							}
						}

						if len(results) > 0 {
							ch <- &Arecord{record: results, id: id, ok: true}
						} else if saveUnmatched {
							if onlyPositiveStrand {
								results = append(results, string(record.Format(config.LineWidth)))
							} else {
								record.Seq.RevComInplace()
								results = append(results, string(record.Format(config.LineWidth)))
							}
							ch <- &Arecord{record: results, id: id, ok: true}
						} else {
							ch <- &Arecord{record: results, id: id, ok: false}
						}

					}(record.Clone(), id)
				}
				fastxReader.Close()

				config.LineWidth = lineWidth
			}

			wg.Wait()
			close(ch)
			<-done

			return
		}

		var record *fastx.Record

		var j int
		var finder *AmpliconFinder
		var loc []int

		var strand string
		var tmpSeq *seq.Seq
		var primer [3][]byte
		var matched bool
		var name0 string
		var start1, end1 int

		for _, file := range files {
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
				if fastxReader.IsFastq {
					config.LineWidth = 0
					fastx.ForcelyOutputFastq = true
				}

				matched = false
				name0 = string(record.Name)

				for _, strand = range strands {
					if strand == "-" {
						if onlyPositiveStrand {
							continue
						}
						record.Seq.RevComInplace()
					}

					for j, primer = range primers {
						// finder, err = NewAmpliconFinder(record.Seq.Seq, primer[1], primer[2], maxMismatch)
						// checkError(err)
						finder = finderPools[j].Get().(*AmpliconFinder)
						finder.Reset(record.Seq.Seq, maxMismatch)

						if usingRegion {
							loc, _, err = finder.LocateRange(begin, end, fregion, strict)
						} else {
							loc, _, err = finder.Locate()
						}
						checkError(err)

						if loc == nil {
							continue
						}

						matched = true

						if outFmtBED {
							start1, end1 = loc[0]-1, loc[1]
							if strand == "-" {
								start1, end1 = len(record.Seq.Seq)-loc[1], len(record.Seq.Seq)-loc[0]+1
							}
							if outputMismatches {
								fmt.Fprintf(outfh,
									"%s\t%d\t%d\t%s\t%d\t%s\t%s\t%d\t%d\t%d\n",
									record.ID,
									start1,
									end1,
									primer[0],
									0,
									strand,
									record.Seq.SubSeq(loc[0], loc[1]).Seq,
									0,
									0,
									0,
								)
							} else {
								fmt.Fprintf(outfh,
									"%s\t%d\t%d\t%s\t%d\t%s\t%s\n",
									record.ID,
									start1,
									end1,
									primer[0],
									0,
									strand,
									record.Seq.SubSeq(loc[0], loc[1]).Seq)
							}

							continue
						}
						tmpSeq = record.Seq

						record.Seq = record.Seq.SubSeq(loc[0], loc[1])
						if outputMismatches {
							record.Name = []byte(fmt.Sprintf("%s mismatches=%d(%d+%d)", name0, 0, 0, 0))
						}
						record.FormatToWriter(outfh, config.LineWidth)

						record.Seq = tmpSeq

						finderPools[j].Put(finder)
					}
				}

				if saveUnmatched && !matched {
					if !onlyPositiveStrand {
						record.Seq.RevComInplace()
					}
					record.FormatToWriter(outfh, config.LineWidth)
				}
			}
			fastxReader.Close()

			config.LineWidth = lineWidth
		}
	},
}

func init() {
	RootCmd.AddCommand(ampliconCmd)

	ampliconCmd.Flags().StringP("forward", "F", "", "forward primer (5'-primer-3'), degenerate bases allowed")
	ampliconCmd.Flags().StringP("reverse", "R", "", "reverse primer (5'-primer-3'), degenerate bases allowed")
	ampliconCmd.Flags().IntP("max-mismatch", "m", 0, "max mismatch when matching primers, no degenerate bases allowed")
	ampliconCmd.Flags().BoolP("output-mismatches", "M", false, "append the total mismatches and mismatches of 5' end and 3' end")
	ampliconCmd.Flags().StringP("primer-file", "p", "", "3- or 2-column tabular primer file, with first column as primer name")

	ampliconCmd.Flags().StringP("region", "r", "", `specify region to return. type "seqkit amplicon -h" for detail`)
	ampliconCmd.Flags().BoolP("flanking-region", "f", false, "region is flanking region")
	ampliconCmd.Flags().BoolP("strict-mode", "s", false, "strict mode, i.e., discarding seqs not fully matching (shorter) given region range")
	ampliconCmd.Flags().BoolP("only-positive-strand", "P", false, "only search on positive strand")
	ampliconCmd.Flags().BoolP("bed", "", false, "output in BED6+1 format with amplicon as the 7th column")
	ampliconCmd.Flags().BoolP("immediate-output", "I", false, "print output immediately, do not use write buffer")
	ampliconCmd.Flags().BoolP("save-unmatched", "u", false, "also save records that do not match any primer")
}

// only used in this command
func amplicon_mismatches(s1, s2 []byte) int {
	var n int
	for i, a := range s1 {
		if a != s2[i] {
			n++
		}
	}
	return n
}

func loadPrimers(file string) ([][3]string, error) {
	fh, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("load primers from '%s': %s", file, err)
	}

	var text string
	var items []string
	lists := make([][3]string, 0, 100)
	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		text = strings.TrimSpace(scanner.Text())
		if text == "" || text[0] == '#' {
			continue
		}

		items = strings.Split(text, "\t")
		switch len(items) {
		case 3:
			lists = append(lists, [3]string{items[0], items[1], items[2]})
		case 2:
			lists = append(lists, [3]string{items[0], items[1], ""})
		default:
			continue
		}
	}
	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("load primers from '%s': %s", file, err)
	}

	return lists, nil
}

func parsePrimers(primers [][3]string) ([][3][]byte, error) {
	list := make([][3][]byte, 0, len(primers))

	for _, items := range primers {
		name := []byte(items[0])

		forward := []byte(items[1])
		if seq.DNAredundant.IsValid(forward) != nil {
			return nil, fmt.Errorf("invalid primer sequence: %s", forward)
		}

		reverse := []byte(items[2])
		if seq.DNAredundant.IsValid(reverse) != nil {
			return nil, fmt.Errorf("invalid primer sequence: %s", reverse)
		}

		// compute revcom of reverse
		s, _ := seq.NewSeq(seq.DNAredundant, reverse)
		reverse = s.RevCom().Seq

		list = append(list, [3][]byte{name, forward, reverse})
	}
	return list, nil
}

// AmpliconFinder is a struct for locating amplicon via primer(s).
type AmpliconFinder struct {
	Seq []byte
	F   []byte // Forward primer
	R   []byte // R should be reverse complementary sequence of reverse primer

	MaxMismatch int
	FMindex     *fmi.FMIndex

	searched, found bool
	iBegin, iEnd    int // 0-based
	mis5, mis3      int

	rF, rR *regexp.Regexp
}

func (finder *AmpliconFinder) Reset(sequence []byte, maxMismatch int) error {
	if len(sequence) == 0 {
		return fmt.Errorf("non-blank sequence needed")
	}
	finder.Seq = bytes.ToUpper(sequence)
	if maxMismatch > 0 { // using FM-index
		index := fmi.NewFMIndex()
		_, err := index.Transform(finder.Seq)
		if err != nil {
			return err
		}
		finder.MaxMismatch = maxMismatch
		finder.FMindex = index
	}

	finder.searched, finder.found = false, false
	return nil
}

// NewAmpliconFinder returns a AmpliconFinder struct.
func NewAmpliconFinder(sequence, forwardPrimer, reversePrimerRC []byte, maxMismatch int) (*AmpliconFinder, error) {
	if len(sequence) == 0 {
		return nil, fmt.Errorf("non-blank sequence needed")
	}
	if len(forwardPrimer) == 0 && len(reversePrimerRC) == 0 {
		return nil, fmt.Errorf("at least one primer needed")
	}

	if len(forwardPrimer) == 0 { // F = R.revcom()
		forwardPrimer = reversePrimerRC
		reversePrimerRC = nil
	}

	finder := &AmpliconFinder{
		Seq: bytes.ToUpper(sequence), // to upper case
		F:   bytes.ToUpper(forwardPrimer),
		R:   bytes.ToUpper(reversePrimerRC),
	}

	if maxMismatch > 0 { // using FM-index
		index := fmi.NewFMIndex()
		_, err := index.Transform(finder.Seq)
		if err != nil {
			return nil, err
		}
		finder.MaxMismatch = maxMismatch
		finder.FMindex = index
	} else {
		if seq.DNA.IsValid(finder.F) != nil { // containing degenerate base
			if maxMismatch > 0 {
				checkError(fmt.Errorf("it does not support both degenerate base and mismatch"))
			}
			s, _ := seq.NewSeq(seq.DNA, finder.F)
			rF, err := regexp.Compile(s.Degenerate2Regexp())
			if err != nil {
				return nil, fmt.Errorf("fail to parse primer containing degenerate base: %s", finder.F)
			}
			finder.rF = rF
		}

		if seq.DNA.IsValid(finder.R) != nil { // containing degenerate base
			if maxMismatch > 0 {
				checkError(fmt.Errorf("it does not support both degenerate base and mismatch"))
			}
			s, _ := seq.NewSeq(seq.DNA, finder.R)
			rR, err := regexp.Compile(s.Degenerate2Regexp())
			if err != nil {
				return nil, fmt.Errorf("fail to parse primer containing degenerate base: %s", finder.R)
			}
			finder.rR = rR
		}
	}
	return finder, nil
}

// LocateRange returns location of the range (begin:end, 1-based).
func (finder *AmpliconFinder) LocateRange(begin, end int, flanking bool, strictMode bool) ([]int, []int, error) {
	if begin == 0 || end == 0 {
		checkError(fmt.Errorf("both begin and end in region should not be 0"))
	}

	if !finder.searched {
		_, _, err := finder.Locate()
		if err != nil {
			return nil, nil, err
		}
	}
	if !finder.found {
		return nil, nil, nil
	}

	var b, e int
	var ok bool
	if flanking {
		b, e, ok = SubLocationFlanking(len(finder.Seq), finder.iBegin, finder.iEnd, begin, end, strictMode)
	} else {
		b, e, ok = SubLocationInner(len(finder.Seq), finder.iBegin, finder.iEnd, begin, end, strictMode)
	}

	if ok {
		return []int{b, e}, []int{finder.mis5, finder.mis3}, nil
	}

	return nil, nil, nil
}

// SubLocationInner returns location of a range (begin:end, relative to amplicon).
// B/E: 0-based, location of amplicon.
// begin/end: 1-based, begin: relative location to 5' end of amplicon,
// end: relative location to 3' end of amplicon.
// Returned locations are 1-based.
//
//	            F
//	-----===============-----
//	     1 3 5                    x/y
//	              -5-3-1          x/y
//	     F             R
//	-----=====-----=====-----     x:y
//
//	     ===============          1:-1
//	     =======                  1:7
//	       =====                  3:7
//	          =====               6:10
//	          =====             -10:-6
//	             =====           -7:-3
//	                             -x:y (invalid)
func SubLocationInner(length, B, E, begin, end int, strictMode bool) (int, int, bool) {
	if begin == 0 || end == 0 {
		return 0, 0, false
	}

	if begin < 0 && end > 0 {
		checkError(fmt.Errorf("invalid inner region (-x:y): %d:%d", begin, end))
	}

	if length == 0 || B < 0 || B > length-1 || E < 0 || E > length-1 {
		return 0, 0, false
	}

	var b, e int

	if begin > 0 {
		b = B + begin
	} else {
		b = E + begin + 2
	}
	if b > length {
		if strictMode {
			return 0, 0, false
		}
		b = length
	} else if b < 1 {
		if strictMode {
			return 0, 0, false
		}
		b = 1
	}

	if end > 0 {
		e = B + end
	} else {
		e = E + end + 2
	}
	if e > length {
		if strictMode {
			return 0, 0, false
		}
		e = length
	} else if e < 1 {
		if strictMode {
			return 0, 0, false
		}
		e = 1
	}

	if b > e {
		return b, e, false
	}

	return b, e, true
}

// SubLocationFlanking returns location of a flanking range (begin:end, relative to amplicon).
// B/E: 0-based, location of amplicon.
// begin/end: 1-based, begin: relative location to 5' end of amplicon,
// end: relative location to 3' end of amplicon.
// Returned locations are 1-based.
//
//	            F
//	-----===============-----
//	 -3-1                        x/y
//	                    1 3 5    x/y
//	     F             R
//	-----=====-----=====-----
//	=====                        -5:-1
//	===                          -5:-3
//	                    =====     1:5
//	                      ===     3:5
//	    =================        -1:1
//	=========================    -5:5
//	                              x:-y (invalid)
func SubLocationFlanking(length, B, E, begin, end int, strictMode bool) (int, int, bool) {
	if begin == 0 || end == 0 {
		return 0, 0, false
	}

	if begin > 0 && end < 0 {
		checkError(fmt.Errorf("invalid flanking region (x:-y): %d:%d", begin, end))
	}

	if length == 0 || B < 0 || B > length-1 || E < 0 || E > length-1 {
		return 0, 0, false
	}

	var b, e int
	var flag bool // 5' flanking is shorter than -begin

	if begin > 0 {
		b = E + begin + 1
	} else {
		b = B + begin + 1
	}
	if b > length {
		// b = length
		return 0, 0, false
	} else if b < 1 {
		if strictMode {
			return 0, 0, false
		}
		b = 1
		flag = true
		// return 0, 0, false
	}

	if end > 0 {
		e = E + end + 1
	} else {
		e = B + end + 1
	}
	if e > length {
		if strictMode {
			return 0, 0, false
		}
		e = length
	} else if e < 1 {
		if strictMode {
			return 0, 0, false
		}
		if flag {
			return 0, 0, false
		}
		e = 1
	}

	if b > e {
		return b, e, false
	}

	return b, e, true
}

// Locate returns location of amplicon.
// Locations are 1-based, nil returns if not found.
func (finder *AmpliconFinder) Locate() ([]int, []int, error) {
	if finder.searched {
		if finder.found {
			return []int{finder.iBegin, finder.iEnd}, []int{finder.mis5, finder.mis3}, nil
		}
		return nil, nil, nil
	}

	if finder.MaxMismatch <= 0 { // exactly matching
		// search F
		var i int

		if finder.rF == nil {
			i = bytes.Index(finder.Seq, finder.F)
			if i < 0 { // not found
				finder.searched, finder.found = true, false
				return nil, nil, nil
			}
		} else {
			loc := finder.rF.FindSubmatchIndex(finder.Seq)
			if len(loc) == 0 {
				finder.searched, finder.found = true, false
				return nil, nil, nil
			}
			i = loc[0]
		}

		if len(finder.R) == 0 { // only forward primer, returns location of F
			finder.searched, finder.found = true, true
			finder.iBegin, finder.iEnd = i, i+len(finder.F)-1
			finder.mis5 = amplicon_mismatches(finder.Seq[i:i+len(finder.F)], finder.F)
			finder.mis3 = 0
			return []int{i + 1, i + len(finder.F)},
				[]int{finder.mis5, finder.mis3},
				nil
		}

		// two primers given, need to search R
		var j int
		if finder.rR == nil {
			j = bytes.Index(finder.Seq, finder.R)
			if j < 0 {
				finder.searched, finder.found = true, false
				return nil, nil, nil
			}

			for {
				if j+1 >= len(finder.Seq) {
					break
				}
				k := bytes.Index(finder.Seq[j+1:], finder.R)
				if k < 0 {
					break
				}
				j += k + 1
			}
		} else {
			loc := finder.rR.FindAllSubmatchIndex(finder.Seq, -1)
			if len(loc) == 0 {
				finder.searched, finder.found = true, false
				return nil, nil, nil
			}
			j = loc[len(loc)-1][0]
		}

		if j < i { // wrong location of F and R:  5' ---R-----F---- 3'
			finder.searched, finder.found = true, false
			return nil, nil, nil
		}
		finder.searched, finder.found = true, true
		finder.iBegin, finder.iEnd = i, j+len(finder.R)-1
		finder.mis5 = amplicon_mismatches(finder.Seq[i:i+len(finder.F)], finder.F)
		finder.mis3 = amplicon_mismatches(finder.Seq[j:j+len(finder.R)], finder.R)
		return []int{i + 1, j + len(finder.R)},
			[]int{finder.mis5, finder.mis3},
			nil
	}

	// search F
	locsI, err := finder.FMindex.Locate(finder.F, finder.MaxMismatch)
	if err != nil {
		return nil, nil, err
	}
	if len(locsI) == 0 { // F not found
		finder.searched, finder.found = true, false
		return nil, nil, nil
	}
	if len(finder.R) == 0 { // returns location of F
		sort.Ints(locsI) // remain the first location
		finder.searched, finder.found = true, true
		finder.iBegin, finder.iEnd = locsI[0], locsI[0]+len(finder.F)-1
		finder.mis5 = amplicon_mismatches(finder.Seq[locsI[0]:locsI[0]+len(finder.F)], finder.F)
		finder.mis3 = 0
		return []int{locsI[0] + 1, locsI[0] + len(finder.F)},
			[]int{finder.mis5, finder.mis3},
			nil
	}

	// search R
	locsJ, err := finder.FMindex.Locate(finder.R, finder.MaxMismatch)
	if err != nil {
		return nil, nil, err
	}
	if len(locsJ) == 0 {
		finder.searched, finder.found = true, false
		return nil, nil, nil
	}
	sort.Ints(locsI) // to remain the FIRST location
	sort.Ints(locsJ) // to remain the LAST location
	finder.searched, finder.found = true, true
	finder.iBegin, finder.iEnd = locsI[0], locsJ[len(locsJ)-1]+len(finder.R)-1
	finder.mis5 = amplicon_mismatches(finder.Seq[locsI[0]:locsI[0]+len(finder.F)], finder.F)
	finder.mis3 = amplicon_mismatches(finder.Seq[locsJ[len(locsJ)-1]:locsJ[len(locsJ)-1]+len(finder.R)], finder.R)
	return []int{locsI[0] + 1, locsJ[len(locsJ)-1] + len(finder.R)},
		[]int{finder.mis5, finder.mis3},
		nil
}

// Location returns location of amplicon.
// Locations are 1-based, nil returns if not found.
func (finder *AmpliconFinder) Location() ([]int, []int, error) {
	if !finder.searched {
		_, _, err := finder.Locate()
		if err != nil {
			return nil, nil, err
		}
	}
	if !finder.found {
		return nil, nil, nil
	}

	return []int{finder.iBegin + 1, finder.iEnd + 1}, []int{finder.mis5, finder.mis3}, nil
}
