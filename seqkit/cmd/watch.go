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
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/biogo/hts/bam"
	"github.com/biogo/hts/sam"
	"github.com/bsipos/thist"
	"github.com/spf13/cobra"
)

// watchCmd represents the hist command
var watchCmd = &cobra.Command{
	Use:   "bam",
	Short: "plot online histograms of average quality/length/GC content/GC skew)",
	Long:  "plot online histograms of average quality/length/GC content/GC skew)",
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		idRegexp := config.IDRegexp
		_ = idRegexp
		outFile := config.OutFile
		runtime.GOMAXPROCS(config.Threads)

		files := getFileList(args)

		mapQual := getFlagInt(cmd, "map-qual")
		field := getFlagString(cmd, "field")
		printFreq := getFlagInt(cmd, "print-freq")
		rangeMin := getFlagFloat64(cmd, "range-min")
		rangeMax := getFlagFloat64(cmd, "range-max")
		printCount := getFlagBool(cmd, "count")
		_ = printCount
		printScatter := getFlagString(cmd, "scatter")
		_ = printScatter
		printDump := getFlagBool(cmd, "dump")
		printLog := getFlagBool(cmd, "log")
		printDelay := getFlagInt(cmd, "delay")
		printStat := getFlagBool(cmd, "stat")
		_ = printStat
		printIdxStat := getFlagBool(cmd, "idx-stat")
		_ = printIdxStat
		printIdxCount := getFlagBool(cmd, "idx-count")
		_ = printIdxCount
		if printDelay < 0 {
			printDelay = 0
		}
		printReset := getFlagBool(cmd, "reset")
		printBins := getFlagInt(cmd, "bins")
		printPass := getFlagBool(cmd, "pass")
		printPrim := getFlagBool(cmd, "prim-only")

		outw := os.Stdout
		if outFile != "-" {
			tw, err := os.Create(outFile)
			checkError(err)
			outw = tw
		}
		outfh := bufio.NewWriter(outw)

		defer outw.Close()

		transform := func(x float64) float64 { return x }
		if printLog {
			transform = func(x float64) float64 {
				return math.Log10(x + 1)
			}
		}

		validFields := []string{"Read", "Ref", "MapQual", "ReadLen", "RefLen", "RefAln", "RefCov", "ReadAln", "ReadCov", "Strand", "LeftClip", "RightClip"}
		_ = validFields

		type fieldInfo struct {
			Title    string
			Generate func(*sam.Record) float64
		}

		fmap := make(map[string]fieldInfo)

		getLeftClip := func(r *sam.Record) float64 {
			if r.Cigar[0].Type() == sam.CigarSoftClipped || r.Cigar[0].Type() == sam.CigarHardClipped {
				return float64(r.Cigar[0].Len())
			}
			return 0
		}

		getRightClip := func(r *sam.Record) float64 {
			last := len(r.Cigar) - 1
			if r.Cigar[last].Type() == sam.CigarSoftClipped || r.Cigar[last].Type() == sam.CigarHardClipped {
				return float64(r.Cigar[last].Len())
			}
			return 0
		}
		getLeftSoftClip := func(r *sam.Record) float64 {
			if r.Cigar[0].Type() == sam.CigarSoftClipped {
				return float64(r.Cigar[0].Len())
			}
			return 0
		}
		_ = getLeftSoftClip

		getRightSoftClip := func(r *sam.Record) float64 {
			last := len(r.Cigar) - 1
			if r.Cigar[last].Type() == sam.CigarSoftClipped {
				return float64(r.Cigar[last].Len())
			}
			return 0
		}
		_ = getRightSoftClip

		getHardClipped := func(r *sam.Record) float64 {
			var hc float64
			last := len(r.Cigar) - 1
			if r.Cigar[last].Type() == sam.CigarHardClipped {
				hc += float64(r.Cigar[last].Len())
			}
			if r.Cigar[0].Type() == sam.CigarHardClipped {
				hc += float64(r.Cigar[0].Len())
			}
			return hc
		}

		var bamReader *bam.Reader

		refMap := make(map[string]float64)
		for _, ref := range bamReader.Header().Refs() {
			refMap[ref.Name()] = float64(ref.Len())
		}

		var bamWriter *bam.Writer
		if printPass {
			bw, err := bam.NewWriter(outfh, bamReader.Header(), 1)
			checkError(err)
			bamWriter = bw
			outfh.Flush()
		}

		getRead := func(r *sam.Record) string {
			return r.Name
		}

		getRef := func(r *sam.Record) string {
			return r.Ref.Name()
		}

		fmap["MapQual"] = fieldInfo{
			"Mapping quality",
			func(r *sam.Record) float64 {
				return float64(int(r.MapQ))
			},
		}

		fmap["ReadLen"] = fieldInfo{
			"Aligned read length",
			func(r *sam.Record) float64 {
				sl := float64(len(r.Seq.Seq)) + getHardClipped(r)
				return float64(sl)
			},
		}

		fmap["RefLen"] = fieldInfo{
			"Aligned read length",
			func(r *sam.Record) float64 {
				return refMap[r.Ref.Name()]
			},
		}

		fmap["RefAln"] = fieldInfo{
			"Aligned refence length",
			func(r *sam.Record) float64 {
				return float64(r.Len())
			},
		}

		fmap["RefCov"] = fieldInfo{
			"Refence coverage",
			func(r *sam.Record) float64 {
				return float64(r.Len()) / float64(r.Ref.Len()) * 100
			},
		}

		fmap["ReadAln"] = fieldInfo{
			"Aligned refence length",
			func(r *sam.Record) float64 {
				sl := fmap["ReadLen"].Generate(r)
				return (float64(sl) - getLeftClip(r) - getRightClip(r))
			},
		}

		fmap["ReadCov"] = fieldInfo{
			"Aligned refence length",
			func(r *sam.Record) float64 {
				sl := fmap["ReadLen"].Generate(r)
				return float64(100 * (float64(sl) - getLeftClip(r) - getRightClip(r)) / float64(sl))
			},
		}

		fmap["LeftClip"] = fieldInfo{
			"Aligned refence length",
			func(r *sam.Record) float64 {
				return getLeftClip(r)
			},
		}

		fmap["RightClip"] = fieldInfo{
			"Aligned refence length",
			func(r *sam.Record) float64 {
				return getRightClip(r)
			},
		}

		fmap["Strand"] = fieldInfo{
			"Aligned refence length",
			func(r *sam.Record) float64 {
				if r.Strand() < int8(0) {
					return -1.0
				}
				return 1.0
			},
		}

		marshall := func(r *sam.Record, fields []string) []byte {
			tmp := make([]string, len(fields))
			for i, f := range fields {
				if f == "Read" {
					tmp[i] = getRead(r)
					continue
				}
				if f == "Ref" {
					tmp[i] = getRef(r)
					continue
				}
				p := transform(fmap[f].Generate(r))
				digits := 3
				if p-float64(int(p)) == 0 {
					digits = 0
				}
				tmp[i] = fmt.Sprintf("%."+strconv.Itoa(digits)+"f", p)
			}

			return []byte(strings.Join(tmp, "\t") + "\n")

		}

		fields := strings.Split(field, ",")
		if field == "" {
			fields = validFields
		}

		if len(fields) > 1 || field == "Read" || field == "Ref" {
			os.Stderr.Write([]byte(strings.Join(fields, "\t") + "\n"))
			for {
				record, err := bamReader.Read()

				if err == io.EOF {
					break
				}
				checkError(err)

				if record.Flags&sam.Unmapped == 0 {

					if printPrim && record.Flags&sam.Supplementary == 1 {
						continue
					}
					if printPrim && record.Flags&sam.Secondary == 1 {
						continue
					}

					if int(record.MapQ) < mapQual {
						continue
					}
					os.Stderr.Write(marshall(record, fields))

				} else {

				}

				if printPass {
					bamWriter.Write(record)
				}
			}
			bamWriter.Close()
			return
		}

		binMode := "termfit"
		if printBins > 0 {
			binMode = "fixed"
		}
		h := thist.NewHist([]float64{}, fmap[field].Title, binMode, printBins, true)

		var count int
		var unmapped int

		if files[0] == "-" && h.BinMode != "fixed" {
			h.BinMode = "fixed"
			h.MaxBins = 80
		}

		for {
			record, err := bamReader.Read()

			if err == io.EOF {
				break
			}
			checkError(err)

			if record.Flags&sam.Unmapped == 0 {
				if printPrim && record.Flags&sam.Supplementary == 1 {
					continue
				}
				if printPrim && record.Flags&sam.Secondary == 1 {
					continue
				}

				if int(record.MapQ) < mapQual {
					continue
				}

				p := transform(fmap[field].Generate(record))

				if !math.IsNaN(rangeMin) && p < rangeMin {
					continue
				}
				if !math.IsNaN(rangeMax) && p >= rangeMax {
					continue
				}

				count++
				h.Update(p)

				if printPass {
					bamWriter.Write(record)
				}

				if printFreq > 0 && count%printFreq == 0 {
					if printDump {
						os.Stderr.Write([]byte(h.Dump()))
					} else {
						os.Stderr.Write([]byte(thist.ClearScreenString()))
						os.Stderr.Write([]byte(h.Draw()))
					}
					time.Sleep(time.Duration(printDelay) * time.Second)
					if printReset {
						h = thist.NewHist([]float64{}, fmap[field].Title, binMode, printBins, true)
					}
				}
			} else {
				unmapped++
			}
		} // records

		if printFreq < 0 || count%printFreq != 0 {
			if printDump {
				os.Stderr.Write([]byte(h.Dump()))
			} else {
				os.Stderr.Write([]byte(thist.ClearScreenString()))
				os.Stderr.Write([]byte(h.Draw()))
			}
		}
		bamWriter.Close()
		outfh.Flush()

	},
}

