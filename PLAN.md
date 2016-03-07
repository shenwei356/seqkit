# fakit

FASTA kit

## Plan

1. github.com/shenwei356/bio

    - adding attribute for FastaRecord: `Head`

    - adding support for reading from stdin, '-' for stdin

    - rewrite fasta reader (ref:breader)

        - chunked, adding attribute `ID` to keep the output order
        - change api and the error handles,
        - adding cancellation

    - fasta reader data flow:

        - read line by line, returns chunk of lines, but not create `Seq` object
        - parallelly create `Seq` object and fan in a chanel


2. commands except subseq and faidx

    data flow:

    - parallelly process fasta records and fan in

3. github.com/shenwei356/bio

    New features

        create github.com/shenwei356/bio/featio/bed
        create github.com/shenwei356/bio/featio/gff

4. finishing commands: subseq and faidx

5. create web pages

6. write paper

## Framework

    No API, just CLI

## Subcommands

    [x] seq         revserse, complement, --no-names, --full-name
    faidx       fasta index
    subseq      sub seqs, --bed,
    stat        counts, length,

    split       split into one-seq file
    [x] extract     extract seqs by names or seqs/motifs
    sample      sampling seqs
    [x] common      find common seqs by names or seqs
    [x] locate      locate subseq/motif in seqs
    [x] rmdup       remove duplicated sequence by names or seqs
    sort        sort fasta records
    fa2tab      covert to tabular format, --length, --base-content
    tab2fa      covert from tabular format

    sliding     sliding window
    clean       remove gaps

global option

    --gz
    -b --buffer-size
    --mask-by-case
    - from stdin

## Features

- Cross-platform, lite, no dependencies, no compilation
- Fast
- Automatically recognize compressed input file

## Similar tools

- [fastx_toolkit](http://hannonlab.cshl.edu/fastx_toolkit/)
- [pyfaidx](https://github.com/mdshw5/pyfaidx)
- [seqtk](https://github.com/lh3/seqtk)
- [fqtools](https://github.com/alastair-droop/fqtools)
- [fasta_utilities](https://github.com/jimhester/fasta_utilities)
