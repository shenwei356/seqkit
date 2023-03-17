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
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fai"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/breader"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// faidxCmd represents the faidx command
var faidxCmd = &cobra.Command{
	Use:   "faidx",
	Short: "create FASTA index file and extract subsequence",
	Long: fmt.Sprintf(`create FASTA index file and extract subsequence

This command is similar with "samtools faidx" but has some extra features:

  1. output full header line with the flag -f
  2. support regular expression as sequence ID with the flag -r
  3. if you have large number of IDs, you can use:
        seqkit faidx seqs.fasta -l IDs.txt

Attentions:
  1. The flag -U/--update-faidx is recommended to ensure the .fai file matches the FASTA file.

The definition of region is 1-based and with some custom design.

Examples:
%s
`, regionExample),
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		runtime.GOMAXPROCS(config.Threads)
		fai.MapWholeFile = false
		quiet := config.Quiet

		fullHead := getFlagBool(cmd, "full-head")
		ignoreCase := getFlagBool(cmd, "ignore-case")
		useRegexp := getFlagBool(cmd, "use-regexp")
		regionFile := getFlagString(cmd, "region-file")

		immediateOutput := getFlagBool(cmd, "immediate-output")

		updateFaidx := getFlagBool(cmd, "update-faidx")

		files := getFileListFromArgsAndFile(cmd, args, false, "infile-list", false)

		file := files[0]

		if file == "-" {
			checkError(fmt.Errorf("stdin not supported"))
		}

		if strings.HasSuffix(strings.ToLower(file), ".gz") {
			checkError(fmt.Errorf("gzipped file not supported"))
		}
		if strings.HasSuffix(strings.ToLower(file), ".xz") {
			checkError(fmt.Errorf("xz compressed file not supported"))
		}
		if strings.HasSuffix(strings.ToLower(file), ".zst") {
			checkError(fmt.Errorf("zstd compressed file not supported"))
		}

		var err error
		regions := make([]string, 0, 256)
		if regionFile != "" {
			var reader *breader.BufferedReader
			reader, err = breader.NewDefaultBufferedReader(regionFile)
			checkError(err)
			var data interface{}
			var r string
			for chunk := range reader.Ch {
				checkError(chunk.Err)
				for _, data = range chunk.Data {
					r = data.(string)
					if r == "" {
						continue
					}
					regions = append(regions, r)
				}
			}
			if !quiet {
				if len(regions) == 0 {
					log.Warningf("%d patterns loaded from file", len(regions))
				} else {
					log.Infof("%d patterns loaded from file", len(regions))
				}
			}
		}

		outfh, err := xopen.Wopen(config.OutFile)
		checkError(err)
		defer outfh.Close()

		// create and read .fai
		var idx fai.Index
		var fileFai string
		var idRegexp string
		if fullHead {
			fileFai = file + ".seqkit.fai"
			idRegexp = "^(.+)$"
		} else {
			fileFai = file + ".fai"
			idRegexp = config.IDRegexp
		}

		if !quiet {
			log.Infof("create or read FASTA index ...")
		}

		if FileExists(fileFai) && updateFaidx {
			checkError(os.RemoveAll(fileFai))
			if !quiet {
				log.Infof("delete the old FASTA index file: %s", fileFai)
			}
		}

		if fileNotExists(fileFai) {
			if !quiet {
				log.Infof("create FASTA index for %s", file)
			}
			idx, err = fai.CreateWithIDRegexp(file, fileFai, idRegexp)
			checkError(err)
		} else {
			if !quiet {
				log.Infof("read FASTA index from %s", fileFai)
			}
			idx, err = fai.Read(fileFai)
			checkError(err)

			if len(idx) == 0 {
				log.Warningf("0 records loaded from %s, please check if it matches the FASTA file, or switch on the flag -U/--update-faidx", fileFai)
				return
			} else if !quiet {
				log.Infof("  %d records loaded from %s", len(idx), fileFai)
			}
		}

		if len(files) == 1 { // just creat .fai file
			if len(regions) == 0 {
				return
			}
		}

		var faidx *fai.Faidx
		faidx, err = fai.NewWithIndex(file, idx)
		checkError(err)
		defer faidx.Close()

		// save id and header in a map(id:head)
		id2head := make(map[string]string)
		var idRe *regexp.Regexp
		if fullHead {
			idRe, _ = regexp.Compile(config.IDRegexp)
		}
		var id string
		for head := range idx {
			if fullHead {
				id = string(fastx.ParseHeadID(idRe, []byte(head)))
			} else {
				id = head
			}
			if ignoreCase {
				id = strings.ToLower(id)
			}
			id2head[id] = head
		}

		// handle queries
		queries := make([]string, len(regions), len(regions)+len(files)-1)
		if len(regions) > 0 {
			copy(queries, regions)
		}
		queries = append(queries, files[1:]...)

		faidxQueries := make([]faidxQuery, 0, len(queries))
		var region [2]int

		var ok bool
		if !useRegexp {
			var begin, end int
			for _, query := range queries {
				id, begin, end = parseRegion(query)

				if ignoreCase {
					id = strings.ToLower(id)
				}
				if _, ok = id2head[id]; !ok {
					log.Warningf("sequence not found: %s", id)
					continue
				}

				faidxQueries = append(faidxQueries, faidxQuery{ID: id, Region: [2]int{begin, end}})
			}
		} else {
			queriesRe := make([]*regexp.Regexp, len(queries))
			for i, query := range queries {
				queriesRe[i], err = regexp.Compile(query)
				if err != nil {
					checkError(fmt.Errorf("invalid regular expression: %s", query))
				}
			}
			var re *regexp.Regexp
			// no need for optimization with goroutine
			for id = range id2head {
				for _, re = range queriesRe {
					if re.MatchString(id) {
						faidxQueries = append(faidxQueries, faidxQuery{ID: id, Region: [2]int{1, -1}})
						break
					}
				}
			}
		}

		var head string
		var subseq []byte
		var text []byte
		var buffer *bytes.Buffer
		var _s *seq.Seq
		var alphabet *seq.Alphabet
		for _, faidxQ := range faidxQueries {
			head = id2head[faidxQ.ID]
			region = faidxQ.Region

			if (region[0] == 1 && region[1] == -1) || (region[0] > 0 && region[1] < 0) { // full record or region like [5, -5].
				subseq, _ = faidx.SubSeq(head, region[0], region[1])

				outfh.WriteString(fmt.Sprintf(">%s\n", head))
			} else if region[0] <= region[1] {
				subseq, _ = faidx.SubSeq(head, region[0], region[1])

				outfh.WriteString(fmt.Sprintf(">%s:%d-%d\n", head, region[0], region[1]))
			} else { // reverse complement sequence
				subseq, _ = faidx.SubSeq(head, region[1], region[0])
				alphabet = config.Alphabet
				if alphabet == nil {
					alphabet = seq.DNAredundant
					if bytes.ContainsAny(subseq, "uU") {
						alphabet = seq.RNAredundant
					}
				}
				_s, err = seq.NewSeqWithoutValidation(alphabet, subseq)
				if err != nil {
					checkError(fmt.Errorf("fail to compute reverse complemente sequence for region: %s:%d-%d", head, region[0], region[1]))
				}
				subseq = _s.RevComInplace().Seq

				outfh.WriteString(fmt.Sprintf(">%s:%d-%d\n", head, region[0], region[1]))
			}

			text, buffer = wrapByteSlice(subseq, config.LineWidth, buffer)
			outfh.Write(text)

			if immediateOutput {
				outfh.Flush()
			}

			outfh.WriteString("\n")
		}
	},
}

