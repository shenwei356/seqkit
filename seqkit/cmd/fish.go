// Copyright © 2019 Oxford Nanopore Technologies.
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
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/biogo/hts/bam"
	"github.com/biogo/hts/sam"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// AlnParams holds the alignment parameters.
type AlnParams struct {
	Match     int
	Mismatch  int
	GapOpen   int
	GapExtend int
}

// parseAlnParams parses alignment parameter string.
func parseAlnParams(s string) *AlnParams {
	p := new(AlnParams)
	sp := strings.Split(s, ",")
	if len(sp) != 4 {
		panic("Invalid alignment parameters: " + s)
	}
	for i, tok := range sp {
		sp[i] = strings.TrimSpace(tok)
	}
	var err error
	p.Match, err = strconv.Atoi(sp[0])
	checkError(err)
	p.Mismatch, err = strconv.Atoi(sp[1])
	checkError(err)
	p.GapOpen, err = strconv.Atoi(sp[2])
	checkError(err)
	p.GapExtend, err = strconv.Atoi(sp[3])
	checkError(err)
	return p
}

// parseRanges parses ranges string into a slice of ranges.
func parseRanges(rf string) Ranges {
	res := Ranges{}
	if len(rf) == 0 {
		return Ranges{Range{math.NaN(), math.NaN()}}
	}
	rst := strings.Split(strings.TrimSpace(rf), ",")
	for _, t := range rst {
		t = strings.TrimSpace(t)
		if len(t) == 0 {
			continue
		}
		rt := strings.Split(t, ":")
		if len(rt) != 2 {
			panic("invalid range: " + t)
		}
		var err error
		var ts, te string
		var s, e float64
		ts = strings.TrimSpace(rt[0])
		te = strings.TrimSpace(rt[1])
		if len(ts) == 0 {
			s = math.NaN()
		} else {
			s, err = strconv.ParseFloat(ts, 64)
			checkError(err)
		}
		if len(te) == 0 {
			e = math.NaN()
		} else {
			e, err = strconv.ParseFloat(te, 64)
			checkError(err)
		}
		if e > 0 && s > 0 && e < s {
			panic("invalid range: " + t)
		}
		res = append(res, Range{s, e})
	}
	return res
}

