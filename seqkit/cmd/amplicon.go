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
	"github.com/shenwei356/bwt/fmi"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// ampliconCmd represents the amplicon command
var ampliconCmd = &cobra.Command{
	Use:   "amplicon",
	Short: "extact amplicon via primer(s)",
	Long: `extact amplicon via primer(s).

Examples:
  1. Typical two primers:

        F                R
        =====--------=====
        1 >>>>>>>>>>>>> -1      1:-1
            a >>>>>>>> -b       a:-b
            a >>>>> b           a:b

  2. Sequence around one primer:
  
        F
        ======---------
        1 >>>>>>>>>>> b         1:b
            a >>>>>>> b         a:b

                      F
        ---------======
        -a <<<<<<<<< -1         -a:-1
        -a <<<<<<< -b           -a:-b

               F
        -----=======---
        -a >>>>>>>>>> b         -a:b

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

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		region := getFlagString(cmd, "region")

		forward0 := getFlagString(cmd, "forward")
		reverse0 := getFlagString(cmd, "reverse")
		maxMismatch := getFlagNonNegativeInt(cmd, "max-mismatch")

		forward := []byte(forward0)
		reverse := []byte(reverse0)

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
				checkError(fmt.Errorf("both begin and end should not be 0"))
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
					loc, err = finder.LocateRange(begin, end)
				} else {
					loc, err = finder.Locate()
				}
				checkError(err)

				fmt.Printf("found loc: %v\n", loc)
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

	ampliconCmd.Flags().StringP("region", "r", "", "region")
}

// AmpliconFinder is a struct for locating amplicon via primer(s).
type AmpliconFinder struct {
	Seq []byte
	F   []byte
	R   []byte
	Rrc []byte

	MaxMismatch int
	FMindex     *fmi.FMIndex

	searched, found bool
	iBegin, iEnd    int // 0-based
}

// NewAmpliconFinder returns a AmpliconFinder struct.
func NewAmpliconFinder(sequence, forwardPrimer, reversePrimer []byte, maxMismatch int) (*AmpliconFinder, error) {
	if len(sequence) == 0 {
		return nil, fmt.Errorf("non-blank sequence needed")
	}
	if len(forwardPrimer) == 0 && len(reversePrimer) == 0 {
		return nil, fmt.Errorf("at least one primer needed")
	}

	if len(forwardPrimer) == 0 { // F = R.revcom()
		s, err := seq.NewSeq(seq.DNAredundant, reversePrimer)
		if err != nil {
			return nil, err
		}

		forwardPrimer = s.RevComInplace().Seq
		reversePrimer = nil
	}

	finder := &AmpliconFinder{
		Seq: bytes.ToUpper(sequence), // to upper case
		F:   bytes.ToUpper(forwardPrimer),
		R:   bytes.ToUpper(reversePrimer),
	}

	if len(reversePrimer) > 0 { // R.revcom()
		s, err := seq.NewSeq(seq.DNAredundant, finder.R)
		if err != nil {
			return nil, err
		}
		finder.Rrc = s.RevComInplace().Seq
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
func (finder *AmpliconFinder) LocateRange(begin, end int) ([]int, error) {
	if begin == 0 || end == 0 {
		checkError(fmt.Errorf("both begin and end should not be 0"))
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

	length := finder.iEnd - finder.iBegin + 1
	fmt.Printf("length: %d\n", length)
	if len(finder.Rrc) > 0 { // two primers given
		b, e, ok := SubLocation(length, begin, end)
		fmt.Println(b, e, ok)
		if ok {
			return []int{finder.iBegin + b, finder.iBegin + e}, nil
		}
		return nil, nil
	}

	return nil, nil
}

func SubLocation(length, start, end int) (int, int, bool) {
	if length == 0 {
		return 0, 0, false
	}
	if start < 1 {
		if start == 0 {
			start = 1
		} else if start < 0 {
			if end < 0 && start > end {
				return start, end, false
			}

			if -start > length {
				return start, end, false
			}
			start = length + start + 1
		}
	}
	if start > length {
		return start, end, false
	}

	if end > length {
		end = length
	}
	if end < 1 {
		if end == 0 {
			end = -1
		}
		end = length + end + 1
	}

	if start-1 > end {
		return start - 1, end, false
	}
	return start, end, true
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
		if len(finder.Rrc) == 0 { // only forward primer, returns location of F
			finder.searched, finder.found = true, true
			finder.iBegin, finder.iEnd = i, i+len(finder.F)-1
			return []int{i + 1, i + len(finder.F)}, nil
		}

		// two primers given, need to search R
		j := bytes.Index(finder.Seq, finder.Rrc)
		if j < 0 {
			finder.searched, finder.found = true, false
			return nil, nil
		}

		if j < i { // wrong location of F and R:  5' ---R-----F---- 3'
			finder.searched, finder.found = true, false
			return nil, nil
		}
		finder.searched, finder.found = true, true
		finder.iBegin, finder.iEnd = i, j+len(finder.Rrc)-1
		return []int{i + 1, j + len(finder.Rrc)}, nil
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
	if len(finder.Rrc) == 0 { // returns location of F
		sort.Ints(locsI) // remain the first location
		finder.searched, finder.found = true, true
		finder.iBegin, finder.iEnd = locsI[0], locsI[0]+len(finder.F)-1
		return []int{locsI[0] + 1, locsI[0] + len(finder.F)}, nil
	}

	// search R
	locsJ, err := finder.FMindex.Locate(finder.Rrc, finder.MaxMismatch)
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
	finder.iBegin, finder.iEnd = locsI[0], locsI[0]+len(finder.Rrc)-1
	return []int{locsI[0] + 1, locsJ[len(locsJ)-1] + len(finder.Rrc)}, nil
}
