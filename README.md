# fakit - a cross-platform and efficient suit for FASTA/Q file manipulation

Documents  : [http://shenwei356.github.io/fakit](http://shenwei356.github.io/fakit)

Source code: [https://github.com/shenwei356/fakit](https://github.com/shenwei356/fakit)

## About the name

Origionally, `fakit` (abbreviation of `FASTA kit`) was designed to handle FASTA
format. And the name was remained after adding seamless support for FASTQ fromat.

## Introduction

FASTA and FASTQ are basic formats for storing nucleotide and protein sequences.
The manipulation of FASTA/Q file includes converting, clipping, searching,
filtering, deduplication, splitting, shuffling, sampling and so on.
Existed tools only implemented parts of the functions,
and some of them are only available for specific operating systems.
Furthermore, the complicated installation process of dependencies packages and
running environment also make them less friendly to common users.

fakit is a cross-platform, efficient, and practical FASTA/Q manipulations tool
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
- **Practical functions supported by 18 subcommands** (see subcommands and
  [usage](http://shenwei356.github.io/fakit/usage/) )
- **Well documented** (detailed [usage](http://shenwei356.github.io/fakit/usage/)
  and [benchmark](http://shenwei356.github.io/fakit/benchmark/) )
- **Seamlessly parses both FASTA and FASTQ formats**
- **Support STDIN and gziped input/output file, easy being used in pipe**
- **Support custom sequence ID regular expression** (especially useful for quering with ID list)
- Reproducible results (configurable rand seed in `sample` and `shuffle`)
- Well organized source code, friendly to use and easy to extend.

**Features comparison**

Features         | fakit    | fasta_utilities | fastx_toolkit | pyfaidx | seqmagick | seqtk
:--------------- | :------: | :-------------: | :-----------: | :-----: | :-------: | :----
Cross-platform   | Yes      | Partly          | Partly        | Yes     | Yes       | Yes
Mutli-line FASTA | Yes      | Yes             | --            | Yes     | Yes       | Yes
Read FASTQ       | Yes      | Yes             | Yes           | --      | Yes       | Yes
Mutli-line FASTQ | Yes      | Yes             | --            | --      | Yes       | Yes
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

# Installation

`fakit` is implemented in [Golang](https://golang.org/) programming language,
 executable binary files **for most popular operating system** are freely available
  in [release](https://github.com/shenwei356/fakit/releases) page.

Just [download](https://github.com/shenwei356/fakit/releases) gzip-compressed
executable file of your operating system, and uncompress it with `gzip -d *.gz` command,
rename it to `fakit.exe` (Windows) or `fakit` (other operating systems) for convenience.

You may need to add executable permision by `chmod a+x fakit`.

You can also add the directory of the executable file to environment variable
`PATH`, so you can run `fakit` anywhere.

1. For windows, the simplest way is copy it to `C:\WINDOWS\system32`.

2. For Linux, type:

      chmod a+x /PATH/OF/FASTCOV/fakit
      echo export PATH=\$PATH:/PATH/OF/FASTCOV >> ~/.bashrc

  or simply copy it to `/usr/local/bin`


## Subcommands (18 in total)

**Sequence and subsequence**

- `seq`        transform sequences (revserse, complement, extract ID...)
- `subseq`     get subsequences by region/gtf/bed, including flanking sequences
- `sliding`    sliding sequences, circle genome supported
- `stat`       simple statistics of FASTA files
- `faidx`      create FASTA index file

**Format conversion**

- `fx2tab`     covert FASTA/Q to tabular format (and length/GC content/GC skew) to filter and sort
- `tab2fx`     covert tabular format to FASTA/Q format
- `fq2fa`      covert FASTQ to FASTA

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
- `rename`     rename duplicated IDs

**Ordering**

- `shuffle`    shuffle sequences
- `sort`       sort sequences by id/name/sequence


**Global Flags**

```
      --alphabet-guess-seq-length int   length of sequence prefix of the first FASTA record based on which fakit guesses the sequence type (default 10000)
  -b, --buffer-size int                 buffer size of chunks (default value is the CPUs number of your computer) (default 4)
  -c, --chunk-size int                  chunk size (attention: unit is FASTA records not lines) (default 1000)
      --id-ncbi                         FASTA head is NCBI-style, e.g. >gi|110645304|ref|NC_002516.2| Pseud...
      --id-regexp string                regular expression for parsing ID (default "^([^\\s]+)\\s?")
  -w, --line-width int                  line width when outputing FASTA format (0 for no wrap) (default 60)
  -o, --out-file string                 out file ("-" for stdout, suffix .gz for gzipped out) (default "-")
      --quiet                           be quiet and do not show extra information
  -t, --seq-type string                 sequence type (dna|rna|protein|unlimit|auto) (for auto, it automatically detect by the first sequence) (default "auto")
  -j, --threads int                     number of CPUs. (default value is the CPUs number of your computer) (default 4)

```

## Technical details and guides for use

### Reading FASTA/Q

fakit use author's bioinformatics packages [bio](https://github.com/shenwei356/bio)
for FASTA/Q parsing, which **asynchronously parse FASTA/Q records and buffer them
in chunks**. The parser return one chunk of records for each call.

**Asynchronous parsing** saves much time because these's no waiting interval for
parsed records being handled.
The strategy of **records chunks** reduces data exchange in parallelly handling
of sequences, which could also improve performance.

Since using of buffers and chunks, the memory occupation will be higher than
cases of reading sequence one by one.
The default value of chunk size (configurable by global flag `-c` or `--chunk-size`)
is 1, which is suitable for most of cases. 
But for manipulating short sequences, e.g. FASTQ or FASTA of short sequences,
you could set higher value, e.g. 1000.
For big genomes like human genome, smaller chunk size is prefered, e.g. 1.
And the buffer size is configurable by  global flag `-b` or `--buffer-size`
(default value is 1), therefore, you may set with higher
value for short sequences to imporve performance.

***In summary, set smaller value for `-c` and `-b` when handling big FASTA file
like human genomes.***

### FASTA index

For some commands, e.g. `subseq`, `sort` and `shuffle`,
when input files are plain FASTA files, FASTA index file .fai will be created for
rapid acccess of sequences and reducing memory occupation.

### Parallelization of CPU intensive jobs

Most of the manipulations of FASTA/Q files are I/O intensive, to improve the
performance, asynchronous parsing strategy is used.

For CPU intensive jobs like `grep` with regular expressions and `locate` with
sequence motifs. The processes are parallelized
with MapReduce model by multiple goroutine in golang, similar to but much
lighter weight than threads. The concurrency number is configurable with global
flag `-j` or `--threads`.

Most of the time you can just use the default value. i.e. the number of CPUs
of your computer.

### Memory occupation

Most of the subcommands do not read whole FASTA/Q records in to memory,
including `stat`, `fq2fa`, `fx2tab`, `tab2fx`, `grep`, `locate`, `replace`,
 `seq`, `sliding`, `subseq`. They just temporarily buffer chunks of records.
 
Note that when using `subseq --gtf | --bed`, if the GTF/BED files are too
big, the memory usage will increase. 
You could use `--chr` to specify chromesomes and `--feature` to limit features.

Some subcommands need to store sequences or heads in memory, but there are
strategy to reduce memory occupation, including `rmdup` and `common`.
When comparing with sequences, MD5 digest could be used to replace sequence by
flag `-m` (`--md5`).

Some subcommands could either read all records or read the files twice by flag
`-2` (`--two-pass`), including `sample` and `split`.

Two subcommands must read all records in memory right now, `shuffle` and `sort`.
But I'll improve this later.

### Reproducibility

Subcommands `sample` and `shuffle` use random function, random seed could be
given by flag `-s` (`--rand-seed`). This make sure that sample result could be
reproduced in different environments with same random seed.

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
