// Copyright © 2019 Oxford Nanopore Technologies.
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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/biogo/hts/bam"
	"github.com/biogo/hts/sam"
	"github.com/bsipos/thist"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// RefCounts is a structure holding read count information for a given reference.
type RefCounts struct {
	Ref      *sam.Reference
	Count    float64
	SecCount float64
	SupCount float64
}

// ReadCounts holds read counts for all references.
type ReadCounts []*RefCounts

// NewReadCounts initializes a new read count slice.
func NewReadCounts(refs []*sam.Reference) ReadCounts {
	res := make(ReadCounts, len(refs))
	for i, _ := range res {
		res[i] = &RefCounts{Ref: refs[i]}
	}
	return res
}

// byCountRev is a utility type for sorting count structures in a decreasing order.
type byCountRev ReadCounts

func (s byCountRev) Len() int {
	return len(s)
}
func (s byCountRev) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byCountRev) Less(i, j int) bool {
	return s[i].Count > s[j].Count
}

// Sorted created a sorted copy of a read counts slice.
func (c ReadCounts) Sorted() ReadCounts {
	sc := make(ReadCounts, len(c))
	copy(sc, c)
	sort.Stable(byCountRev(sc))
	return sc
}

// load list of IDs from file
func loadIdList(f string) map[string]bool {
	fh, err := os.Open(f)
	checkError(err)
	r := bufio.NewReader(fh)
	res := make(map[string]bool)

	for {
		b, err := r.ReadBytes(byte('\n'))
		if err == nil {
			res[strings.TrimRight(string(b), "\n")] = true
		} else if err == io.EOF {
			break
		} else {
			checkError(err)
		}
	}
	return res
}

// filter records by ID
func filterById(id string, include map[string]bool, exclude map[string]bool) bool {
	if include != nil {
		if !include[id] {
			return true
		}
	}
	if exclude != nil {
		if exclude[id] {
			return true
		}
	}
	return false
}

// reportCounts prints per referecne count information.
func reportCounts(readCounts ReadCounts, countFile string, field string, rangeMin float64, rangeMax float64, printLog bool, printBins int, binMode string, printDump bool, title string, printPdf string, count int, printQuiet bool) {

	outw := os.Stdout
	if countFile != "-" {
		tw, err := os.Create(countFile)
		checkError(err)
		outw = tw
	}
	outfh := bufio.NewWriter(outw)

	prec := 0
	transform := func(x float64) float64 { return x }
	if printLog {
		prec = 6
		transform = func(x float64) float64 {
			return math.Log10(x + 1)
		}
	}
	digits := strconv.Itoa(prec)

	sortedCounts := readCounts.Sorted()

	totalCounts := make([]float64, 0, len(sortedCounts))
	outfh.WriteString("Ref\tCount\tSecCount\tSupCount\n")
	for _, cr := range sortedCounts {
		p := transform(cr.Count)
		if !math.IsNaN(rangeMax) && p >= rangeMax {
			continue
		}
		if !math.IsNaN(rangeMin) && p < rangeMin {
			break
		}
		totalCounts = append(totalCounts, p)
		outfh.WriteString(fmt.Sprintf("%s\t%."+digits+"f\t%."+digits+"f\t%."+digits+"f\n", cr.Ref.Name(), cr.Count, cr.SecCount, cr.SupCount))

	}

	outfh.Flush()
	if countFile != "-" {
		outw.Close()
	}

	h := thist.NewHist(totalCounts, fmt.Sprintf("%s - records: %d", title, count), binMode, printBins, true)
	if printDump {
		os.Stderr.Write([]byte(h.Dump()))
	} else {
		if !printQuiet {
			os.Stderr.Write([]byte(thist.ClearScreenString()))
			os.Stderr.Write([]byte(h.Draw()))
		}
		if printPdf != "" {
			h.SaveImage(printPdf)
		}

	}
}

