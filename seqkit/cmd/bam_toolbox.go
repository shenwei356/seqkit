// Copyright © 2020 Oxford Nanopore Technologies, 2020 Botond Sipos.

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
	"os"
	"regexp"
	"sort"

	"github.com/biogo/hts/bam"
	"github.com/biogo/hts/sam"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fai"
	"github.com/shenwei356/bio/seqio/fastx"
	syaml "github.com/smallfish/simpleyaml"
	"gopkg.in/yaml.v2"
)

type BamTool struct {
	Name string
	Desc string
	Use  func(params *BamToolParams)
}

type BamToolParams struct {
	Yaml    *syaml.Yaml
	InChan  chan *sam.Record
	OutChan chan *sam.Record
	Quiet   bool
	Silent  bool
	Threads int
	Rank    int
	Shed    Toolshed
}

type Toolshed map[string]BamTool

func (s Toolshed) String() string {
	tools := make([]string, 0, len(s))
	for t := range s {
		tools = append(tools, t)
	}
	sort.Strings(tools)
	res := "Tool\tDescription\n"
	res += "----\t-----------\n"
	for _, t := range tools {
		res += fmt.Sprintf("%s\t%s\n", s[t].Name, s[t].Desc)
	}
	return res
}

func NewToolshed() Toolshed {
	ts := map[string]BamTool{
		"AlnContext": BamTool{Name: "AlnContext", Desc: "filter records by the sequence context at start and end", Use: BamToolAlnContext},
		"AccStats":   BamTool{Name: "AccStats", Desc: "calculates mean accuracy weighted by aligment lengths", Use: BamToolAccStats},
		"Dump":       BamTool{Name: "Dump", Desc: "dump various record properties in TSV format", Use: BamToolDump},
		"help":       BamTool{Name: "help", Desc: "list all tools with description", Use: ListTools},
	}
	return ts
}

// NewBamReaderChan streams records and closes the reader before signaling EOF.
func NewBamReaderChan(inFile string, cp int, buff int, threads int) (chan *sam.Record, *bam.Reader) {
	outChan := make(chan *sam.Record, cp)
	fh, err := os.Stdin, error(nil)
	if inFile != "-" {
		fh, err = os.Open(inFile)
		checkError(err)
	}

	r, err := bam.NewReader(bufio.NewReaderSize(fh, buff), threads)
	if err != nil {
		if fh != os.Stdin {
			fh.Close()
		}
		checkError(fmt.Errorf("%s: %w", inFile, err))
	}
	go func() {
		defer close(outChan)
		if fh != os.Stdin {
			defer fh.Close()
		}
		defer r.Close()
		for {
			rec, err := r.Read()
			if err == io.EOF {
				return
			}
			if err != nil {
				checkError(fmt.Errorf("%s: %w", inFile, err))
			}
			outChan <- rec
		}
	}()
	return outChan, r
}

func NewBamSinkChan(cp int) (chan *sam.Record, chan bool) {
	outChan := make(chan *sam.Record, cp)
	doneChan := make(chan bool)
	go func() {
		for rec := range outChan {
			_ = rec
		}
		doneChan <- true
	}()

	return outChan, doneChan
}

// NewBamWriterChan reports completion only after compression and buffered writes succeed.
func NewBamWriterChan(inFile string, head *sam.Header, cp int, buff int, threads int) (chan *sam.Record, chan bool) {
	outChan := make(chan *sam.Record, cp)
	doneChan := make(chan bool)
	fh, err := os.Stdout, error(nil)
	if inFile != "-" {
		fh, err = os.Create(inFile)
		checkError(err)
	}

	bio := bufio.NewWriterSize(fh, buff)
	w, err := bam.NewWriter(bio, head, threads)
	checkError(err)
	go func() {
		for rec := range outChan {
			err := w.Write(rec)
			checkError(err)
		}
		checkError(w.Close())
		checkError(bio.Flush())
		if fh != os.Stdout {
			checkError(fh.Close())
		}
		doneChan <- true
	}()
	return outChan, doneChan
}

