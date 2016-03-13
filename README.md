# fakit - Practical FASTA kit

Documents  : [http://shenwei356.github.io/fakit](http://shenwei356.github.io/fakit)

Source code: [https://github.com/shenwei356/fakit](https://github.com/shenwei356/fakit)

## Features

- Cross-platform (Linux/Windows/Mac OS X/OpenBSD/FreeBSD,
  see [download](http://shenwei356.github.io/fakit/download/))
- Out-of-the-box, no dependencies, without compilation, light weight
  see [download](http://shenwei356.github.io/fakit/download/))
- Fast (see benchmark)
- Practical functions (see subcommands and
  [examples](http://shenwei356.github.io/fakit/usage/) )
- Support STDIN and gziped input/output file, could be used in pipe
- Support custom sequence ID regular expression (especially useful for quering with ID list)
- Reproducible results (configurable rand seed in `sample` and `shuffle`)

## Subcommands

Basic

- `seq`        transform sequence (revserse, complement, extract ID...)
- `subseq`     get subsequence by region
- `stat`       simple statistics of FASTA files

Format convert

- `fa2tab`     covert FASTA to tabular format, and provide various information
- `tab2fa`     covert tabular format to FASTA format

More

- `extract`    extract sequences by pattern(s) of name or sequence motifs
- `common`     find common sequences of multiple files by id/name/sequence
- `rmdup`      remove duplicated sequences by id/name/sequence
- `split`      split sequences into files by id/seq region/size/parts
- `sample`     sample sequences by number or proportion
- `shuffle`    shuffle sequences
- `locate`     locate subsequences/motifs
- `sliding`    sliding sequences, circle genome supported

Global Flags

```
-c, --chunk-size int     chunk size (attention: unit is FASTA records not lines) (default 1000)
    --id-regexp string   regular expression for parsing ID (default "^([^\\s]+)\\s?")
-w, --line-width int     line width (0 for no wrap) (default 60)
-o, --out-file string    out file ("-" for stdout, suffix .gz for gzipped out) (default "-")
    --quiet              be quiet and do not show extra information
-t, --seq-type string    sequence type (dna|rna|protein|unlimit|auto) (for auto, it automatically detect by the first sequence) (default "auto")
-j, --threads int        number of CPUs (default value depends on your device) (default 4)
```

## Usage && Examples

[http://shenwei356.github.io/fakit/usage/](http://shenwei356.github.io/fakit/usage/)

## Benchmark

[http://blog.shenwei.me/fakit/benchmark/](http://blog.shenwei.me/fakit/benchmark/)

## Contact

Email me for any problem when using fakit. shenwei356(at)gmail.com

[Create an issue](https://github.com/shenwei356/fakit/issues) to report bugs,
propose new functions or ask for help.

## License

[MIT License](https://github.com/shenwei356/bio_scripts/blob/master/LICENSE)