// CountReads counts total, secondary and supplementary reads mapped to each reference.
func CountReads(bamReader *bam.Reader, bamWriter *bam.Writer, countFile string, field string, rangeMin, rangeMax float64, printPass bool, printPrim bool, printLog bool, printBins int, binMode string, mapQual int, printFreq int, printDump bool, printDelay int, printPdf string, execBefore, execAfter string, includeIds map[string]bool, excludeIds map[string]bool, printQuiet bool) {
	readCounts := NewReadCounts(bamReader.Header().Refs())
	validFields := []string{"Count", "SecCount", "SupCount"}
	fields := strings.Split(field, ",")
	_ = fields
	if field == "" {
		fields = validFields
	}

	title := "Read count distribution"

	var count int
	var unmapped int

	for {
		record, err := bamReader.Read()

		if err == io.EOF {
			break
		}
		checkError(err)

		if filterById(record.Name, includeIds, excludeIds) {
			continue
		}

		if record.Flags&sam.Unmapped == 0 {
			if printPrim && (record.Flags&sam.Supplementary != 0) {
				continue
			}
			if printPrim && (record.Flags&sam.Secondary != 0) {
				continue
			}

			if int(record.MapQ) < mapQual {
				continue
			}

			count++

			readCounts[record.RefID()].Count++
			if record.Flags&sam.Supplementary != 0 {
				readCounts[record.RefID()].SupCount++
			}
			if record.Flags&sam.Secondary != 0 {
				readCounts[record.RefID()].SecCount++
			}

			if printPass {
				checkError(bamWriter.Write(record))
			}

			if printFreq > 0 && count%printFreq == 0 {
				if execBefore != "" {
					BashExec(execBefore)
				}
				reportCounts(readCounts, countFile, field, rangeMin, rangeMax, printLog, printBins, binMode, printDump, title, printPdf, count, printQuiet)
				time.Sleep(time.Duration(printDelay) * time.Second)
				if execAfter != "" {
					BashExec(execAfter)
				}

			}
		} else {
			unmapped++
			if printPass {
				bamWriter.Write(record)
			}
		}
	} // records

	if printFreq < 0 || count%printFreq != 0 {
		if execBefore != "" {
			BashExec(execBefore)
		}
		reportCounts(readCounts, countFile, field, rangeMin, rangeMax, printLog, printBins, binMode, printDump, title, printPdf, count, printQuiet)
		time.Sleep(time.Duration(printDelay) * time.Second)
		if execAfter != "" {
			BashExec(execAfter)
		}
	}
	if printPass {
		bamWriter.Close()
	}

}

// bamStatRec is a structure holding BAM statistics.
type bamStatRec struct {
	PrimAlnPerc  float64
	MultimapPerc float64
	PrimAln      int
	SecAln       int
	SupAln       int
	Unmapped     int
	TotalRec     int
	File         string
}

// String generates string representatION for a pointer to bamStatRec.
func (r *bamStatRec) String() string {
	return fmt.Sprintf("%.2f\t%d\t%d\t%d\t%d\t%.2f\t%d\t%s", r.PrimAlnPerc, r.PrimAln, r.SecAln, r.SupAln, r.Unmapped, r.MultimapPerc, r.TotalRec, r.File)
}

// bamIdxStats extracts rough statistics form the BAM index.
func bamIdxStats(file string) *bamStatRec {
	idx := file + ".bai"
	var err error
	_, err = os.Stat(idx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Index file not found for file %s! Please run samtools index on the sorted file!\n", file)
		os.Exit(1)
	}
	checkError(err)
	fh, err := xopen.Ropen(idx)
	checkError(err)
	i, err := bam.ReadIndex(fh)
	res := new(bamStatRec)
	um, ok := i.Unmapped()
	if !ok {
		fmt.Fprintf(os.Stderr, "Unmapped counts are invalid in the index of %s!\n", file)
		os.Exit(1)
	}
	res.Unmapped = int(um)
	for j := 0; j < i.NumRefs(); j++ {
		st, ok := i.ReferenceStats(j)
		if !ok {
			continue
		}
		res.PrimAln += int(st.Mapped)
	}
	res.File = file
	return res
}

// bamIdxCount extracts rough count information from the BAM index. Secondary and supplementary alignments are not mapped.
func bamIdxCount(file string) {
	idx := file + ".bai"
	var err error
	_, err = os.Stat(idx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Index file not found for file %s! Please run samtools index on the sorted file!\n", file)
		os.Exit(1)
	}
	checkError(err)
	fh, err := xopen.Ropen(idx)
	checkError(err)
	i, err := bam.ReadIndex(fh)
	checkError(err)
	bamReader := NewBamReader(file, 1)

	fmt.Fprintf(os.Stderr, "Ref\tCount\tSecCount\tSupCount\n")
	for j := 0; j < i.NumRefs(); j++ {
		st, ok := i.ReferenceStats(j)
		if !ok {
			continue
		}
		fmt.Fprintf(os.Stderr, "%s\t%d\t\t\n", bamReader.Header().Refs()[j].Name(), int(st.Mapped))
	}
}

