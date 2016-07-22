# seqkit - a cross-platform and efficient toolkit for FASTA/Q file manipulation


Documents: [http://shenwei356.github.io/seqkit](http://shenwei356.github.io/seqkit)

Source code: [https://github.com/shenwei356/seqkit](https://github.com/shenwei356/seqkit)

Latest version: [![Latest Version](https://img.shields.io/github/release/shenwei356/seqkit.svg?style=flat)](https://github.com/shenwei356/seqkit/releases)


## Introduction

FASTA and FASTQ are basic formats for storing nucleotide and protein sequences.
The manipulations of FASTA/Q file include converting, clipping, searching,
filtering, deduplication, splitting, shuffling, sampling and so on.
Existed tools only implemented parts of the functions,
and some of them are only available for specific operating systems.
Furthermore, the complicated installation process of dependencies packages and
running environment also make them less friendly to common users.

seqkit is a cross-platform, efficient, and practical FASTA/Q manipulations tool
that is friendly for researchers to complete wide ranges of FASTA file processing.
The toolkit supports plain or gzip-compressed input and output
from either standard stream or files,
therefore, it could be easily used in command-line pipe.

## Features

- **Cross-platform** (Linux/Windows/Mac OS X/OpenBSD/FreeBSD,
  see [download](http://shenwei356.github.io/seqkit/download/))
- **Light weight and out-of-the-box, no dependencies, no compilation, no configuration**
  (see [download](http://shenwei356.github.io/seqkit/download/))
- **Very Fast** (see [benchmark](/#benchmark)),
  **multiple-CPUs supported**.
- **Practical functions supported by 20 subcommands** (see subcommands and
  [usage](http://shenwei356.github.io/seqkit/usage/) )
- **Well documented** (detailed [usage](http://shenwei356.github.io/seqkit/usage/)
  and [benchmark](http://shenwei356.github.io/seqkit/benchmark/) )
- **Seamlessly parses both FASTA and FASTQ formats**
- **Support STDIN and gziped input/output file, easy being used in pipe**
- **Support custom sequence ID regular expression** (especially useful for searching with ID list)
- Reproducible results (configurable rand seed in `sample` and `shuffle`)
- Well organized source code, friendly to use and easy to extend.

### Features comparison

Categories          |Features               |seqkit  |fasta_utilities|fastx_toolkit|pyfaidx|seqmagick|seqtk
:-------------------|:----------------------|:------:|:-------------:|:-----------:|:-----:|:-------:|:---:
**Formats supports**|Multi-line FASTA       |Yes     |Yes            |--           |Yes    |Yes      |Yes
                    |FASTQ                  |Yes     |Yes            |Yes          |--     |Yes      |Yes
                    |Multi-line  FASTQ      |Yes     |Yes            |--           |--     |Yes      |Yes
                    |Validating sequences   |Yes     |--             |Yes          |Yes    |--       |--
                    |Supporting RNA         |Yes     |Yes            |--           |--     |Yes      |Yes
**Functions**       |Searching by motifs    |Yes     |Yes            |--           |--     |Yes      |--
                    |Sampling               |Yes     |--             |--           |--     |Yes      |Yes
                    |Extracting sub-sequence|Yes     |Yes            |--           |Yes    |Yes      |Yes
                    |Removing duplicates    |Yes     |--             |--           |--     |Partly   |--
                    |Splitting              |Yes     |Yes            |--           |Partly |--       |--
                    |Splitting by seq       |Yes     |--             |Yes          |Yes    |--       |--
                    |Shuffling              |Yes     |--             |--           |--     |--       |--
                    |Sorting                |Yes     |Yes            |--           |--     |Yes      |--
                    |Locating motifs        |Yes     |--             |--           |--     |--       |--
                    |Common sequences       |Yes     |--             |--           |--     |--       |--
                    |Cleaning bases         |Yes     |Yes            |Yes          |Yes    |--       |--
                    |Transcription          |Yes     |Yes            |Yes          |Yes    |Yes      |Yes
                    |Translation            |--      |Yes            |Yes          |Yes    |Yes      |--
                    |Filtering by size      |Indirect|Yes            |--           |Yes    |Yes      |--
                    |Renaming header        |Yes     |Yes            |--           |--     |Yes      |Yes 
**Other features**  |Cross-platform         |Yes     |Partly         |Partly       |Yes    |Yes      |Yes
                    |Reading STDIN          |Yes     |Yes            |Yes          |--     |Yes      |Yes
                    |Reading gzipped file   |Yes     |Yes            |--           |--     |Yes      |Yes
                    |Writing gzip file      |Yes     |--             |--           |--     |Yes      |--

**Note 1**: See [version information](http://127.0.0.1:8000/benchmark/#softwares) of the softwares.

**Note 2**: See [usage](http://shenwei356.github.io/seqkit/usage/) for detailed options of seqkit. 

## Installation

Go to [Download Page](http://shenwei356.github.io/seqkit/downlaod) for more download options and changelogs.

`seqkit` is implemented in [Golang](https://golang.org/) programming language,
 executable binary files **for most popular operating systems** are freely available
  in [release](https://github.com/shenwei356/seqkit/releases) page.

Just [download](https://github.com/shenwei356/seqkit/releases) compressed
executable file of your operating system, and uncompress it with `tar -zxvf *.tar.gz` command.

You can add the directory of the executable file to environment variable
`PATH`, so you can run `seqkit` anywhere.


1. For windows, the simplest way is copy it to `C:\WINDOWS\system32`.

2. For Linux, type:

        chmod a+x /PATH/OF/FAKIT/seqkit
        echo export PATH=\$PATH:/PATH/OF/seqkit >> ~/.bashrc

    or simply copy it to `/usr/local/bin`

For Go developer, just one command:

    go get -u github.com/shenwei356/seqkit/seqkit

## Subcommands

20 subcommands in total.

**Sequence and subsequence**

- `seq`        transform sequences (revserse, complement, extract ID...)
- `subseq`     get subsequences by region/gtf/bed, including flanking sequences
- `sliding`    sliding sequences, circular genome supported
- `stat`       simple statistics of FASTA files
- `faidx`      create FASTA index file

**Format conversion**

- `fx2tab`     covert FASTA/Q to tabular format (and length/GC content/GC skew)
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
- `head`       print first N FASTA/Q records

**Edit**

- `replace`    replace name/sequence by regular expression
- `rename`     rename duplicated IDs

**Ordering**

- `shuffle`    shuffle sequences
- `sort`       sort sequences by id/name/sequence

**Misc**

- `version`   print version information and check for update


## Technical details and guides for use

### FASTA/Q format parsing

seqkit uses author's lightweight and high-performance bioinformatics packages
[bio](https://github.com/shenwei356/bio) for FASTA/Q parsing,
which has [high performance](https://github.com/shenwei356/bio#compare-to-kseqh-klib)
close to the
famous C lib [kseq.h](https://github.com/attractivechaos/klib/blob/master/kseq.h).

### Sequence formats and types

seqkit seamlessly support FASTA and FASTQ format. 
Sequence format is automatically detected.
All subcommands except for `faidx` can handle both formats.
And only when some commands (`subseq`, `split`, `sort` and `shuffle`)
which utilise FASTA index to improve perfrmance for large files in two pass mode
(by flag `--two-pass`), only FASTA format is supported.


Sequence type (DNA/RNA/Protein) is automatically detected by leading subsequences
of the first sequences in file or STDIN. The length of the leading subsequences
is configurable by global flag `--alphabet-guess-seq-length` with default value
of 10000. If length of the sequences is less than that, whole sequences will
be checked.

### Sequence ID

By default, most softwares, including `seqkit`, take the leading non-space
letters as sequence identifier (ID). For example,

|   FASTA header                                                  |     ID                                            |
|:--------------------------------------------------------------|:--------------------------------------------------|
| >123456 gene name                                             | 123456                                            |
| >longname                                                     | longname                                          |
| >gi&#124;110645304&#124;ref&#124;NC_002516.2&#124; Pseudomona | gi&#124;110645304&#124;ref&#124;NC_002516.2&#124; |

But for some sequences from NCBI,
e.g. `>gi|110645304|ref|NC_002516.2| Pseudomona`, the ID is `NC_002516.2`.
In this case, we could set sequence ID parsing regular expression by global flag
`--id-regexp "\|([^\|]+)\| "` or just use flag `--id-ncbi`. If you want
the `gi` number, then use `--id-regexp "^gi\|([^\|]+)\|"`.

### FASTA index

For some commands, including `subseq`, `split`, `sort` and `shuffle`,
when input files are (plain or gzipped) FASTA files,
FASTA index would be optional used for
rapid access of sequences and reducing memory occupation.

ATTENTION: the `.seqkit.fai` file created by seqkit is a little different from `.fai` file
created by `samtools`. seqkit uses full sequence head instead of just ID as key.

### Parallelization of CPU intensive jobs

The validation of sequences bases and complement process of sequences
are parallelized for large sequences.

Parsing of line-based files, including BED/GFF file and ID list file are also parallelized.

The Parallelization is implemented by multiple goroutines in golang
 which are similar to but much
lighter weight than threads. The concurrency number is configurable with global
flag `-j` or `--threads` (default value: 1 for single-CPU PC, 2 for others).

### Memory occupation

Most of the subcommands do not read whole FASTA/Q records in to memory,
including `stat`, `fq2fa`, `fx2tab`, `tab2fx`, `grep`, `locate`, `replace`,
 `seq`, `sliding`, `subseq`.

Note that when using `subseq --gtf | --bed`, if the GTF/BED files are too
big, the memory usage will increase.
You could use `--chr` to specify chromesomes and `--feature` to limit features.

Some subcommands need to store sequences or heads in memory, but there are
strategy to reduce memory occupation, including `rmdup` and `common`.
When comparing with sequences, MD5 digest could be used to replace sequence by
flag `-m` (`--md5`).

Some subcommands could either read all records or read the files twice by flag
`-2` (`--two-pass`), including `sample`, `split`, `shuffle` and `sort`.
They use FASTA index for rapid acccess of sequences and reducing memory occupation.

### Reproducibility

Subcommands `sample` and `shuffle` use random function, random seed could be
given by flag `-s` (`--rand-seed`). This makes sure that sampling result could be
reproduced in different environments with same random seed.

## Usage && Examples

[Usage and examples](http://shenwei356.github.io/seqkit/usage/)

[Tutorial](http://shenwei356.github.io/seqkit/tutorial/)

## Benchmark

More details: [http://shenwei356.github.io/seqkit/benchmark/](http://shenwei356.github.io/seqkit/benchmark/)

Datasets:

    $ seqkit stat *.fa
    file           seq_format   seq_type   num_seqs   min_len        avg_len       max_len
    dataset_A.fa   FASTA        DNA          67,748        56       41,442.5     5,976,145                  
    dataset_B.fa   FASTA        DNA             194       970   15,978,096.5   248,956,422

seqkit version: v0.2.8

![benchmark-5tests.csv.png](benchmark/benchmark.5tests.csv.png)


## Citation

**W Shen**, S Le, Y Li\*, F Hu\*. seqkit: a cross-platform and efficient toolkit for FASTA/Q file manipulation.
***PLOS ONE***. revised.

## Contact

Email me for any problem when using seqkit. shenwei356(at)gmail.com

[Create an issue](https://github.com/shenwei356/seqkit/issues) to report bugs,
propose new functions or ask for help.

## License

[MIT License](https://github.com/shenwei356/seqkit/blob/master/LICENSE)
