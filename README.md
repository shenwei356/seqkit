# fakit - a cross-platform and efficient suit for FASTA file manipulation

Documents  : [http://shenwei356.github.io/fakit](http://shenwei356.github.io/fakit)

Source code: [https://github.com/shenwei356/fakit](https://github.com/shenwei356/fakit)

## Introduction

FASTA is a basic format for storing nucleotide and protein sequences.
The manipulation of FASTA file includes converting, clipping, searching, filtering,
deduplication, splitting, shuffling, sampling and so on.
Existed tools only implemented parts of the functions,
and some of them are only available for specific operating systems.
Furthermore, the complicated installation process of dependencies packages and
running environment also make them less friendly to common users.

Fakit is a cross-platform, efficient, and practical FASTA manipulations tool
that is friendly for researchers to complete wide ranges of FASTA file processing.
The suite supports plain or gzip-compressed input and output
from either standard stream or files, therefore, it could be easily used in pipelines.

## Features

- **Cross-platform** (Linux/Windows/Mac OS X/OpenBSD/FreeBSD,
  see [download](http://shenwei356.github.io/fakit/download/))
- **Light weight and out-of-the-box, no dependencies, no compilation, no configuration**
  (see [download](http://shenwei356.github.io/fakit/download/))
- **Fast** (see [benchmark](/#performance-comparison-with-other-tools)),
  **multiple-CPUs supported** (see [benchmark](/#speedup-with-multi-threads)).
- **Practical functions supported by 14 subcommands** (see subcommands and
  [usage](http://shenwei356.github.io/fakit/usage/) )
- **Well documented** (detailed [usage](http://shenwei356.github.io/fakit/usage/)
  and [benchmark](http://shenwei356.github.io/fakit/benchmark/) )
- **Support STDIN and gziped input/output file, easy being used in pipe**
- **Support custom sequence ID regular expression** (especially useful for quering with ID list)
- Reproducible results (configurable rand seed in `sample` and `shuffle`)
- Well organized source code, friendly to use and easy to extend.

**Features comparison**

Features         | fakit    | fasta_utilities | fastx_toolkit | pyfaidx | seqmagick | seqtk
:--------------- | :------: | :-------------: | :-----------: | :-----: | :-------: | :----
Cross-platform   | Yes      | Partly          | Partly        | Yes     | Yes       | Yes
Mutli-line FASTA | Yes      | Yes             | --            | Yes     | Yes       | Yes
Validate bases   | Yes      | --              | Yes           | Yes     | --        | --
Recognize RNA    | Yes      | Yes             | --            | --      | Yes       | Yes
Read STDIN       | Yes      | Yes             | Yes           | --      | Yes       | Yes
Read gzip        | Yes      | Yes             | --            | --      | Yes       | Yes
Write gzip       | Yes      | --              | --            | --      | Yes       | --
Search by motifs | Yes      | Yes             | --            | --      | Yes       | Yes
Sample seqs      | Yes      | Yes             | --            | --      | Yes       | Yes
Subseq           | Yes      | Yes             | --            | Yes     | Yes       | Yes
Deduplicate seqs | Yes      | --              | --            | --      | Partly    | --
Split seqs       | Yes      | Yes             | --            | Partly  | --        | --
Split by seq     | Yes      | --              | Yes           | Yes     | --        | --
Shuffle seqs     | Yes      | --              | --            | --      | --        | --
Sort seqs        | Yes      | Yes             | --            | --      | Yes       | --
Locate motifs    | Yes      | --              | --            | --      | --        | --
Common seqs      | Yes      | --              | --            | --      | --        | --
Clean bases      | Yes      | Yes             | Yes           | Yes     | --        | --
Transcribe       | Yes      | Yes             | Yes           | Yes     | Yes       | Yes
Translate        | --       | Yes             | Yes           | Yes     | Yes       | --
Size select      | Indirect | Yes             | --            | Yes     | Yes       | --
Rename head      | Yes      | Yes             | --            | --      | Yes       | Yes

# Download

`fakit` is implemented in [Golang](https://golang.org/) programming language,
 executable binary files **for most popular operating system** are freely available
  in [release](https://github.com/shenwei356/fakit/releases) page.

Just [download](https://github.com/shenwei356/fakit/releases) executable file
 of your operating system and rename it to `fakit.exe` (Windows) or
 `fakit` (other operating systems) for convenience,
 and then run it in command-line interface, no dependencies,
 no complicated compilation process.


## Subcommands

**Sequence and subsequence**

- `seq`        transform sequences (revserse, complement, extract ID...)
- `subseq`     get subsequences by region/gtf/bed, including flanking sequences
- `sliding`    sliding sequences, circle genome supported
- `stat`       simple statistics of FASTA files

**Format conversion**

- `fa2tab`     covert FASTA to tabular format (and length/GC content/GC skew) to filter and sort
- `tab2fa`     covert tabular format to FASTA format

**Searching**

- `grep`       search sequences by pattern(s) of name or sequence motifs
- `locate`     locate subsequences/motifs

**Set operations**

- `rmdup`      remove duplicated sequences by id/name/sequence
- `common`     find common sequences of multiple files by id/name/sequence
- `split`      split sequences into files by id/seq region/size/parts
- `sample`     sample sequences by number or proportion

**Edit**

- `replace`    replace name/sequence/by regular expression

**Ordering**

- `shuffle`    shuffle sequences
- `sort`       sort sequences by id/name/sequence


**Global Flags**

```
      --alphabet-guess-seq-length int   length of sequence prefix of the first FASTA record based on which fakit guesses the sequence type (default 10000)
  -c, --chunk-size int                  chunk size (attention: unit is FASTA records not lines) (default 1000)
      --id-ncbi                         FASTA head is NCBI-style, e.g. >gi|110645304|ref|NC_002516.2| Pseud...
      --id-regexp string                regular expression for parsing ID (default "^([^\\s]+)\\s?")
  -w, --line-width int                  line width (0 for no wrap) (default 60)
  -o, --out-file string                 out file ("-" for stdout, suffix .gz for gzipped out) (default "-")
      --quiet                           be quiet and do not show extra information
  -t, --seq-type string                 sequence type (dna|rna|protein|unlimit|auto) (for auto, it automatically detect by the first sequence) (default "auto")
  -j, --threads int                     number of CPUs. (default value depends on your device) (default 4)

```

## Usage && Examples

[Usage and examples](http://shenwei356.github.io/fakit/usage/)

[Tutorial](http://shenwei356.github.io/fakit/tutorial/)

## Benchmark

Details: [http://shenwei356.github.io/fakit/benchmark/](http://shenwei356.github.io/fakit/benchmark/)

All tests were repeated 4 times.

### Performance comparison with other tools

Missing data indicates that the tool does not have the function.

Result also shows that **the self-implemented FASTA parsing module has better performance than
the [biogo](https://github.com/biogo/biogo)**, a bioinformatics library for Go.

For the revese complementary sequence test,
the `fasta_utilities`, `seqmagick` and `seqtk` do not validate the bases/residues, which save some times.

![benchmark_colorful.png](benchmark/benchmark_colorful.png)

### Acceleration with multi-CPUs

![benchmark_colorful.png](benchmark/fakit_multi_threads/benchmark_colorful.png)

## Contact

Email me for any problem when using fakit. shenwei356(at)gmail.com

[Create an issue](https://github.com/shenwei356/fakit/issues) to report bugs,
propose new functions or ask for help.

## License

[MIT License](https://github.com/shenwei356/fakit/blob/master/LICENSE)
