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
	"path"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/biogo/hts/bam"
	"github.com/biogo/hts/sam"
	"github.com/botond-sipos/thist"
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
	for i := range res {
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
	TotalReads   int
	File         string
}

// String generates string representatION for a pointer to bamStatRec.
func (r *bamStatRec) String() string {
	return fmt.Sprintf("%.2f\t%d\t%d\t%d\t%d\t%.2f\t%d\t%d\t%s", r.PrimAlnPerc, r.PrimAln, r.SecAln, r.SupAln, r.Unmapped, r.MultimapPerc, r.TotalRec, r.TotalReads, r.File)
}

func (r *bamStatRec) StatFields() ([]string, []string) {
	fields := []string{"PrimAlnPerc", "MultimapPerc", "PrimAln", "SecAln", "SupAln", "Unmapped", "TotalReads", "TotalRecords", "File"}
	res := make([]string, len(fields))
	for i, f := range fields {
		switch f {
		case "PrimAlnPerc":
			res[i] = fmt.Sprintf("%.2f", r.PrimAlnPerc)
		case "MultimapPerc":
			res[i] = fmt.Sprintf("%.2f", r.MultimapPerc)
		case "PrimAln":
			res[i] = fmt.Sprintf("%d", r.PrimAln)
		case "SecAln":
			res[i] = fmt.Sprintf("%d", r.SecAln)
		case "SupAln":
			res[i] = fmt.Sprintf("%d", r.SupAln)
		case "Unmapped":
			res[i] = fmt.Sprintf("%d", r.Unmapped)
		case "TotalReads":
			res[i] = fmt.Sprintf("%d", r.TotalReads)
		case "TotalRecords":
			res[i] = fmt.Sprintf("%d", r.TotalRec)
		case "File":
			res[i] = r.File
		default:
			panic(f)
		}
	}
	return fields, res
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
	checkError(err)
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
	res.TotalReads = res.PrimAln + res.Unmapped
	return res
}

// bamStats calculates detailed statistics for multiple BAM files and prints to stderr.
func bamStats(files []string, mapQual int, includeIds map[string]bool, excludeIds map[string]bool, threads int, pretty bool) {
	width := 0
	if pretty {
		width = -1
	}
	var fields []string
	var out [][]string
	for _, f := range files {
		s := bamStatsOnce(f, mapQual, includeIds, excludeIds, threads)
		fi, data := s.StatFields()
		if fields == nil {
			fields = fi
			out = make([][]string, len(fi))
			for i := range out {
				out[i] = make([]string, 0)
			}
		}
		for i, d := range data {
			out[i] = append(out[i], d)
		}
	}
	color := true
	if width == 0 {
		color = false
	}
	fs, brush := PrettyPrintTsv(fields, out, width, color)
	brush.WrapWriter(os.Stderr).Write([]byte(fs))
}

