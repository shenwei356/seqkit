# Download

SeqKit is implemented in [Go](https://golang.org/) programming language,
 executable binary files **for most popular operating systems** are freely available
  in [release](https://github.com/shenwei356/seqkit/releases) page.

## Latest Version

- [SeqKit v0.13.2](https://github.com/shenwei356/seqkit/releases/tag/v0.13.2)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.13.2/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.13.2)
    - `seqkit sana`: fix bug causing hanging on empty files. [#149](https://github.com/shenwei356/seqkit/pull/149)

### Please cite

- **W Shen**, S Le, Y Li\*, F Hu\*. SeqKit: a cross-platform and ultrafast toolkit for FASTA/Q file manipulation.
***PLOS ONE***. [doi:10.1371/journal.pone.0163962](https://doi.org/10.1371/journal.pone.0163962).

### Links

***Tips***

- run `seqkit version` to check update !!!
- run `seqkit genautocomplete` to update Bash completion !!!


OS     |Arch      |File, 中国镜像                                                                                                                                                                                  |Download Count
:------|:---------|:------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Linux  |32-bit    |[seqkit_linux_386.tar.gz](https://github.com/shenwei356/seqkit/releases/download/v0.13.2/seqkit_linux_386.tar.gz), <br/> [中国镜像](http://app.shenwei.me/data/seqkit/seqkit_linux_386.tar.gz)                            |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/seqkit/latest/seqkit_linux_386.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/seqkit/releases/download/v0.13.2/seqkit_linux_386.tar.gz)
Linux  |**64-bit**|[**seqkit_linux_amd64.tar.gz**](https://github.com/shenwei356/seqkit/releases/download/v0.13.2/seqkit_linux_amd64.tar.gz), <br/> [中国镜像](http://app.shenwei.me/data/seqkit/seqkit_linux_amd64.tar.gz)                  |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/seqkit/latest/seqkit_linux_amd64.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/seqkit/releases/download/v0.13.2/seqkit_linux_amd64.tar.gz)
OS X   |32-bit    |[seqkit_darwin_386.tar.gz](https://github.com/shenwei356/seqkit/releases/download/v0.13.2/seqkit_darwin_386.tar.gz), <br/> [中国镜像](http://app.shenwei.me/data/seqkit/seqkit_darwin_386.tar.gz)                         |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/seqkit/latest/seqkit_darwin_386.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/seqkit/releases/download/v0.13.2/seqkit_darwin_386.tar.gz)
OS X   |**64-bit**|[**seqkit_darwin_amd64.tar.gz**](https://github.com/shenwei356/seqkit/releases/download/v0.13.2/seqkit_darwin_amd64.tar.gz), <br/> [中国镜像](http://app.shenwei.me/data/seqkit/seqkit_darwin_amd64.tar.gz)               |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/seqkit/latest/seqkit_darwin_amd64.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/seqkit/releases/download/v0.13.2/seqkit_darwin_amd64.tar.gz)
Windows|32-bit    |[seqkit_windows_386.exe.tar.gz](https://github.com/shenwei356/seqkit/releases/download/v0.13.2/seqkit_windows_386.exe.tar.gz), <br/> [中国镜像](http://app.shenwei.me/data/seqkit/seqkit_windows_386.exe.tar.gz)          |[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/seqkit/latest/seqkit_windows_386.exe.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/seqkit/releases/download/v0.13.2/seqkit_windows_386.exe.tar.gz)
Windows|**64-bit**|[**seqkit_windows_amd64.exe.tar.gz**](https://github.com/shenwei356/seqkit/releases/download/v0.13.2/seqkit_windows_amd64.exe.tar.gz), <br/> [中国镜像](http://app.shenwei.me/data/seqkit/seqkit_windows_amd64.exe.tar.gz)|[![Github Releases (by Asset)](https://img.shields.io/github/downloads/shenwei356/seqkit/latest/seqkit_windows_amd64.exe.tar.gz.svg?maxAge=3600)](https://github.com/shenwei356/seqkit/releases/download/v0.13.2/seqkit_windows_amd64.exe.tar.gz)


## Installation

#### Method 1: Download binaries (latest stable/dev version)

Just [download](https://github.com/shenwei356/seqkit/releases) compressed
executable file of your operating system,
and decompress it with `tar -zxvf *.tar.gz` command or other tools.
And then:

1. **For Linux-like systems**
    1. If you have root privilege simply copy it to `/usr/local/bin`:

            sudo cp seqkit /usr/local/bin/

    1. Or copy to anywhere in the environment variable `PATH`:

            mkdir -p $HOME/bin/; cp seqkit $HOME/bin/

1. **For windows**, just copy `seqkit.exe` to `C:\WINDOWS\system32`.

#### Method 2: Install via conda (latest stable version)  [![Anaconda Cloud](	https://anaconda.org/bioconda/seqkit/badges/version.svg)](https://anaconda.org/bioconda/seqkit) [![downloads](https://anaconda.org/bioconda/seqkit/badges/downloads.svg)](https://anaconda.org/bioconda/seqkit)

    conda install -c bioconda seqkit

#### Method 3: Install via homebrew (latest stable version)

    brew install brewsci/bio/seqkit

#### Method 4: For Go developer (latest stable/dev version)

    go get -u github.com/shenwei356/seqkit/seqkit

#### Method 5: Docker based installation (latest stable/dev version)

[Install Docker](https://docs.docker.com/engine/installation/#supported-platforms)

git clone this repo:

    git clone https://github.com/shenwei356/seqkit

Run the following commands:

    cd seqkit
    docker build -t shenwei356/seqkit .
    docker run -it shenwei356/seqkit:latest

## Bash-completion

Note: The current version supports Bash only.
This should work for *nix systems with Bash installed.

Howto:

1. run: `seqkit genautocomplete`

2. create and edit `~/.bash_completion` file if you don't have it.

        nano ~/.bash_completion

    add the following:

        for bcfile in ~/.bash_completion.d/* ; do
          . $bcfile
        done

## Release History

- [SeqKit v0.13.1](https://github.com/shenwei356/seqkit/releases/tag/v0.13.1)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.13.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.13.1)
    - `seqkit sana`: fix bug causing hanging on empty files. [#148](https://github.com/shenwei356/seqkit/pull/148)
- [SeqKit v0.13.0](https://github.com/shenwei356/seqkit/releases/tag/v0.13.0)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.13.0/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.13.0)
    - `seqkit`: fix a rare FASTA/Q parser bug. [#127](https://github.com/shenwei356/seqkit/issues/127)
    - `seqkit seq`: output sequence or quality in single line when `-s/--seq` or `-q/--qual` is on. [#132](https://github.com/shenwei356/seqkit/issues/132)
    - `seqkit translate`: delete debug info, [#133](https://github.com/shenwei356/seqkit/issues/133), and fix typo. [#134](https://github.com/shenwei356/seqkit/issues/134)
    - `seqkit split2`: tiny performance improvement. [#137](https://github.com/shenwei356/seqkit/issues/137)
    - `seqkit stats`: new flag `-i/--stdin-label` for replacing default "-" for stdin. [#139](https://github.com/shenwei356/seqkit/issues/139)
    - `seqkit fx2tab`: new flag `-s/--seq-hash` for printing hash of sequence (case sensitive). [#144](https://github.com/shenwei356/seqkit/issues/144)
    - `seqkit amplicon`:
        - fix bug of missing searching reverse strand. [#140](https://github.com/shenwei356/seqkit/issues/140)
        - supporting degenerate bases now. [#83](https://github.com/shenwei356/seqkit/issues/83)
        - new flag `-p/--primer-file` for reading list of primer pairs. [#142](https://github.com/shenwei356/seqkit/issues/142)
        - new flag `--bed` for outputing in BED6+1 format. [#141](https://github.com/shenwei356/seqkit/issues/141)
    - New features and improvements by [@bsipos](https://github.com/bsipos). [#130](https://github.com/shenwei356/seqkit/pull/130), [#147](https://github.com/shenwei356/seqkit/pull/147)
        - new command `seqkit scat`, for real-time robust concatenation of fastx files. 
        - Rewrote the parser behind the `sana` subcommand, now it supports robust parsing of fasta file as well.
        - Added a "toolbox" feature to the `bam` subcommand (`-T`), which is a collection of filters acting on streams of BAM records configured through a YAML string (see the docs for more).
        - Added the `SEQKIT_THREADS` environmental variable to override the default number of threads.
- [SeqKit v0.12.1](https://github.com/shenwei356/seqkit/releases/tag/v0.12.1)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.12.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.12.1)
    - `seqkit bam`: add colorised and pretty printed output, by [@bsipos](https://github.com/bsipos). [#110](https://github.com/shenwei356/seqkit/pull/110)
    - `seqkit locate/grep`: fix bug of `-m`, when query contains letters not in subject sequences. [#124](https://github.com/shenwei356/seqkit/issues/124)
    - `seqkit split2`: new flag `-l/--by-length` for splitting into chunks of N bases.
    - `seqkit fx2tab`:
        - new flag `-I/--case-sensitive` for calculating case sensitive base content. [#108](https://github.com/shenwei356/seqkit/issues/108)
        - add missing column name for averge quality for `-H -q`. [#115](https://github.com/shenwei356/seqkit/issues/115)
        - fix output of `-n/--only-name`, do not write empty columns of sequence and quality. [#104](https://github.com/shenwei356/seqkit/issues/104), [#115](https://github.com/shenwei356/seqkit/issues/115)
    - `seqkit seq`: new flag `-k/--color`: colorize sequences.
- [SeqKit v0.12.0](https://github.com/shenwei356/seqkit/releases/tag/v0.12.0)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.12.0/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.12.0)
    - `seqkit`:
        - fix checking input file existence.
        - new global flag `--infile-list` for long list of input files, if given, they are appended to files from cli arguments.
    - `seqkit faidx`: supporting "truncated" (no ending newline charactor) file.
    - `seqkit seq`:
        - **do not force switching on `-g` when using `-m/-M`**.
        - show recommendation if flag `-t/--seq-type` is not DNA/RNA when computing complement sequence. [#103](https://github.com/shenwei356/seqkit/issues/103)
    - `seqkit translate`: supporting multiple frames. [#96](https://github.com/shenwei356/seqkit/issues/96)
    - `seqkit grep/locate`:
        - add detection and warning for space existing in search pattern/sequence.
        - **speed improvement (2X) for `-m/--max-mismatch`**. [shenwei356/bwt/issues/3](https://github.com/shenwei356/bwt/issues/3)
    - `seqkit locate`:
        - new flag `-M/--hide-matched` for hiding matched sequences. [#98](https://github.com/shenwei356/seqkit/issues/98)
        - **new flag `-r/--use-regexp` for explicitly using regular expression, so improve speed of default `index` operation. And you have to switch this on if using regexp now**.
        [#101](https://github.com/shenwei356/seqkit/issues/101)
        - **new flag `-F/--use-fmi` for improving search speed for lots of sequence patterns**.
    - `seqkit rename`: making IDs unique across multiple files, and can write into multiple files. [#100](https://github.com/shenwei356/seqkit/issues/100)
    - `seqkit sample`: fix stdin checking for flag `-2`. [#102](https://github.com/shenwei356/seqkit/issues/102).
    - `seqkit rename/split/split2`: fix detection of existed outdir.
    - `split split`: fix bug of `seqkit split -i -2` and parallizing it.
    - `seqkit version`: checking update is optional (`-u`).
- [SeqKit v0.11.0](https://github.com/shenwei356/seqkit/releases/tag/v0.11.0)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.11.0/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.11.0)
    - `seqkit`: fix hanging when reading from truncated gzip file.
    - new commands:
        - `seqkit amplicon`: retrieve amplicon (or specific region around it) via primer(s).
    - [new commands by @bsipos](https://github.com/shenwei356/seqkit/pull/81):
        - `seqkit watch`: monitoring and online histograms of sequence features.
        - `seqkit sana`: sanitize broken single line fastq files.
        - `seqkit fish`: look for short sequences in larger sequences using local alignment.
        - `seqkit bam`: monitoring and online histograms of BAM record features.
    - `seqkit grep/locate`: reduce memory occupation when using flag `-m/--max-mismatch`.
    - `seqkit seq`: fix panic of computing complement sequence for long sequences containing illegal letters without flag `-v` on. [#84](https://github.com/shenwei356/seqkit/issues/84)
- [SeqKit v0.10.2](https://github.com/shenwei356/seqkit/releases/tag/v0.10.2)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.10.2/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.10.2)
    - `seqkit`: fix bug of parsing sequence ID delimited by tab (`\t`). [#78](https://github.com/shenwei356/seqkit/issues/78)
    - `seqkit grep`: better logic of `--delete-matched`.
    - `seqkit common/rmdup/split`: use xxhash to replace MD5 when comparing with sequence, discard flag `-m/--md5`.
    - `seqkit stats`: new flag `-b/--basename` for outputting basename instead of full path.
- [SeqKit v0.10.1](https://github.com/shenwei356/seqkit/releases/tag/v0.10.1)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.10.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.10.1)
    - `seqkit fx2tab`: new option `-q/--avg-qual` for outputting average read quality. [#60](https://github.com/shenwei356/seqkit/issues/60)
    - `seqkit grep/locate`: fix support of `X` when using `-d/--degenerate`. [#61](https://github.com/shenwei356/seqkit/issues/61)
    - `seqkit translate`:
        - new flag `-M/--init-codon-as-M` to translate initial codon at beginning to 'M'. [#62](https://github.com/shenwei356/seqkit/issues/62)
        - translates `---` to `-` for aligned DNA/RNA, flag `-X` needed. [#63](https://github.com/shenwei356/seqkit/issues/63)
        - supports codons containing ambiguous bases, e.g., `GGN->G`, `ATH->I`. [#64](https://github.com/shenwei356/seqkit/issues/64)
        - new flag `-l/--list-transl-table` to show details of translate table N
        - new flag `-l/--list-transl-table-with-amb-codons` to show details of translate table N (including ambigugous codons)
    - `seqkit split/split2`, fix bug of ignoring `-O` when reading from stdin.
- [SeqKit v0.10.0](https://github.com/shenwei356/seqkit/releases/tag/v0.10.0)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.10.0/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.10.0)
    - `seqkit`: report error when input is directory.
    - new command `seqkit mutate`: edit sequence (point mutation, insertion, deletion).
- [SeqKit v0.9.3](https://github.com/shenwei356/seqkit/releases/tag/v0.9.3)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.9.3/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.9.3)
    - `seqkit stats`: fix panic for empty file. [#57](https://github.com/shenwei356/seqkit/issues/57)
    - `seqkit translate`: add flag `-x/--allow-unknown-codon` to translate unknown codon to `X`. [#56](https://github.com/shenwei356/seqkit/issues/56)
- [SeqKit v0.9.2](https://github.com/shenwei356/seqkit/releases/tag/v0.9.2)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.9.2/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.9.2)
    - `seqkit`: stricter checking for value of global flag `-t/--seq-type`.
    - `seqkit sliding`: fix bug for flag `-g/--greedy`. [#54](https://github.com/shenwei356/seqkit/issues/54)
    - `seqkit translate`: fix bug for frame < 0. [#55](https://github.com/shenwei356/seqkit/issues/55)
    - `seqkit seq`: add TAB to default blank characters (flag `-G/--gap-letters`), and fix filter result when using flag `-g/--remove-gaps` along with `-m/--min-len` or `-M/--max-len`
- [SeqKit v0.9.1](https://github.com/shenwei356/seqkit/releases/tag/v0.9.1)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.9.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.9.1)
    - `seqkit faidx`: fix bug of retrieving subsequence with multiple regions on same sequence. [#48](https://github.com/shenwei356/seqkit/issues/48)
    - `seqkit sort/shuffle/split`: fix bug when using `-2/--two-pass` to process `.gz` files. [#52](https://github.com/shenwei356/seqkit/issues/52)
- [SeqKit v0.9.0](https://github.com/shenwei356/seqkit/releases/tag/v0.9.0)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.9.0/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.9.0)
    - `seqkit`: better handle of empty file, no error message shown. [#36](https://github.com/shenwei356/seqkit/issues/36)
    - new subcommand `seqkit split2`: split sequences into files by size/parts (FASTA, PE/SE FASTQ).  [#35](https://github.com/shenwei356/seqkit/issues/35)
    - new subcommand `seqkit translate`: translate DNA/RNA to protein sequence. [#28](https://github.com/shenwei356/seqkit/issues/28)
    - `seqkit sort`: fix bug when using `-2 -i`, and add support for sorting in natural order. [#39](https://github.com/shenwei356/seqkit/issues/39)
    - `seqkit grep` and `seqkit locate`: add experimental support of mismatch when searching subsequences. [#14](https://github.com/shenwei356/seqkit/issues/14)
    - `seqkit stats`: add stats of Q20 and Q30 for FASTQ. [#45](https://github.com/shenwei356/seqkit/issues/45)
- [SeqKit v0.8.1](https://github.com/shenwei356/seqkit/releases/tag/v0.8.1)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.8.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.8.1)
    - `seqkit`: do not call `pigz` or `gzip` for decompressing gzipped file any more. But you can still utilize `pigz` or `gzip` by `pigz -d -c seqs.fq.gz | seqkit xxx`.
    - `seqkit subseq`: fix bug of missing quality when using `--gtf` or `--bed`
    - `seqkit stats`: parallelize counting files, it's much faster for lots of small files, especially for files on SSD
- [SeqKit v0.8.0](https://github.com/shenwei356/seqkit/releases/tag/v0.8.0)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.8.0/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.8.0)
    - `seqkit`, **stricter FASTA/Q format requirement**, i.e., must starting with `>` or `@`.
    - `seqkit`, *fix output format for FASTQ files containing zero-length records*, yes this [happens](https://github.com/lh3/seqtk/issues/109).
    - `seqkit`, add amino acid code `O` (pyrrolysine) and `U` (selenocysteine).
    - `seqkit replace`, *add flag `--nr-width` to fill leading 0s for `{nr}`*, useful for preparing sequence submission (">strain_00001 XX", ">strain_00002 XX").
    - `seqkit subseq`, require BED file to be tab-delimited.
- [SeqKit v0.7.2](https://github.com/shenwei356/seqkit/releases/tag/v0.7.2)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.7.2/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.7.2)
    - `seqkit tab2fx`: fix a concurrency bug that occurs in low proprobability
    when only 1-column data provided.
    - `seqkit stats`: add quartiles of sequence length
    - `seqkit faidx`: add support for retrieving subsequence using seq ID and region,
    which is similar with "samtools faidx" but has some extra features
- [SeqKit v0.7.1](https://github.com/shenwei356/seqkit/releases/tag/v0.7.1)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.7.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.7.1)
    - `seqkit convert`: fix bug of read quality containing only 3 or less values.  [shenwei356/bio/issues/3](https://github.com/shenwei356/bio/issues/3)
    - `seqkit stats`: add option `-T/--tabular` to output in machine-friendly tabular format.   [#23](https://github.com/shenwei356/seqkit/issues/23)
    - `seqkit common`: increase speed and decrease memory occupation, and add some notes.
    - fix some typos. [#22](https://github.com/shenwei356/seqkit/issues/22)
    - suggestion: please **install [pigz](http://zlib.net/pigz/) to gain better parsing performance for gzipped data**.
- [SeqKit v0.7.0](https://github.com/shenwei356/seqkit/releases/tag/v0.7.0)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.7.0/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.7.0)
    - add new command `convert` for converting FASTQ quality encoding between Sanger, Solexa and Illumina. Thanks suggestion from [@cviner](https://github.com/cviner) ( [#18](https://github.com/shenwei356/seqkit/issues/18)). [usage & example](http://bioinf.shenwei.me/seqkit/usage/#convert).
    - add new command `range` for printing FASTA/Q records in a range (start:end). [#19](https://github.com/shenwei356/seqkit/issues/19). [usage & example](http://bioinf.shenwei.me/seqkit/usage/#range).
    - add new command `concate` for concatenating sequences with same ID from multiple files. [usage & example](http://bioinf.shenwei.me/seqkit/usage/#concate).
- [SeqKit v0.6.0](https://github.com/shenwei356/seqkit/releases/tag/v0.6.0)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.6.0/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.6.0)
    - add new command `genautocomplete` to generate **shell autocompletion** script! ([#17](https://github.com/shenwei356/seqkit/issues/17))
    - add new command `seqkit dup` for duplicating sequences ([#16](https://github.com/shenwei356/seqkit/issues/16))
    - `seqkit stats -a` does not show L50 which may brings confusion ([#15](https://github.com/shenwei356/seqkit/issues/15))
    - `seqkit subseq --bed`: more robust for bad BED files
- [SeqKit v0.5.5](https://github.com/shenwei356/seqkit/releases/tag/v0.5.5)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.5.5/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.5.5)
    - Increasing speed of reading `.gz` file by utilizing `gzip` (1.3X),
        it would be much faster if you installed `pigz` (2X).
    - ***Fixing colorful output in Windows***
    - `seqkit locate`: ***add flag `--gtf` and `--bed` to output GTF/BED6 format,
        so the result can be used in `seqkit subseq`***.
    - `seqkit subseq`: fix bug of `--bed`, add checking coordinate.
- [SeqKit v0.5.4](https://github.com/shenwei356/seqkit/releases/tag/v0.5.4)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.5.4/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.5.4)
    - `seqkit subseq --gtf`, add flag `--gtf-tag` to set tag that's outputted as sequence comment
    - fix `seqkit split` and `seqkit sample`: forget not to wrap sequence and quality in output for FASTQ format
    - compile with go1.8.1
- [SeqKit v0.5.3](https://github.com/shenwei356/seqkit/releases/tag/v0.5.3)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.5.3/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.5.3)
    -  `seqkit grep`: fix bug when using `seqkit grep -r -f patternfile`:
      all records will be retrived due to failing to discarding the blank pattern (`""`). #11
- [SeqKit v0.5.2](https://github.com/shenwei356/seqkit/releases/tag/v0.5.2)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.5.2/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.5.2)
    - `seqkit stats -a` and `seqkit seq -g -G`: change default gap letters from '- ' to '- .'
    - `seqkit subseq`: fix bug of range overflow when using `-d/--down-stream`
        or `-u/--up-stream` for retieving subseq using BED (`--beb`) or GTF (`--gtf`) file.
    - `seqkit locate`: add flag `-G/--non-greedy`, non-greedy mode,
     faster but may miss motifs overlaping with others.
- [SeqKit v0.5.1](https://github.com/shenwei356/seqkit/releases/tag/v0.5.1)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.5.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.5.1)
    - `seqkit restart`: fix bug of flag parsing
- [SeqKit v0.5.0](https://github.com/shenwei356/seqkit/releases/tag/v0.5.0)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.5.0/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.5.0)
    - **new command `seqkit restart`, for resetting start position for circular genome**.
    - `seqkit sliding`: add flag `-g/--greedy`, exporting last subsequences even shorter than windows size.
    - `seqkit seq`:
        - **add flag `-m/--min-len` and `-M/--max-len` to filter sequences by length**.
        - rename flag `-G/--gap-letter` to `-G/--gap-letters`.
    - `seqkit stat`:
        - renamed to `seqkit stats`, don't worry, old name is still available as an alias.
        - **add new flag `-a/all`, for all statistics, including `sum_gap`, `N50`, and `L50`**.
- [SeqKit v0.4.5](https://github.com/shenwei356/seqkit/releases/tag/v0.4.5)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.4.5/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.4.5)
    - `seqkit seq`: fix bug of failing to reverse quality of FASTQ sequence
- [SeqKit v0.4.4](https://github.com/shenwei356/seqkit/releases/tag/v0.4.4)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.4.4/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.4.4)
    - `seqkit locate`: fix bug of missing regular-expression motifs containing
     non-DNA characters (e.g., `ACT.{6,7}CGG`) from motif file (`-f`).
    - compiled with go v1.8.
- [SeqKit v0.4.3](https://github.com/shenwei356/seqkit/releases/tag/v0.4.3)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.4.3/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.4.3)
    - fix bug of `seqkit stat`: `min_len` always be `0` in versions: v0.4.0, v0.4.1, v0.4.2
- [SeqKit v0.4.2](https://github.com/shenwei356/seqkit/releases/tag/v0.4.2)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.4.2/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.4.2)
    - fix header information of `seqkit subseq` when restriving up- and down-steam
sequences using GTF/BED file.
- [SeqKit v0.4.1](https://github.com/shenwei356/seqkit/releases/tag/v0.4.1)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.4.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.4.1)
    - enchancement: remove redudant regions for `seqkit locate`.
- [SeqKit v0.4.0](https://github.com/shenwei356/seqkit/releases/tag/v0.4.0)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.4.0/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.4.0)
    - fix bug of `seqkit locate`, e.g, only find two locations
(`1-4`, `7-10`, missing `4-7`) of `ACGA` in `ACGACGACGA`.
    - better output of `seqkit stat` for empty file.
- [SeqKit v0.3.9](https://github.com/shenwei356/seqkit/releases/tag/v0.3.9)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.3.9/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.3.9)
    - fix bug of region selection for blank sequences. affected commands include
`seqkit subseq --region`, `seqkit grep --region`, `seqkit split --by-region`.
    - compile with go1.8beta1.
- [SeqKit v0.3.8.1](https://github.com/shenwei356/seqkit/releases/tag/v0.3.8.1)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.3.8.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.3.8.1)
    - enhancement and bugfix of `seqkit common`: two or more same files allowed,
fix log information of number of extracted sequences in the first file.
- [SeqKit v0.3.8](https://github.com/shenwei356/seqkit/releases/tag/v0.3.8)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.3.8/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.3.8)
    - enhancement of `seqkit common`: better handling of files containing replicated sequences
- [SeqKit v0.3.7](https://github.com/shenwei356/seqkit/releases/tag/v0.3.7)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.3.7/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.3.7)
    - fix bug in `seqkit split --by-id` when sequence ID contains invalid characters for system path.
    - add more flags validation for `seqkit replace`.
    - enhancement: raise error when key pattern matches multiple targes in cases of replacing with key-value files and more controls are added.
    - changes: do not wrap sequence and quality in output for FASTQ  format.
- [SeqKit v0.3.6](https://github.com/shenwei356/seqkit/releases/tag/v0.3.6)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.3.6/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.3.6)
    - add new feature for `seqkit grep`: new flag `-R` (`--region`) for specifying sequence region for searching.
- [SeqKit v0.3.5](https://github.com/shenwei356/seqkit/releases/tag/v0.3.5)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.3.5/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.3.5)
    - fig bug of `seqkit grep`: flag `-i` (`--ignore-case`) did not work when not using regular expression
- [SeqKit v0.3.4.1](https://github.com/shenwei356/seqkit/releases/tag/v0.3.4.1)
[![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.3.4.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.3.4.1)
    - improve performance of reading (~10%) and writing (100%) gzip-compressed file
    by using `github.com/klauspost/pgzip` package
    - add citation
- [SeqKit v0.3.4](https://github.com/shenwei356/seqkit/releases/tag/v0.3.4)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.3.4/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.3.4)
    - bugfix: `seq` wrongly handles only the first one sequence file when multiple files given
    - new feature: `fx2tab` can output alphabet letters of a sequence by flag `-a` (`--alphabet`)
    - new feature: new flag `-K` (`--keep-key`) for `replace`,  when replacing
    with key-value file, one can choose keeping the key as value or not.
- [SeqKit v0.3.3](https://github.com/shenwei356/seqkit/releases/tag/v0.3.3)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.3.3/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.3.3)
    - fix bug of `seqkit replace`, wrongly starting from 2 when using `{nr}`
      in `-r` (`--replacement`)
    - new feature: `seqkit replace` supports replacement symbols `{nr}` (record number)
      and `{kv}` (corresponding value of the key ($1) by key-value file)
- [SeqKit v0.3.2](https://github.com/shenwei356/seqkit/releases/tag/v0.3.2)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.3.2/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.3.2)
    - fix bug of `seqkit split`, error when target file is in a directory.
    - improve performance of `seqkit spliding` for big sequences, and output
      last part even if it's shorter than window sze,
      output of FASTQ is also supported.
- [SeqKit v0.3.1.1](https://github.com/shenwei356/seqkit/releases/tag/v0.3.1.1)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.3.1.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.3.1.1)
    - compile with go1.7rc5, with ***higher performance and smaller size of binary file***
- [SeqKit v0.3.1](https://github.com/shenwei356/seqkit/releases/tag/v0.3.1)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.3.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.3.1)
    - improve speed of `seqkit locate`
- [SeqKit v0.3.0](https://github.com/shenwei356/seqkit/releases/tag/v0.3.0)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.3.0/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.3.0)
    - use fork of github.com/brentp/xopen, using `zcat` for speedup of .gz file
      reading on \*nix systems.
    - improve speed of parsing sequence ID when creating FASTA index
    - reduce memory usage of `seqkit subseq --gtf`
    - fix bug of `seqkit subseq` when using flag `--id-ncbi`
    - fix bug of `seqkit split`, outdir error
    - fix bug of `seqkit seq -p`, last base is wrongly failed to convert when
      sequence length is odd.
    - add "sum_len" result for output of `seqkit stat`
- [seqkit v0.2.9](https://github.com/shenwei356/seqkit/releases/tag/v0.2.9)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.2.9/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.2.9)
    - fix minor bug of `seqkit split` and `seqkit shuffle`,
      header name error due to improper use of pointer
    - add option `-O (--out-dir)` to `seqkit split`
- [seqkit v0.2.8](https://github.com/shenwei356/seqkit/releases/tag/v0.2.8)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.2.8/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.2.8)
    - improve speed of parsing sequence ID, not using regular expression for default `--id-regexp`
    - improve speed of record outputing for small-size sequences
    - fix minor bug: `seqkit seq` for blank record
    - update benchmark result
- [seqkit v0.2.7](https://github.com/shenwei356/seqkit/releases/tag/v0.2.7)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.2.7/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.2.7)
    - ***reduce memory usage*** by optimize the outputing of sequences.
      detail: using [`BufferedByteSliceWrapper`](https://godoc.org/github.com/shenwei356/util/byteutil#BufferedByteSliceWrapper) to resuse bytes.Buffer.
    - ***reduce memory usage and improve speed*** by using custom buffered
     reading mechanism, instead of using standard library `bufio`,
      which is slow for large genome sequence.
    - discard strategy of "buffer" and "chunk" of FASTA/Q records,
      just parse records one by one.
    - delete global flags `-c (--chunk-size)` and `-b (--buffer-size)`.
    - add function testing scripts
- [seqkit v0.2.6](https://github.com/shenwei356/seqkit/releases/tag/v0.2.6)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.2.6/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.2.6)
    - fix bug of `seqkit subseq`: Inplace subseq method leaded to wrong result
- [seqkit v0.2.5.1](https://github.com/shenwei356/seqkit/releases/tag/v0.2.5.1)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.2.5.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.2.5.1)
    - fix a bug of `seqkit subseq`: chromesome name was not be converting to lower case when using `--gtf` or `--bed`
- [seqkit v0.2.5](https://github.com/shenwei356/seqkit/releases/tag/v0.2.5)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.2.5/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.2.5)
    - fix a serious bug brought in `v0.2.3`, using unsafe method to convert `string` to `[]byte`
    - add awk-like built-in variable of record number (`{NR}`) for `seqkit replace`
- [seqkit v0.2.4.1](https://github.com/shenwei356/seqkit/releases/tag/v0.2.4.1)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.2.4.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.2.4.1)
    - fix several bugs from library `bio`, affected situations:
        - Locating patterns in sequences by pattern FASTA file: `seqkit locate -f`
        - Reading FASTQ file with record of which the quality starts with `+`
    - add command `version`
- [seqkit v0.2.4](https://github.com/shenwei356/seqkit/releases/tag/v0.2.4)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.2.4/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.2.4)
    - add subcommand `head`
- [seqkit v0.2.3](https://github.com/shenwei356/seqkit/releases/tag/v0.2.3)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.2.3/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.2.3)
    - reduce memory occupation by avoid copy data when convert `string` to `[]byte`
    - speedup reverse-complement by avoid repeatly calling functions
- [seqkit v0.2.2](https://github.com/shenwei356/seqkit/releases/tag/v0.2.2)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.2.2/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.2.2)
    - reduce memory occupation of subcommands that use FASTA index
- [seqkit v0.2.1](https://github.com/shenwei356/seqkit/releases/tag/v0.2.1)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.2.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.2.1)
    - improve performance of outputing.
    - fix bug of `seqkit seq -g` for FASTA fromat
    - some other minor fix of code and docs
    - update benchmark results
- [seqkit v0.2.0](https://github.com/shenwei356/seqkit/releases/tag/v0.2.0)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.2.0/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.2.0)
    - ***reduce memory usage of writing output***
    - fix bug of `subseq`, `shuffle`, `sort` when reading from stdin
    - reduce memory usage of `faidx`
    - make validating sequences an optional option in `seq` command, it saves some time.
- [seqkit v0.1.9](https://github.com/shenwei356/seqkit/releases/tag/v0.1.9)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.1.9/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.1.9)
    - using custom FASTA index file extension: `.seqkit.fai`
    - reducing memory usage of `sample --number --two-pass`
    - ***change default CPU number to 2 for multi-cpus computer, and 1 for single-CPU computer***
- [seqkit v0.1.8](https://github.com/shenwei356/seqkit/releases/tag/v0.1.8)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.1.8/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.1.8)
    - add subcommand `rename` to rename duplicated IDs
    - add subcommand `faidx` to create FASTA index file
    - ***utilize faidx to improve performance of `subseq`***
    - *`shuffle`, `sort` and split support two-pass mode (by flag `-2`) with faidx to reduce memory usage.*
    - document update
- [seqkit v0.1.7](https://github.com/shenwei356/seqkit/releases/tag/v0.1.7)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.1.7/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.1.7)
    - ***add support for (multi-line) FASTQ format***
    - update document, add technical details
    - rename subcommands `fa2tab` and `tab2fa` to `fx2tab` and `tab2fx`
    - add subcommand `fq2fa`
    - add column "seq_format" to `stat`
    - add global flag `-b` (`--bufer-size`)
    - little change of flag in `subseq` and some other commands
- [seqkit v0.1.6](https://github.com/shenwei356/seqkit/releases/tag/v0.1.6)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.1.6/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.1.6)
    - add subcommand `replace`
- [seqkit v0.1.5.2](https://github.com/shenwei356/seqkit/releases/tag/v0.1.5.2)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.1.5.2/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.1.5.2)
    - fix bug of `grep`, when not using flag `-r`, flag `-i` will not take effect.
- [seqkit v0.1.5.1](https://github.com/shenwei356/seqkit/releases/tag/v0.1.5.1)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.1.5.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.1.5.1)
    - fix result of `seqkit sample -n`
    - fix benchmark script
- [seqkit v0.1.5](https://github.com/shenwei356/seqkit/releases/tag/v0.1.5)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.1.5/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.1.5)
    - add global flag `--id-ncbi`
    - add flag `-d` (`--dup-seqs-file`) and `-D` (`--dup-num-file`) for subcommand `rmdup`
    - make using MD5 as an optional flag `-m` (`--md5`) in subcommand `rmdup` and `common`
    - fix file name suffix of `seqkit split` result
    - minor modification of `sliding` output
- [seqkit v0.1.4.1](https://github.com/shenwei356/seqkit/releases/tag/v0.1.4.1)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.1.4.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.1.4.1)
    - change alignment of `stat` output
    - preciser CPUs number control
- [seqkit v0.1.4](https://github.com/shenwei356/seqkit/releases/tag/v0.1.4)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.1.4/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.1.4)
    - add subcommand `sort`
    - improve subcommand `subseq`: supporting of getting subsequences by GTF and BED files
    - change name format of `sliding` result
    - prettier output of `stat`
- [seqkit v0.1.3.1](https://github.com/shenwei356/seqkit/releases/tag/v0.1.3.1)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.1.3.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.1.3.1)
    - Performance improvement by reducing time of cleaning spaces
    - Document update
- [seqkit v0.1.3](https://github.com/shenwei356/seqkit/releases/tag/v0.1.3)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.1.3/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.1.3)
    - **Further performance improvement**
    - Rename sub command `extract` to `grep`
    - Change default value of flag `--threads` back CPU number of current device,
      change default value of flag `--chunk-size` back 10000 sequences.
    - Update benchmark
- [seqkit v0.1.2](https://github.com/shenwei356/seqkit/releases/tag/v0.1.2)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.1.2/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.1.2)
    - Add flag `--dna2rna` and `--rna2dna` to subcommand `seq`.
- [seqkit v0.1.1](https://github.com/shenwei356/seqkit/releases/tag/v0.1.1)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.1.1/total.svg)](https://github.com/shenwei356/seqkit/releases/download/v0.1.1)
    - **5.5X speedup of FASTA file parsing** by avoid using regular expression to remove spaces ([detail](https://github.com/shenwei356/bio/commit/2457b877cf1b8d79d05adb1a8952f2dff9046eaf) ) and using slice indexing instead of map to validate letters ([detail](https://github.com/shenwei356/bio/commit/0f5912f6a3c6d737faacf9212f62d11c94e5044a))
    - Change default value of global flag `-- thread` to 1. Since most of the subcommands are I/O intensive,  For computation intensive jobs, like extract and locate, you may set a bigger value.
    - Change default value of global flag `--chunk-size` to 100.
    - Add subcommand `stat`
    - Fix bug of failing to automatically detect alphabet when only one record in file.
- [seqkit v0.1](https://github.com/shenwei356/seqkit/releases/tag/v0.1)
  [![Github Releases (by Release)](https://img.shields.io/github/downloads/shenwei356/seqkit/v0.1/total.svg)](https://github.com/shenwei356/seqkit/releases/tag/v0.1)
    - first release of seqkit

<div id="disqus_thread"></div>
<script>

/**
*  RECOMMENDED CONFIGURATION VARIABLES: EDIT AND UNCOMMENT THE SECTION BELOW TO INSERT DYNAMIC VALUES FROM YOUR PLATFORM OR CMS.
*  LEARN WHY DEFINING THESE VARIABLES IS IMPORTANT: https://disqus.com/admin/universalcode/#configuration-variables*/
/*
var disqus_config = function () {
this.page.url = PAGE_URL;  // Replace PAGE_URL with your page's canonical URL variable
this.page.identifier = PAGE_IDENTIFIER; // Replace PAGE_IDENTIFIER with your page's unique identifier variable
};
*/
(function() { // DON'T EDIT BELOW THIS LINE
var d = document, s = d.createElement('script');
s.src = '//seqkit.disqus.com/embed.js';
s.setAttribute('data-timestamp', +new Date());
(d.head || d.body).appendChild(s);
})();
</script>
<noscript>Please enable JavaScript to view the <a href="https://disqus.com/?ref_noscript">comments powered by Disqus.</a></noscript>
