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

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/cespare/xxhash/v2"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/breader"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// replaceCmd represents the replace command
var replaceCmd = &cobra.Command{
	GroupID: "edit",

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
    {fn}    File name
    {fbn}   File base name
    {fbne}  File base name without any extension
    {kv}    Corresponding value of the key (captured variable $n) by key-value file,
            n can be specified by flag -I (--key-capt-idx) (default: 1)
            
Special cases:
  1. If replacements contain '$', 
    a). If using '{kv}', you need use '$$$$' instead of a single '$':
            -r '{kv}' -k <(sed 's/\$/$$$$/' kv.txt)
    b). If not, use '$$':
            -r 'xxx$$xx'

Filtering records to edit:
  You can use flags similar to those in "seqkit grep" to choose partly records to edit.

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
		keepUntouch := getFlagBool(cmd, "keep-untouch")
		keyCaptIdx := getFlagPositiveInt(cmd, "key-capt-idx")
		keyMissRepl := getFlagString(cmd, "key-miss-repl")

		bySeq := getFlagBool(cmd, "by-seq")
		// byName := getFlagBool(cmd, "by-name")
		ignoreCase := getFlagBool(cmd, "ignore-case")

		if pattern == "" {
			checkError(fmt.Errorf("flags -p (--pattern) needed"))
		}

		// check pattern with unquoted comma
		if reUnquotedComma.MatchString(pattern) {
			if outFile == "-" {
				defer log.Warningf(helpUnquotedComma)
			} else {
				log.Warningf(helpUnquotedComma)
			}
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

		var replaceWithFN bool
		if reFN.Match(replacement) {
			replaceWithFN = true
		}

		var replaceWithFBN bool
		if reFBN.Match(replacement) {
			replaceWithFBN = true
		}

		var replaceWithFBNE bool
		if reFBNE.Match(replacement) {
			replaceWithFBNE = true
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

		// -------------------
		// filter for records to edit

		fpattern := getFlagStringSlice(cmd, "f-pattern")
		fpatternFile := getFlagString(cmd, "f-pattern-file")
		fuseRegexp := getFlagBool(cmd, "f-use-regexp")
		finvertMatch := getFlagBool(cmd, "f-invert-match")
		fbySeq := getFlagBool(cmd, "f-by-seq")
		fbyName := getFlagBool(cmd, "f-by-name")
		fignoreCase := getFlagBool(cmd, "f-ignore-case")
		fonlyPositiveStrand := getFlagBool(cmd, "f-only-positive-strand")

		// check pattern with unquoted comma
		hasUnquotedComma := false
		for _, _pattern := range fpattern {
			if reUnquotedComma.MatchString(_pattern) {
				hasUnquotedComma = true
				break
			}
		}
		if hasUnquotedComma {
			if outFile == "-" {
				defer log.Warningf(helpUnquotedComma)
			} else {
				log.Warningf(helpUnquotedComma)
			}
		}

		// prepare pattern
		usingDefaultIDRegexp := config.IDRegexp == fastx.DefaultIDRegexp

		patternsR := make(map[uint64]*regexp.Regexp, 1<<10)
		patternsN := make(map[uint64]interface{}, 1<<20)
		// patternsS := make(map[string]interface{}, 1<<10)
		patternsS := make([][]byte, 0, 16)

		var pbyte []byte
		if fpatternFile != "" {
			var reader *breader.BufferedReader
			reader, err = breader.NewDefaultBufferedReader(fpatternFile)
			checkError(err)
			for chunk := range reader.Ch {
				checkError(chunk.Err)
				for _, data := range chunk.Data {
					p := data.(string)
					if p == "" {
						continue
					}

					if !quiet {
						if p[0] == '>' {
							log.Warningf(`symbol ">" detected, it should not be a part of the sequence ID/name: %s`, p)
						} else if p[0] == '@' {
							log.Warningf(`symbol "@" detected, it should not be a part of the sequence ID/name. %s`, p)
						} else if !fbyName && usingDefaultIDRegexp && strings.ContainsAny(p, "\t ") {
							log.Warningf("space found in pattern, you may need use --f-by-name: %s", p)
						}
					}

					if fuseRegexp {
						if fignoreCase {
							p = "(?i)" + p
						}
						r, err := regexp.Compile(p)
						checkError(err)
						patternsR[xxhash.Sum64String(p)] = r
					} else if fbySeq {
						pbyte = []byte(p)
						if seq.DNAredundant.IsValid(pbyte) == nil ||
							seq.RNAredundant.IsValid(pbyte) == nil ||
							seq.Protein.IsValid(pbyte) == nil { // legal sequence
							if fignoreCase {
								// patternsS[strings.ToLower(p)] = struct{}{}
								patternsS = append(patternsS, bytes.ToLower(pbyte))
							} else {
								// patternsS[p] = struct{}{}
								patternsS = append(patternsS, pbyte)
							}
						} else {
							checkError(fmt.Errorf("illegal DNA/RNA/Protein sequence: %s", p))
						}
					} else {
						if fignoreCase {
							patternsN[xxhash.Sum64String(strings.ToLower(p))] = struct{}{}
						} else {
							patternsN[xxhash.Sum64String(p)] = struct{}{}
						}
					}
				}
			}
			if !quiet {
				if len(patternsR)+len(patternsN)+len(patternsS) == 0 {
					log.Warningf("%d patterns loaded from file", 0)
				} else {
					log.Infof("%d patterns loaded from file", len(patternsR)+len(patternsN)+len(patternsS))
				}
			}
		} else if len(fpattern) > 0 {
			for _, p := range fpattern {
				if !quiet {
					if p[0] == '>' {
						log.Warningf(`symbol ">" detected, it should not be a part of the sequence ID/name: %s`, p)
					} else if p[0] == '@' {
						log.Warningf(`symbol "@" detected, it should not be a part of the sequence ID/name. %s`, p)
					} else if !fbyName && usingDefaultIDRegexp && strings.ContainsAny(p, "\t ") {
						log.Warningf("space found in pattern, you may need use --f-by-name: %s", p)
					}
				}

				if fuseRegexp {
					if fignoreCase {
						p = "(?i)" + p
					}
					r, err := regexp.Compile(p)
					checkError(err)
					patternsR[xxhash.Sum64String(p)] = r
				} else if fbySeq {
					pbyte = []byte(p)
					if seq.DNAredundant.IsValid(pbyte) == nil ||
						seq.RNAredundant.IsValid(pbyte) == nil ||
						seq.Protein.IsValid(pbyte) == nil { // legal sequence
						if fignoreCase {
							// patternsS[strings.ToLower(p)] = struct{}{}
							patternsS = append(patternsS, bytes.ToLower(pbyte))
						} else {
							// patternsS[p] = struct{}{}
							patternsS = append(patternsS, pbyte)
						}
					} else {
						checkError(fmt.Errorf("illegal DNA/RNA/Protein sequence: %s", p))
					}
				} else {
					if fignoreCase {
						patternsN[xxhash.Sum64String(strings.ToLower(p))] = struct{}{}
					} else {
						patternsN[xxhash.Sum64String(p)] = struct{}{}
					}
				}
			}
		}

		useFilter := len(fpattern) > 0 || fpatternFile != ""

		// -------------------

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var r []byte
		var founds [][][]byte
		var found [][]byte
		var k, v string
		var ok bool
		var doNotChange bool
		var record *fastx.Record
		nrFormat := fmt.Sprintf("%%0%dd", nrWidth)

		var count int
		var target []byte
		var hit bool
		// var k string
		var k2 []byte
		var re *regexp.Regexp
		var h uint64

		var fileBase string

		for _, file := range files {
			fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)

			nr := 0
			bFile := []byte(file)
			fileBase = filepath.Base(file)
			bFileBase := []byte(fileBase)
			var bFileBaseWithoutExtension []byte
			if i := strings.Index(fileBase, "."); i >= 0 {
				bFileBaseWithoutExtension = []byte(fileBase[:i])
			} else {
				bFileBaseWithoutExtension = []byte(fileBase)
			}

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

				// filter

				if useFilter {
					if fbyName {
						target = record.Name
					} else if fbySeq {
						target = record.Seq.Seq
					} else {
						target = record.ID
					}

					hit = false

					if fuseRegexp {
						for h, re = range patternsR {
							if re.Match(target) {
								hit = true
								break
							}
						}
					} else if fbySeq {
						if fignoreCase {
							target = bytes.ToLower(target)
						}
						for _, k2 = range patternsS {
							if bytes.Contains(target, k2) {
								hit = true
								break
							}
						}
						// search the reverse complement seq
						if !hit && !fonlyPositiveStrand {
							target = record.Seq.RevCom().Seq
							if fignoreCase {
								target = bytes.ToLower(target)
							}
							for _, k2 = range patternsS {
								if bytes.Contains(target, k2) {
									hit = true
									break
								}
							}
						}
					} else {
						h = xxhash.Sum64(target)
						if ignoreCase {
							h = xxhash.Sum64(bytes.ToLower(target))
						}
						if _, ok = patternsN[h]; ok {
							hit = true
						}
					}

					if finvertMatch {
						hit = !hit
					}

					if !hit {
						record.FormatToWriter(outfh, config.LineWidth)
						continue
					}

					count++
				}

				// edit

				if bySeq {
					if fastxReader.IsFastq {
						checkError(fmt.Errorf("editing FASTQ is not supported"))
					}
					record.Seq.Seq = patternRegexp.ReplaceAll(record.Seq.Seq, replacement)
				} else {
					doNotChange = false

					r = replacement

					if replaceWithNR {
						r = reNR.ReplaceAll(r, []byte(fmt.Sprintf(nrFormat, nr)))
					}

					if replaceWithFN {
						r = reFN.ReplaceAll(r, bFile)
					}
					if replaceWithFBN {
						r = reFBN.ReplaceAll(r, bFileBase)
					}
					if replaceWithFBNE {
						r = reFBNE.ReplaceAll(r, bFileBaseWithoutExtension)
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
							if v, ok = kvs[k]; ok {
								r = reKV.ReplaceAll(r, []byte(v))
							} else if keepUntouch {
								doNotChange = true
							} else if keepKey {
								r = reKV.ReplaceAll(r, found[keyCaptIdx])
							} else {
								r = reKV.ReplaceAll(r, []byte(keyMissRepl))
							}
						} else {
							doNotChange = true
						}
					}

					if !doNotChange {
						record.Name = patternRegexp.ReplaceAll(record.Name, r)
					}
				}

				record.FormatToWriter(outfh, config.LineWidth)
			}
			fastxReader.Close()

			config.LineWidth = lineWidth
		}

		if !config.Quiet && useFilter {
			log.Infof("%d records matched by the filter", count)
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
			`use the \ escape character. Record number and file name is also supported by "{nr}" and "{fn}".`+
			`use ${1} instead of $1 when {kv} given!`)
	replaceCmd.Flags().IntP("nr-width", "", 1, `minimum width for {nr} in flag -r/--replacement. e.g., formatting "1" to "001" by --nr-width 3`)
	// replaceCmd.Flags().BoolP("by-name", "n", false, "replace full name instead of just id")
	replaceCmd.Flags().BoolP("by-seq", "s", false, "replace seq (only FASTA)")
	replaceCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
	replaceCmd.Flags().StringP("kv-file", "k", "",
		`tab-delimited key-value file for replacing key with value when using "{kv}" in -r (--replacement) (only for sequence name)`)
	replaceCmd.Flags().BoolP("keep-untouch", "U", false, "do not change anything when no value found for the key (only for sequence name)")
	replaceCmd.Flags().BoolP("keep-key", "K", false, "keep the key as value when no value found for the key (only for sequence name)")
	replaceCmd.Flags().IntP("key-capt-idx", "I", 1, "capture variable index of key (1-based)")
	replaceCmd.Flags().StringP("key-miss-repl", "m", "", "replacement for key with no corresponding value")

	replaceCmd.Flags().StringSliceP("f-pattern", "", []string{""}, `[target filter] search pattern (multiple values supported. Attention: use double quotation marks for patterns containing comma, e.g., -p '"A{2,}"')`)
	replaceCmd.Flags().StringP("f-pattern-file", "", "", "[target filter] pattern file (one record per line)")
	replaceCmd.Flags().BoolP("f-use-regexp", "", false, "[target filter] patterns are regular expression")
	replaceCmd.Flags().BoolP("f-invert-match", "", false, "[target filter] invert the sense of matching, to select non-matching records")
	replaceCmd.Flags().BoolP("f-by-name", "", false, "[target filter] match by full name instead of just ID")
	replaceCmd.Flags().BoolP("f-by-seq", "", false, "[target filter] search subseq on seq, both positive and negative strand are searched, and mismatch allowed using flag -m/--max-mismatch")
	replaceCmd.Flags().BoolP("f-ignore-case", "", false, "[target filter] ignore case")
	replaceCmd.Flags().BoolP("f-only-positive-strand", "", false, "[target filter] only search on positive strand")
}

var reNR = regexp.MustCompile(`\{(NR|nr)\}`)
var reKV = regexp.MustCompile(`\{(KV|kv)\}`)
var reFN = regexp.MustCompile(`\{(FN|fn)\}`)
var reFBN = regexp.MustCompile(`\{(FBN|fbn)\}`)
var reFBNE = regexp.MustCompile(`\{(FBNE|fbne)\}`)
