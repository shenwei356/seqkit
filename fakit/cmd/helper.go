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
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/brentp/xopen"
	"github.com/cznic/sortutil"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fasta"
	"github.com/spf13/cobra"
)

func checkError(err error) {
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}
}

func getFileList(args []string) []string {
	files := []string{}
	if len(args) == 0 {
		files = append(files, "-")
	} else {
		for _, file := range files {
			if file == "-" {
				continue
			}
			if _, err := os.Stat(file); os.IsNotExist(err) {
				checkError(err)
			}
		}
		files = args
	}
	return files
}

func getFlagInt(cmd *cobra.Command, flag string) int {
	value, err := cmd.Flags().GetInt(flag)
	checkError(err)
	return value
}

func getFlagBool(cmd *cobra.Command, flag string) bool {
	value, err := cmd.Flags().GetBool(flag)
	checkError(err)
	return value
}

func getFlagString(cmd *cobra.Command, flag string) string {
	value, err := cmd.Flags().GetString(flag)
	checkError(err)
	return value
}

func getFlagFloat64(cmd *cobra.Command, flag string) float64 {
	value, err := cmd.Flags().GetFloat64(flag)
	checkError(err)
	return value
}

func getFlagInt64(cmd *cobra.Command, flag string) int64 {
	value, err := cmd.Flags().GetInt64(flag)
	checkError(err)
	return value
}

func getFlagStringSlice(cmd *cobra.Command, flag string) []string {
	value, err := cmd.Flags().GetStringSlice(flag)
	checkError(err)
	return value
}

func getAlphabet(cmd *cobra.Command, t string) *seq.Alphabet {
	value, err := cmd.Flags().GetString(t)
	checkError(err)

	switch strings.ToLower(value) {
	case "dna":
		return seq.DNAredundant
	case "rna":
		return seq.RNAredundant
	case "protein":
		return seq.Protein
	case "unlimit":
		return seq.Unlimit
	case "auto":
		return nil
	default:
		return nil
	}
}

func sortFastaRecordChunkMapID(chunks map[uint64]fasta.FastaRecordChunk) sortutil.Uint64Slice {
	ids := make(sortutil.Uint64Slice, len(chunks))
	i := 0
	for id := range chunks {
		ids[i] = id
		i++
	}
	sort.Sort(ids)
	return ids
}

// MD5 of a slice
func MD5(s []byte) string {
	h := md5.New()
	h.Write(s)
	return hex.EncodeToString(h.Sum(nil))
}

func filepathTrimExtension(file string) (string, string) {
	extension := filepath.Ext(file)
	name := file[0 : len(file)-len(extension)]
	return name, extension
}

var reRegion = regexp.MustCompile(`\-?\d+:\-?\d+`)

var regionExample = `
 0-based index    0 1 2 3 4 5 6 7 8 9
 1-based index    1 2 3 4 5 6 7 8 9 10
negative index    0-9-8-7-6-5-4-3-2-1
           seq    A C G T N a c g t n
           1:1    A
           2:4        G T N
         -4:-2                c g t
         -4:-1                c g t n
         -1:-1                      n
          2:-2      C G T N a c g t
          1:-1    A C G T N a c g t n
`

func writeSeqs(records []*fasta.FastaRecord, file string, lineWidth int, quiet bool, dryRun bool) error {
	if !quiet {
		log.Infof("write %d sequences to file: %s\n", len(records), file)
	}
	if dryRun {
		return nil
	}

	outfh, err := xopen.Wopen(file)
	checkError(err)
	defer outfh.Close()

	for _, record := range records {
		outfh.WriteString(fmt.Sprintf(">%s\n%s\n", record.Name, record.FormatSeq(lineWidth)))
	}

	return nil
}
