# Benchmark

## Softwares

1. [fakit](https://github.com/shenwei356/fakit). (Go).
  Version [v0.1.2](https://github.com/shenwei356/fakit/releases/tag/v0.1.2)
1. [pyfaidx](https://github.com/mdshw5/pyfaidx). (Python).
  Version [0.4.7.1](https://pypi.python.org/packages/source/p/pyfaidx/pyfaidx-0.4.7.1.tar.gz#md5=f33604a3550c2fa115ac7d33b952127d)
1. [seqmagick](http://seqmagick.readthedocs.org/en/latest/index.html). (Python).
  Version 0.6.1
1. [seqtk](https://github.com/lh3/seqtk). (C).
Version [1.0-r82-dirty](https://github.com/lh3/seqtk/commit/4feb6e81444ab6bc44139dd3a125068f81ae4ad8)

Unused:

1. [fasta_utilities](https://github.com/jimhester/fasta_utilities). (Perl).  Very difficult to install.
  Version
  [329f7ca](https://github.com/jimhester/fasta_utilities/commit/329f7ca9266d4a0a96cb5576c464c1bd106865a0)
1. [fastx_toolkit](http://hannonlab.cshl.edu/fastx_toolkit/). (Perl). Can't handle multi-line FASTA files.
Version [0.0.13](http://hannonlab.cshl.edu/fastx_toolkit/fastx_toolkit_0.0.13_binaries_Linux_2.6_amd64.tar.bz2)

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

 file                   | type  |  num_seqs   |     min_len |  avg_len    |  max_len
:----------------------:|:-----:|:-----------:|:-----------:|:-----------:|:---------:      
dataset_A.fa (261.7M)   | DNA   |    175364   |    900      | 1419.6      |  3725
dataset_B.fa (346.5M)   | DNA   |    6        |   46709983  | 59698489.0  |  80373285


## Reverse Complement

### Commands

Before each run, run `su -c 'sync; echo 1 > /proc/sys/vm/drop_caches'` to empty page caches.

1. fakit:   `for f in *.fa; do time fakit seq -r -p $f > /dev/null; done`
1. pyfaidx: `for f in *.fa; do time faidx -c -r $f > /dev/null; done`.
Two runs needed, first run creates fasta index file, and the second one evaluates.
1. seqmagick: `for f in *.fa; do time seqmagick convert --reverse-complement $f - > /dev/null; done`
1. seqtk:   `for f in *.fa; do time seqtk seq -r $f > /dev/null; done`
1. `revcom_biogo` using [biogo](https://github.com/biogo/biogo)
([source](files/revcom_biogo.go), [binary](files/revcom_biogo.gz) ). (Go).
`for f in *.fa; do time ./revcom_biogo $f > /dev/null; done`


### Results:

 datasets     |  fakit   | pyfaidx | seqmagick | seqtk   | revcom_biogo
:------------:|:--------:|:-------:|:---------:|:-------:|:------------:
dataset_A.fa  |  8.40s   | 26.43s  |  13.61s   | 0.84s   | 9.95s
dataset_B.fa  |  14.15s  | 18.30s  |  9.98s    | 1.44s   | 10.43s

## Extract sequencs by ID list

### ID list

ID list comes from sampling 80% of dataset_A and shuffling. n=140261.

    $ fakit sample -p 0.8 dataset_A.fa | fakit shuffle | fakit seq -n -i > ids.txt
    $ head -n 5 ids.txt
    GQ103704.1.1352
    FR853054.1.1478
    GU214562.1.1781
    DQ796266.1.1393
    HM309604.1.1340

### Commands

Before each run, run `su -c 'sync; echo 1 > /proc/sys/vm/drop_caches'` to empty page caches.

1. fakit:
    1. default parameters: `time fakit extract -f ids.txt dataset_A.fa -j 1 -c 100 > /dev/null`
    1. single thread and bigger chunk-sze: `time fakit extract -f ids.txt dataset_A.fa -j 1 -c 10000 > /dev/null`;
    1. multiple threads and bigger chunk-size: `time fakit extract -f ids.txt dataset_A.fa -j 4 -c 10000 > /dev/null`
1. pyfaidx: unsupported
1. seqmagick: only support single pattern by `--pattern-include`
1. seqtk: `time seqtk subseq dataset_A.fa ids.txt > /dev/null`

### Results

datasets      |  fakit (-j 1 -c 100)  | fakit (-j 1 -c 10000) | fakit (-j 4 -c 10000)  | seqtk     
:------------:|:---------------------:|:---------------------:|:----------------------:|:-------:
dataset_A.fa  |  7.87s                | 6.14s                 | 4.72s                  | 2.34s


## Deduplication

### Dataset

Randomly extract 10% sequences from dataset_A.fa and merge back and then shuffle. Number: 175364 + 17411 = 192775

    $ cat <(fakit sample -p 0.1 dataset_A.fa) dataset_A.fa | fakit shuffle > dataset_A_dup.fa

Unique seqs:

    $ fakit fa2tab dataset_A_dup.fa  | cut -f 2 | sort | uniq | wc -l
    161864

### Commands

1. fakit: `time fakit rmdup -s dataset_A_dup.fa > /dev/null`
2. seqmagick: `time seqmagick convert --deduplicate-sequences dataset_A_dup.fa - > /dev/null`

### Results

datasets          |  fakit  | seqmagick     
:-----------------|:--------|:---------
dataset_A_dup.fa  |  8.10s  | 13.31s

## TODO