// parseBamToolbox retains the declared pipeline order rather than Go map order.
func parseBamToolbox(data []byte) (*syaml.Yaml, []string, error) {
	var ordered yaml.MapSlice
	if err := yaml.UnmarshalStrict(data, &ordered); err != nil {
		return nil, nil, err
	}
	keys := make([]string, 0, len(ordered))
	seen := make(map[string]bool, len(ordered))
	for _, item := range ordered {
		key, ok := item.Key.(string)
		if !ok || seen[key] {
			return nil, nil, fmt.Errorf("toolbox: invalid or repeated tool name: %v", item.Key)
		}
		seen[key] = true
		keys = append(keys, key)
	}
	y, err := syaml.NewYaml(data)
	return y, keys, err
}

// BamToolbox runs a validated pipeline using full input-path checks.
func BamToolbox(toolYaml string, inFile string, outFile string, quiet bool, silent bool, threads int) {
	runBamToolbox(toolYaml, inFile, outFile, quiet, silent, threads, nil, true)
}

// runBamToolbox also protects CLI auxiliary inputs and honors --skip-file-check.
func runBamToolbox(toolYaml string, inFile string, outFile string, quiet bool, silent bool, threads int, auxiliaryInputs []string, checkInputFiles bool) {
	if toolYaml == "help" {
		toolYaml = "help: true"
	}
	y, keys, err := parseBamToolbox([]byte(toolYaml))
	checkError(err)
	if y.Get("Yaml").IsFound() {
		if len(keys) != 1 {
			checkError(fmt.Errorf("toolbox: Yaml must be the only configuration key"))
		}
		conf, err := y.Get("Yaml").String()
		checkError(err)
		cb, err := os.ReadFile(conf)
		checkError(err)
		auxiliaryInputs = append(auxiliaryInputs, conf)
		y, keys, err = parseBamToolbox(cb)
		checkError(err)
	}

	shed := NewToolshed()
	if len(keys) == 1 && keys[0] == "help" {
		ListTools(&BamToolParams{Shed: shed})
		return
	}
	sink := false
	if y.Get("Sink").IsFound() {
		sink, err = y.Get("Sink").Bool()
		checkError(err)
	}
	outputs := []string{}
	if !sink {
		outputs = append(outputs, outFile)
	}
	stages := make([]string, 0, len(keys))
	var indexes, newIndexes []string
	for _, key := range keys {
		if key == "Sink" {
			continue
		}
		if _, ok := shed[key]; !ok || key == "help" {
			checkError(fmt.Errorf("toolbox: unknown tool: %s", key))
		}
		stages = append(stages, key)
		params := y.Get(key)
		if params.Get("Tsv").IsFound() {
			tsv, err := params.Get("Tsv").String()
			checkError(err)
			if tsv == "" {
				checkError(fmt.Errorf("%s: Tsv must not be empty", key))
			}
			outputs = append(outputs, tsv)
		}
		switch key {
		case "Dump":
			dumpFields(params)
		case "AlnContext":
			ref, err := params.Get("Ref").String()
			checkError(err)
			_, err = params.Get("LeftShift").Int()
			checkError(err)
			_, err = params.Get("RightShift").Int()
			checkError(err)
			for _, key := range []string{"RegexStart", "RegexEnd"} {
				if params.Get(key).IsFound() {
					pattern, err := params.Get(key).String()
					checkError(err)
					_, err = regexp.Compile(pattern)
					checkError(err)
				}
			}
			for _, key := range []string{"Stranded", "Invert"} {
				if params.Get(key).IsFound() {
					_, err = params.Get(key).Bool()
					checkError(err)
				}
			}
			_, err = os.Stat(ref)
			checkError(err)
			index := ref + ".seqkit.fai"
			checkError(checkFASTAIndexFile(index))
			auxiliaryInputs = append(auxiliaryInputs, ref)
			indexes = append(indexes, index)
			if fileNotExists(index) {
				newIndexes = append(newIndexes, index)
			}
		}
	}
	if len(stages) == 0 {
		checkError(fmt.Errorf("toolbox: no tool specified"))
	}
	checkOutputFiles([]string{inFile}, append(outputs, newIndexes...), checkInputFiles)
	checkOutputFiles(auxiliaryInputs, append(outputs, newIndexes...), true)
	checkOutputFiles(indexes, outputs, true)
	checkBAMOutputPaths(append(outputs, newIndexes...))

	chanCap := 5000
	ioBuff := 1024 * 128
	inChan, bamReader := NewBamReaderChan(inFile, chanCap, ioBuff, threads)
	var lastOut chan *sam.Record
	var doneChan chan bool
	if sink {
		lastOut, doneChan = NewBamSinkChan(chanCap)
	} else {
		lastOut, doneChan = NewBamWriterChan(outFile, bamReader.Header(), chanCap, ioBuff, threads)
	}
	for rank, tool := range stages {
		nextOut := lastOut
		if rank < len(stages)-1 {
			nextOut = make(chan *sam.Record, chanCap)
		}
		params := &BamToolParams{
			Yaml: y.Get(tool), InChan: inChan, OutChan: nextOut,
			Quiet: quiet, Silent: silent, Threads: threads, Rank: rank, Shed: shed,
		}
		go shed[tool].Use(params)
		inChan = nextOut
	}
	<-doneChan
}

