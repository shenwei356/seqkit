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
	"os"
	"runtime"
	"sort"
	"strings"

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
	default:
		return seq.Unlimit
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

func getSeqsAsMap(alphabet *seq.Alphabet, file string) map[string]*seq.Seq {
	sequences := make(map[string]*seq.Seq)
	fastaReader, err := fasta.NewFastaReader(alphabet, file, 1000, runtime.NumCPU(), "")
	checkError(err)
	for chunk := range fastaReader.Ch {
		checkError(chunk.Err)
		for _, record := range chunk.Data {
			sequences[string(record.Name)] = record.Seq
		}
	}
	return sequences
}
