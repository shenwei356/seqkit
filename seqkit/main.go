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

package main

import (
	"io"
	"os"
	"runtime"

	// "runtime/pprof"

	colorable "github.com/mattn/go-colorable"
	"github.com/shenwei356/go-logging"
	"github.com/shenwei356/seqkit/seqkit/cmd"
)

var logFormat = logging.MustStringFormatter(
	// `%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	`%{color}[%{level:.4s}]%{color:reset} %{message}`,
)

func init() {
	var stderr io.Writer = os.Stderr
	if runtime.GOOS == "windows" {
		stderr = colorable.NewColorableStderr()
	}
	backend := logging.NewLogBackend(stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, logFormat)
	logging.SetBackend(backendFormatter)
}

func main() {
	// go tool pprof /usr/local/bin/seqkit pprof
	// f, _ := os.Create("pprof")
	// pprof.StartCPUProfile(f)
	// defer pprof.StopCPUProfile()

	cmd.Execute()

	// go tool pprof --alloc_space /usr/local/bin/seqkit mprof
	// go tool pprof --inuse_space /usr/local/bin/seqkit mprof
	// f2, _ := os.Create("mprof")
	// pprof.WriteHeapProfile(f2)
	// defer f2.Close()
}