func ListTools(p *BamToolParams) {
	os.Stderr.WriteString(p.Shed.String())
	os.Exit(0)
}

// BamToolAlnContext filters alignments by flanking reference sequences.
func BamToolAlnContext(p *BamToolParams) {
	// Signal completion only after TSV output and the reference have closed.
	defer close(p.OutChan)
	ref, err := p.Yaml.Get("Ref").String()
	checkError(err)
	idx := NewRefWitdFaidx(ref, false, p.Silent)
	defer closeOutput(idx.faidx)
	leftShift, err := p.Yaml.Get("LeftShift").Int()
	checkError(err)
	rightShift, err := p.Yaml.Get("RightShift").Int()
	checkError(err)
	var regStart *regexp.Regexp
	var regEnd *regexp.Regexp
	checkError(err)
	regStrStart, err := p.Yaml.Get("RegexStart").String()
	if err == nil {
		regStart, err = regexp.Compile(regStrStart)
		checkError(err)
	}
	regStrEnd, err := p.Yaml.Get("RegexEnd").String()
	if err == nil {
		regEnd, err = regexp.Compile(regStrEnd)
		checkError(err)
	}
	stranded, invert := false, false
	_ = invert
	stranded, _ = p.Yaml.Get("Stranded").Bool()
	invert, _ = p.Yaml.Get("Invert").Bool()
	tsvFh := os.Stderr
	tsvFile, err := p.Yaml.Get("Tsv").String()
	if err == nil && tsvFile != "-" {
		tsvFh, err = os.Create(tsvFile)
		checkError(err)
		defer closeOutput(tsvFh)
	}

	head := "Read\tRef\tStrand\tStartSeq\tStartMatch\tEndSeq\tEndMatch\n"
	_, err = tsvFh.WriteString(head)
	checkError(err)

	for r := range p.InChan {
		if !GetSamMapped(r) || r.Ref == nil {
			p.OutChan <- r
			continue
		}
		chrom := r.Ref.Name()
		startPos, endPos := r.Pos, r.End()
		startSeq, err := idx.IdxSubSeq(chrom, addBamOffset(startPos, leftShift), addBamOffset(startPos, rightShift))
		checkError(err)
		endSeq, err := idx.IdxSubSeq(chrom, addBamOffset(endPos, leftShift), addBamOffset(endPos, rightShift))
		checkError(err)
		strand := 1
		if GetSamReverse(r) {
			strand = -1
			if stranded {
				startSeq, endSeq = RevCompDNA(endSeq), RevCompDNA(startSeq)
			}
		}

		yes, no := -1, 1
		if invert {
			no, yes = yes, no
		}
		startMatch := no
		endMatch := no
		if regStart != nil {
			if regStart.MatchString(startSeq) {
				startMatch = yes
			}
		}

		if regEnd != nil {
			if regEnd.MatchString(endSeq) {
				endMatch = yes
			}
		}

		match := (startMatch == yes) || (endMatch == yes)

		info := fmt.Sprintf("%s\t%s\t%d\t%s\t%d\t%s\t%d\n", GetSamName(r), GetSamRef(r), strand, startSeq, startMatch, endSeq, endMatch)

		if match && !invert {
			p.OutChan <- r
			_, err = tsvFh.WriteString(info)
			checkError(err)
		} else if !match && invert {
			p.OutChan <- r
		} else {
			_, err = tsvFh.WriteString(info)
			checkError(err)

		}

	}
}

