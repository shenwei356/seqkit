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
	"strconv"
	"strings"

	"github.com/shenwei356/bio/featio/gtf"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fai"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/util/byteutil"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// subseqCmd represents the subseq command
var subseqCmd = &cobra.Command{
	Use:   "subseq",
	Short: "get subsequences by region/gtf/bed, including flanking sequences",
	Long: fmt.Sprintf(`get subsequences by region/gtf/bed, including flanking sequences.

Recommendation: use plain FASTA file, so seqkit could utilize FASTA index.

The definition of region is 1-based and with some custom design.

Examples:
%s
`, regionExample),
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		lineWidth := config.LineWidth
		outFile := config.OutFile
		quiet := config.Quiet
		seq.AlphabetGuessSeqLenghtThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		gtf.Threads = config.Threads
		fai.MapWholeFile = false
		Threads = config.Threads
		runtime.GOMAXPROCS(config.Threads)

		files := getFileList(args)

		chrs := getFlagStringSlice(cmd, "chr")
		chrs2 := make([]string, len(chrs))
		for i, chr := range chrs {
			chrs2[i] = chr
		}
		chrs = chrs2
		chrsMap := make(map[string]struct{}, len(chrs))
		for _, chr := range chrs {
			chrsMap[strings.ToLower(chr)] = struct{}{}
		}
		region := getFlagString(cmd, "region")

		gtfFile := getFlagString(cmd, "gtf")
		bedFile := getFlagString(cmd, "bed")
		gtfTag := getFlagString(cmd, "gtf-tag")
		choosedFeatures := getFlagStringSlice(cmd, "feature")
		choosedFeatures2 := make([]string, len(choosedFeatures))
		for i, f := range choosedFeatures {
			choosedFeatures2[i] = strings.ToLower(f)
		}
		choosedFeatures = choosedFeatures2

		upStream := getFlagNonNegativeInt(cmd, "up-stream")
		downStream := getFlagNonNegativeInt(cmd, "down-stream")
		onlyFlank := getFlagBool(cmd, "only-flank")
		if onlyFlank {
			if upStream > 0 && downStream > 0 {
				checkError(fmt.Errorf("when flag -f (--only-flank) given," +
					" only one of flags -u (--up-stream) and -d (--down-stream) is allowed"))
			} else if upStream == 0 && downStream == 0 {
				checkError(fmt.Errorf("when flag -f (--only-flank) given," +
					" one of flags -u (--up-stream) and -d (--down-stream) should be given"))
			}
		}
		if region != "" {
			if upStream > 0 || downStream > 0 || onlyFlank {
				checkError(fmt.Errorf("when flag -r (--region) given," +
					" any of flags -u (--up-stream), -d (--down-stream) and -f (--only-flank) is not allowed"))
			}
		}

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		idRe, err := regexp.Compile(idRegexp)
		if err != nil {
			checkError(fmt.Errorf("fail to compile regexp: %s", idRegexp))
		}

		var start, end int

		var gtfFeaturesMap map[string]type2gtfFeatures
		var bedFeatureMap map[string][]BedFeature

		if region != "" {
			if !reRegion.MatchString(region) {
				checkError(fmt.Errorf(`invalid region: %s. type "seqkit subseq -h" for more examples`, region))
			}
			r := strings.Split(region, ":")
			start, err = strconv.Atoi(r[0])
			checkError(err)
			end, err = strconv.Atoi(r[1])
			checkError(err)
			if start == 0 || end == 0 {
				checkError(fmt.Errorf("both start and end should not be 0"))
			}
			if start < 0 && end > 0 {
				checkError(fmt.Errorf("when start < 0, end should not > 0"))
			}
		} else if gtfFile != "" {
			if !quiet {
				log.Info("read GTF file ...")
			}
			gtfFeaturesMap = make(map[string]type2gtfFeatures)

			gtf.Threads = config.Threads // threads of gtf.ReadFeatures
			var features []gtf.Feature
			var err error
			if len(chrs) > 0 || len(choosedFeatures) > 0 {
				features, err = gtf.ReadFilteredFeatures(gtfFile, chrs, choosedFeatures, []string{gtfTag})
			} else {
				features, err = gtf.ReadFilteredFeatures(gtfFile, []string{}, []string{}, []string{gtfTag})
			}
			checkError(err)

			var chr, feat string
			for _, feature := range features {
				chr = strings.ToLower(feature.SeqName)
				if _, ok := gtfFeaturesMap[chr]; !ok {
					gtfFeaturesMap[chr] = make(map[string][]gtf.Feature)
				}
				feat = strings.ToLower(feature.Feature)
				if _, ok := gtfFeaturesMap[chr][feat]; !ok {
					gtfFeaturesMap[chr][feat] = []gtf.Feature{}
				}
				gtfFeaturesMap[chr][feat] = append(gtfFeaturesMap[chr][feat], feature)
			}
			if !quiet {
				log.Infof("%d GTF features loaded", len(features))
			}
		} else if bedFile != "" {
			if !quiet {
				log.Info("read BED file ...")
			}
			if len(choosedFeatures) > 0 {
				checkError(fmt.Errorf("when given flag -b (--bed), flag -f (--feature) is not allowed"))
			}
			bedFeatureMap = make(map[string][]BedFeature)
			Threads = config.Threads // threads of ReadBedFeatures

			var features []BedFeature
			var err error
			if len(chrs) > 0 {
				features, err = ReadBedFilteredFeatures(bedFile, chrs)
			} else {
				features, err = ReadBedFeatures(bedFile)
			}
			checkError(err)

			var chr string
			for _, feature := range features {
				chr = strings.ToLower(feature.Chr)
				if _, ok := bedFeatureMap[chr]; !ok {
					bedFeatureMap[chr] = []BedFeature{}
				}
				bedFeatureMap[chr] = append(bedFeatureMap[chr], feature)
			}
			if !quiet {
				log.Infof("%d BED features loaded", len(features))
			}
		}

		for _, file := range files {
			// plain fasta, using Faidx
			if !isStdin(file) && isPlainFile(file) {
				// check seq format, ignoring fastq
				alphabet2, isFastq, err := fastx.GuessAlphabet(file)
				checkError(err)

				id2name := make(map[string][]byte)

				if !isFastq { // ok, it's fasta!
					// faidx, err := fai.New(file)
					// checkError(err)
					faidx := getFaidx(file, "^(.+)$")

					var id string
					for head := range faidx.Index {
						id = strings.ToLower(string(fastx.ParseHeadID(idRe, []byte(head))))
						id2name[id] = []byte(head)
					}

					var chr2 string
					if region != "" {
						if len(chrs) > 0 {
							for _, chr := range chrs {
								chr2 = string(id2name[strings.ToLower(chr)])
								r, ok := faidx.Index[chr2]
								if !ok {
									log.Warningf(`sequence (%s) not found in file: %s`, chr2, file)
									continue
								}

								s, e, _ := seq.SubLocation(r.Length, start, end)
								subseq := subseqByFaix(faidx, chr2, r, start, end)
								outfh.WriteString(fmt.Sprintf(">%s_%d-%d %s\n",
									chr, s, e, chr2))
								outfh.Write(byteutil.WrapByteSlice(subseq, config.LineWidth))
								outfh.WriteString("\n")
							}
							continue
						} else {
							// read all sequence
						}

					} else if gtfFile != "" {
						for chr := range gtfFeaturesMap {
							if len(chrs) > 0 { // selected chrs
								if _, ok := chrsMap[strings.ToLower(chr)]; !ok {
									continue
								}
							}

							chr = string(id2name[chr])

							r, ok := faidx.Index[chr]
							if !ok {
								log.Warningf(`sequence (%s) not found in file: %s`, chr, file)
								continue
							}

							subseq := subseqByFaix(faidx, chr, r, 1, -1)
							record, err := fastx.NewRecord(alphabet2, fastx.ParseHeadID(idRe, []byte(chr)), []byte(chr), subseq)
							checkError(err)

							subseqByGTFFile(outfh, record, config.LineWidth,
								gtfFeaturesMap, choosedFeatures,
								onlyFlank, upStream, downStream, gtfTag)
						}

						continue
					} else if bedFile != "" {
						for chr := range bedFeatureMap {
							if len(chrs) > 0 { // selected chrs
								if _, ok := chrsMap[strings.ToLower(chr)]; !ok {
									continue
								}
							}

							chr = string(id2name[chr])

							r, ok := faidx.Index[chr]
							if !ok {
								log.Warningf(`sequence (%s) not found in file: %s`, chr, file)
								continue
							}

							subseq := subseqByFaix(faidx, chr, r, 1, -1)
							record, err := fastx.NewRecord(alphabet2, fastx.ParseHeadID(idRe, []byte(chr)), []byte(chr), subseq)
							checkError(err)

							subSeqByBEDFile(outfh, record, config.LineWidth,
								bedFeatureMap,
								onlyFlank, upStream, downStream)
						}

						continue
					}

					checkError(faidx.Close())
				}
			}

			var record *fastx.Record
			var fastxReader *fastx.Reader
			// Parse all sequences
			fastxReader, err = fastx.NewReader(alphabet, file, idRegexp)
			checkError(err)

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
				}

				if region != "" {
					subseqByRegion(outfh, record, config.LineWidth, start, end)

				} else if gtfFile != "" {
					seqname := strings.ToLower(string(record.ID))
					if _, ok := gtfFeaturesMap[seqname]; !ok {
						continue
					}

					subseqByGTFFile(outfh, record, config.LineWidth,
						gtfFeaturesMap, choosedFeatures,
						onlyFlank, upStream, downStream, gtfTag)

				} else if bedFile != "" {
					seqname := strings.ToLower(string(record.ID))
					if _, ok := bedFeatureMap[seqname]; !ok {
						continue
					}

					subSeqByBEDFile(outfh, record, config.LineWidth,
						bedFeatureMap,
						onlyFlank, upStream, downStream)
				}
			}

			config.LineWidth = lineWidth
		}
	},
}

