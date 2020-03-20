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
	"sort"

	"github.com/biogo/hts/bam"
	"github.com/biogo/hts/sam"
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
	outChan := make(chan *sam.Record, buff)
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
			checkError(err)
			outChan <- rec
		}
	}()
	return outChan, r
}

func NewBamWriterChan(inFile string, head *sam.Header, cp int, buff int, threads int) (chan *sam.Record, chan bool) {
	outChan := make(chan *sam.Record, buff)
	doneChan := make(chan bool, 0)
	fh, err := os.Stderr, error(nil)
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
	chanCap := 5000
	ioBuff := 1024 * 128
	switch len(ty) {
	case 0:
		log.Fatal("toolbox: not tool specified!")
	default:
		tkeys, err := y.GetMapKeys()
		checkError(err)
		shed := NewToolshed()
		var inChan, outChan chan *sam.Record
		var bamReader *bam.Reader
		var doneChan chan bool
		if tkeys[0] != "help" {
			inChan, bamReader = NewBamReaderChan(inFile, chanCap, ioBuff, threads)
			outChan, doneChan = NewBamWriterChan(inFile, bamReader.Header(), chanCap, ioBuff, threads)
		}
		nextIn, lastOut := inChan, outChan
		for rank, tool := range tkeys {
			var wt BamTool
			var ok bool
			if wt, ok = shed[tool]; !ok {
				log.Info("Unknown tool:", tool)
			}
			nextOut := make(chan *sam.Record, chanCap)
			if rank == len(tkeys)-1 {
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
			go wt.Use(params)

		}
		<-doneChan
	}

}

func ListTools(p *BamToolParams) {
	os.Stderr.WriteString(p.Shed.String())
	os.Exit(0)
}

func BamToolAlnContext(p *BamToolParams) {
	for r := range p.InChan {
		log.Info(r.Name)
	}
	close(p.OutChan)
}