type RefWithFaidx struct {
	Fasta   string
	IdxFile string
	idx     fai.Index
	faidx   *fai.Faidx
	Cache   bool
}

// addBamOffset saturates extreme offsets so clipping cannot wrap to the other end.
func addBamOffset(position, offset int) int {
	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1
	if offset > 0 && position > maxInt-offset {
		return maxInt
	}
	if offset < 0 && position < minInt-offset {
		return minInt
	}
	return position + offset
}

// IdxSubSeq clips absolute, zero-based half-open coordinates to the reference.
func (idx *RefWithFaidx) IdxSubSeq(chrom string, start, end int) (string, error) {
	record, ok := idx.idx[chrom]
	if !ok {
		return "", fmt.Errorf("reference not found: %s", chrom)
	}
	start, end = max(0, start), min(record.Length, end)
	if start >= end {
		return "", nil
	}
	b, err := idx.faidx.SubSeq(chrom, start+1, end)
	return string(b), err
}

func NewRefWitdFaidx(file string, cache bool, quiet bool) *RefWithFaidx {
	fileFai := file + ".seqkit.fai"
	idRegexp := fastx.DefaultIDRegexp
	var idx fai.Index
	var err error
	if fileNotExists(fileFai) {
		if !quiet {
			log.Infof("create FASTA index for %s", file)
		}
		idx, err = fai.CreateWithIDRegexp(file, fileFai, idRegexp)
		checkError(err)
	} else {
		idx, err = fai.Read(fileFai)
		checkError(err)
	}

	var faidx *fai.Faidx
	faidx, err = fai.NewWithIndex(file, idx)
	checkError(err)

	i := &RefWithFaidx{
		Fasta:   file,
		IdxFile: fileFai,
		idx:     idx,
		faidx:   faidx,
		Cache:   cache,
	}
	return i
}

// BamToolAccStats reports mean and length-weighted alignment accuracy.
func BamToolAccStats(p *BamToolParams) {
	defer close(p.OutChan)
	totalLen := 0
	accSum := 0.0
	wAccSum := 0.0
	nr := 0.0
	tsvFh := os.Stderr
	tsvFile, err := p.Yaml.Get("Tsv").String()
	if err == nil && tsvFile != "-" {
		tsvFh, err = os.Create(tsvFile)
		checkError(err)
		defer closeOutput(tsvFh)
	}
	for r := range p.InChan {
		if GetSamMapped(r) {
			info := GetSamAlnDetails(r)
			totalLen += info.Len
			wAccSum += info.WAcc
			accSum += info.Acc
			nr++
		}
		p.OutChan <- r
	}
	var WeightedAcc, MeanAcc float64
	if totalLen > 0 {
		WeightedAcc = wAccSum / float64(totalLen)
	}
	if nr > 0 {
		MeanAcc = accSum / nr
	}
	_, err = fmt.Fprintf(tsvFh, "AccMean\tWeightedAccMean\n%.3f\t%.3f\n", MeanAcc, WeightedAcc)
	checkError(err)
}

type AlnDetails struct {
	Match         int
	Mismatch      int
	MatchMismatch int
	Insertion     int
	Deletion      int
	Skip          int
	Len           int
	Acc           float64
	WAcc          float64
}

// GetSamAlnDetails computes NM-dependent alignment statistics.
func GetSamAlnDetails(r *sam.Record) *AlnDetails { return samAlnDetails(r, true) }