// fishCmd represents the fish command
var fishCmd = &cobra.Command{
	GroupID: "search",

	Use:   "fish",
	Short: "look for short sequences in larger sequences using local alignment",
	Long: `look for short sequences in larger sequences using local alignment

Attention:
  1. output coordinates are BED-like 0-based, left-close and right-open.
  2. alignment information are printed to STDERR.

`,

	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		lineWidth := config.LineWidth
		outFile := config.OutFile

		flagMinQual := getFlagFloat64(cmd, "min-qual")
		flagCutoff := 1 - math.Pow(10, flagMinQual/-10)
		_ = flagCutoff

		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		runtime.GOMAXPROCS(config.Threads)

		flagPass := getFlagBool(cmd, "pass")
		flagAln := getFlagBool(cmd, "print-aln")
		flagAlnParams := getFlagString(cmd, "aln-params")
		queryFastx := getFlagString(cmd, "query-fastx")
		flagSeq := getFlagString(cmd, "query-sequences")
		flagBam := getFlagString(cmd, "out-bam")
		flagRange := getFlagString(cmd, "ranges")
		flagNullMode := "self"
		flagStranded := getFlagBool(cmd, "stranded")
		flagAll := getFlagBool(cmd, "all")
		flagDesc := getFlagBool(cmd, "print-desc")
		flagInvert := getFlagBool(cmd, "invert")

		ranges := parseRanges(flagRange)
		alnParams := parseAlnParams(flagAlnParams)

		validateSeq := getFlagBool(cmd, "validate-seq")

		seq.ValidateSeq = validateSeq
		seq.ValidSeqThreads = config.Threads

		if !(alphabet == nil || alphabet == seq.Unlimit) {
			seq.ValidateSeq = true
		}

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		var alns []*AlignedSeq
		if len(files) == 0 {
			files = []string{"-"}
		}

		detector := NewSeqDetector(flagAll, flagStranded, flagNullMode, flagCutoff, alnParams)
		if queryFastx != "" {
			detector.LoadQueries(queryFastx)
		}
		if flagSeq != "" {
			detector.AddAnonQueries(strings.Split(flagSeq, ","))
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)

		var checkSeqType bool
		var isFastq bool
		var printQual bool
		var head []byte
		var text []byte
		var buffer *bytes.Buffer
		var record *fastx.Record
		var sequence *seq.Seq
		var count int
		var refMap map[string]int
		var samRefs []*sam.Reference
		if flagBam != "" {
			alns = make([]*AlignedSeq, 0, 1024)
			samRefs = make([]*sam.Reference, 0, 1024)
			refMap = make(map[string]int, 1024)
		}
		first := true

		for _, file := range files {
			fastxReader, err := fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)

			checkSeqType = true
			printQual = false
			for {
				record, err = fastxReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					checkError(err)
					break
				}

				if checkSeqType {
					isFastq = fastxReader.IsFastq
					if isFastq {
						config.LineWidth = 0
						printQual = true
					}
					checkSeqType = false
				}

				refId := string(record.Name)
				if !flagDesc {
					refId = strings.Split(refId, " ")[0]
				}

				hits := detector.Detect(&Reference{refId, string(record.Seq.Seq), ranges}, flagAll)

				if !flagInvert {
					for _, h := range hits {
						if first {
							fmt.Fprintf(os.Stderr, "%s\n", strings.Join(h.Fields(), "\t"))
						}
						first = false
						fmt.Fprintf(os.Stderr, "%s\n", h)
						if flagAln {
							fmt.Fprintf(os.Stderr, "%s\n", h.AlnString())
						}
						h.Ref.Seq = ""
					}

					if flagBam != "" {
						nr, _ := sam.NewReference(refId, "", "", len(record.Seq.Seq), nil, nil)
						samRefs = append(samRefs, nr)
						refMap[refId] = count
						alns = append(alns, hits...)
					}
				} else {
					if first {
						fmt.Fprintf(os.Stderr, "Ref\n")
						first = false
					}
					if len(hits) == 0 {
						fmt.Fprintf(os.Stderr, "%s\n", refId)
					}
				}

				count++
				if !flagPass {
					continue
				}

				head = record.Name
				sequence = record.Seq

				if isFastq {
					outfh.Write(_mark_fastq)
					outfh.Write(head)
					outfh.Write(_mark_newline)
				} else {
					outfh.Write(_mark_fasta)
					outfh.Write(head)
					outfh.Write(_mark_newline)
				}

				text, buffer = wrapByteSlice(sequence.Seq, config.LineWidth, buffer)
				outfh.Write(text)

				outfh.Write(_mark_newline)

				if printQual {
					outfh.Write(_mark_plus_newline)

					text, buffer = wrapByteSlice(sequence.Qual, config.LineWidth, buffer)
					outfh.Write(text)

					outfh.Write(_mark_newline)
				}
			} // record
			fastxReader.Close()
			config.LineWidth = lineWidth

		} //file

		if flagBam != "" {
			saveBam(flagBam, samRefs, refMap, alns)
		}
		outfh.Close()
	},
}

// saveBam writes alignment records to a BAM file.
func saveBam(bamFile string, refs []*sam.Reference, refMap map[string]int, alns []*AlignedSeq) {
	var err error
	var bamWriter *bam.Writer
	//fh, err := xopen.Wopen(bamFile)
	fh, err := os.Create(bamFile)
	checkError(err)

	checkError(err)
	var h *sam.Header
	h, err = sam.NewHeader([]byte{}, refs)
	checkError(err)
	fish := sam.NewProgram("seqkit", "seqkit", "seqkit fish", "-", "1.0")
	h.AddProgram(fish)
	bamWriter, err = bam.NewWriter(fh, h, 50)
	checkError(err)
	for _, a := range alns {
		mq := -10 * math.Log10(1.0-(a.Score/a.Query.NullScore))
		if math.IsNaN(mq) || mq > 60 {
			mq = 60
		}
		var record *sam.Record
		pg, err := sam.NewAux(sam.NewTag("PG"), 0)
		record, err = NewSAMRecordFromAln(a.Query.Name, refs[refMap[a.Ref.Name]], a.RefStart, a.RefEnd, a.QueryStart, a.QueryEnd, a.RefAln, a.QueryAln, a.Query.Strand, byte(uint8(mq)), a.Query.Seq, nil, []sam.Aux{pg})
		checkError(err)
		if !sam.IsValidRecord(record) {
			panic("failed to build BAM record from raw alignment: \n" + a.String() + "\n" + record.String())
		}
		if !a.Best {
			record.Flags |= sam.Secondary
		}

		err = bamWriter.Write(record)
		checkError(err)
	}
	checkError(bamWriter.Close())
	checkError(fh.Close())
}

