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
	//"github.com/biogo/hts/bam"
	"fmt"
	"github.com/biogo/hts/sam"
	syaml "github.com/smallfish/simpleyaml"
	"os"
)

type BamTool struct {
	Name string
	Desc string
	Use  func(params *BamToolParams)
}

type BamToolParams struct {
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
	res := "Tool\t\tDescription\n"
	res += "----\t\t-----------\n"
	for _, tool := range s {
		res += fmt.Sprintf("%s\t\t%s\n", tool.Name, tool.Desc)
	}
	return res
}

func NewToolshed() Toolshed {
	ts := map[string]BamTool{
		"AlnContext": BamTool{Name: "AlnContext", Desc: "lilter records by the sequence context at start and end", Use: BamToolAlnContext},
		"help":       BamTool{Name: "help", Desc: "list all tools with description", Use: ListTools},
	}
	return ts
}

func BamToolbox(toolYaml string, inFile string, outFile string, quiet bool, silent bool, threads int) {
	if toolYaml == "help" {
		toolYaml = "help: true"
	}
	y, err := syaml.NewYaml([]byte(toolYaml))
	checkError(err)
	ty, err := y.GetMapKeys()
	checkError(err)
	switch len(ty) {
	case 0:
		log.Fatal("toolbox: not tool specified!")
	default:
		tkeys, err := y.GetMapKeys()
		checkError(err)
		shed := NewToolshed()
		for rank, tool := range tkeys {
			var wt BamTool
			var ok bool
			if wt, ok = shed[tool]; !ok {

			}
			params := &BamToolParams{
				Shed: shed,
				Rank: rank,
			}
			wt.Use(params)
		}
	}

}

func ListTools(p *BamToolParams) {
	os.Stderr.WriteString(p.Shed.String())
	os.Exit(0)
}

func BamToolAlnContext(p *BamToolParams) {

}