// samAlnDetails reads NM only for fields that actually depend on it.
func samAlnDetails(r *sam.Record, needNM bool) *AlnDetails {
	var mismatch int
	res := new(AlnDetails)
	var mm, ins, del, skip int
	if needNM {
		aux, ok := r.Tag([]byte("NM"))
		if !ok {
			checkError(fmt.Errorf("%s: required NM tag is missing", r.Name))
		}

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
			checkError(fmt.Errorf("%s: could not parse NM tag: %s", r.Name, aux.String()))
		}
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
	res.MatchMismatch = mm
	// NM includes insertions and deletions; Mismatch counts substitutions only.
	if needNM {
		res.Mismatch = mismatch - ins - del
	}
	res.Insertion = ins
	res.Deletion = del
	res.Skip = skip
	res.Len = mm + ins + del
	res.Match = res.MatchMismatch - res.Mismatch
	alnLen := float64(mm + ins + del)
	if alnLen > 0 {
		res.Acc = (1.0 - float64(mismatch)/alnLen) * 100
		res.WAcc = res.Acc * float64(res.Len)
	} else {
		res.Acc = 0.0
		res.WAcc = 0.0
	}
	return res
}

// BamToolDump writes selected record fields and forwards the records unchanged.
func BamToolDump(p *BamToolParams) {
	defer close(p.OutChan)
	keys := dumpFields(p.Yaml)
	tsvFh := os.Stderr
	tsvFile, err := p.Yaml.Get("Tsv").String()
	if err == nil && tsvFile != "-" {
		tsvFh, err = os.Create(tsvFile)
		checkError(err)
		defer closeOutput(tsvFh)
	}

	_, err = tsvFh.WriteString(PrintTsvLine(keys))
	checkError(err)
	for r := range p.InChan {
		if GetSamMapped(r) {
			_, err = tsvFh.WriteString(PrintTsvLine(SamDumper(keys, r)))
			checkError(err)
		}
		p.OutChan <- r
	}
}

func SamDumper(fields []string, r *sam.Record) []string {
	res := make([]string, len(fields))
	for i, f := range fields {
		res[i] = GetSamDump(f, r)
	}
	return res
}

func GetSamDump(field string, r *sam.Record) string {
	var acc *AlnDetails
	switch field {
	case "Acc", "Match", "Mismatch":
		acc = samAlnDetails(r, true)
	case "Ins", "Del", "AlnLen":
		acc = samAlnDetails(r, false)
	}
	switch field {
	case "Read":
		return fmt.Sprintf("%s", GetSamName(r))
	case "Ref":
		return fmt.Sprintf("%s", GetSamRef(r))
	case "Pos":
		return fmt.Sprintf("%d", GetSamPos(r))
	case "EndPos":
		return fmt.Sprintf("%d", GetSamEndPos(r))
	case "MapQual":
		return fmt.Sprintf("%d", GetSamMapQual(r))
	case "Acc":
		return fmt.Sprintf("%.3f", acc.Acc)
	case "Match":
		return fmt.Sprintf("%d", acc.Match)
	case "Mismatch":
		return fmt.Sprintf("%d", acc.Mismatch)
	case "Ins":
		return fmt.Sprintf("%d", acc.Insertion)
	case "Del":
		return fmt.Sprintf("%d", acc.Deletion)
	case "AlnLen":
		return fmt.Sprintf("%d", acc.Len)
	case "ReadLen":
		return fmt.Sprintf("%d", GetSamReadLen(r))
	case "RefLen":
		return fmt.Sprintf("%d", GetSamRefLen(r))
	case "RefAln":
		return fmt.Sprintf("%d", GetSamRefAln(r))
	case "RefCov":
		return fmt.Sprintf("%.3f", GetSamRefCov(r))
	case "ReadAln":
		return fmt.Sprintf("%d", GetSamReadAln(r))
	case "ReadCov":
		return fmt.Sprintf("%.3f", GetSamReadCov(r))
	case "Strand":
		return fmt.Sprintf("%d", GetSamStrand(r))
	case "MeanQual":
		return fmt.Sprintf("%.3f", GetSamMeanBaseQual(r))
	case "LeftClip":
		return fmt.Sprintf("%d", GetSamLeftClip(r))
	case "RightClip":
		return fmt.Sprintf("%d", GetSamRightClip(r))
	case "Flags":
		return fmt.Sprintf("%d", r.Flags)
	case "IsSec":
		return fmt.Sprintf("%d", GetSamIsSec(r))
	case "IsSup":
		return fmt.Sprintf("%d", GetSamIsSup(r))
	case "ReadSeq":
		return fmt.Sprintf("%s", GetSamReadSeq(r))
	case "ReadAlnSeq":
		return fmt.Sprintf("%s", GetSamReadAlnSeq(r))
	case "LeftSoftClipSeq":
		return fmt.Sprintf("%s", GetSamLeftSoftClipSeq(r))
	case "RightSoftClipSeq":
		return fmt.Sprintf("%s", GetSamRightSoftClipSeq(r))
	case "LeftSoftClip":
		return fmt.Sprintf("%d", GetSamLeftSoftClip(r))
	case "RightSoftClip":
		return fmt.Sprintf("%d", GetSamRightSoftClip(r))
	case "LeftHardClip":
		return fmt.Sprintf("%d", GetSamLeftHardClip(r))
	case "RightHardClip":
		return fmt.Sprintf("%d", GetSamRightHardClip(r))
	default:
		return "<!INVALID_FIELD!>"

	}
}

