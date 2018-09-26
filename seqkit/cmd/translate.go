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
	"runtime"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

// translateCmd represents the head command
var translateCmd = &cobra.Command{
	Use:   "translate",
	Short: "translate DNA/RNA to protein sequence",
	Long: `translate DNA/RNA to protein sequence

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
		frame := getFlagInt(cmd, "frame")
		if !(frame == 1 || frame == 2 || frame == 3 || frame == -1 || frame == -2 || frame == -3) {
			checkError(fmt.Errorf("invalid frame: %d. available: 1, 2, 3, -1, -2, -3", frame))
		}
		trim := getFlagBool(cmd, "trim")
		clean := getFlagBool(cmd, "clean")

		files := getFileList(args)

		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()

		var record *fastx.Record
		var fastxReader *fastx.Reader
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

				record.Seq, err = record.Seq.Translate(translTable, frame, trim, clean)
				checkError(err)

				record.FormatToWriter(outfh, config.LineWidth)

			}

		}
	},
}

func init() {
	RootCmd.AddCommand(translateCmd)
	translateCmd.Flags().IntP("transl-table", "T", 1, `translate table/genetic code, type 'seqkit translate --help' for more details`)
	translateCmd.Flags().IntP("frame", "", 1, "frame to translate, available value: 1, 2, 3, -1, -2, -3")
	translateCmd.Flags().BoolP("trim", "", false, "removes all 'X' and '*' characters from the right end of the translation")
	translateCmd.Flags().BoolP("clean", "", false, "changes all STOP codon positions from the '*' character to 'X' (an unknown residue)")
}
