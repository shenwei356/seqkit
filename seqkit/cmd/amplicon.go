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

                      R
        ---------======
        -a <<<<<<<<< -1         -a:-1
        -a <<<<<<< -b           -a:-b

               F/R
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

		var start, end int

		if region != "" {
			if !reRegion.MatchString(region) {
				checkError(fmt.Errorf(`invalid region: %s. type "seqkit amplicon -h" for more examples`, region))
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

		var record *fastx.Record
		var fastxReader *fastx.Reader

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

				if region != "" {

				}
				finder, err := NewAmpliconFinder(record.Seq.Seq, forward, reverse, maxMismatch)
				checkError(err)

				loc, err := finder.Locate()
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

	ampliconCmd.Flags().StringP("region", "r", "", "region")
}

type AmpliconFinder struct {
	Seq []byte
	F   []byte
	R   []byte
	Rrc []byte

	MaxMismatch int
	FMindex     *fmi.FMIndex
}

func NewAmpliconFinder(sequence, forwardPrimer, reversePrimer []byte, maxMismatch int) (*AmpliconFinder, error) {
	if len(sequence) == 0 {
		return nil, fmt.Errorf("non-blank sequence needed")
	}
	if len(forwardPrimer) == 0 && len(reversePrimer) == 0 {
		return nil, fmt.Errorf("at least one primer needed")
	}

	if len(forwardPrimer) == 0 {
		s, err := seq.NewSeq(seq.DNAredundant, reversePrimer)
		if err != nil {
			return nil, err
		}

		forwardPrimer = s.RevComInplace().Seq
		reversePrimer = nil
	}

	finder := &AmpliconFinder{
		Seq: bytes.ToUpper(sequence),
		F:   bytes.ToUpper(forwardPrimer),
		R:   bytes.ToUpper(reversePrimer),
	}

	if len(reversePrimer) > 0 {
		s, err := seq.NewSeq(seq.DNAredundant, finder.R)
		if err != nil {
			return nil, err
		}
		finder.Rrc = s.RevComInplace().Seq
	}

	if maxMismatch > 0 {
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

func (finder AmpliconFinder) Locate() ([]int, error) {
	if finder.MaxMismatch <= 0 {
		i := bytes.Index(finder.Seq, finder.F)
		if i < 0 {
			return nil, nil
		}
		if len(finder.Rrc) == 0 { // only forward primer
			return []int{i + 1, i + len(finder.F)}, nil
		}

		j := bytes.Index(finder.Seq, finder.Rrc)
		if j < 0 {
			return nil, nil
		}
		if j < i {
			return nil, nil
		}
		return []int{i + 1, j + len(finder.Rrc)}, nil
	}

	locsI, err := finder.FMindex.Locate(finder.F, finder.MaxMismatch)
	if err != nil {
		return nil, err
	}
	if len(locsI) == 0 {
		return nil, nil
	}
	if len(finder.Rrc) == 0 {
		sort.Ints(locsI)
		return []int{locsI[0] + 1, locsI[0] + len(finder.F)}, nil
	}
	locsJ, err := finder.FMindex.Locate(finder.Rrc, finder.MaxMismatch)
	if err != nil {
		return nil, err
	}
	if len(locsJ) == 0 {
		return nil, nil
	}
	sort.Ints(locsI)
	sort.Ints(locsJ)
	return []int{locsI[0] + 1, locsJ[len(locsJ)-1] + len(finder.Rrc)}, nil
}

func (finder AmpliconFinder) Range(begin, end int) []byte {
	return nil
}
