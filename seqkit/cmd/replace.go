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
	"io"
	"regexp"
	"runtime"
	"strings"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// replaceCmd represents the replace command
var replaceCmd = &cobra.Command{
	Use:   "replace",
	Short: "replace name/sequence by regular expression",
	Long: `replace name/sequence by regular expression.

Note that the replacement supports capture variables.
e.g. $1 represents the text of the first submatch.
ATTENTION: use SINGLE quote NOT double quotes in *nix OS.

Examples: Adding space to all bases.

    seqkit replace -p "(.)" -r '$1 ' -s

Or use the \ escape character.

    seqkit replace -p "(.)" -r "\$1 " -s

more on: http://bioinf.shenwei.me/seqkit/usage/#replace

Special replacement symbols (only for replacing name not sequence):

    {nr}    Record number, starting from 1
    {kv}    Corresponding value of the key (captured variable $n) by key-value file,
            n can be specified by flag -I (--key-capt-idx) (default: 1)

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		lineWidth := config.LineWidth
		outFile := config.OutFile
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		quiet := config.Quiet
		runtime.GOMAXPROCS(config.Threads)

		pattern := getFlagString(cmd, "pattern")
		replacement := []byte(getFlagString(cmd, "replacement"))
		nrWidth := getFlagPositiveInt(cmd, "nr-width")
		kvFile := getFlagString(cmd, "kv-file")
		keepKey := getFlagBool(cmd, "keep-key")
		keyCaptIdx := getFlagPositiveInt(cmd, "key-capt-idx")
		keyMissRepl := getFlagString(cmd, "key-miss-repl")

		bySeq := getFlagBool(cmd, "by-seq")
		// byName := getFlagBool(cmd, "by-name")
		ignoreCase := getFlagBool(cmd, "ignore-case")

		if pattern == "" {
			checkError(fmt.Errorf("flags -p (--pattern) needed"))
		}
		p := pattern
		if ignoreCase {
			p = "(?i)" + p
		}
		patternRegexp, err := regexp.Compile(p)
		checkError(err)

		if kvFile != "" {
			if len(replacement) == 0 {
				checkError(fmt.Errorf("flag -r (--replacement) needed when given flag -k (--kv-file)"))
			}
			if !reKV.Match(replacement) {
				checkError(fmt.Errorf(`replacement symbol "{kv}"/"{KV}" not found in value of flag -r (--replacement) when flag -k (--kv-file) given`))
			}
		}
		var replaceWithNR bool
		if reNR.Match(replacement) {
			replaceWithNR = true
		}

		var replaceWithKV bool
		var kvs map[string]string
		if reKV.Match(replacement) {
			replaceWithKV = true
			if !regexp.MustCompile(`\(.+\)`).MatchString(pattern) {
				checkError(fmt.Errorf(`value of -p (--pattern) must contains "(" and ")" to capture data which is used specify the KEY`))
			}
			if bySeq {
				checkError(fmt.Errorf(`replaceing with key-value pairs was not supported for sequence`))
			}
			if kvFile == "" {
				checkError(fmt.Errorf(`since replacement symbol "{kv}"/"{KV}" found in value of flag -r (--replacement), tab-delimited key-value file should be given by flag -k (--kv-file)`))
			}
			if !quiet {
				log.Infof("read key-value file: %s", kvFile)
			}
			kvs, err = readKVs(kvFile, ignoreCase)
			if err != nil {
				checkError(fmt.Errorf("read key-value file: %s", err))
			}
			if len(kvs) == 0 {
				checkError(fmt.Errorf("no valid data in key-value file: %s", kvFile))
			}
			if !quiet {
				log.Infof("%d pairs of key-value loaded", len(kvs))
			}
		}

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var r []byte
		var founds [][][]byte
		var found [][]byte
		var k string
		var ok bool
		var record *fastx.Record
		var fastxReader *fastx.Reader
		nrFormat := fmt.Sprintf("%%0%dd", nrWidth)
		for _, file := range files {
			fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)
			nr := 0
			for {
				record, err = fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}
				if fastxReader.IsFastq {
					config.LineWidth = 0
					fastx.ForcelyOutputFastq = true
				}

				nr++
				if bySeq {
					record.Seq.Seq = patternRegexp.ReplaceAll(record.Seq.Seq, replacement)
				} else {
					r = replacement

					if replaceWithNR {
						r = reNR.ReplaceAll(r, []byte(fmt.Sprintf(nrFormat, nr)))
					}

					if replaceWithKV {
						founds = patternRegexp.FindAllSubmatch(record.Name, -1)
						if len(founds) > 1 {
							checkError(fmt.Errorf(`pattern "%s" matches multiple targets in "%s", this will cause chaos`, p, record.Name))
						}
						if len(founds) > 0 {
							found = founds[0]
							if keyCaptIdx > len(found)-1 {
								checkError(fmt.Errorf("value of flag -I (--key-capt-idx) overflows"))
							}
							k = string(found[keyCaptIdx])
							if ignoreCase {
								k = strings.ToLower(k)
							}
							if _, ok = kvs[k]; ok {
								r = reKV.ReplaceAll(r, []byte(kvs[k]))
							} else if keepKey {
								r = reKV.ReplaceAll(r, found[keyCaptIdx])
							} else {
								r = reKV.ReplaceAll(r, []byte(keyMissRepl))
							}
						}
					}

					record.Name = patternRegexp.ReplaceAll(record.Name, r)
				}

				record.FormatToWriter(outfh, config.LineWidth)
			}

			config.LineWidth = lineWidth
		}
	},
}

func init() {
	RootCmd.AddCommand(replaceCmd)
	replaceCmd.Flags().StringP("pattern", "p", "", "search regular expression")
	replaceCmd.Flags().StringP("replacement", "r", "",
		"replacement. supporting capture variables. "+
			" e.g. $1 represents the text of the first submatch. "+
			"ATTENTION: for *nix OS, use SINGLE quote NOT double quotes or "+
			`use the \ escape character. Record number is also supported by "{nr}".`+
			`use ${1} instead of $1 when {kv} given!`)
	replaceCmd.Flags().IntP("nr-width", "", 1, `minimum width for {nr} in flag -r/--replacement. e.g., formating "1" to "001" by --nr-width 3`)
	// replaceCmd.Flags().BoolP("by-name", "n", false, "replace full name instead of just id")
	replaceCmd.Flags().BoolP("by-seq", "s", false, "replace seq")
	replaceCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
	replaceCmd.Flags().StringP("kv-file", "k", "",
		`tab-delimited key-value file for replacing key with value when using "{kv}" in -r (--replacement) (only for sequence name)`)
	replaceCmd.Flags().BoolP("keep-key", "K", false, "keep the key as value when no value found for the key (only for sequence name)")
	replaceCmd.Flags().IntP("key-capt-idx", "I", 1, "capture variable index of key (1-based)")
	replaceCmd.Flags().StringP("key-miss-repl", "m", "", "replacement for key with no corresponding value")
}

var reNR = regexp.MustCompile(`\{(NR|nr)\}`)
var reKV = regexp.MustCompile(`\{(KV|kv)\}`)