func GetSamMapped(r *sam.Record) bool {
	return (r.Flags&sam.Unmapped == 0)
}

func GetSamReverse(r *sam.Record) bool {
	return (r.Flags&sam.Reverse != 0)
}

func GetSamRef(r *sam.Record) string {
	return r.Ref.Name()
}

func GetSamName(r *sam.Record) string {
	return r.Name
}

func GetSamReadAlnSeq(r *sam.Record) string {
	res := GetSamReadSeq(r)
	if len(res) > 0 {
		return res[GetSamLeftSoftClip(r) : len(res)-GetSamRightSoftClip(r)]
	}
	return ""
}

func GetSamReadSeq(r *sam.Record) string {
	if r.Seq.Length > 0 {
		return string(r.Seq.Expand())
	}
	return ""
}

func GetSamLeftSoftClipSeq(r *sam.Record) string {
	seq := GetSamReadSeq(r)
	if len(seq) > 0 {
		return seq[:GetSamLeftSoftClip(r)]
	}
	return ""
}

func GetSamRightSoftClipSeq(r *sam.Record) string {
	seq := GetSamReadSeq(r)
	if len(seq) > 0 {
		return seq[(len(seq) - GetSamRightSoftClip(r)):]
	}
	return ""
}

func GetSamMapQual(r *sam.Record) int {
	return int(r.MapQ)
}

func GetSamHardClipped(r *sam.Record) int {
	var hc int
	if len(r.Cigar) == 0 {
		return 0
	}
	last := len(r.Cigar) - 1
	if r.Cigar[last].Type() == sam.CigarHardClipped {
		hc += r.Cigar[last].Len()
	}
	if r.Cigar[0].Type() == sam.CigarHardClipped {
		hc += r.Cigar[0].Len()
	}
	return hc
}

func GetSamLeftHardClip(r *sam.Record) int {
	var hc int
	if len(r.Cigar) > 0 && r.Cigar[0].Type() == sam.CigarHardClipped {
		hc += r.Cigar[0].Len()
	}
	return hc
}

func GetSamLeftClip(r *sam.Record) int {
	return GetSamLeftSoftClip(r) + GetSamLeftHardClip(r)
}

func GetSamRightClip(r *sam.Record) int {
	return GetSamRightSoftClip(r) + GetSamRightHardClip(r)
}

func GetSamRightSoftClip(r *sam.Record) int {
	var hc int
	if len(r.Cigar) > 0 {
		last := len(r.Cigar) - 1
		if r.Cigar[last].Type() == sam.CigarSoftClipped {
			hc += r.Cigar[last].Len()
		} else if len(r.Cigar) > 1 && r.Cigar[last].Type() == sam.CigarHardClipped && r.Cigar[last-1].Type() == sam.CigarSoftClipped {
			hc += r.Cigar[last-1].Len()
		}
	}
	return hc
}

func GetSamLeftSoftClip(r *sam.Record) int {
	var hc int
	if len(r.Cigar) > 0 {
		if r.Cigar[0].Type() == sam.CigarSoftClipped {
			hc += r.Cigar[0].Len()
		} else if len(r.Cigar) > 1 && r.Cigar[0].Type() == sam.CigarHardClipped && r.Cigar[1].Type() == sam.CigarSoftClipped {
			hc += r.Cigar[1].Len()
		}
	}
	return hc
}

