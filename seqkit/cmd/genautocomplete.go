// Copyright © 2016-2021 Wei Shen <shenwei356@gmail.com>
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
	Short: "generate shell autocompletion script (bash|zsh|fish|powershell)",
	Long: `generate shell autocompletion script

Supported shell: bash|zsh|fish|powershell

Bash:

    # generate completion shell
    seqkit genautocomplete --shell bash

    # configure if never did.
    # install bash-completion if the "complete" command is not found.
    echo "for bcfile in ~/.bash_completion.d/* ; do source \$bcfile; done" >> ~/.bash_completion
    echo "source ~/.bash_completion" >> ~/.bashrc

Zsh:

    # generate completion shell
    seqkit genautocomplete --shell zsh --file ~/.zfunc/_seqkit

    # configure if never did
    echo 'fpath=( ~/.zfunc "${fpath[@]}" )' >> ~/.zshrc
    echo "autoload -U compinit; compinit" >> ~/.zshrc

fish:

    seqkit genautocomplete --shell fish --file ~/.config/fish/completions/seqkit.fish

`,
	Run: func(cmd *cobra.Command, args []string) {
		outfile := getFlagString(cmd, "file")
		shell := getFlagString(cmd, "shell")

		dir := filepath.Dir(outfile)
		ok, err := pathutil.DirExists(dir)
		checkError(err)
		if !ok {
			os.MkdirAll(dir, 0744)
		}

		switch shell {
		case "bash":
			checkError(cmd.Root().GenBashCompletionFile(outfile))
		case "zsh":
			checkError(cmd.Root().GenZshCompletionFile(outfile))
		case "fish":
			checkError(cmd.Root().GenFishCompletionFile(outfile, true))
		case "powershell":
			checkError(cmd.Root().GenPowerShellCompletionFile(outfile))
		default:
			checkError(fmt.Errorf("unsupported shell: %s", shell))
		}

		log.Infof("%s completion file for seqkit saved to %s", shell, outfile)
	},
}

func init() {
	RootCmd.AddCommand(genautocompleteCmd)
	defaultCompletionFile, err := homedir.Expand("~/.bash_completion.d/seqkit.sh")
	checkError(err)
	genautocompleteCmd.Flags().StringP("file", "", defaultCompletionFile, "autocompletion file")
	genautocompleteCmd.Flags().StringP("shell", "", "bash", "autocompletion type (bash|zsh|fish|powershell)")
}