type type2gtfFeatures map[string][]gtf.Feature

func subseqByRegion(outfh *xopen.Writer, record *fastx.Record, lineWidth int, start, end int) {
	record.Seq = record.Seq.SubSeq(start, end)
	record.FormatToWriter(outfh, lineWidth)
}

func subseqByGTFFile(outfh *xopen.Writer, record *fastx.Record, lineWidth int,
	gtfFeaturesMap map[string]type2gtfFeatures, choosedFeatures []string,
	onlyFlank bool, upStream int, downStream int, gtfTag string) {

	seqname := strings.ToLower(string(record.ID))

	var strand, tag, outname, flankInfo string
	var s, e int
	var subseq *seq.Seq

	featsMap := make(map[string]struct{}, len(choosedFeatures))
	for _, chr := range choosedFeatures {
		featsMap[chr] = struct{}{}
	}

	for featureType := range gtfFeaturesMap[seqname] {
		if len(choosedFeatures) > 0 {
			if _, ok := featsMap[strings.ToLower(featureType)]; !ok {
				continue
			}
		}
		for _, feature := range gtfFeaturesMap[seqname][featureType] {
			s, e = feature.Start, feature.End
			if feature.Strand != nil && *feature.Strand == "-" {
				if onlyFlank {
					if upStream > 0 {
						s = feature.End + 1
						e = feature.End + upStream
					} else {
						s = feature.Start - downStream
						e = feature.Start - 1
					}
				} else {
					s = feature.Start - downStream // seq.SubSeq will check it
					e = feature.End + upStream
				}
				if s < 1 {
					s = 1
				}
				if e > len(record.Seq.Seq) {
					e = len(record.Seq.Seq)
				}
				subseq = record.Seq.SubSeq(s, e).RevComInplace()
			} else {
				if onlyFlank {
					if upStream > 0 {
						s = feature.Start - upStream
						e = feature.Start - 1
					} else {
						s = e + 1
						e = e + downStream
					}
				} else {
					s = feature.Start - upStream
					e = feature.End + downStream
				}
				if s < 1 {
					s = 1
				}
				if e > len(record.Seq.Seq) {
					e = len(record.Seq.Seq)
				}
				subseq = record.Seq.SubSeq(s, e)
			}

			if feature.Strand == nil {
				strand = "."
			} else {
				strand = *feature.Strand
			}
			tag = ""
			for _, arrtribute := range feature.Attributes {
				if arrtribute.Tag == gtfTag {
					tag = arrtribute.Value
					break
				}
			}
			if upStream > 0 {
				if onlyFlank {
					flankInfo = fmt.Sprintf("_usf:%d", upStream)
				} else if downStream > 0 {
					flankInfo = fmt.Sprintf("_us:%d_ds:%d", upStream, downStream)
				} else {
					flankInfo = fmt.Sprintf("_us:%d", upStream)
				}
			} else if downStream > 0 {
				if onlyFlank {
					flankInfo = fmt.Sprintf("_dsf:%d", downStream)
				} else if upStream > 0 {
					flankInfo = fmt.Sprintf("_us:%d_ds:%d", upStream, downStream)
				} else {
					flankInfo = fmt.Sprintf("_ds:%d", downStream)
				}
			} else {
				flankInfo = ""
			}
			outname = fmt.Sprintf("%s_%d-%d:%s%s %s", record.ID, feature.Start, feature.End, strand, flankInfo, tag)
			newRecord, err := fastx.NewRecord(record.Seq.Alphabet, []byte(outname), []byte(outname), subseq.Seq)
			checkError(err)
			outfh.Write(newRecord.Format(lineWidth))
		}
	}
}

