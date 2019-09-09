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
	"io"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/shenwei356/bio/featio/gtf"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fai"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/bwt"
	"github.com/shenwei356/bwt/fmi"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// ampliconCmd represents the amplicon command
var ampliconCmd = &cobra.Command{
	Use:   "amplicon",
	Short: "retrieve amplicon (or specific region around it) via primer(s)",
	Long: `retrieve amplicon (or specific region around it) via primer(s).

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

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		region := getFlagString(cmd, "region")
		fregion := getFlagBool(cmd, "flanking-region")

		forward0 := getFlagString(cmd, "forward")
		reverse0 := getFlagString(cmd, "reverse")
		maxMismatch := getFlagNonNegativeInt(cmd, "max-mismatch")
		strict := getFlagBool(cmd, "strict-mode")

		forward := []byte(forward0)
		if seq.DNAredundant.IsValid(forward) != nil {
			checkError(fmt.Errorf("invalid primer sequence: %s", forward))
		}

		reverse := []byte(reverse0)
		if seq.DNAredundant.IsValid(reverse) != nil {
			checkError(fmt.Errorf("invalid primer sequence: %s", reverse))
		}

		// compute revcom of reverse
		s, _ := seq.NewSeq(seq.DNAredundant, reverse)
		reverse = s.RevCom().Seq

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

		var record *fastx.Record
		var fastxReader *fastx.Reader

		var finder *AmpliconFinder
		var loc []int

		for _, file := range files {
			fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
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

				finder, err = NewAmpliconFinder(record.Seq.Seq, forward, reverse, maxMismatch)
				checkError(err)

				if usingRegion {
					loc, err = finder.LocateRange(begin, end, fregion, strict)
				} else {
					loc, err = finder.Locate()
				}
				checkError(err)

				if loc == nil {
					continue
				}

				record.Seq.SubSeqInplace(loc[0], loc[1])
				record.FormatToWriter(outfh, config.LineWidth)
			}

			config.LineWidth = lineWidth
		}
	},
}

func init() {
	RootCmd.AddCommand(ampliconCmd)

	ampliconCmd.Flags().StringP("forward", "F", "", "forward primer")
	ampliconCmd.Flags().StringP("reverse", "R", "", "reverse primer")
	ampliconCmd.Flags().IntP("max-mismatch", "m", 0, "max mismatch when matching primers")

	ampliconCmd.Flags().StringP("region", "r", "", `specify region to return. type "seqkit amplicon -h" for detail`)
	ampliconCmd.Flags().BoolP("flanking-region", "f", false, "region is flanking region")
	ampliconCmd.Flags().BoolP("strict-mode", "s", false, "strict mode, i.e., discarding seqs not fully matching (shorter) given region range")
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
	}
	return finder, nil
}

// LocateRange returns location of the range (begin:end, 1-based).
func (finder *AmpliconFinder) LocateRange(begin, end int, flanking bool, strictMode bool) ([]int, error) {
	if begin == 0 || end == 0 {
		checkError(fmt.Errorf("both begin and end in region should not be 0"))
	}

	if !finder.searched {
		_, err := finder.Locate()
		if err != nil {
			return nil, err
		}
	}
	if !finder.found {
		return nil, nil
	}

	var b, e int
	var ok bool
	if flanking {
		b, e, ok = SubLocationFlanking(len(finder.Seq), finder.iBegin, finder.iEnd, begin, end, strictMode)
	} else {
		b, e, ok = SubLocationInner(len(finder.Seq), finder.iBegin, finder.iEnd, begin, end, strictMode)
	}

	if ok {
		return []int{b, e}, nil
	}

	return nil, nil
}

// SubLocationInner returns location of a range (begin:end, relative to amplicon).
// B/E: 0-based, location of amplicon.
// begin/end: 1-based, begin: relative location to 5' end of amplicon,
// end: relative location to 3' end of amplicon.
// Returned locations are 1-based.
//
//                     F
//         -----===============-----
//              1 3 5                    x/y
//                       -5-3-1          x/y
//              F             R
//         -----=====-----=====-----     x:y
//
//              ===============          1:-1
//              =======                  1:7
//                =====                  3:7
//                   =====               6:10
//                   =====             -10:-6
//                      =====           -7:-3
//                                      -x:y (invalid)
//
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
//                     F
//         -----===============-----
//          -3-1                        x/y
//                             1 3 5    x/y
//              F             R
//         -----=====-----=====-----
//         =====                        -5:-1
//         ===                          -5:-3
//                             =====     1:5
//                               ===     3:5
//             =================        -1:1
//         =========================    -5:5
//                                       x:-y (invalid)
//
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
func (finder *AmpliconFinder) Locate() ([]int, error) {
	if finder.searched {
		if finder.found {
			return []int{finder.iBegin, finder.iEnd}, nil
		}
		return nil, nil
	}

	if finder.MaxMismatch <= 0 { // exactly matching
		// search F
		i := bytes.Index(finder.Seq, finder.F)
		if i < 0 { // not found
			finder.searched, finder.found = true, false
			return nil, nil
		}
		if len(finder.R) == 0 { // only forward primer, returns location of F
			finder.searched, finder.found = true, true
			finder.iBegin, finder.iEnd = i, i+len(finder.F)-1
			return []int{i + 1, i + len(finder.F)}, nil
		}

		// two primers given, need to search R
		j := bytes.Index(finder.Seq, finder.R)
		if j < 0 {
			finder.searched, finder.found = true, false
			return nil, nil
		}

		if j < i { // wrong location of F and R:  5' ---R-----F---- 3'
			finder.searched, finder.found = true, false
			return nil, nil
		}
		finder.searched, finder.found = true, true
		finder.iBegin, finder.iEnd = i, j+len(finder.R)-1
		return []int{i + 1, j + len(finder.R)}, nil
	}

	// search F
	locsI, err := finder.FMindex.Locate(finder.F, finder.MaxMismatch)
	if err != nil {
		return nil, err
	}
	if len(locsI) == 0 { // F not found
		finder.searched, finder.found = true, false
		return nil, nil
	}
	if len(finder.R) == 0 { // returns location of F
		sort.Ints(locsI) // remain the first location
		finder.searched, finder.found = true, true
		finder.iBegin, finder.iEnd = locsI[0], locsI[0]+len(finder.F)-1
		return []int{locsI[0] + 1, locsI[0] + len(finder.F)}, nil
	}

	// search R
	locsJ, err := finder.FMindex.Locate(finder.R, finder.MaxMismatch)
	if err != nil {
		return nil, err
	}
	if len(locsJ) == 0 {
		finder.searched, finder.found = true, false
		return nil, nil
	}
	sort.Ints(locsI) // to remain the FIRST location
	sort.Ints(locsJ) // to remain the LAST location
	finder.searched, finder.found = true, true
	finder.iBegin, finder.iEnd = locsI[0], locsI[0]+len(finder.R)-1
	return []int{locsI[0] + 1, locsJ[len(locsJ)-1] + len(finder.R)}, nil
}
