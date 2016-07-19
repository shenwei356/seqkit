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
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/brentp/xopen"
	"github.com/cznic/sortutil"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fai"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/breader"
	"github.com/shenwei356/util/byteutil"
	"github.com/spf13/cobra"
)

// VERSION of fakit
const VERSION = "0.2.8"

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
			if isStdin(file) {
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

func getFlagPositiveInt(cmd *cobra.Command, flag string) int {
	value, err := cmd.Flags().GetInt(flag)
	checkError(err)
	if value <= 0 {
		checkError(fmt.Errorf("value of flag --%s should be greater than 0", flag))
	}
	return value
}

func getFlagNonNegativeInt(cmd *cobra.Command, flag string) int {
	value, err := cmd.Flags().GetInt(flag)
	checkError(err)
	if value < 0 {
		checkError(fmt.Errorf("value of flag --%s should be greater than 0", flag))
	}
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

func getIDRegexp(cmd *cobra.Command, flag string) string {
	var idRegexp string
	f := getFlagBool(cmd, "id-ncbi")
	if f {
		// e.g. >gi|110645304|ref|NC_002516.2| Pseudomonas aeruginosa PAO1 chromosome, complete genome
		// NC_002516.2 is ID
		idRegexp = `\|([^\|]+)\| `
	} else {
		idRegexp = getFlagString(cmd, "id-regexp")
	}
	return idRegexp
}

func getAlphabet(cmd *cobra.Command, flag string) *seq.Alphabet {
	value, err := cmd.Flags().GetString(flag)
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

func getFlagAlphabetGuessSeqLength(cmd *cobra.Command, flag string) int {
	alphabetGuessSeqLength := getFlagNonNegativeInt(cmd, flag)
	if alphabetGuessSeqLength > 0 && alphabetGuessSeqLength < 1000 {
		checkError(fmt.Errorf("value of flag --%s too small, should >= 1000", flag))
	}
	return alphabetGuessSeqLength
}

func getFlagValidateSeqLength(cmd *cobra.Command, flag string) int {
	validateSeqLength := getFlagNonNegativeInt(cmd, flag)
	if validateSeqLength > 0 && validateSeqLength < 1000 {
		checkError(fmt.Errorf("value of flag --%s too small, should >= 1000", flag))
	}
	return validateSeqLength
}

// Config is the global falgs
type Config struct {
	Alphabet               *seq.Alphabet
	ChunkSize              int
	BufferSize             int
	Threads                int
	LineWidth              int
	IDRegexp               string
	IDNCBI                 bool
	OutFile                string
	Quiet                  bool
	AlphabetGuessSeqLength int
	ValidateSeqLength      int
}

func getConfigs(cmd *cobra.Command) Config {
	return Config{
		Alphabet:  getAlphabet(cmd, "seq-type"),
		Threads:   getFlagPositiveInt(cmd, "threads"),
		LineWidth: getFlagNonNegativeInt(cmd, "line-width"),
		IDRegexp:  getIDRegexp(cmd, "id-regexp"),
		IDNCBI:    getFlagBool(cmd, "id-ncbi"),
		OutFile:   getFlagString(cmd, "out-file"),
		Quiet:     getFlagBool(cmd, "quiet"),
		AlphabetGuessSeqLength: getFlagAlphabetGuessSeqLength(cmd, "alphabet-guess-seq-length"),
	}

}

func sortRecordChunkMapID(chunks map[uint64]fastx.RecordChunk) sortutil.Uint64Slice {
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
	gz := strings.HasSuffix(file, ".gz") || strings.HasSuffix(file, ".GZ")
	if gz {
		file = file[0 : len(file)-3]
	}
	extension := filepath.Ext(file)
	name := file[0 : len(file)-len(extension)]
	if gz {
		extension += ".gz"
	}
	return name, extension
}

func isPlainFile(file string) bool {
	return file != "" && !strings.HasSuffix(strings.ToLower(file), ".gz")
}

func fileNotExists(file string) bool {
	_, err := os.Stat(file)
	return os.IsNotExist(err)
}

func copySeqs(file, newFile string) {
	fh, err := xopen.Ropen(file)
	checkError(err)
	defer fh.Close()

	outfh, err := xopen.Wopen(newFile)
	checkError(err)
	defer outfh.Close()

	_, err = io.Copy(outfh, fh)
	checkError(err)
}

func getFaidx(file string, idRegexp string) *fai.Faidx {
	var idx fai.Index
	var err error
	fileFai := file + ".fakit.fai"
	if fileNotExists(fileFai) {
		log.Infof("create FASTA index for %s", file)
		idx, err = fai.CreateWithIDRegexp(file, fileFai, idRegexp)
		checkError(err)
	} else {
		idx, err = fai.Read(fileFai)
		checkError(err)
	}
	faidx, err := fai.NewWithIndex(file, idx)
	checkError(err)
	return faidx
}

func subseqByFaix(faidx *fai.Faidx, chrs string, r fai.Record, start, end int) []byte {
	start, end, ok := seq.SubLocation(r.Length, start, end)
	if !ok {
		return []byte("")
	}
	subseq, err := faidx.SubSeq(chrs, start, end)
	checkError(err)
	return subseq
}

func subseqByFaixNotCleaned(faidx *fai.Faidx, chrs string, r fai.Record, start, end int) []byte {
	start, end, ok := seq.SubLocation(r.Length, start, end)
	if !ok {
		return []byte("")
	}
	subseq, err := faidx.SubSeqNotCleaned(chrs, start, end)
	checkError(err)
	return subseq
}

func getSeqIDAndLengthFromFaidxFile(file string) ([]string, []int, error) {
	ids := []string{}
	lengths := []int{}
	type idAndLength struct {
		id     string
		length int
	}
	fn := func(line string) (interface{}, bool, error) {
		if len(line) == 0 {
			return nil, false, nil
		}
		items := strings.Split(strings.TrimRight(line, "\r\n"), "\t")
		if len(items) != 5 {
			return nil, false, nil
		}

		length, err := strconv.Atoi(items[1])
		if err != nil {
			return nil, false, fmt.Errorf("seq length should be integer: %s", items[1])
		}
		return idAndLength{id: items[0], length: length}, true, nil
	}
	reader, err := breader.NewBufferedReader(file, runtime.NumCPU(), 100, fn)
	if err != nil {
		return ids, lengths, err
	}
	var info idAndLength
	for chunk := range reader.Ch {
		if chunk.Err != nil {
			return ids, lengths, err
		}
		for _, data := range chunk.Data {
			info = data.(idAndLength)
			ids = append(ids, info.id)
			lengths = append(lengths, info.length)
		}
	}

	return ids, lengths, nil
}

var reRegion = regexp.MustCompile(`\-?\d+:\-?\d+`)

var regionExample = `
 1-based index    1 2 3 4 5 6 7 8 9 10
negative index    0-9-8-7-6-5-4-3-2-1
           seq    A C G T N a c g t n
           1:1    A
           2:4      C G T
         -4:-2                c g t
         -4:-1                c g t n
         -1:-1                      n
          2:-2      C G T N a c g t
          1:-1    A C G T N a c g t n
`

func writeSeqs(records []*fastx.Record, file string, lineWidth int, quiet bool, dryRun bool) error {
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
		record.FormatToWriter(outfh, lineWidth)
	}

	return nil
}

func isStdin(file string) bool {
	return file == "-"
}

var defaultBytesBufferSize = 1 << 20

var bufferedByteSliceWrapper *byteutil.BufferedByteSliceWrapper

// func init() {
// 	bufferedByteSliceWrapper = byteutil.NewBufferedByteSliceWrapper(1, defaultBytesBufferSize)
// }
