## fakit

FASTA kit

Documents  : http://shenwei356.github.io/fakit

Source code: https://github.com/shenwei356/fakit

## Features

- Cross-platform (Linux/Windows/Mac OS X/OpenBSD/FreeBSD)
- Lite weight, out-of-the-box, no dependencies, without compilation
- Fast (see benchmark)
- Practical functions (see subcommands)
- Reproducible results (sample and shuffle)
- Support STDIN and gziped input/output file, could be used in pipe

## Subcommands

Basic

- `seq`        transform sequence (revserse, complement, extract ID...)

Format convert

- `fa2tab`     covert FASTA to tabular format, and provide various information
- `tab2fa`     covert tabular format to FASTA format

More

- `extract`    extract sequences by patterns/motifs
- `common`     find common sequences of multiple files
- `rmdup`      remove duplicated sequences
- `split`      split sequences into files by id/seq region/size/parts
- `sample`     sample sequences
- `shuffle`    shuffle sequences
- `locate`     locate sub-sequences/motifs
- `sliding`    sliding sequences

TODO

- `faidx`      create fasta index file
- `subseq`     extract sub-sequences by region/bed/gff

## Benchmarks

## Examples

## Usage

[detailed command line usage](http://shenwei356.github.io/fakit)

## License

[MIT License](https://github.com/shenwei356/bio_scripts/blob/master/LICENSE)

## Similar tools

- [fastx_toolkit](http://hannonlab.cshl.edu/fastx_toolkit/)
- [seqtk](https://github.com/lh3/seqtk)
- [pyfaidx](https://github.com/mdshw5/pyfaidx)
- [fqtools](https://github.com/alastair-droop/fqtools)
- [fasta_utilities](https://github.com/jimhester/fasta_utilities)