func subSeqByBEDFile(outfh *xopen.Writer, record *fastx.Record, lineWidth int,
	bedFeatureMap map[string][]BedFeature,
	onlyFlank bool, upStream, downStream int) {
	seqname := strings.ToLower(string(record.ID))

	var strand, geneID, outname, flankInfo string
	var s, e int
	var subseq *seq.Seq

	for _, feature := range bedFeatureMap[seqname] {
		s, e = feature.Start, feature.End
		if feature.Strand != nil && *feature.Strand == "-" {
			if onlyFlank {
				if upStream > 0 {
					s = feature.End + 1
					e = feature.End + upStream
				} else {
					s = feature.Start - downStream
					e = feature.Start - 1
				}
			} else {
				s = feature.Start - downStream // seq.SubSeq will check it
				e = feature.End + upStream
			}
			if s < 1 {
				s = 1
			}
			if e > len(record.Seq.Seq) {
				e = len(record.Seq.Seq)
			}
			subseq = record.Seq.SubSeq(s, e).RevComInplace()
		} else {
			if onlyFlank {
				if upStream > 0 {
					s = feature.Start - upStream
					e = feature.Start - 1
				} else {
					s = e + 1
					e = e + downStream
				}
			} else {
				s = feature.Start - upStream
				e = feature.End + downStream
			}
			if s < 1 {
				s = 1
			}
			if e > len(record.Seq.Seq) {
				e = len(record.Seq.Seq)
			}
			subseq = record.Seq.SubSeq(s, e)
		}

		if feature.Strand == nil {
			strand = "."
		} else {
			strand = *feature.Strand
		}
		geneID = ""
		if feature.Name != nil {
			geneID = *feature.Name
		}
		if upStream > 0 {
			if onlyFlank {
				flankInfo = fmt.Sprintf("_usf:%d", upStream)
			} else if downStream > 0 {
				flankInfo = fmt.Sprintf("_us:%d_ds:%d", upStream, downStream)
			} else {
				flankInfo = fmt.Sprintf("_us:%d", upStream)
			}
		} else if downStream > 0 {
			if onlyFlank {
				flankInfo = fmt.Sprintf("_dsf:%d", downStream)
			} else if upStream > 0 {
				flankInfo = fmt.Sprintf("_us:%d_ds:%d", upStream, downStream)
			} else {
				flankInfo = fmt.Sprintf("_ds:%d", downStream)
			}
		} else {
			flankInfo = ""
		}
		outname = fmt.Sprintf("%s_%d-%d:%s%s %s", record.ID, feature.Start, feature.End, strand, flankInfo, geneID)
		newRecord, err := fastx.NewRecord(record.Seq.Alphabet, []byte(outname), []byte(outname), subseq.Seq)
		checkError(err)
		outfh.Write(newRecord.Format(lineWidth))
	}
}

func init() {
	RootCmd.AddCommand(subseqCmd)

	subseqCmd.Flags().StringSliceP("chr", "", []string{}, "select limited sequence with sequence IDs (multiple value supported, case ignored)")
	subseqCmd.Flags().StringP("region", "r", "", "by region. "+
		"e.g 1:12 for first 12 bases, -12:-1 for last 12 bases,"+
		` 13:-1 for cutting first 12 bases. type "seqkit subseq -h" for more examples`)

	subseqCmd.Flags().StringP("gtf", "", "", "by GTF (version 2.2) file")
	subseqCmd.Flags().StringSliceP("feature", "", []string{}, `select limited feature types (multiple value supported, case ignored, only works with GTF)`)
	subseqCmd.Flags().IntP("up-stream", "u", 0, "up stream length")
	subseqCmd.Flags().IntP("down-stream", "d", 0, "down stream length")
	subseqCmd.Flags().BoolP("only-flank", "f", false, "only return up/down stream sequence")
	subseqCmd.Flags().StringP("bed", "", "", "by BED file")
	subseqCmd.Flags().StringP("gtf-tag", "", "gene_id", `output this tag as sequence comment`)
}