type faidxQuery struct {
	ID     string
	Region [2]int
}

func init() {
	RootCmd.AddCommand(faidxCmd)

	faidxCmd.Flags().BoolP("use-regexp", "r", false, "IDs are regular expression. But subseq region is not supported here.")
	faidxCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
	faidxCmd.Flags().BoolP("full-head", "f", false, "print full header line instead of just ID. New fasta index file ending with .seqkit.fai will be created")
	faidxCmd.Flags().StringP("region-file", "l", "", "file containing a list of regions")

	faidxCmd.Flags().BoolP("immediate-output", "I", false, "print output immediately, do not use write buffer")
	faidxCmd.Flags().BoolP("update-faidx", "U", false, "update the fasta index file if it exists. Use this if you are not sure whether the fasta file changed")

	faidxCmd.SetUsageTemplate(`Usage:{{if .Runnable}}
  {{if .HasAvailableFlags}}{{appendIfNotPresent .UseLine "[flags]"}}{{else}}{{.UseLine}}{{end}}{{end}}{{if .HasAvailableSubCommands}}
  {{ .CommandPath}} [command]{{end}} <fasta-file> [regions...]{{if gt .Aliases 0}}

Aliases:
  {{.NameAndAliases}}
{{end}}{{if .HasExample}}

Examples:
{{ .Example }}{{end}}{{ if .HasAvailableSubCommands}}
Available Commands:{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsagesWrapped 110 | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsagesWrapped 110 | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsHelpCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableSubCommands }}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)

}

var reRegionFull = regexp.MustCompile(`^(.+?):(\-?\d+)\-(\-?\d+)$`)
var reRegionOneBase = regexp.MustCompile(`^(.+?):(\d+)$`)
var reRegionOnlyBegin = regexp.MustCompile(`^(.+?):(\-?\d+)\-$`)
var reRegionOnlyEnd = regexp.MustCompile(`^(.+?):\-(\-?\d+)$`)

func parseRegion(region string) (id string, begin int, end int) {
	var found []string
	if reRegionFull.MatchString(region) {
		found = reRegionFull.FindStringSubmatch(region)
		id = found[1]
		begin, _ = strconv.Atoi(found[2])
		end, _ = strconv.Atoi(found[3])
	} else if reRegionOneBase.MatchString(region) {
		found = reRegionOneBase.FindStringSubmatch(region)
		id = found[1]
		begin, _ = strconv.Atoi(found[2])
		end = begin
	} else if reRegionOnlyBegin.MatchString(region) {
		found = reRegionOnlyBegin.FindStringSubmatch(region)
		id = found[1]
		begin, _ = strconv.Atoi(found[2])
		end = -1
	} else if reRegionOnlyEnd.MatchString(region) {
		found = reRegionOnlyEnd.FindStringSubmatch(region)
		id = found[1]
		begin = 1
		end, _ = strconv.Atoi(found[2])
	} else {
		id = region
		begin, end = 1, -1
	}
	return
}