func GetSamRightHardClip(r *sam.Record) int {
	var hc int
	if len(r.Cigar) > 0 {
		last := len(r.Cigar) - 1
		if r.Cigar[last].Type() == sam.CigarHardClipped {
			hc += r.Cigar[last].Len()
		}
	}
	return hc
}

func GetSamReadLen(r *sam.Record) int {
	if r.Seq.Length > 0 {
		sl := int(r.Seq.Length) + GetSamHardClipped(r)
		return sl
	}
	var ql int
	for _, op := range r.Cigar {
		ql += op.Len() * op.Type().Consumes().Query
	}
	return ql + GetSamHardClipped(r)
}

func GetSamRefAln(r *sam.Record) int {
	return r.Len()
}

func GetSamRefLen(r *sam.Record) int {
	return r.Ref.Len()
}

func GetSamRefCov(r *sam.Record) float64 {
	if r.Ref == nil || r.Ref.Len() == 0 {
		return 0
	}
	return float64(r.Len()) / float64(r.Ref.Len()) * 100
}

func GetSamReadCov(r *sam.Record) float64 {
	sl := float64(GetSamReadLen(r))
	if sl > 0 {
		return float64(100 * (sl - float64(GetSamLeftClip(r)+GetSamRightClip(r))) / float64(sl))
	}
	return 0
}

func GetSamReadAln(r *sam.Record) int {
	if r.Flags&sam.Unmapped != 0 {
		return 0
	}
	sl := GetSamReadLen(r)
	return (sl - GetSamLeftClip(r) - GetSamRightClip(r))
}

func GetSamStrand(r *sam.Record) int {
	if GetSamReverse(r) {
		return -1
	}
	return 1
}

func GetSamIsSup(r *sam.Record) int {
	if r.Flags&sam.Supplementary != 0 {
		return 1
	}
	return 0
}

func GetSamIsSec(r *sam.Record) int {
	if r.Flags&sam.Secondary != 0 {
		return 1
	}
	return 0
}

func GetSamMeanBaseQual(r *sam.Record) float64 {
	if len(r.Qual) != r.Seq.Length {
		return 0.0
	}
	s := &seq.Seq{Qual: r.Qual}
	return s.AvgQual(0)
}

func GetSamPos(r *sam.Record) int {
	return r.Pos
}

func GetSamEndPos(r *sam.Record) int {
	return r.Pos + r.Len()
}

func GetSamAcc(r *sam.Record) float64 {
	var mismatch int
	aux, ok := r.Tag([]byte("NM"))
	if !ok {
		checkError(fmt.Errorf("%s: required NM tag is missing", r.Name))
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
		checkError(fmt.Errorf("%s: could not parse NM tag: %s", r.Name, aux.String()))
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
	alnLen := float64(mm + ins + del)
	if alnLen > 0 {
		return (1.0 - float64(mismatch)/alnLen) * 100
	}
	return 0.0
}

// dumpFields rejects unknown names before any stage starts processing.
func dumpFields(y *syaml.Yaml) []string {
	valid := []string{"Read", "Ref", "Pos", "EndPos", "MapQual", "Acc", "Match", "Mismatch", "Ins", "Del", "AlnLen", "ReadLen", "RefLen", "RefAln", "RefCov", "ReadAln", "ReadCov", "Strand", "MeanQual", "LeftClip", "RightClip", "Flags", "IsSec", "IsSup", "ReadSeq", "ReadAlnSeq", "LeftSoftClipSeq", "RightSoftClipSeq", "RightSoftClip", "LeftHardClip", "RightHardClip", "LeftSoftClip"}
	fields, err := y.Get("Fields").Array()
	checkError(err)
	if len(fields) == 1 && fields[0] == "*" {
		return valid
	}
	keys := make([]string, 0, len(fields))
	for _, field := range fields {
		name, ok := field.(string)
		found := false
		if ok {
			for _, allowed := range valid {
				if name == allowed {
					found = true
					break
				}
			}
		}
		if !found {
			checkError(fmt.Errorf("Dump: unknown field: %v", field))
		}
		keys = append(keys, name)
	}
	return keys
}