// bamStatsOnce calculates detailed statistics for a single BAM file.
func bamStatsOnce(f string, mapQual int, includeIds map[string]bool, excludeIds map[string]bool, threads int) *bamStatRec {
	bamReader := NewBamReader(f, threads)
	res := new(bamStatRec)
	res.File = f
	for {
		record, err := bamReader.Read()

		if err == io.EOF {
			break
		}
		checkError(err)

		if filterById(record.Name, includeIds, excludeIds) {
			continue
		}

		res.TotalRec++

		if record.Flags&sam.Unmapped == 0 {
			if record.Flags&sam.Supplementary != 0 {
				res.SupAln++
				continue
			} else if record.Flags&sam.Secondary != 0 {
				res.SecAln++
				continue
			} else {
				if record.MapQ == 0 {
					res.MultimapPerc++
				}
				if int(record.MapQ) < mapQual {
					continue
				}
				res.PrimAln++
			}

		} else {
			res.Unmapped++
		}
	} // records
	res.PrimAlnPerc = 100 * float64(res.PrimAln) / float64(res.PrimAln+res.Unmapped)
	res.MultimapPerc = 100 * res.MultimapPerc / float64(res.PrimAln)
	return res
}

// bamStats calculates detailed statistics for multiple BAM files and prints to stderr.
func bamStats(files []string, mapQual int, includeIds map[string]bool, excludeIds map[string]bool, threads int) {
	fmt.Fprintf(os.Stderr, "PrimAlnPerc\tPrimAln\tSecAln\tSupAln\tUnmapped\tMultimapPerc\tTotalRec\tFile\n")
	for _, f := range files {
		s := bamStatsOnce(f, mapQual, includeIds, excludeIds, threads)
		fmt.Fprintf(os.Stderr, "%s\n", s.String())
	}
}

// idxStats print rough statistics for multiple BAM files to stderr.
func idxStats(files []string) {
	fmt.Fprintf(os.Stderr, "Aligned\tUnmapped\tTotalRec\tFile\n")
	for _, f := range files {
		s := bamIdxStats(f)
		fmt.Fprintf(os.Stderr, "%d\t%d\t%d\t%s\n", s.PrimAln, s.Unmapped, s.PrimAln+s.Unmapped, s.File)
	}
}

// topEntry is a struct holding a SAM rcord along with a calculated field.
type topEntry struct {
	Record *sam.Record
	Value  float64
}

// TopBuffer is a slice of topEntries.
type TopBuffer []topEntry

