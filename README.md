# fakit - Swiss army knife of FASTA format

Documents  : [http://shenwei356.github.io/fakit](http://shenwei356.github.io/fakit)

Source code: [https://github.com/shenwei356/fakit](https://github.com/shenwei356/fakit)

## Introduction



## Features

- **Cross-platform** (Linux/Windows/Mac OS X/OpenBSD/FreeBSD,
  see [download](http://shenwei356.github.io/fakit/download/))
- **Out-of-the-box, no dependencies, without compilation, light weight**
  (see [download](http://shenwei356.github.io/fakit/download/))
- **Fast** (see [benchmark](http://shenwei356.github.io/fakit/benchmark/)),
  **multiple-threads supported**, more significant speedup for `grep` and `locate`.
- **Practical functions by 14 subcommands** (see subcommands and
  [usage](http://shenwei356.github.io/fakit/usage/) )
- **Well documented** (detailed [usage](http://shenwei356.github.io/fakit/usage/)
  and [benchmark](http://shenwei356.github.io/fakit/benchmark/) )
- **Support STDIN and gziped input/output file, could be used in pipe**
- Support custom sequence ID regular expression (especially useful for quering with ID list)
- Reproducible results (configurable rand seed in `sample` and `shuffle`)
- Well organized source code, friendly to use and easy to extend.

## Subcommands

Sequence and subsequence

- `seq`        transform sequences (revserse, complement, extract ID...)
- `subseq`     get subsequences by region/gtf/bed, including flanking sequences
- `sliding`    sliding sequences, circle genome supported
- `stat`       simple statistics of FASTA files

Format conversion

- `fa2tab`     covert FASTA to tabular format (and length/GC content/GC skew) to filter and sort
- `tab2fa`     covert tabular format to FASTA format

Searching

- `grep`       search sequences by pattern(s) of name or sequence motifs
- `locate`     locate subsequences/motifs

Set operations

- `rmdup`      remove duplicated sequences by id/name/sequence
- `common`     find common sequences of multiple files by id/name/sequence
- `split`      split sequences into files by id/seq region/size/parts
- `sample`     sample sequences by number or proportion

Ordering

- `shuffle`    shuffle sequences
- `sort`       sort sequences by id/name/sequence


Global Flags

```
    --alphabet-guess-seq-length int   length of sequence prefix of the first FASTA record based on which fakit guess the sequence type (default 10000)
-c, --chunk-size int                  chunk size (attention: unit is FASTA records not lines) (default 1000)
    --id-regexp string                regular expression for parsing ID (default "^([^\\s]+)\\s?")
-w, --line-width int                  line width (0 for no wrap) (default 60)
-o, --out-file string                 out file ("-" for stdout, suffix .gz for gzipped out) (default "-")
    --quiet                           be quiet and do not show extra information
-t, --seq-type string                 sequence type (dna|rna|protein|unlimit|auto) (for auto, it automatically detect by the first sequence) (default "auto")
-j, --threads int                     number of CPUs. (default value depends on your device) (default 4)
```

## Usage && Examples

[http://shenwei356.github.io/fakit/usage/](http://shenwei356.github.io/fakit/usage/)

## Benchmark

[http://shenwei356.github.io/fakit/benchmark/](http://blog.shenwei.me/fakit/benchmark/)

## Contact

Email me for any problem when using fakit. shenwei356(at)gmail.com

[Create an issue](https://github.com/shenwei356/fakit/issues) to report bugs,
propose new functions or ask for help.

## License

[MIT License](https://github.com/shenwei356/bio_scripts/blob/master/LICENSE)
