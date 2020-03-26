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
	"io/ioutil"
	"os"
	"regexp"
	"sort"

	"github.com/biogo/hts/bam"
	"github.com/biogo/hts/sam"
	"github.com/shenwei356/bio/seqio/fai"
	"github.com/shenwei356/bio/seqio/fastx"
	syaml "github.com/smallfish/simpleyaml"
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
	for t, _ := range s {
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
		"help":       BamTool{Name: "help", Desc: "list all tools with description", Use: ListTools},
	}
	return ts
}

func NewBamReaderChan(inFile string, cp int, buff int, threads int) (chan *sam.Record, *bam.Reader) {
	outChan := make(chan *sam.Record, cp)
	fh, err := os.Stdin, error(nil)
	if inFile != "-" {
		fh, err = os.Open(inFile)
		checkError(err)
	}

	r, err := bam.NewReader(bufio.NewReaderSize(fh, buff), threads)
	go func() {
		for {
			rec, err := r.Read()
			if err == io.EOF {
				close(outChan)
				return
			}
			if err != nil {
				close(outChan)
			}
			checkError(err)
			outChan <- rec
		}
	}()
	return outChan, r
}

func NewBamSinkChan(cp int) (chan *sam.Record, chan bool) {
	outChan := make(chan *sam.Record, cp)
	doneChan := make(chan bool, 0)
	go func() {
		for rec := range outChan {
			_ = rec
		}
		doneChan <- true
	}()

	return outChan, doneChan
}

func NewBamWriterChan(inFile string, head *sam.Header, cp int, buff int, threads int) (chan *sam.Record, chan bool) {
	outChan := make(chan *sam.Record, buff)
	doneChan := make(chan bool, 0)
	fh, err := os.Stdout, error(nil)
	if inFile != "-" {
		fh, err = os.Open(inFile)
		checkError(err)
	}

	bio := bufio.NewWriterSize(fh, buff)
	w, err := bam.NewWriter(bio, head, threads)
	go func() {
		for rec := range outChan {
			err := w.Write(rec)
			checkError(err)
		}
		w.Close()
		bio.Flush()
		fh.Close()
		doneChan <- true
	}()
	return outChan, doneChan
}

func BamToolbox(toolYaml string, inFile string, outFile string, quiet bool, silent bool, threads int) {
	if toolYaml == "help" {
		toolYaml = "help: true"
	}
	y, err := syaml.NewYaml([]byte(toolYaml))
	checkError(err)
	ty, err := y.GetMapKeys()
	checkError(err)
	if ty[0] == "Yaml" {
		conf, err := y.Get("Yaml").String()
		checkError(err)
		cb, err := ioutil.ReadFile(conf)
		checkError(err)
		y, err = syaml.NewYaml(cb)
		checkError(err)
	}

	chanCap := 5000
	ioBuff := 1024 * 128

	paramFields := map[string]bool{
		"Sink": true,
	}

	switch len(ty) {
	case 0:
		log.Fatal("toolbox: not tool specified!")
	default:
		tkeys, err := y.GetMapKeys()
		checkError(err)
		shed := NewToolshed()
		var inChan, outChan, lastOut chan *sam.Record
		var bamReader *bam.Reader
		var doneChan chan bool
		var sink bool
		log.Info(sink)
		if tkeys[0] != "help" {
			inChan, bamReader = NewBamReaderChan(inFile, chanCap, ioBuff, threads)
			sink, err = y.Get("Sink").Bool()
			if err == nil && sink {
				lastOut, doneChan = NewBamSinkChan(chanCap)
			} else {
				lastOut, doneChan = NewBamWriterChan(outFile, bamReader.Header(), chanCap, ioBuff, threads)
			}
		}
		outChan = make(chan *sam.Record, chanCap)
		nextIn, nextOut := inChan, outChan
		paramFieldCount := 0 // FIXME
		for rank, tool := range tkeys {
			var wt BamTool
			var ok bool
			if paramFields[tool] {
				paramFieldCount++
				continue
			}
			if wt, ok = shed[tool]; !ok {
				log.Fatal("Unknown tool:", tool)
			}
			if rank == (len(tkeys) - 1 - paramFieldCount) {
				nextOut = lastOut
			}
			params := &BamToolParams{
				Yaml:    y.Get(tool),
				InChan:  nextIn,
				OutChan: nextOut,
				Quiet:   quiet,
				Silent:  silent,
				Threads: threads,
				Rank:    rank,
				Shed:    shed,
			}
			nextIn = nextOut
			nextOut = make(chan *sam.Record, chanCap)
			go wt.Use(params)

		}
		log.Info("Wai")
		<-doneChan
	}

}

func ListTools(p *BamToolParams) {
	os.Stderr.WriteString(p.Shed.String())
	os.Exit(0)
}

func BamToolAlnContext(p *BamToolParams) {
	ref, err := p.Yaml.Get("Ref").String()
	checkError(err)
	leftShift, err := p.Yaml.Get("LeftShift").Int()
	checkError(err)
	rightShift, err := p.Yaml.Get("RightShift").Int()
	checkError(err)
	var regStart *regexp.Regexp
	var regEnd *regexp.Regexp
	checkError(err)
	regStrStart, err := p.Yaml.Get("RegexStart").String()
	if err == nil {
		regStart = regexp.MustCompile(regStrStart)
	}
	regStrEnd, err := p.Yaml.Get("RegexEnd").String()
	if err == nil {
		regEnd = regexp.MustCompile(regStrEnd)
	}
	idx := NewRefWitdFaidx(ref, false, p.Silent)

	for r := range p.InChan {
		chrom := r.Ref.Name()
		startPos, endPos := r.Pos, r.End()

		if regStart != nil {
			startSeq, err := idx.IdxSubSeq(chrom, startPos+leftShift, startPos+rightShift)
			checkError(err)
			if regStart.MatchString(startSeq) {
				//log.Info(startSeq)
				continue
			}
		}

		if regEnd != nil {
			endSeq, err := idx.IdxSubSeq(chrom, endPos+leftShift, endPos+rightShift)
			checkError(err)
			if regEnd.MatchString(endSeq) {
				//log.Info(endSeq)
				continue
			}
		}

		p.OutChan <- r

	}
	close(p.OutChan)
}

type RefWithFaidx struct {
	Fasta   string
	IdxFile string
	idx     fai.Index
	faidx   *fai.Faidx
	Cache   bool
}

func (idx *RefWithFaidx) IdxSubSeq(chrom string, start, end int) (string, error) {
	b, err := idx.faidx.SubSeq(chrom, start, end)
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