// bamCmd represents the bam command
var bamCmd = &cobra.Command{
	Use:   "bam",
	Short: "monitoring and online histograms of BAM record features",
	Long:  "monitoring and online histograms of BAM record features",
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		idRegexp := config.IDRegexp
		_ = idRegexp
		outFile := config.OutFile
		runtime.GOMAXPROCS(config.Threads)

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		mapQual := getFlagInt(cmd, "map-qual")
		field := getFlagString(cmd, "field")
		printFreq := getFlagInt(cmd, "print-freq")
		rangeMin := getFlagFloat64(cmd, "range-min")
		rangeMax := getFlagFloat64(cmd, "range-max")
		printCount := getFlagString(cmd, "count")
		printPdf := getFlagString(cmd, "img")
		//printScatter := getFlagString(cmd, "scatter")
		//_ = printScatter
		printDump := getFlagBool(cmd, "dump")
		printLog := getFlagBool(cmd, "log")
		printDelay := getFlagInt(cmd, "delay")
		printStat := getFlagBool(cmd, "stat")
		printIdxStat := getFlagBool(cmd, "idx-stat")
		printIdxCount := getFlagBool(cmd, "idx-count")
		if printDelay < 0 {
			printDelay = 0
		}
		printReset := getFlagBool(cmd, "reset")
		printBins := getFlagInt(cmd, "bins")
		printPass := getFlagBool(cmd, "pass")
		printPrim := getFlagBool(cmd, "prim-only")
		printHelp := getFlagBool(cmd, "list-fields")
		printQuiet := getFlagBool(cmd, "quiet-mode")
		silentMode := getFlagBool(cmd, "silent-mode")

		execBefore := getFlagString(cmd, "exec-before")
		execAfter := getFlagString(cmd, "exec-after")
		printTop := getFlagString(cmd, "top-bam")
		topSize := getFlagInt(cmd, "top-size")
		includeIdList := getFlagString(cmd, "grep-ids")
		excludeIdList := getFlagString(cmd, "exclude-ids")

		var includeIds map[string]bool
		var excludeIds map[string]bool

		if includeIdList != "" {
			includeIds = loadIdList(includeIdList)
		}

		if excludeIdList != "" {
			excludeIds = loadIdList(excludeIdList)
		}

		if printIdxStat {
			idxStats(files)
			os.Exit(0)
		}

		if printStat {
			bamStats(files, mapQual, includeIds, excludeIds, config.Threads)
			os.Exit(0)
		}

		if printIdxCount {
			bamIdxCount(files[0])
			os.Exit(0)
		}

		binMode := "termfit"
		if printBins > 0 {
			binMode = "fixed"
		}

		if printPass && printCount == "-" {
			fmt.Fprintf(os.Stderr, "Cannot enable pass-through mode when count output is stdout!\n")
			os.Exit(1)
		}

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

		validFields := []string{"Read", "Ref", "Pos", "EndPos", "MapQual", "Acc", "ReadLen", "RefLen", "RefAln", "RefCov", "ReadAln", "ReadCov", "Strand", "MeanQual", "LeftClip", "RightClip", "Flags", "IsSec", "IsSup"}

		fields := strings.Split(field, ",")
		if field == "" {
			fields = validFields
		}

		type fieldInfo struct {
			Title    string
			Generate func(*sam.Record) float64
		}

		fmap := make(map[string]fieldInfo)

		getLeftClip := func(r *sam.Record) float64 {
			if r.Flags&sam.Unmapped != 0 {
				return 0
			}
			if r.Cigar[0].Type() == sam.CigarSoftClipped || r.Cigar[0].Type() == sam.CigarHardClipped {
				return float64(r.Cigar[0].Len())
			}
			return 0
		}

		getRightClip := func(r *sam.Record) float64 {
			if r.Flags&sam.Unmapped != 0 {
				return 0
			}
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

		fmap["Acc"] = fieldInfo{
			"Alignment accuracy",
			func(r *sam.Record) float64 {
				var mismatch int
				aux, ok := r.Tag([]byte("NM"))
				if !ok {
					panic("no NM tag")
				}
				var mm int
				var ins int
				var del int
				var skip int
				switch aux.Value().(type) {
				case int:
					mismatch = int(aux.Value().(int))
				case int8:
					mismatch = int(aux.Value().(int8))
				case int16:
					mismatch = int(aux.Value().(int16))
				case int32:
					mismatch = int(aux.Value().(int32))
				case int64:
					mismatch = int(aux.Value().(int64))
				case uint:
					mismatch = int(aux.Value().(uint))
				case uint8:
					mismatch = int(aux.Value().(uint8))
				case uint16:
					mismatch = int(aux.Value().(uint16))
				case uint32:
					mismatch = int(aux.Value().(uint32))
				case uint64:
					mismatch = int(aux.Value().(uint64))
				default:
					panic("Could not parse NM tag: " + aux.String())
				}
				for _, op := range r.Cigar {
					switch op.Type() {
					case sam.CigarMatch, sam.CigarEqual, sam.CigarMismatch:
						mm += op.Len()
					case sam.CigarInsertion:
						ins += op.Len()
					case sam.CigarDeletion:
						del += op.Len()
					case sam.CigarSkipped:
						skip += op.Len()
					default:
						//fmt.Println(op)
					}
				}
				return (1.0 - float64(mismatch)/float64(mm+ins+del)) * 100
			},
		}

		fmap["ReadLen"] = fieldInfo{
			"Read length",
			func(r *sam.Record) float64 {
				if r.Seq.Length > 0 {
					sl := float64(r.Seq.Length) + getHardClipped(r)
					return float64(sl)
				}
				var ql int
				for _, op := range r.Cigar {
					ql += op.Len() * op.Type().Consumes().Query
				}
				return float64(ql)
			},
		}

		fmap["RefLen"] = fieldInfo{
			"Reference length",
			func(r *sam.Record) float64 {
				return float64(r.Ref.Len())
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
			"Aligned read length",
			func(r *sam.Record) float64 {
				sl := fmap["ReadLen"].Generate(r)
				if r.Flags&sam.Unmapped != 0 {
					return 0
				}
				return (float64(sl) - getLeftClip(r) - getRightClip(r))
			},
		}

		fmap["ReadCov"] = fieldInfo{
			"Read coverage",
			func(r *sam.Record) float64 {
				sl := fmap["ReadLen"].Generate(r)
				return float64(100 * (float64(sl) - getLeftClip(r) - getRightClip(r)) / float64(sl))
			},
		}

		fmap["LeftClip"] = fieldInfo{
			"Clipped bases on the left (hard and soft)",
			func(r *sam.Record) float64 {
				return getLeftClip(r)
			},
		}

		fmap["RightClip"] = fieldInfo{
			"Clipped bases on the right (hard and soft)",
			func(r *sam.Record) float64 {
				return getRightClip(r)
			},
		}

		fmap["Strand"] = fieldInfo{
			"Strand",
			func(r *sam.Record) float64 {
				if r.Strand() < int8(0) {
					return -1.0
				}
				return 1.0
			},
		}

		fmap["Flags"] = fieldInfo{
			"SAM record flags as an integer",
			func(r *sam.Record) float64 {
				return float64(r.Flags)
			},
		}

		fmap["IsSec"] = fieldInfo{
			"One if alignment is secondary, zero otherwise",
			func(r *sam.Record) float64 {
				if r.Flags&sam.Secondary != 0 {
					return 1
				}
				return 0
			},
		}

		fmap["IsSup"] = fieldInfo{
			"One if alignment is supplementary, zero otherwise",
			func(r *sam.Record) float64 {
				if r.Flags&sam.Supplementary != 0 {
					return 1
				}
				return 0
			},
		}

		fmap["MeanQual"] = fieldInfo{
			"Mean base quality of the read",
			func(r *sam.Record) float64 {
				if len(r.Qual) != r.Seq.Length {
					return 0.0
				}
				s := &seq.Seq{Qual: r.Qual}
				return s.AvgQual(0)
			},
		}

		fmap["Pos"] = fieldInfo{
			"Leftmost reference position (zero-based)",
			func(r *sam.Record) float64 {
				return float64(r.Pos)
			},
		}

		fmap["EndPos"] = fieldInfo{
			"Rightmost reference position (zero-based)",
			func(r *sam.Record) float64 {
				return float64(r.Pos + r.Len())
			},
		}

		if printHelp {
			for _, f := range validFields {
				fmt.Printf("%-10s\t%s\n", f, fmap[f].Title)
			}
			os.Exit(0)
		}

		bamReader := NewBamReader(files[0], config.Threads)
		bamHeader := bamReader.Header()

		var bamWriter *bam.Writer
		var topBuffer TopBuffer

		if printTop != "" {
			topBuffer = make(TopBuffer, topSize)
			for i := range topBuffer {
				topBuffer[i].Value = math.NaN()
			}
		}

		if printPass {
			bw, err := bam.NewWriter(outfh, bamHeader, 1)
			checkError(err)
			bamWriter = bw
			outfh.Flush()
		}

		if printCount != "" {
			CountReads(bamReader, bamWriter, printCount, field, rangeMin, rangeMax, printPass, printPrim, printLog, printBins, binMode, mapQual, printFreq, printDump, printDelay, printPdf, execBefore, execAfter, includeIds, excludeIds, printQuiet)
			outfh.Flush()
			outw.Close()
			return
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
				if fmap[f].Generate == nil {
					fmt.Fprintf(os.Stderr, "Invalid field: %s\n", f)
					fmt.Fprintf(os.Stderr, "The valid fields are:\n")
					for _, ff := range validFields {
						fmt.Printf("%-10s\t%s\n", ff, fmap[ff].Title)
					}
					os.Exit(1)
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

		if len(fields) > 1 || field == "Read" || field == "Ref" {
			if execBefore != "" {
				BashExec(execBefore)
			}
			if !silentMode {
				os.Stderr.Write([]byte(strings.Join(fields, "\t") + "\n"))
			}
			for {
				record, err := bamReader.Read()

				if err == io.EOF {
					break
				}
				checkError(err)

				if filterById(record.Name, includeIds, excludeIds) {
					continue
				}

				if record.Flags&sam.Unmapped == 0 {

					if printPrim && (record.Flags&sam.Supplementary != 0) {
						continue
					}
					if printPrim && (record.Flags&sam.Secondary != 0) {
						continue
					}

					if int(record.MapQ) < mapQual {
						continue
					}
					if !silentMode {
						os.Stderr.Write(marshall(record, fields))
					}

				} else {

				}

				if printPass {
					bamWriter.Write(record)
				}
			}
			if printPass {
				bamWriter.Close()
			}
			if execAfter != "" {
				BashExec(execAfter)
			}

			return
		}

		if fmap[field].Generate == nil {
			fmt.Fprintf(os.Stderr, "Invalid field: %s\n", field)
			fmt.Fprintf(os.Stderr, "The valid fields are:\n")
			for _, ff := range validFields {
				fmt.Printf("%-10s\t%s\n", ff, fmap[ff].Title)
			}
			os.Exit(1)
		}
		h := thist.NewHist([]float64{}, fmap[field].Title, binMode, printBins, true)

		var count int
		var unmapped int

		for {
			record, err := bamReader.Read()

			if err == io.EOF {
				break
			}
			checkError(err)

			if filterById(record.Name, includeIds, excludeIds) {
				continue
			}

			if record.Flags&sam.Unmapped == 0 {
				if printPrim && (record.Flags&sam.Supplementary != 0) {
					continue
				}
				if printPrim && (record.Flags&sam.Secondary != 0) {
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
				topBuffer = updateTop(record, p, topBuffer, printTop)

				if printPass {
					bamWriter.Write(record)
				}

				if printFreq > 0 && count%printFreq == 0 {
					if execBefore != "" {
						BashExec(execBefore)
					}
					if printDump {
						os.Stderr.Write([]byte(h.Dump()))
					} else {
						if !printQuiet {
							switch field {
							case "RefCov", "ReadCov":
								h.Title = fmt.Sprintf("%s (>= 80: %.1f)", fmap[field].Title, gtThanPerc(h, 80))
							}
							os.Stderr.Write([]byte(thist.ClearScreenString()))
							os.Stderr.Write([]byte(h.Draw()))
						}
					}
					if printPdf != "" {
						h.SaveImage(printPdf)
					}
					dumpTop(printTop, bamHeader, topBuffer)
					time.Sleep(time.Duration(printDelay) * time.Second)
					if execAfter != "" {
						BashExec(execAfter)
					}
					if printReset {
						h = thist.NewHist([]float64{}, fmap[field].Title, binMode, printBins, true)
					}
				}
			} else {
				unmapped++
				if printPass {
					bamWriter.Write(record)
				}
			}
		} // records

		if printFreq < 0 || count%printFreq != 0 {
			if execBefore != "" {
				BashExec(execBefore)
			}
			if printDump {
				os.Stderr.Write([]byte(h.Dump()))
			} else {
				if !printQuiet {
					switch field {
					case "RefCov", "ReadCov":
						h.Title = fmt.Sprintf("%s (>= 80: %.1f)", fmap[field].Title, gtThanPerc(h, 80))
					}
					os.Stderr.Write([]byte(thist.ClearScreenString()))
					os.Stderr.Write([]byte(h.Draw()))
				}
			}
			if printPdf != "" {
				h.SaveImage(printPdf)
			}
			dumpTop(printTop, bamHeader, topBuffer)
			if execAfter != "" {
				BashExec(execAfter)
			}
		}
		if printPass {
			bamWriter.Close()
		}
		outfh.Flush()

	},
}

// gtThanPerc calculates the number of elements in a slice greater than a specified value.
func gtThanPerc(h *thist.Hist, percent float64) float64 {
	var count float64
	for v, c := range h.DataMap {
		if v >= percent {
			count += c
		}
	}
	return count * 100 / float64(h.DataCount)
}

// NewBamReader creates a new BAM reader from file.
func NewBamReader(bamFile string, nrProc int) *bam.Reader {
	fh, err := os.Stdin, error(nil)
	if bamFile != "-" {
		fh, err = os.Open(bamFile)
		checkError(err)
	}

	reader, err := bam.NewReader(bufio.NewReader(fh), nrProc)
	checkError(err)

	return reader
}

// dumpTop saves to entries to a BAM files.
func dumpTop(printTop string, bamHeader *sam.Header, topBuffer TopBuffer) {
	if printTop == "" {
		return
	}
	var topBam *bam.Writer
	var topfh *os.File
	var err error
	topfh, err = os.Create(printTop)
	checkError(err)
	topbuff := bufio.NewWriter(topfh)
	topBam, err = bam.NewWriter(topbuff, bamHeader, 1)
	checkError(err)
	for _, r := range topBuffer {
		if r.Record != nil {
			topBam.Write(r.Record)
		}
	}

	topBam.Close()
	topbuff.Flush()
	topfh.Close()

}

// byTopValue is a utility type for sorting top entries.
type byTopValue TopBuffer

func (a byTopValue) Len() int           { return len(a) }
func (a byTopValue) Less(i, j int) bool { return a[i].Value < a[j].Value }
func (a byTopValue) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func updateTop(record *sam.Record, value float64, topBuffer TopBuffer, printTop string) TopBuffer {
	if printTop == "" {
		return nil
	}
	topBuffer = append(topBuffer, topEntry{record, value})
	sort.Stable(byTopValue(topBuffer))
	return topBuffer[1:]

}

func init() {
	RootCmd.AddCommand(bamCmd)

	bamCmd.Flags().IntP("map-qual", "q", 0, "minimum mapping quality")
	bamCmd.Flags().StringP("field", "f", "", "target fields")
	//bamCmd.Flags().StringP("scatter", "S", "", "scatter plot of two numerical fields")
	bamCmd.Flags().StringP("img", "O", "", "save histogram to this PDF/image file")
	bamCmd.Flags().IntP("print-freq", "p", -1, "print/report after this many records (-1 for print after EOF)")
	bamCmd.Flags().IntP("delay", "W", 1, "sleep this many seconds after plotting")
	bamCmd.Flags().IntP("bins", "B", -1, "number of histogram bins")
	bamCmd.Flags().Float64P("range-min", "m", math.NaN(), "discard record with field (-f) value less than this flag")
	bamCmd.Flags().Float64P("range-max", "M", math.NaN(), "discard record with field (-f) value greater than this flag")
	bamCmd.Flags().BoolP("dump", "y", false, "print histogram data to stderr instead of plotting")
	bamCmd.Flags().BoolP("stat", "s", false, "print BAM satistics of the input files")
	bamCmd.Flags().BoolP("idx-stat", "i", false, "fast statistics based on the BAM index")
	bamCmd.Flags().BoolP("idx-count", "C", false, "fast read per reference counting based on the BAM index")
	bamCmd.Flags().StringP("count", "c", "", "count reads per reference and save to this file")
	bamCmd.Flags().BoolP("log", "L", false, "log10(x+1) transform numeric values")
	bamCmd.Flags().BoolP("reset", "R", false, "reset histogram after every report")
	bamCmd.Flags().BoolP("pass", "x", false, "passthrough mode (forward filtered BAM to output)")
	bamCmd.Flags().BoolP("prim-only", "F", false, "filter out non-primary alignment records")
	bamCmd.Flags().BoolP("quiet-mode", "Q", false, "supress all plotting to stderr")
	bamCmd.Flags().BoolP("silent-mode", "Z", false, "supress TSV output to stderr")
	bamCmd.Flags().BoolP("list-fields", "H", false, "list all available BAM record features")
	bamCmd.Flags().StringP("exec-after", "e", "", "execute command after reporting")
	bamCmd.Flags().StringP("exec-before", "E", "", "execute command before reporting")
	bamCmd.Flags().StringP("top-bam", "@", "", "save the top -? records to this bam file")
	bamCmd.Flags().StringP("grep-ids", "g", "", "only keep records with IDs contained in this file")
	bamCmd.Flags().StringP("exclude-ids", "G", "", "exclude records with IDs contained in this file")
	bamCmd.Flags().IntP("top-size", "?", 100, "size of the top-mode buffer")
}