// idxStats print rough statistics for multiple BAM files to stderr.
func idxStats(files []string, pretty bool) {
	width := 0
	if pretty {
		width = -1
	}
	fields := []string{"AlnPerc", "Aligned", "Unmapped", "TotalRec", "File"}
	data := make([][]string, len(fields))
	for _, f := range files {
		s := bamIdxStats(f)
		for i, fi := range fields {
			switch fi {
			case "AlnPerc":
				data[i] = append(data[i], fmt.Sprintf("%.2f", float64(s.PrimAln*100)/float64(s.PrimAln+s.Unmapped)))
			case "Aligned":
				data[i] = append(data[i], fmt.Sprintf("%d", s.PrimAln))
			case "Unmapped":
				data[i] = append(data[i], fmt.Sprintf("%d", s.Unmapped))
			case "TotalRec":
				data[i] = append(data[i], fmt.Sprintf("%d", s.PrimAln+s.Unmapped))
			case "File":
				data[i] = append(data[i], s.File)
			}
		}
	}
	color := true
	if width == 0 {
		color = false
	}
	fs, brush := PrettyPrintTsv(fields, data, width, color)
	brush.WrapWriter(os.Stderr).Write([]byte(fs))
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
		prettyTSV := getFlagBool(cmd, "pretty")
		rangeMin := getFlagFloat64(cmd, "range-min")
		rangeMax := getFlagFloat64(cmd, "range-max")
		printCount := getFlagString(cmd, "count")
		printBundle := getFlagInt(cmd, "bundle")
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
		toolYaml := getFlagString(cmd, "tool")
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
			idxStats(files, prettyTSV)
			os.Exit(0)
		}

		if printStat {
			bamStats(files, mapQual, includeIds, excludeIds, config.Threads, prettyTSV)
			os.Exit(0)
		}

		if printIdxCount {
			bamIdxCount(files[0])
			os.Exit(0)
		}

		if toolYaml != "" {
			if toolYaml == "help" && len(toolYaml) == 0 {
				files = []string{"-"}

			}
			if len(files) != 1 {
				log.Fatal("The BAM toolbox takes exactly one input file!")
			}
			BamToolbox(toolYaml, files[0], outFile, printQuiet, silentMode, config.Threads)
			os.Exit(0)
		}

		if printBundle != 0 {
			if len(files) != 1 {
				log.Fatal("The BAM bundler takes exactly one input file!")
			}
			if outFile == "-" {
				outFile = path.Base(files[0]) + "_bundles"
			}
			Bam2Bundles(files[0], outFile, printBundle, config.Threads, printQuiet, silentMode)
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
			GetSamAcc,
		}

		fmap["ReadLen"] = fieldInfo{
			"Read length",
			func(r *sam.Record) float64 {
				if r.Seq.Length > 0 {
					sl := float64(r.Seq.Length) + float64(GetSamHardClipped(r))
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
			"Aligned reference length",
			func(r *sam.Record) float64 {
				return float64(r.Len())
			},
		}

		fmap["RefCov"] = fieldInfo{
			"Reference coverage",
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
			bw, err := bam.NewWriter(outfh, bamHeader, config.Threads)
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

type Locus struct {
	Chrom     string
	Start     int
	End       int
	Order     int
	NrRecords int
	Size      int
}

func Bam2Bundles(inBam string, outDir string, minBundle int, nrProcBam int, quiet, silent bool) {
	bamReader := NewBamReader(inBam, nrProcBam)
	if _, err := os.Stat(outDir); err == nil {
		log.Fatal("Cannot create output directory as it already exists:", outDir)
	}
	err := os.MkdirAll(outDir, 0750)
	if err != nil {
		log.Fatal("Could not create output directory for BAM bundles:", err)
	}
	bamHeader := bamReader.Header()
	var chrom string
	bStart := -1
	bEnd := -1
	recCache := make([]*sam.Record, 0, 20000)
	if !silent {
		log.Info("Creating BAM bundles from file:", inBam)
		log.Info("Minimum reads per bundle:", minBundle)
		log.Info("Output directory:", outDir)
	}
	var recCount, bundleCount int
	locusCount, bundleLoci := 0, 0
	if !quiet && !silent {
		os.Stderr.WriteString("Bundle\tChrom\tStart\tEnd\tNrRecs\tNrLoci\n")
	}
	for {
		rec, err := bamReader.Read()
		if err == io.EOF {
			break
		}
		checkError(err)
		if rec.Flags&sam.Unmapped != 0 {
			continue
		}
		recCount += 1
		if bStart == -1 || bEnd == -1 {
			bStart = rec.Start()
			bEnd = rec.End()
			chrom = rec.Ref.Name()
		}
		if (rec.Start() > bEnd) || (rec.Ref.Name() != chrom) {
			locusCount++
			bundleLoci++
			if minBundle == -1 || len(recCache) >= minBundle {
				bundleName := fmt.Sprintf("%09d_%s:%d:%d_bundle.bam", bundleCount, chrom, bStart, bEnd)
				outName := path.Join(outDir, bundleName)
				outFh, err := os.Create(outName)
				checkError(err)
				bamWriter, err := bam.NewWriter(outFh, bamHeader, nrProcBam)
				checkError(err)

				for _, r := range recCache {
					bamWriter.Write(r)
				}

				bamWriter.Close()
				outFh.Close()
				if !quiet && !silent {
					os.Stderr.WriteString(fmt.Sprintf("%d\t%s\t%d\t%d\t%d\t%d\n", bundleCount, chrom, bStart, bEnd, len(recCache), bundleLoci))
				}
				recCache = recCache[:0]
				bStart = rec.Start()
				bEnd = rec.End()
				chrom = rec.Ref.Name()
				bundleLoci = 0
				bundleCount++
			} else {
				if rec.Ref.Name() != chrom {
					chrom = rec.Ref.Name()
					bStart = rec.Start()
					bEnd = rec.End()
				}
			}
		} else {
			if len(recCache) > 0 && rec.Start() < recCache[len(recCache)-1].Start() {
				log.Fatal("BAM file is not sorted! Offending records:", recCache[len(recCache)-1].Ref.Name(), rec.Ref.Name())
			}
		}
		if rec.End() > bEnd {
			bEnd = rec.End()
		}
		recCache = append(recCache, rec)
	}

	//Write out final batch:
	if len(recCache) > 0 {
		bundleLoci++
		locusCount++
		bundleName := fmt.Sprintf("%09d_%s:%d:%d_bundle.bam", bundleCount, chrom, bStart, bEnd)
		outName := path.Join(outDir, bundleName)
		outFh, err := os.Create(outName)
		checkError(err)
		bamWriter, err := bam.NewWriter(outFh, bamHeader, nrProcBam)
		checkError(err)

		for _, r := range recCache {
			bamWriter.Write(r)
		}
		if !quiet && !silent {
			os.Stderr.WriteString(fmt.Sprintf("%d\t%s\t%d\t%d\t%d\t%d\n", bundleCount, chrom, bStart, bEnd, len(recCache), bundleLoci))
		}

		bamWriter.Close()
		outFh.Close()
		recCache = nil
	}

	if !silent {
		log.Info(fmt.Sprintf("Written %d BAM records to %d loci and %d bundles.", recCount, locusCount, bundleCount+1))
	}
}

func init() {
	RootCmd.AddCommand(bamCmd)

	bamCmd.Flags().IntP("map-qual", "q", 0, "minimum mapping quality")
	bamCmd.Flags().StringP("field", "f", "", "target fields")
	bamCmd.Flags().StringP("img", "O", "", "save histogram to this PDF/image file")
	bamCmd.Flags().IntP("print-freq", "p", -1, "print/report after this many records (-1 for print after EOF)")
	bamCmd.Flags().IntP("delay", "W", 1, "sleep this many seconds after plotting")
	bamCmd.Flags().IntP("bins", "B", -1, "number of histogram bins")
	bamCmd.Flags().IntP("bundle", "N", 0, "partition BAM file into loci (-1) or bundles with this minimum size")
	bamCmd.Flags().Float64P("range-min", "m", math.NaN(), "discard record with field (-f) value less than this flag")
	bamCmd.Flags().Float64P("range-max", "M", math.NaN(), "discard record with field (-f) value greater than this flag")
	bamCmd.Flags().BoolP("dump", "y", false, "print histogram data to stderr instead of plotting")
	bamCmd.Flags().BoolP("stat", "s", false, "print BAM satistics of the input files")
	bamCmd.Flags().BoolP("idx-stat", "i", false, "fast statistics based on the BAM index")
	bamCmd.Flags().BoolP("idx-count", "C", false, "fast read per reference counting based on the BAM index")
	bamCmd.Flags().StringP("count", "c", "", "count reads per reference and save to this file")
	bamCmd.Flags().StringP("tool", "T", "", "invoke toolbox in YAML format (see documentation)")
	bamCmd.Flags().BoolP("log", "L", false, "log10(x+1) transform numeric values")
	bamCmd.Flags().BoolP("reset", "R", false, "reset histogram after every report")
	bamCmd.Flags().BoolP("pass", "x", false, "passthrough mode (forward filtered BAM to output)")
	bamCmd.Flags().BoolP("prim-only", "F", false, "filter out non-primary alignment records")
	bamCmd.Flags().BoolP("quiet-mode", "Q", false, "supress all plotting to stderr")
	bamCmd.Flags().BoolP("silent-mode", "Z", false, "supress TSV output to stderr")
	bamCmd.Flags().BoolP("list-fields", "H", false, "list all available BAM record features")
	bamCmd.Flags().BoolP("pretty", "k", false, "pretty print certain TSV outputs")
	bamCmd.Flags().StringP("exec-after", "e", "", "execute command after reporting")
	bamCmd.Flags().StringP("exec-before", "E", "", "execute command before reporting")
	bamCmd.Flags().StringP("top-bam", "@", "", "save the top -? records to this bam file")
	bamCmd.Flags().StringP("grep-ids", "g", "", "only keep records with IDs contained in this file")
	bamCmd.Flags().StringP("exclude-ids", "G", "", "exclude records with IDs contained in this file")
	bamCmd.Flags().IntP("top-size", "?", 100, "size of the top-mode buffer")
}