func init() {
	RootCmd.AddCommand(fishCmd)

	fishCmd.Flags().BoolP("validate-seq", "v", false, "validate bases according to the alphabet")
	fishCmd.Flags().BoolP("pass", "x", false, "pass through mode (write input to stdout)")
	fishCmd.Flags().BoolP("all", "a", false, "search all")
	fishCmd.Flags().StringP("query-fastx", "f", "", "query fasta")
	fishCmd.Flags().StringP("aln-params", "p", "4,-4,-2,-1", "alignment parameters in format \"<match>,<mismatch>,<gap_open>,<gap_extend>\"")
	fishCmd.Flags().StringP("query-sequences", "F", "", "query sequences")
	fishCmd.Flags().StringP("out-bam", "b", "", "save aligmnets to this BAM file (memory intensive)")
	fishCmd.Flags().BoolP("stranded", "s", false, "search + strand only")
	fishCmd.Flags().StringP("ranges", "r", "", "target ranges, for example: \":10,30:40,-20:\"")
	fishCmd.Flags().BoolP("print-desc", "D", false, "print full sequence header")
	fishCmd.Flags().BoolP("print-aln", "g", false, "print sequence alignments")
	fishCmd.Flags().BoolP("invert", "i", false, "print out references not matching with any query")
	fishCmd.Flags().Float64P("min-qual", "q", 5.0, "minimum mapping quality")
}

// NewRecordFromAln builds a new SAM record based on the provided local alignment and its reference/query coordinates.
func NewSAMRecordFromAln(name string, ref *sam.Reference, refStart, refEnd, queryStart, queryEnd int, refAln, queryAln string, strand string, mapQ byte, seq string, qual []byte, aux []sam.Aux) (*sam.Record, error) {
	if len(refAln) != len(queryAln) {
		panic("alignment length mismatch")
	}
	if len(refAln) == 0 {
		panic("empty alignment")
	}
	gap := byte('-')

	rawCo := make([]sam.CigarOp, 0, len(seq))
	co := make([]sam.CigarOp, 0, len(seq))
	var consumed int
	var nm int

	// Building the CIGAR in two steps for clarity.
	if queryStart > 0 {
		rawCo = append(rawCo, sam.NewCigarOp(sam.CigarSoftClipped, queryStart))
	}
	for i := range refAln {
		if queryAln[i] == gap {
			rawCo = append(rawCo, sam.NewCigarOp(sam.CigarDeletion, 1))
			nm++
			continue

		} else if refAln[i] == gap {
			rawCo = append(rawCo, sam.NewCigarOp(sam.CigarInsertion, 1))
			consumed++
			nm++
			continue
		} else {
			rawCo = append(rawCo, sam.NewCigarOp(sam.CigarMatch, 1))
			consumed++
			if queryAln[i] != refAln[i] {
				nm++
			}
			continue

		}
	} // refAln

	leftover := len(seq) - queryStart - consumed
	if leftover > 0 {
		rawCo = append(rawCo, sam.NewCigarOp(sam.CigarSoftClipped, leftover))
	}

	cop := rawCo[0].Type()
	length := rawCo[0].Len()
	var o sam.CigarOp
	for i := 1; i < len(rawCo); i++ {
		o = rawCo[i]
		if o.Type() == cop {
			length++
			continue
		}
		co = append(co, sam.NewCigarOp(cop, length))
		length = o.Len()
		cop = o.Type()
	}
	co = append(co, sam.NewCigarOp(o.Type(), length))

	switch strand {
	case "-":
	case "+":
	default:
		panic("Invalid strand: " + strand)

	}

	nmTag, _ := sam.NewAux(sam.NewTag("NM"), nm)
	aux = append(aux, nmTag)
	res, err := sam.NewRecord(name, ref, nil, refStart, -1, 0, mapQ, co, []byte(seq), qual, aux)
	if err != nil {
		panic(err)
	}
	if strand == "-" {
		res.Flags |= sam.Reverse
	}
	return res, err
}
