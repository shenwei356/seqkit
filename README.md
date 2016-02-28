# fakit

FASTA kit

## Plain

1. github.com/shenwei356/bio

    Improvments

        adding attribute for FastaRecord: Head
        adding support for reading from stdin, '-' for stdin
        rewrite fasta reader, change api and the error handles, adding cancellation, ref:breader

2. commands except subseq and faidx

3. github.com/shenwei356/bio

    New features

        create github.com/shenwei356/bio/featio/bed
        create github.com/shenwei356/bio/featio/gff

4. finishing commands: subseq and faidx

## Framework

    No API, just CLI

## Subcommands

    seq         revserse, complement, --no-names, --full-name
    faidx       fasta index
    subseq      sub seqs, --bed,
    stat        counts, length,

    split       split into one-seq file
    extract     extract seqs by names or seqs/motifs
    sample      sampling seqs
    common      find common seqs by names or seqs
    locate      locate seq/motif by names or seqs
    rmdup       remove duplicated sequence by names or seqs
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
