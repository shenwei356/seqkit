# SeqKit - a cross-platform and ultrafast toolkit for FASTA/Q file manipulation


- **Documents:** [http://bioinf.shenwei.me/seqkit](http://bioinf.shenwei.me/seqkit)
([**Usage**](http://bioinf.shenwei.me/seqkit/usage/),
[**FAQ (New!)**](http://bioinf.shenwei.me/seqkit/faq/),
[**Tutorial**](http://bioinf.shenwei.me/seqkit/tutorial/),
[**Benchmark**](http://bioinf.shenwei.me/seqkit/benchmark/)
and [**Development Notes**](http://bioinf.shenwei.me/seqkit/note/))
- **Source code:** [https://github.com/shenwei356/seqkit](https://github.com/shenwei356/seqkit)
[![GitHub stars](https://img.shields.io/github/stars/shenwei356/seqkit.svg?style=social&label=Star&?maxAge=2592000)](https://github.com/shenwei356/seqkit)
[![license](https://img.shields.io/github/license/shenwei356/seqkit.svg?maxAge=2592000)](https://github.com/shenwei356/seqkit/blob/master/LICENSE)
[![Built with GoLang](https://img.shields.io/badge/powered_by-go-6362c2.svg?style=flat)](https://golang.org)
[![Go Report Card](https://goreportcard.com/badge/github.com/shenwei356/seqkit)](https://goreportcard.com/report/github.com/shenwei356/seqkit)
- **Latest version:** [![Latest Version](https://img.shields.io/github/release/shenwei356/seqkit.svg?style=flat?maxAge=86400)](https://github.com/shenwei356/seqkit/releases)
[![Github Releases](https://img.shields.io/github/downloads/shenwei356/seqkit/latest/total.svg?maxAge=3600)](http://bioinf.shenwei.me/seqkit/download/)
[![Cross-platform](https://img.shields.io/badge/platform-any-ec2eb4.svg?style=flat)](http://bioinf.shenwei.me/seqkit/download/)
[![Install-with-conda](	https://anaconda.org/bioconda/seqkit/badges/installer/conda.svg)](http://bioinf.shenwei.me/seqkit/download/)
[![Anaconda Cloud](	https://anaconda.org/bioconda/seqkit/badges/version.svg)](https://anaconda.org/bioconda/seqkit)
- **[Citation](/#citation):** [![doi](https://img.shields.io/badge/doi-10.1371%2Fjournal.pone.0163962-blue.svg?style=flat)](https://doi.org/10.1371/journal.pone.0163962)


## Introduction

FASTA and FASTQ are basic and ubiquitous formats for storing nucleotide and
protein sequences. Common manipulations of FASTA/Q file include converting,
searching, filtering, deduplication, splitting, shuffling, and sampling.
Existing tools only implement some of these manipulations,
and not particularly efficiently, and some are only available for certain
operating systems. Furthermore, the complicated installation process of
required packages and running environments can render these programs less
user friendly.

This project describes a cross-platform ultrafast comprehensive
toolkit for FASTA/Q processing. SeqKit provides executable binary files for
all major operating systems, including Windows, Linux, and Mac OS X, and can
be directly used without any dependencies or pre-configurations.
SeqKit demonstrates competitive performance in execution time and memory
usage compared to similar tools. The efficiency and usability of SeqKit
enable researchers to rapidly accomplish common FASTA/Q file manipulations.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

## Table of Contents

- [Features](#features)
    - [Features comparison](#features-comparison)
- [Installation](#installation)
    - [Method 1: Download binaries](#method-1-download-binaries)
    - [Method 2: Install via conda](#method-2-install-via-conda)
    - [Method 3: For Go developer](#method-3-for-go-developer)
- [Subcommands](#subcommands)
- [Technical details and guides for use](#technical-details-and-guides-for-use)
    - [FASTA/Q format parsing](#fastaq-format-parsing)
    - [Sequence formats and types](#sequence-formats-and-types)
    - [Sequence ID](#sequence-id)
    - [FASTA index](#fasta-index)
    - [Parallelization of CPU intensive jobs](#parallelization-of-cpu-intensive-jobs)
    - [Memory occupation](#memory-occupation)
    - [Reproducibility](#reproducibility)
- [Usage && Examples](#usage--examples)
- [Benchmark](#benchmark)
- [Citation](#citation)
- [Acknowledgements](#acknowledgements)
- [Contact](#contact)
- [License](#license)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->


## Features

- **Cross-platform** (Linux/Windows/Mac OS X/OpenBSD/FreeBSD,
  see [download](http://bioinf.shenwei.me/seqkit/download/))
- **Light weight and out-of-the-box, no dependencies, no compilation, no configuration**
  (see [download](http://bioinf.shenwei.me/seqkit/download/))
- **UltraFast** (see [benchmark](/#benchmark)),
  **multiple-CPUs supported**.
- **Practical functions supported by 20 subcommands** (see subcommands and
  [usage](http://bioinf.shenwei.me/seqkit/usage/) )
- **Well documented** (detailed [usage](http://bioinf.shenwei.me/seqkit/usage/)
  and [benchmark](http://bioinf.shenwei.me/seqkit/benchmark/) )
- **Seamlessly parses both FASTA and FASTQ formats**
- **Support STDIN and gziped input/output file, easy being used in pipe**
- **Support custom sequence ID regular expression** (especially useful for searching with ID list)
- Reproducible results (configurable rand seed in `sample` and `shuffle`)
- Well organized source code, friendly to use and easy to extend.

### Features comparison

Categories          |Features               |seqkit  |fasta_utilities|fastx_toolkit|pyfaidx|seqmagick|seqtk
:-------------------|:----------------------|:------:|:-------------:|:-----------:|:-----:|:-------:|:---:
**Formats support** |Multi-line FASTA       |Yes     |Yes            |--           |Yes    |Yes      |Yes
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

**Note 1**: See [version information](http://bioinf.shenwei.me/seqkit/benchmark/#softwares) of the softwares.

**Note 2**: See [usage](http://bioinf.shenwei.me/seqkit/usage/) for detailed options of seqkit.

## Installation

Go to [Download Page](http://bioinf.shenwei.me/seqkit/download) for more download options and changelogs.

`SeqKit` is implemented in [Go](https://golang.org/) programming language,
 executable binary files **for most popular operating systems** are freely available
  in [release](https://github.com/shenwei356/seqkit/releases) page.

#### Method 1: Download binaries

Just [download](https://github.com/shenwei356/seqkit/releases) compressed
executable file of your operating system,
and decompress it with `tar -zxvf *.tar.gz` command or other tools.
And then:

1. **For Linux-like systems**
    1. If you have root privilege simply copy it to `/usr/local/bin`:

            sudo cp seqkit /usr/local/bin/

    1. Or add the current directory of the executable file to environment variable
    `PATH`:

            echo export PATH=\$PATH:\"$(pwd)\" >> ~/.bashrc
            source ~/.bashrc


1. **For windows**, just copy `seqkit.exe` to `C:\WINDOWS\system32`.

#### Method 2: Install via conda [![Install-with-conda](https://anaconda.org/bioconda/seqkit/badges/installer/conda.svg)](http://bioinf.shenwei.me/seqkit/download/) [![Anaconda Cloud](	https://anaconda.org/bioconda/seqkit/badges/version.svg)](https://anaconda.org/bioconda/seqkit) [![downloads](https://anaconda.org/bioconda/seqkit/badges/downloads.svg)](https://anaconda.org/bioconda/seqkit)

    conda install -c bioconda seqkit

#### Method 3: For Go developer

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

SeqKit uses author's lightweight and high-performance bioinformatics packages
[bio](https://github.com/shenwei356/bio) for FASTA/Q parsing,
which has [high performance](https://github.com/shenwei356/bio#fastaq-parsing)
close to the
famous C lib [klib](https://github.com/attractivechaos/klib/) ([kseq.h](https://github.com/attractivechaos/klib/blob/master/kseq.h)).

![](https://github.com/shenwei356/bio/raw/master/benchmark/benchmark.tsv.png)

### Sequence formats and types

SeqKit seamlessly support FASTA and FASTQ format.
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

ATTENTION: the `.seqkit.fai` file created by SeqKit is slightly different from `.fai` file
created by `samtools`. SeqKit uses full sequence head instead of just ID as key.

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

[Usage and examples](http://bioinf.shenwei.me/seqkit/usage/)

[Tutorial](http://bioinf.shenwei.me/seqkit/tutorial/)

## Benchmark

More details: [http://bioinf.shenwei.me/seqkit/benchmark/](http://bioinf.shenwei.me/seqkit/benchmark/)

Datasets:

    $ seqkit stat *.fa
    file          format  type   num_seqs        sum_len  min_len       avg_len      max_len
    dataset_A.fa  FASTA   DNA      67,748  2,807,643,808       56      41,442.5    5,976,145
    dataset_B.fa  FASTA   DNA         194  3,099,750,718      970  15,978,096.5  248,956,422
    dataset_C.fq  FASTQ   DNA   9,186,045    918,604,500      100           100          100  

SeqKit version: v0.3.1.1

FASTA:

![benchmark-5tests.tsv.png](benchmark/benchmark.5tests.tsv.png)

FASTQ:

![benchmark-5tests.tsv.png](benchmark/benchmark.5tests.tsv.C.png)

## Citation

**W Shen**, S Le, Y Li\*, F Hu\*. SeqKit: a cross-platform and ultrafast toolkit for FASTA/Q file manipulation.
***PLOS ONE***. [doi:10.1371/journal.pone.0163962](https://doi.org/10.1371/journal.pone.0163962).

## Acknowledgements

We thank [Lei Zhang](https://github.com/jameslz) for testing of SeqKit,
and also thank [Jim Hester](https://github.com/jimhester/),
author of [fasta_utilities](https://github.com/jimhester/fasta_utilities),
for advice on early performance improvements of for FASTA parsing
and [Brian Bushnell](https://twitter.com/BBToolsBio),
author of [BBMaps](https://sourceforge.net/projects/bbmap/),
for advice on naming SeqKit and adding accuracy evaluation in benchmarks.
We also thank Nicholas C. Wu from the Scripps Research Institute,
USA for commenting on the manuscript
and [Guangchuang Yu](http://guangchuangyu.github.io/)
from State Key Laboratory of Emerging Infectious Diseases,
The University of Hong Kong, HK for advice on the manuscript.

We thank [Li Peng](https://github.com/penglbio) for reporting many bugs.

## Contact

Email me for any problem when using seqkit. shenwei356(at)gmail.com

[Create an issue](https://github.com/shenwei356/seqkit/issues) to report bugs,
propose new functions or ask for help.

## License

[MIT License](https://github.com/shenwei356/seqkit/blob/master/LICENSE)
