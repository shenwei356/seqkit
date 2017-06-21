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
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/shenwei356/util/pathutil"
	"github.com/spf13/cobra"
)

// genautocompleteCmd represents the fq2fa command
var genautocompleteCmd = &cobra.Command{
	Use:   "genautocomplete",
	Short: "generate shell autocompletion script",
	Long: `generate shell autocompletion script

Note: The current version supports Bash only.
This should work for *nix systems with Bash installed.

Howto:

1. run: seqkit genautocomplete

2. create and edit ~/.bash_completion file if you don't have it.

        nano ~/.bash_completion

    add the following:

        for bcfile in ~/.bash_completion.d/* ; do
          . $bcfile
        done

`,
	Run: func(cmd *cobra.Command, args []string) {
		autocompleteTarget := getFlagString(cmd, "file")
		autocompleteType := getFlagString(cmd, "type")

		if autocompleteType != "bash" {
			checkError(fmt.Errorf("only Bash is supported for now"))
		}

		dir := filepath.Dir(autocompleteTarget)
		ok, err := pathutil.DirExists(dir)
		checkError(err)
		if !ok {
			os.MkdirAll(dir, 0744)
		}
		checkError(cmd.Root().GenBashCompletionFile(autocompleteTarget))

		log.Infof("bash completion file for SeqKit saved to %s", autocompleteTarget)
	},
}

func init() {
	RootCmd.AddCommand(genautocompleteCmd)
	defaultCompletionFile, err := homedir.Expand("~/.bash_completion.d/seqkit.sh")
	checkError(err)
	genautocompleteCmd.Flags().StringP("file", "", defaultCompletionFile, "autocompletion file")
	genautocompleteCmd.Flags().StringP("type", "", "bash", "autocompletion type (currently only bash supported)")
}
