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
	"os"
	"runtime"

	"github.com/shenwei356/bio/seqio/fasta"
	"github.com/spf13/cobra"
)

var cfgFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "fakit",
	Short: "FASTA kit",
	Long: `fakit -- FASTA kit

Version: 0.1.1

Author: Wei Shen <shenwei356@gmail.com>

Documents  : http://shenwei356.github.io/fakit
Source code: https://github.com/shenwei356/fakit

`,
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringP("seq-type", "t", "auto", "sequence type (dna|rna|protein|unlimit|auto) (for auto, it automatically detect by the first sequence)")
	RootCmd.PersistentFlags().IntP("chunk-size", "c", 1000, "chunk size (attention: unit is FASTA records not lines)")
	RootCmd.PersistentFlags().IntP("threads", "j", runtime.NumCPU(), "number of CPUs (default value depends on your device)")
	RootCmd.PersistentFlags().IntP("line-width", "w", 60, "line width (0 for no wrap)")
	RootCmd.PersistentFlags().StringP("id-regexp", "", fasta.DefaultIDRegexp, "regular expression for parsing ID")
	RootCmd.PersistentFlags().StringP("out-file", "o", "-", `out file ("-" for stdout, suffix .gz for gzipped out)`)
	RootCmd.PersistentFlags().BoolP("quiet", "", false, "be quiet and do not show extra information")
}
