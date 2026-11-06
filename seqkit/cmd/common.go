// Copyright © 2016-2026 Wei Shen <shenwei356@gmail.com>
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
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"

	"github.com/cespare/xxhash/v2"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// commonCmd represents the common command
var commonCmd = &cobra.Command{
	GroupID: "set",

	Use:   "common",
	Short: "find common/shared sequences of multiple files by id/name/sequence",
	Long: `find common/shared sequences of multiple files by id/name/sequence

Note:
  1. 'seqkit common' is designed to support 2 and MORE files.
     Results always come from the first input, including stdin.
  2. When comparing by sequences,
     a) Both positive and negative strands are compared. You can switch on
        -P/--only-positive-strand for considering the positive strand only.
     b) You can switch on -e/--check-embedded-seqs to check embedded sequences.
          e.g, for file A and B, the reverse complement sequence of CCCC from file B
          is a part of TTGGGGTT from file A, we will extract and output GGGG from file A.
          If sequences CCC exist in other files except file A, we will skip it,
          as it is an embedded subsequence of GGGG.
        It is recommended to put the smallest file as the first file, for saving
        memory usage.
        With -s -e, empty records are skipped, but every input still participates
        in the intersection. Fragments retain their corresponding qualities and
        are ordered by first-file record, then decreasing length and increasing start.
  3. For 2 files, 'seqkit grep' is much faster and consumes lesser memory:
       seqkit grep -f <(seqkit seq -n -i small.fq.gz) big.fq.gz # by seq ID
     But note that searching by sequence would be much slower, as it's partly
     string matching.
       seqkit grep -s -f <(seqkit seq -s small.fq.gz) big.fq.gz # much slower!!!!
  4. Some records in one file may have same sequences/IDs. They will ALL be
     retrieved if the sequence/ID was shared in multiple files.
     So the records number may be larger than that of the smallest file.

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		// lineWidth := config.LineWidth
		outFile := config.OutFile
		quiet := config.Quiet
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		bySeq := getFlagBool(cmd, "by-seq")
		byName := getFlagBool(cmd, "by-name")
		ignoreCase := getFlagBool(cmd, "ignore-case")
		checkembeddedSeqs := getFlagBool(cmd, "check-embedded-seqs")

		if bySeq && byName {
			checkError(fmt.Errorf("only one/none of the flags -s (--by-seq) and -n (--by-name) is allowed"))
		}

		// revcom := getFlagBool(cmd, "consider-revcom")
		revcom := !getFlagBool(cmd, "only-positive-strand")

		if !revcom && !bySeq {
			checkError(fmt.Errorf("flag -s (--by-seq) needed when using -P (--only-positive-strand)"))
		}

		if checkembeddedSeqs && !bySeq {
			checkError(fmt.Errorf("flag -s (--by-seq) needed when using -e (--check-embedded-seqs)"))
		}

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", !config.SkipFileCheck)

		if len(files) < 2 {
			checkError(errors.New("at least 2 files needed"))
		}
		if !config.SkipFileCheck {
			for _, file := range files {
				checkIfFilesAreTheSame(file, outFile, "input", "output")
			}
		}

		firstLabel := files[0]
		var spool string
		// Cache bytes rather than parsed records so stdin preserves every header and quality.
		if isStdin(files[0]) {
			temp, err := os.CreateTemp("", "seqkit-common-*")
			checkError(err)
			spool = temp.Name()
			defer os.Remove(spool)
			_, copyErr := io.Copy(temp, os.Stdin)
			closeErr := temp.Close()
			if copyErr != nil || closeErr != nil {
				os.Remove(spool)
				checkError(errors.Join(copyErr, closeErr))
			}
			files[0] = spool
		}
		check := func(err error) {
			if err != nil {
				if spool != "" {
					os.Remove(spool)
				}
				checkError(err)
			}
		}

		outfh, err := xopen.Wopen(outFile)
		check(err)
		defer closeOutput(outfh)
		if bySeq && checkembeddedSeqs {
			err := commonEmbedded(files, firstLabel, outfh, config, revcom, ignoreCase)
			if err != nil {
				closeOutput(outfh)
				check(err)
			}
			return
		}
		var fastxReader *fastx.Reader
		var record *fastx.Record
		var rc *seq.Seq
		var _seq, _seqRC []byte
		var subject, subjectRC uint64
		isFirstFile := true
		firstFile := files[0]
		var ok bool

		// ------------------------- for other situations  --------------------------------

		// target -> file idx -> struct{}
		counter := make(map[uint64]map[int]struct{}, 1000)
		var _counter map[int]struct{} // file idx -> struct{}

		// target -> seqnames in firstFile
		// note that it's []string, i.e., records may have same sequences
		names := make(map[uint64][]string, 1024)

		for i, file := range files {
			if !quiet {
				label := file
				if i == 0 {
					label = firstLabel
				}
				log.Infof("read file %d/%d: %s", i+1, len(files), label)
			}

			fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
			check(err)

			for {
				record, err = fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					check(err)
					break
				}

				if bySeq {
					_seq = record.Seq.Seq
					if ignoreCase {
						_seq = bytes.ToLower(_seq)
					}
					subject = xxhash.Sum64(_seq)

					if revcom {
						rc = record.Seq.RevCom()
						_seqRC = rc.Seq
						if ignoreCase {
							_seqRC = bytes.ToLower(_seqRC)
						}
						subjectRC = xxhash.Sum64(_seqRC)

						if subjectRC < subject { // only save the smaller one, like keeping canonical k-mers
							subject = subjectRC
						}
					}
				} else if byName {
					if ignoreCase {
						subject = xxhash.Sum64(bytes.ToLower(record.Name))
					} else {
						subject = xxhash.Sum64(record.Name)
					}
				} else { // byID
					if ignoreCase {
						subject = xxhash.Sum64(bytes.ToLower(record.ID))
					} else {
						subject = xxhash.Sum64(record.ID)
					}
				}

				if _counter, ok = counter[subject]; !ok {
					_counter = make(map[int]struct{})
					counter[subject] = _counter
				}
				_counter[i] = struct{}{}

				if isFirstFile {
					if _, ok = names[subject]; !ok {
						names[subject] = make([]string, 0, 1)
					}
					names[subject] = append(names[subject], string(record.Name))
				}
			}
			fastxReader.Close()

			if isFirstFile {
				isFirstFile = false
			}
		}

		// find common seqs
		if !quiet {
			log.Info("find common seqs ...")
		}
		fileNum := len(files)

		var nHashes, nOriginalRecords, nOutput int
		for subject, presence := range counter {
			if len(presence) != fileNum {
				continue
			}

			nHashes++                               // number of hash values
			nOriginalRecords += len(names[subject]) // number of records

		}

		var t string
		if byName {
			t = "sequence headers"
		} else if bySeq {
			t = "sequences"
		} else {
			t = "sequence IDs"
		}
		if nHashes == 0 {
			log.Infof("no common %s found", t)
			return
		}
		if !quiet {
			log.Infof("%d unique %s found in %d files, which belong to %d records in the first file: %s",
				nHashes, t, fileNum, nOriginalRecords, firstLabel)
		}

		// retrieve
		fastxReader, err = fastx.NewReader(alphabet, firstFile, idRegexp)
		check(err)
		checkFormat := true
		for {
			record, err = fastxReader.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				check(err)
				break
			}
			if checkFormat {
				checkFormat = false
				if fastxReader.IsFastq {
					if !config.LineWidthChanged {
						config.LineWidth = 0
					}
					fastx.ForcelyOutputFastq = true
				}
			}

			subject = commonRecordHash(record, bySeq, byName, revcom, ignoreCase)
			if len(counter[subject]) == fileNum {
				nOutput++
				record.FormatToWriter(outfh, config.LineWidth)
			}
		}
		fastxReader.Close()

		if !quiet {
			log.Infof("%d common/shared sequences saved to: %s", nOutput, outFile)
		}
	},
}

func init() {
	RootCmd.AddCommand(commonCmd)

	commonCmd.Flags().BoolP("by-name", "n", false, "match by full name instead of just id")
	commonCmd.Flags().BoolP("by-seq", "s", false, "match by sequence")
	commonCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")
	// commonCmd.Flags().BoolP("consider-revcom", "r", false, "considering the reverse compelment sequence")
	commonCmd.Flags().BoolP("only-positive-strand", "P", false, "only considering the positive strand when comparing by sequence")
	commonCmd.Flags().BoolP("check-embedded-seqs", "e", false, "check embedded sequences, e.g., if a sequence is part of another one, we'll keep the shorter one")
}

// commonRecordHash uses the same canonical comparison for input and retrieval.
func commonRecordHash(record *fastx.Record, bySeq, byName, revcom, ignoreCase bool) uint64 {
	text := record.ID
	if byName {
		text = record.Name
	} else if bySeq {
		text = record.Seq.Seq
	}
	if ignoreCase {
		text = bytes.ToLower(text)
	}
	hash := xxhash.Sum64(text)
	if bySeq && revcom {
		rc := record.Seq.RevCom().Seq
		if ignoreCase {
			rc = bytes.ToLower(rc)
		}
		hash = min(hash, xxhash.Sum64(rc))
	}
	return hash
}

// commonSpan is a half-open interval in an unchanged first-file record.
type commonSpan struct{ begin, end int }

// commonEmbedded intersects snapshots, so records within one file cannot remove
// candidates needed by later records from that same file.
func commonEmbedded(files []string, firstLabel string, out *xopen.Writer, config Config, revcom, ignoreCase bool) error {
	originals, err := readRecordsChecked(files[0], config.Alphabet, config.IDRegexp)
	if err != nil {
		return err
	}
	candidates := make([][]commonSpan, len(originals))
	normalized := make([][]byte, len(originals))
	seenIDs := make(map[string]bool)
	normalize := func(text []byte) []byte {
		if ignoreCase {
			return bytes.ToLower(text)
		}
		return text
	}
	for i, record := range originals {
		if len(record.Seq.Seq) == 0 {
			continue
		}
		if seenIDs[string(record.ID)] {
			return fmt.Errorf("duplicated ID not allowed: %s", record.ID)
		}
		seenIDs[string(record.ID)] = true
		normalized[i] = normalize(record.Seq.Seq)
		candidates[i] = []commonSpan{{0, len(record.Seq.Seq)}}
	}
	for fileIndex, file := range files[1:] {
		if !config.Quiet {
			log.Infof("read file %d/%d: %s", fileIndex+2, len(files), file)
		}
		reader, err := fastx.NewReader(config.Alphabet, file, config.IDRegexp)
		if err != nil {
			return err
		}
		next := make([]map[commonSpan]bool, len(originals))
		// Equal-length candidates use hashes; only unequal lengths need containment scans.
		type targetSpan struct {
			record int
			span   commonSpan
			text   []byte
		}
		var targets []targetSpan
		exact := make(map[uint64][]targetSpan)
		for i, spans := range candidates {
			for _, span := range spans {
				target := targetSpan{i, span, normalized[i][span.begin:span.end]}
				targets = append(targets, target)
				key := canonicalCommonSequence(target.text, originals[i].Seq.Alphabet, revcom)
				hash := xxhash.Sum64(key)
				exact[hash] = append(exact[hash], target)
			}
		}
		sort.Slice(targets, func(i, j int) bool { return len(targets[i].text) < len(targets[j].text) })
		keep := func(target targetSpan, span commonSpan) {
			if next[target.record] == nil {
				next[target.record] = make(map[commonSpan]bool)
			}
			next[target.record][span] = true
		}

		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				reader.Close()
				return fmt.Errorf("%s: %w", file, err)
			}
			if len(record.Seq.Seq) == 0 {
				continue
			}
			query := normalize(record.Seq.Seq)
			var queryRC []byte
			if revcom {
				queryRC = normalize(record.Seq.RevCom().Seq)
			}
			key := query
			if revcom && bytes.Compare(queryRC, key) < 0 {
				key = queryRC
			}
			for _, target := range exact[xxhash.Sum64(key)] {
				if bytes.Equal(target.text, query) || revcom && bytes.Equal(target.text, queryRC) {
					keep(target, target.span)
				}
			}
			shorter := sort.Search(len(targets), func(i int) bool { return len(targets[i].text) >= len(query) })
			longer := sort.Search(len(targets), func(i int) bool { return len(targets[i].text) > len(query) })
			for _, target := range targets[:shorter] {
				if bytes.Contains(query, target.text) || revcom && bytes.Contains(queryRC, target.text) {
					keep(target, target.span)
				}
			}
			for _, target := range targets[longer:] {
				position := bytes.Index(target.text, query)
				if position < 0 && revcom {
					position = bytes.Index(target.text, queryRC)
				}
				if position >= 0 {
					keep(target, commonSpan{target.span.begin + position, target.span.begin + position + len(query)})
				}
			}

		}
		reader.Close()
		remaining := 0
		for i := range candidates {
			candidates[i] = nil
			for span := range next[i] {
				candidates[i] = append(candidates[i], span)
			}
			if len(candidates[i]) > 0 {
				remaining++
			}
		}
		if !config.Quiet {
			log.Infof("%d first-file records with candidate fragments left", remaining)
		}
	}
	unique := make(map[uint64][][]byte)
	uniqueCount := 0
	contributed, written := 0, 0
	fastx.ForcelyOutputFastq = false
	for i, record := range originals {
		spans := candidates[i]
		sort.Slice(spans, func(a, b int) bool {
			x, y := spans[a], spans[b]
			if x.end-x.begin != y.end-y.begin {
				return x.end-x.begin > y.end-y.begin
			}
			return x.begin < y.begin
		})
		retained := make([]commonSpan, 0, len(spans))
		for _, span := range spans {
			contained := false
			for _, previous := range retained {
				if span.begin >= previous.begin && span.end <= previous.end {
					contained = true
					break
				}
			}
			if contained {
				continue
			}
			retained = append(retained, span)
			// Views preserve the original record without copying its full sequence for each hit.
			fragment := *record
			sequence := *record.Seq
			sequence.Seq = record.Seq.Seq[span.begin:span.end]
			if len(record.Seq.Qual) > 0 {
				sequence.Qual = record.Seq.Qual[span.begin:span.end]
			}
			if len(record.Seq.QualValue) > 0 {
				sequence.QualValue = record.Seq.QualValue[span.begin:span.end]
			}
			fragment.Seq = &sequence
			if span.begin != 0 || span.end != len(record.Seq.Seq) {
				fragment.Name = []byte(fmt.Sprintf("%s:%d-%d", record.ID, span.begin+1, span.end))
				if len(record.Desc) > 0 {
					fragment.Name = append(fragment.Name, []byte(" "+string(record.Desc))...)
				}
			}
			width := config.LineWidth
			if len(record.Seq.Qual) > 0 && !config.LineWidthChanged {
				width = 0
			}
			fragment.FormatToWriter(out, width)
			key := canonicalCommonSequence(normalized[i][span.begin:span.end], record.Seq.Alphabet, revcom)
			hash := xxhash.Sum64(key)
			exists := false
			for _, previous := range unique[hash] {
				if bytes.Equal(previous, key) {
					exists = true
					break
				}
			}
			if !exists {
				unique[hash] = append(unique[hash], key)
				uniqueCount++
			}

			written++
		}
		if len(retained) > 0 {
			contributed++
		}
	}
	if !config.Quiet {
		log.Infof("%d original records, %d distinct common sequences, %d output fragments from first file: %s", contributed, uniqueCount, written, firstLabel)
	}
	return nil
}

// canonicalCommonSequence returns a view or reverse complement for exact comparison.
func canonicalCommonSequence(text []byte, alphabet *seq.Alphabet, revcom bool) []byte {
	if !revcom {
		return text
	}
	sequence := &seq.Seq{Alphabet: alphabet, Seq: text}
	rc := sequence.RevCom().Seq
	if bytes.Compare(rc, text) < 0 {
		return rc
	}
	return text
}