func init() {
	RootCmd.AddCommand(watchCmd)

	watchCmd.Flags().IntP("map-qual", "q", 0, "minimum mapping quality")
	watchCmd.Flags().StringP("field", "f", "", "target field")
	watchCmd.Flags().StringP("scatter", "S", "", "scatter plot of two fields")
	watchCmd.Flags().IntP("print-freq", "p", -1, "print frequency (-1 for print after parsing)")
	watchCmd.Flags().IntP("delay", "W", 0, "sleep this many seconds after online plotting")
	watchCmd.Flags().IntP("bins", "B", -1, "number of histogram bins")
	watchCmd.Flags().Float64P("range-min", "m", math.NaN(), "number of histogram bins")
	watchCmd.Flags().Float64P("range-max", "M", math.NaN(), "number of histogram bins")
	watchCmd.Flags().BoolP("dump", "y", false, "dump histogram instead of plotting")
	watchCmd.Flags().BoolP("stat", "s", false, "dump histogram instead of plotting")
	watchCmd.Flags().BoolP("idx-stat", "i", false, "dump histogram instead of plotting")
	watchCmd.Flags().BoolP("idx-count", "C", false, "dump histogram instead of plotting")
	watchCmd.Flags().BoolP("count", "c", false, "dump histogram instead of plotting")
	watchCmd.Flags().BoolP("log", "L", false, "log10 transform numeric values")
	watchCmd.Flags().BoolP("reset", "R", false, "reset histogram after every print")
	watchCmd.Flags().BoolP("pass", "x", false, "passthrough mode (print filtered BAM to output)")
	watchCmd.Flags().BoolP("prim-only", "F", false, "passthrough mode (print filtered BAM to output)")
	watchCmd.Flags().BoolP("quiet", "Q", false, "supress all output to stderr)")
}
