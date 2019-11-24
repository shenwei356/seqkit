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
	//"bytes"
	"fmt"
	//"io"
	//"math"
	//"os"
	"runtime"
	//"github.com/shenwei356/bio/seqio/fastx"
	//"github.com/shenwei356/util/byteutil"
	"github.com/fsnotify/fsnotify"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// scatCmd represents the fish command
var scatCmd = &cobra.Command{
	Use:   "scat",
	Short: "look for short sequences in larger sequences using local alignment",
	Long:  "look for short sequences in larger sequences using local alignment",

	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		outFile := config.OutFile
		runtime.GOMAXPROCS(config.Threads)
		qBase := getFlagPositiveInt(cmd, "qual-ascii-base")

		dirs := getFileList(args, true)
		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()
		_ = outFile
		_ = qBase
		fmt.Println(dirs, outFile)
		LauchWatchers(dirs, "")

	},
}

func LauchWatchers(dirs []string, rexp string) {
	for _, dir := range dirs {
		NewFxWatcher(dir, rexp)
	}
}

func NewFxWatcher(dir, rexp string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				fmt.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					fmt.Println("modified file:", event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(dir)
	if err != nil {
		fmt.Println(err)
	}
	<-done
}

func init() {
	RootCmd.AddCommand(scatCmd)

	scatCmd.Flags().StringP("regexp", "r", ".*\\.(fastq|fq)", "output format: auto, fq, fas")
	scatCmd.Flags().StringP("out-format", "f", "auto", "output format: auto, fq, fas")
	scatCmd.Flags().Float64P("time-limit", "T", -1, "quit after inactive for this many hours")
	scatCmd.Flags().Float64P("wait-pid", "p", -1, "after process with this PID exited")
}
