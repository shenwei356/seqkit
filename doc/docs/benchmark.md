# Benchmark

## Softwares

1. [fakit](https://github.com/shenwei356/fakit). (Go).
   Version [v0.1.3](https://github.com/shenwei356/fakit/releases/tag/v0.1.3).
1. [fasta_utilities](https://github.com/jimhester/fasta_utilities). (Perl).
   Version
   [329f7ca](https://github.com/jimhester/fasta_utilities/commit/329f7ca9266d4a0a96cb5576c464c1bd106865a0).
   *Lots of dependencies to install*.
1. [fastx_toolkit](http://hannonlab.cshl.edu/fastx_toolkit/). (Perl).
   Version [0.0.13](http://hannonlab.cshl.edu/fastx_toolkit/fastx_toolkit_0.0.13_binaries_Linux_2.6_amd64.tar.bz2).
   *Can't handle multi-line FASTA files*.
1. [pyfaidx](https://github.com/mdshw5/pyfaidx). (Python).
   Version [0.4.7.1](https://pypi.python.org/packages/source/p/pyfaidx/pyfaidx-0.4.7.1.tar.gz#md5=f33604a3550c2fa115ac7d33b952127d).
1. [seqmagick](http://seqmagick.readthedocs.org/en/latest/index.html). (Python).
   Version 0.6.1
1. [seqtk](https://github.com/lh3/seqtk). (C).
   Version [1.0-r82-dirty](https://github.com/lh3/seqtk/commit/4feb6e81444ab6bc44139dd3a125068f81ae4ad8).


## Features

Features         |  fakit      | fasta_utilities | fastx_toolkit | pyfaidx | seqmagick | seqtk
:----------------|:-----------:|:---------------:|:-------------:|:-------:|:---------:|:-------
Cross platform   |  Yes        |   Partly        |   Partly      |   Yes   |   Yes     |   Yes
Mutli-line FASTA |  Yes        |   Yes           |   --          |   Yes   |   Yes     |   Yes
Recognize RNA    |  Yes        |   Yes           |   --          |   --    |   Yes     |   Yes
Read STDIN       |  Yes        |   Yes           |   Yes         |   --    |   Yes     |   Yes
Read Gzip        |  Yes        |   Yes           |   --          |   --    |   Yes     |   Yes
Grep             |  Yes        |   Yes           |   --          |   --    |   Yes     |   Yes
Mutli-grep       |  Yes        |   Yes           |   --          |   --    |   Yes     |   Yes
Sample           |  Yes        |   Yes           |   --          |   --    |   Yes     |   Yes
Subseq           |  Yes        |   Yes           |   --          |   Yes   |   Yes     |   Yes
Deduplicate      |  Yes        |  Partly         |   --          |   --    |   Partly  |   --
Split            |  Yes        |   Yes           |   --          |  Partly |   --      |   --
Barcode split    |  Yes        |   --            |   Yes         |   Yes   |   --      |   --
Shuffle          |  Yes        |   --            |   --          |   --    |   --      |   --
Locate motifs    |  Yes        |   --            |   --          |   --    |   --      |   --
Common           |  Yes        |   --            |   --          |   --    |   --      |   --
Clean            |  Yes        |   Yes           |   Yes         |   Yes   |   --      |   --
Transcribe       |  Yes        |   Yes           |   Yes         |   Yes   |   Yes     |   Yes
Translate        |  --         |   Yes           |   Yes         |   Yes   |   Yes     |   --
Size select      |  Indirect   |   Yes           |   --          |   Yes   |   Yes     |   --
Rename name      |  --         |   Yes           |   --          |   --    |   Yes     |   Yes

## Datasets

Original datasets included:

- [SILVA_123_SSURef_tax_silva.fasta.gz](http://www.arb-silva.de/fileadmin/silva_databases/current/Exports/SILVA_123_SSURef_tax_silva.fasta.gz)
- [hs_ref_GRCh38.p2_*.mfa.gz](ftp://ftp.ncbi.nlm.nih.gov/refseq/H_sapiens/H_sapiens/Assembled_chromosomes/seq/)

They are so large, so only subsets are used.

1. `dataset_A`. Sampling by proption of 0.1 for `SILVA_123_SSURef_tax_silva.fasta.gz`

        fakit sample SILVA_123_SSURef_tax_silva.fasta.gz -p 0.1 -o dataset_A.fa.gz

2. `dataset_B`. Merging chr18,19,20,21,22,Y to a single file

        zcat hs_ref_GRCh38.p2_chr{18,19,20,21,22,Y}.mfa.gz | pigz -c > dataset_B.fa.gz

And some tools do not support RNA sequences,
 and are not able to directly read .gz file,
 so the files are uncompressed, and convert to DNA by `fakit seq --rna2dna dataset_A.fa.gz > dataset_A.fa`.

 File                   | type  |  num_seqs   |     min_len |  avg_len    |  max_len
:----------------------:|:-----:|:-----------:|:-----------:|:-----------:|:---------:
dataset_A.fa (261.7M)   | DNA   |    175364   |    900      | 1419.6      |  3725
dataset_B.fa (346.5M)   | DNA   |    6        |   46709983  | 59698489.0  |  80373285

## Platform

PC:

- CPU: Intel Core i5-3320M @ 2.60GHz, two cores/4 threads
- RAM: DDR3 1600MHz, 12GB
- SSD: SAMSUNG 850 EVO 250G, SATA-3
- OS: Fedora 23 (Scientific KDE spin),  Kernal: 4.4.3-300.fc23.x86_64

Softwares:

- Perl: perl 5, version 22, subversion 1 (v5.22.1) built for x86_64-linux-thread-multi
- Python: Python 2.7.10 (default, Sep  8 2015, 17:20:17) [GCC 5.1.1 20150618 (Red Hat 5.1.1-4)] on linux2


## Test 1. Reverse Complement

### Commands

1. fakit:   `for f in *.fa; do time fakit seq -r -p $f > /dev/null; done`
1. fasta_utilities: `for f in *.fa; do time reverse_complement.pl $f > /dev/null; done`
1. pyfaidx: `for f in *.fa; do time faidx -c -r $f > /dev/null; done`.
Two runs needed, first run creates fasta index file, and the second one evaluates.
1. seqmagick: `for f in *.fa; do time seqmagick convert --reverse-complement $f - > /dev/null; done`
1. seqtk:   `for f in *.fa; do time seqtk seq -r $f > /dev/null; done`
1. `revcom_biogo` using [biogo](https://github.com/biogo/biogo)
([source](files/revcom_biogo.go), [binary](files/revcom_biogo.gz) ). (Go).
`for f in *.fa; do time ./revcom_biogo $f > /dev/null; done`

### Results:

 Datasets     |  fakit   | fasta_utilities | pyfaidx | seqmagick | seqtk   | revcom_biogo
:------------:|:--------:|:---------------:|:-------:|:---------:|:-------:|:------------:
dataset_A.fa  |  5.38s   |   4.02s         | 25.98s  |  13.10s   | 0.66s   | 10.92s
dataset_B.fa  |  7.68s   |   2.70s         | 17.52s  |  9.42s    | 0.81s   | 10.48s

## Test 2. Extract sequencs by ID list

### ID lists

ID lists come from sampling 80% of dataset_A and shuffling.

    $ fakit sample -p 0.8 dataset_A.fa | fakit shuffle | fakit seq -n -i > ids_A.txt
    $ wc -l ids_A.txt
    140261 ids_A.txt
    $ head -n 2 ids_A.txt
    GQ103704.1.1352
    FR853054.1.1478

    $ time fakit sample -p 0.8 dataset_B.fa | fakit shuffle | fakit seq -n -i --id-regexp 'gi\|(.+?)\|'  > ids_B.txt
    wc -l ids_B.txt
    4 ids_B.txt
    $ head -n 2 ids_B.txt
    568815574
    568815580

### Commands

1. fakit: `for g in A B; do time fakit grep -f ids_$g.txt dataset_$g.fa > /dev/null; done`
1. fasta_utilities: (using an [fixed](https://github.com/shenwei356/fasta_utilities/commit/cac7f14f952fab9bc4a209c6bc2b7cfad47e60d8)
   version of `in_list.pl`)
   `for g in A B; do time in_list.pl -files ids_$g.txt dataset_$g.fa > /dev/null; done`
1. fastx_toolkit: can't handle multi-line FASTA files
1. pyfaidx: unsupported
1. seqmagick: `for g in A B; do time seqmagick convert --include-from-file ids_$g.txt dataset_$g.fa - > /dev/null; done`
1. seqtk: `for g in A B; do time seqtk subseq  dataset_$g.fa ids_$g.txt > /dev/null; done`

### Results

Datasets      |  fakit    | fasta_utilities | seqmagick   | seqtk
:-------------|:----------|:----------------|:------------|:----------
dataset_A.fa  |  5.40s    |    3.88s        |   8.78s     | 2.31s
dataset_A.fa  |  2.82s    |    1.50s        |   4.58s     | 0.49s



## Test 3. Deduplication

### Dataset

Randomly extract 10% sequences from dataset_*.fa and merge back and then shuffle.

    $ cat <(fakit sample -p 0.1 dataset_A.fa) dataset_A.fa | fakit shuffle > dataset_A_dup.fasta
    $ cat <(fakit sample -p 0.2 dataset_B.fa) dataset_B.fa | fakit shuffle > dataset_B_dup.fasta

Numbers:

    dataset_A_dup.fasta: 175364 + 17411 = 192775
    dataset_B_dup.fasta: 6 + 2 = 8

Unique seqs:

    $ fakit fa2tab dataset_A_dup.fasta  | cut -f 2 | sort | uniq | wc -l
    161864
    $ fakit fa2tab dataset_B_dup.fasta  | cut -f 2 | sort | uniq | wc -l
    6

### Commands

By sequence

1. fakit: `for f in *_dup.fasta; do time fakit rmdup -s $f > /dev/null; done`
1. seqmagick: `for f in *_dup.fasta; do time seqmagick convert --deduplicate-sequences $f - > /dev/null; done`

By name

1. fakit: `for f in *_dup.fasta; do time fakit rmdup -n $f > /dev/null; done`
1. fasta_utilities: `for f in *_dup.fasta; do time unique_headers.pl $f > /dev/null; done`

### Results

By sequence

Datasets             |  fakit  | seqmagick
:--------------------|:--------|:---------
dataset_A_dup.fasta  |  5.18s  |  12.08s
dataset_B_dup.fasta  |  5.55s  |  11.75s

By name

Datasets             |  fakit  | fasta_utilities
:--------------------|:--------|:---------
dataset_A_dup.fasta  |  4.86s  |  4.35s
dataset_B_dup.fasta  |  4.76s  |  2.44s

## Test 4. Sampling

### Commands

Sample by number

1. fakit: `for g in A B; do time fakit sample -n 1000 dataset_$g.fa > /dev/null; done`
1. fasta_utilities: (*Not randomly*) `for g in A B; do time subset_fasta.pl -size 1000 dataset_$g.fa > /dev/null; done`
1. fastx_toolkit: unsupported
1. pyfaidx: unsupported
1. seqmagick: `for g in A B; do time seqmagick convert --sample 1000  dataset_$g.fa - > /dev/null; done`
1. seqtk: `for g in A B; do time seqtk sample  dataset_$g.fa 1000 > /dev/null; done`

### Results

Datasets      |  fakit    | fasta_utilities | seqmagick   | seqtk
:-------------|:----------|:----------------|:------------|:----------
dataset_A.fa  |  2.68s    |    3.01s        |   5.75s     | 0.38s
dataset_A.fa  |  3.62s    |    1.96s        |   7.81s     | 0.60s


## Test 5. Spliting

1. fakit: `for g in A B; do time fakit split -n 3 dataset_$g.fa; done`
1. fasta_utilities: failed to run `split_fasta.pl`
1. fastx_toolkit: unsupported
1. pyfaidx: only support to write each region to a separate file by flag `-x`
1. seqmagick: unsupported
1. seqtk: unsupported

Datasets      |  fakit    | fasta_utilities
:-------------|:----------|:---------------
dataset_A.fa  |  4.31s    |    --  
dataset_A.fa  |  3.57s    |    --
