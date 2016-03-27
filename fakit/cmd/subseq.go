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
	"runtime"
	"strconv"
	"strings"

	"github.com/brentp/xopen"
	"github.com/shenwei356/bio/featio/gtf"
	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fasta"
	"github.com/shenwei356/util/byteutil"
	"github.com/spf13/cobra"
)

// subseqCmd represents the seq command
var subseqCmd = &cobra.Command{
	Use:   "subseq",
	Short: "get subsequences by region/gtf/bed, including flanking sequences",
	Long: fmt.Sprintf(`get subsequences by region/gtf/bed, including flanking sequences.

The definition of region is 1-based and with some custom design.

Examples:
%s
`, regionExample),
	Run: func(cmd *cobra.Command, args []string) {
		alphabet := getAlphabet(cmd, "seq-type")
		idRegexp := getIDRegexp(cmd, "id-regexp")
		chunkSize := getFlagPositiveInt(cmd, "chunk-size")
		threads := getFlagPositiveInt(cmd, "threads")
		lineWidth := getFlagNonNegativeInt(cmd, "line-width")
		outFile := getFlagString(cmd, "out-file")
		quiet := getFlagBool(cmd, "quiet")
		seq.AlphabetGuessSeqLenghtThreshold = getFlagalphabetGuessSeqLength(cmd, "alphabet-guess-seq-length")
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(threads)

		files := getFileList(args)

		region := getFlagString(cmd, "region")
		gtfFile := getFlagString(cmd, "gtf")
		bedFile := getFlagString(cmd, "bed")
		choosedFeature := strings.ToLower(getFlagString(cmd, "feature"))
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

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var start, end int
		type type2gtfFeatures map[string][]gtf.Feature
		var gtfFeaturesMap map[string]type2gtfFeatures
		var bedFeatureMap map[string][]BedFeature

		if region != "" {
			if !reRegion.MatchString(region) {
				checkError(fmt.Errorf(`invalid region: %s. type "fakit subseq -h" for more examples`, region))
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

			gtf.Threads = threads // threads of gtf.ReadFeatures
			features, err := gtf.ReadFeatures(gtfFile)
			checkError(err)
			for _, feature := range features {
				if _, ok := gtfFeaturesMap[feature.SeqName]; !ok {
					gtfFeaturesMap[feature.SeqName] = make(map[string][]gtf.Feature)
				}
				if _, ok := gtfFeaturesMap[feature.SeqName][feature.Feature]; !ok {
					gtfFeaturesMap[feature.SeqName][feature.Feature] = []gtf.Feature{}
				}
				gtfFeaturesMap[feature.SeqName][feature.Feature] = append(gtfFeaturesMap[feature.SeqName][feature.Feature], feature)
			}
		} else if bedFile != "" {
			if !quiet {
				log.Info("read BED file ...")
			}
			if choosedFeature != "." {
				checkError(fmt.Errorf("when given flag -b (--bed), flag -f (--feature) is not allowed"))
			}
			bedFeatureMap = make(map[string][]BedFeature)
			Threads = threads // threads of ReadBedFeatures
			features, err := ReadBedFeatures(bedFile)
			checkError(err)
			for _, feature := range features {
				if _, ok := bedFeatureMap[feature.Chr]; !ok {
					bedFeatureMap[feature.Chr] = []BedFeature{}
				}
				bedFeatureMap[feature.Chr] = append(bedFeatureMap[feature.Chr], feature)
			}
		}

		if !quiet {
			log.Info("read sequences ...")
		}
		var strand, geneID, outname, flankInfo string
		var s, e int
		var subseq *seq.Seq
		for _, file := range files {
			fastaReader, err := fasta.NewFastaReader(alphabet, file, threads, chunkSize, idRegexp)
			checkError(err)
			for chunk := range fastaReader.Ch {
				checkError(chunk.Err)

				for _, record := range chunk.Data {
					if region != "" {
						outfh.WriteString(fmt.Sprintf(">%s\n%s\n", record.Name,
							byteutil.WrapByteSlice(record.Seq.SubSeq(start, end).Seq, lineWidth)))
					} else if gtfFile != "" {
						seqname := string(record.ID)
						if _, ok := gtfFeaturesMap[seqname]; !ok {
							continue
						}

						for featureType := range gtfFeaturesMap[seqname] {
							if choosedFeature != "." && strings.ToLower(featureType) != choosedFeature {
								continue
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
									subseq = record.Seq.SubSeq(s, e).RevCom()
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
									subseq = record.Seq.SubSeq(s, e)
								}

								if feature.Strand == nil {
									strand = "."
								} else {
									strand = *feature.Strand
								}
								geneID = ""
								for _, arrtribute := range feature.Attributes {
									if arrtribute.Tag == "gene_id" {
										geneID = arrtribute.Value
										break
									}
								}
								if upStream > 0 {
									if onlyFlank {
										flankInfo = fmt.Sprintf("_usf:%d", upStream)
									} else {
										flankInfo = fmt.Sprintf("_us:%d", upStream)
									}
								} else if downStream > 0 {
									if onlyFlank {
										flankInfo = fmt.Sprintf("_dsf:%d", downStream)
									} else {
										flankInfo = fmt.Sprintf("_ds:%d", downStream)
									}
								} else {
									flankInfo = ""
								}
								outname = fmt.Sprintf("%s_%d:%d:%s%s %s", record.ID, feature.Start, feature.End, strand, flankInfo, geneID)
								outfh.WriteString(fmt.Sprintf(">%s\n%s\n", outname, byteutil.WrapByteSlice(subseq.Seq, lineWidth)))
							}

						}
					} else if bedFile != "" {
						seqname := string(record.ID)
						if _, ok := bedFeatureMap[seqname]; !ok {
							continue
						}

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
								subseq = record.Seq.SubSeq(s, e).RevCom()
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
								} else {
									flankInfo = fmt.Sprintf("_us:%d", upStream)
								}
							} else if downStream > 0 {
								if onlyFlank {
									flankInfo = fmt.Sprintf("_dsf:%d", downStream)
								} else {
									flankInfo = fmt.Sprintf("_ds:%d", downStream)
								}
							} else {
								flankInfo = ""
							}
							outname = fmt.Sprintf("%s_%d:%d:%s%s %s", record.ID, feature.Start, feature.End, strand, flankInfo, geneID)
							outfh.WriteString(fmt.Sprintf(">%s\n%s\n", outname, byteutil.WrapByteSlice(subseq.Seq, lineWidth)))
						}
					}

				}
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(subseqCmd)

	subseqCmd.Flags().StringP("region", "r", "", "by region. "+
		"e.g 1:12 for first 12 bases, -12:-1 for last 12 bases,"+
		` 13:-1 for cutting first 12 bases. type "fakit subseq -h" for more examples`)

	subseqCmd.Flags().StringP("gtf", "g", "", "by GTF (version 2.2) file")
	subseqCmd.Flags().StringP("feature", "T", ".", `feature type ("." for all, case ignored)`)
	subseqCmd.Flags().IntP("up-stream", "u", 0, "up stream length")
	subseqCmd.Flags().IntP("down-stream", "d", 0, "down stream length")
	subseqCmd.Flags().BoolP("only-flank", "f", false, "only return up/down stream sequence")

	subseqCmd.Flags().StringP("bed", "b", "", "by BED file")
}
