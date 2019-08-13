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
	"io"
	"runtime"
	"strconv"
	"strings"

	"github.com/shenwei356/bio/featio/gtf"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fai"
	"github.com/shenwei356/bio/seqio/fastx"
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
				record.FormatToWriter(outfh, config.LineWidth)
			}

			config.LineWidth = lineWidth
		}
	},
}

func init() {
	RootCmd.AddCommand(ampliconCmd)

	ampliconCmd.Flags().StringP("forward", "f", "", "forward primer")
	ampliconCmd.Flags().StringP("reverse", "r", "", "reverse primer")
	ampliconCmd.Flags().IntP("max-mismatch", "m", 0, "max mismatch when matching primers")

	ampliconCmd.Flags().StringP("region", "r", "", "region")
}
