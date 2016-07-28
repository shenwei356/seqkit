package main

import (
	"fmt"
	"io"
	"os"

	"github.com/biogo/biogo/alphabet"
	"github.com/biogo/biogo/io/seqio/fasta"
	"github.com/biogo/biogo/seq/linear"
)

func main() {
	if len(os.Args) != 2 {
		checkError(fmt.Errorf("feed me one uncompressed FASTA file"))
	}

	file := os.Args[1]

	fh, err := os.Open(file)
	checkError(err)

	reader := fasta.NewReader(fh, linear.NewSeq("", nil, alphabet.DNA))
	for {
		if s, err := reader.Read(); err != nil {
			if err == io.EOF {
				break
			} else {
				checkError(fmt.Errorf("Failed to read %q: %s", file, err))
			}

		} else {
			t := s.(*linear.Seq)
			t.RevComp()

			fmt.Printf(">%s %s\n%v\n", t.Name(), t.Description(), t.Seq)
		}
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
