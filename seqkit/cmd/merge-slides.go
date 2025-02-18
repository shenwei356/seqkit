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
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// mergeSlidesCmd represents the mergeSlides command
var mergeSlidesCmd = &cobra.Command{
	GroupID: "misc",

	Use:   "merge-slides",
	Short: "merge sliding windows generated from seqkit sliding",
	Long: `merge sliding windows generated from seqkit sliding

For example,

    ref.contig00001_sliding:454531-454680
    ref.contig00001_sliding:454561-454710
    ref.contig00001_sliding:454591-454740
    ref.contig00002_sliding:362281-362430
    ref.contig00002_sliding:362311-362460
    ref.contig00002_sliding:362341-362490
    ref.contig00002_sliding:495991-496140
    ref.contig00044_sliding:1-150
    ref.contig00044_sliding:31-180
    ref.contig00044_sliding:61-210
    ref.contig00044_sliding:91-240

could be merged into

    ref.contig00001 454530  454740
    ref.contig00002 362280  362490
    ref.contig00002 495990  496140
    ref.contig00044 0       240

Output (BED3 format):
    1. chrom      - chromosome name
    2. chromStart - starting position (0-based)
    3. chromEnd   - ending position (0-based)

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		outFile := config.OutFile
		runtime.GOMAXPROCS(config.Threads)

		commentPrefixes := getFlagStringSlice(cmd, "comment-line-prefix")
		bufferSizeS := getFlagString(cmd, "buffer-size")
		if bufferSizeS == "" {
			checkError(fmt.Errorf("value of buffer size. supported unit: K, M, G"))
		}
		bufferSize, err := ParseByteSize(bufferSizeS)
		if err != nil {
			checkError(fmt.Errorf("invalid value of buffer size. supported unit: K, M, G"))
		}

		maxGap := getFlagNonNegativeInt(cmd, "max-gap")
		limitGap := maxGap > 0
		minOverlap := getFlagPositiveInt(cmd, "min-overlap")
		if !config.Quiet && minOverlap == 1 {
			log.Warningf("you may set -l/--min-overlap as $sliding_step_size - 1")
		}

		reQueryStr := getFlagString(cmd, "regexp")
		reQuery, err := regexp.Compile(reQueryStr)
		checkError(err)

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", !config.SkipFileCheck)

		if !config.Quiet {
			if len(files) == 1 && isStdin(files[0]) {
				log.Info("no files given, reading from stdin")
			} else {
				log.Infof("  %d input file(s) given", len(files))
			}
		}

		outFileClean := filepath.Clean(outFile)
		for _, file := range files {
			if isStdin(file) {
				// checkError(fmt.Errorf("stdin not supported"))
			} else if filepath.Clean(file) == outFileClean {
				checkError(fmt.Errorf("out file should not be one of the input file"))
			}
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var line, p string
		var isCommentLine bool
		var scanner *bufio.Scanner
		var fh *xopen.Reader
		buf := make([]byte, bufferSize)
		var founds [][]string
		var ref, ref0 string
		var begin, end, begin0, end0, begin1, end1 int
		var extend bool

		for _, file := range files {
			fh, err = xopen.Ropen(file)
			checkError(err)

			scanner = bufio.NewScanner(fh)
			scanner.Buffer(buf, int(bufferSize))

			ref0, begin0, end0 = "", 0, 0
			for scanner.Scan() {
				line = strings.TrimRight(scanner.Text(), "\r\n")

				if line == "" {
					continue
				}
				// check comment line
				isCommentLine = false
				for _, p = range commentPrefixes {
					if strings.HasPrefix(line, p) {
						isCommentLine = true
						break
					}
				}
				if isCommentLine {
					continue
				}

				founds = reQuery.FindAllStringSubmatch(line, 3)
				if len(founds) == 0 {
					checkError(fmt.Errorf("no reference and location found in the input  line"))
				}
				ref = founds[0][1]
				begin, _ = strconv.Atoi(founds[0][2])
				end, _ = strconv.Atoi(founds[0][3])

				// ------------------------------------------
				if begin0 > 0 { // not the first record
					extend = false

					// 1. the same chromesome; 2. has overlap
					if ref == ref0 && begin+minOverlap-1 <= end1 {
						if limitGap {
							if begin-begin1 <= maxGap {
								extend = true
							}
						} else {
							extend = true
						}
					}

					if extend {
						end0 = end
					} else { // print previous region
						fmt.Fprintf(outfh, "%s\t%d\t%d\n", ref0, begin0-1, end0)

						ref0, begin0, end0 = ref, begin, end
					}
				} else { // first record
					ref0, begin0, end0 = ref, begin, end
				}

				begin1, end1 = begin, end // last record
			}

			// print the last region
			fmt.Fprintf(outfh, "%s\t%d\t%d\n", ref0, begin0-1, end0)
		}
	},
}

func init() {
	RootCmd.AddCommand(mergeSlidesCmd)

	mergeSlidesCmd.Flags().StringSliceP("comment-line-prefix", "p", []string{"#", "//"}, "comment line prefix")
	mergeSlidesCmd.Flags().StringP("buffer-size", "b", "1G", `size of buffer, supported unit: K, M, G. You need increase the value when "bufio.Scanner: token too long" error reported`)

	mergeSlidesCmd.Flags().StringP("regexp", "r", `^(.+)_sliding:(\d+)\-(\d+)`, `regular expression for extract the reference name and window position.`)

	mergeSlidesCmd.Flags().IntP("max-gap", "g", 0, `maximum distance of starting positions of two adjacent regions, 0 for no limitation, 1 for no merging.`)
	mergeSlidesCmd.Flags().IntP("min-overlap", "l", 1, `minimum overlap of two adjacent regions, recommend $sliding_step_size - 1.`)
}
