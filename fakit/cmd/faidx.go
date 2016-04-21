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
	"fmt"
	"runtime"
	"strings"

	"github.com/shenwei356/bio/seqio/fai"
	"github.com/spf13/cobra"
)

// faidxCmd represents the faidx command
var faidxCmd = &cobra.Command{
	Use:   "faidx",
	Short: "create FASTA index file",
	Long: `create FASTA index file

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		runtime.GOMAXPROCS(config.Threads)

		files := getFileList(args)

		for _, file := range files {
			if file == "-" {
				checkError(fmt.Errorf("stdin not supported"))
			}

			if strings.HasSuffix(strings.ToLower(file), ".gz") {
				checkError(fmt.Errorf("gzipped file not supported"))
			}

			_, err := fai.CreateWithIDRegexp(file, config.IDRegexp)
			checkError(err)
		}
	},
}

func init() {
	RootCmd.AddCommand(faidxCmd)
}
