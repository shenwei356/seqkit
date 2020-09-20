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
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strconv"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/util/byteutil"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// translateCmd represents the head command
var translateCmd = &cobra.Command{
	Use:   "translate",
	Short: "translate DNA/RNA to protein sequence (supporting ambiguous bases)",
	Long: `translate DNA/RNA to protein sequence (supporting ambiguous bases)

Note:

  1. This command supports codons containing any ambiguous base.
     Please switch on flag -L INT for details. e.g., for standard table:

        ACN -> T
        CCN -> P
        CGN -> R
        CTN -> L
        GCN -> A
        GGN -> G
        GTN -> V
        TCN -> S
        
        MGR -> R
        YTR -> L

Translate Tables/Genetic Codes:

    # https://www.ncbi.nlm.nih.gov/Taxonomy/taxonomyhome.html/index.cgi?chapter=tgencodes

     1: The Standard Code
     2: The Vertebrate Mitochondrial Code
     3: The Yeast Mitochondrial Code
     4: The Mold, Protozoan, and Coelenterate Mitochondrial Code and the Mycoplasma/Spiroplasma Code
     5: The Invertebrate Mitochondrial Code
     6: The Ciliate, Dasycladacean and Hexamita Nuclear Code
     9: The Echinoderm and Flatworm Mitochondrial Code
    10: The Euplotid Nuclear Code
    11: The Bacterial, Archaeal and Plant Plastid Code
    12: The Alternative Yeast Nuclear Code
    13: The Ascidian Mitochondrial Code
    14: The Alternative Flatworm Mitochondrial Code
    16: Chlorophycean Mitochondrial Code
    21: Trematode Mitochondrial Code
    22: Scenedesmus obliquus Mitochondrial Code
    23: Thraustochytrium Mitochondrial Code
    24: Pterobranchia Mitochondrial Code
    25: Candidate Division SR1 and Gracilibacteria Code
    26: Pachysolen tannophilus Nuclear Code
    27: Karyorelict Nuclear
    28: Condylostoma Nuclear
    29: Mesodinium Nuclear
    30: Peritrich Nuclear
    31: Blastocrithidia Nuclear

`,
	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		alphabet := config.Alphabet
		idRegexp := config.IDRegexp
		outFile := config.OutFile
		seq.AlphabetGuessSeqLengthThreshold = config.AlphabetGuessSeqLength
		seq.ValidateSeq = false
		runtime.GOMAXPROCS(config.Threads)

		translTable := getFlagPositiveInt(cmd, "transl-table")
		if _, ok := seq.CodonTables[translTable]; !ok {
			checkError(fmt.Errorf("invalid translate table: %d", translTable))
		}
		_frames := getFlagStringSlice(cmd, "frame")
		frames := make([]int, 0, len(_frames))
		for _, _frame := range _frames {
			frame, err := strconv.Atoi(_frame)
			if err != nil {
				checkError(fmt.Errorf("invalid frame(s): %s. available: 1, 2, 3, -1, -2, -3, and 6 for all. multiple frames should be separated by comma", _frame))
			}
			if !(frame == 1 || frame == 2 || frame == 3 || frame == -1 || frame == -2 || frame == -3 || frame == 6) {
				checkError(fmt.Errorf("invalid frame: %d. available: 1, 2, 3, -1, -2, -3, and 6 for all", frame))
			}
			if frame == 6 {
				frames = []int{1, 2, 3, -1, -2, -3}
				break
			}
			frames = append(frames, frame)
		}
		trim := getFlagBool(cmd, "trim")
		clean := getFlagBool(cmd, "clean")
		allowUnknownCodon := getFlagBool(cmd, "allow-unknown-codon")
		markInitCodonAsM := getFlagBool(cmd, "init-codon-as-M")
		listTable := getFlagInt(cmd, "list-transl-table")
		listTableAmb := getFlagInt(cmd, "list-transl-table-with-amb-codons")
		appendFrame := getFlagBool(cmd, "append-frame")

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		if listTableAmb == 0 || listTable == 0 {
			ks := make([]int, len(seq.CodonTables))
			i := 0
			for k := range seq.CodonTables {
				ks[i] = k
				i++
			}
			sort.Ints(ks)
			for _, i = range ks {
				outfh.WriteString(fmt.Sprintf("%d\t%s\n", seq.CodonTables[i].ID, seq.CodonTables[i].Name))
			}
			return
		} else if listTableAmb > 0 {
			if table, ok := seq.CodonTables[listTableAmb]; ok {
				outfh.WriteString(table.StringWithAmbiguousCodons())
			}
			return
		} else if listTable > 0 {
			if table, ok := seq.CodonTables[listTable]; ok {
				outfh.WriteString(table.String())
			}
			return
		}

		files := getFileListFromArgsAndFile(cmd, args, true, "infile-list", true)

		var record *fastx.Record
		var fastxReader *fastx.Reader
		var _seq *seq.Seq
		var frame int
		once := true
		for _, file := range files {
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

				if once {
					if !(record.Seq.Alphabet == seq.DNA || record.Seq.Alphabet == seq.DNAredundant ||
						record.Seq.Alphabet == seq.RNA || record.Seq.Alphabet == seq.RNAredundant) {
						checkError(fmt.Errorf(`command 'seqkit translate' only apply to DNA/RNA sequences`))
					}
					once = false
				}

				for _, frame = range frames {
					_seq, err = record.Seq.Translate(translTable, frame, trim, clean, allowUnknownCodon, markInitCodonAsM)
					if err != nil {
						if err == seq.ErrUnknownCodon {
							log.Error("unknown codon detected, you can use flag -x/--allow-unknown-codon to translate it to 'X'.")
							os.Exit(-1)
						}
						checkError(err)
					}
					checkError(err)

					if appendFrame {
						outfh.WriteString(fmt.Sprintf(">%s_frame=%d %s\n", record.ID, frame, record.Desc))
					} else {
						outfh.WriteString(string(record.Name) + "\n")
					}
					outfh.Write(byteutil.WrapByteSlice(_seq.Seq, config.LineWidth))
					outfh.WriteString("\n")
				}

			}

		}
	},
}

func init() {
	RootCmd.AddCommand(translateCmd)
	translateCmd.Flags().IntP("transl-table", "T", 1, `translate table/genetic code, type 'seqkit translate --help' for more details`)
	translateCmd.Flags().StringSliceP("frame", "f", []string{"1"}, "frame(s) to translate, available value: 1, 2, 3, -1, -2, -3, and 6 for all six frames")
	translateCmd.Flags().BoolP("trim", "", false, "remove all 'X' and '*' characters from the right end of the translation")
	translateCmd.Flags().BoolP("clean", "", false, "change all STOP codon positions from the '*' character to 'X' (an unknown residue)")
	translateCmd.Flags().BoolP("allow-unknown-codon", "x", false, "translate unknown code to 'X'. And you may not use flag --trim which removes 'X'")
	translateCmd.Flags().BoolP("init-codon-as-M", "M", false, "translate initial codon at beginning to 'M'")
	translateCmd.Flags().IntP("list-transl-table", "l", -1, "show details of translate table N, 0 for all")
	translateCmd.Flags().IntP("list-transl-table-with-amb-codons", "L", -1, "show details of translate table N (including ambigugous codons), 0 for all. ")
	translateCmd.Flags().BoolP("append-frame", "F", false, "append frame infomation to sequence ID")
}
