# Benchmark


## Softwares

1. [fakit](https://github.com/shenwei356/fakit). (Go).
   Version [v0.1.4.1](https://github.com/shenwei356/fakit/releases/tag/v0.1.4.1).
1. [fasta_utilities](https://github.com/jimhester/fasta_utilities). (Perl).
   Version [3dcc0bc](https://github.com/jimhester/fasta_utilities/tree/3dcc0bc6bf1e97839476221c26984b1789482579).
   Lots of dependencies to install_.
1. [fastx_toolkit](http://hannonlab.cshl.edu/fastx_toolkit/). (Perl).
   Version [0.0.13](http://hannonlab.cshl.edu/fastx_toolkit/fastx_toolkit_0.0.13_binaries_Linux_2.6_amd64.tar.bz2).
   Can't handle multi-line FASTA files_.
1. [pyfaidx](https://github.com/mdshw5/pyfaidx). (Python).
   Version [0.4.7.1](https://pypi.python.org/packages/source/p/pyfaidx/pyfaidx-0.4.7.1.tar.gz#md5=f33604a3550c2fa115ac7d33b952127d).
1. [seqmagick](http://seqmagick.readthedocs.org/en/latest/index.html). (Python).
   Version 0.6.1
1. [seqtk](https://github.com/lh3/seqtk). (C).
   Version [1.0-r82-dirty](https://github.com/lh3/seqtk/commit/4feb6e81444ab6bc44139dd3a125068f81ae4ad8).

## Features

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
Deduplicate seqs | Yes      | Partly          | --            | --      | Partly    | --
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
Rename head      | --       | Yes             | --            | --      | Yes       | Yes

## Datasets

### dataset_A - large number of short sequences

dataset_A came from [SILVA rRNA database](http://www.arb-silva.de/).

- [`SILVA_123_SSURef_tax_silva.fasta.gz`](http://www.arb-silva.de/fileadmin/silva_databases/current/Exports/SILVA_123_SSURef_tax_silva.fasta.gz)

Only sampled subsets (~ 10%) are used:

```
fakit sample SILVA_123_SSURef_tax_silva.fasta.gz -p 0.1 -o dataset_A.fa.gz
```

Some tools do not support RNA sequences,  and are not able to directly read .gz file,  so the file are uncompressed, and converted to DNA by

```
fakit seq --rna2dna dataset_A.fa.gz > dataset_A.fa
```

### dataset_B - small number of large sequences

Human genome from [ensembl](http://uswest.ensembl.org/info/data/ftp/index.html)

- Genome DNA:  [`Homo_sapiens.GRCh38.dna_sm.primary_assembly.fa.gz`](ftp://ftp.ensembl.org/pub/release-84/fasta/homo_sapiens/dna/Homo_sapiens.GRCh38.dna_sm.primary_assembly.fa.gz)
- GTF:  [`Homo_sapiens.GRCh38.84.gtf.gz`](ftp://ftp.ensembl.org/pub/release-84/gtf/homo_sapiens/Homo_sapiens.GRCh38.84.gtf.gz)
- BED: `Homo_sapiens.GRCh38.84.bed.gz` was converted from `Homo_sapiens.GRCh38.84.gtf.gz` by  [`gtf2bed`](http://bedops.readthedocs.org/en/latest/content/reference/file-management/conversion/gtf2bed.html?highlight=gtf2bed)  with command

        zcat Homo_sapiens.GRCh38.84.gtf.gz | gtf2bed --do-not-sort | gzip -c > Homo_sapiens.GRCh38.84.bed.gz


Only subsets of serveral chromosomes (chr18,19,20,21,22,Y) were used:

```
fakit grep Homo_sapiens.GRCh38.dna_sm.primary_assembly.fa.gz -c 1 -p 18 -p 19 -p 20 -p 21 -p 22 -p Y  -o dataset_B.fa
```

Datasets summary:

```
$ fakit stat *.fa
file            seq_type    num_seqs       min_len       avg_len       max_len
dataset_A.fa         DNA     175,364           900       1,419.6         3,725
dataset_B.fa         DNA           6    46,709,983    59,698,489    80,373,285
```

### Chr1

DNA and gtf/bed data of Chr1 were used for testing of extracting subsequence:

- `chr1.fa.gz`

        fakit grep -p 1 Homo_sapiens.GRCh38.dna_sm.primary_assembly.fa.gz -o chr1.fa.gz


- `chr1.gtf.gz`

        zcat Homo_sapiens.GRCh38.84.gtf.gz | grep -w '^1' | gzip -c > chr1.gtf.gz


- `chr1.bed.gz`

        zcat Homo_sapiens.GRCh38.84.bed.gz | grep -w '^1' | gzip -c > chr1.bed.gz


## Platform

PC:

- CPU: Intel Core i5-3320M @ 2.60GHz, two cores/4 threads
- RAM: DDR3 1600MHz, 12GB
- SSD: SAMSUNG 850 EVO 250G, SATA-3
- OS: Fedora 23 (Scientific KDE spin),  Kernal: 4.4.3-300.fc23.x86_64

Softwares:

- Perl: perl 5, version 22, subversion 1 (v5.22.1) built for x86_64-linux-thread-multi
- Python: Python 2.7.10 (default, Sep  8 2015, 17:20:17) [GCC 5.1.1 20150618 (Red Hat 5.1.1-4)] on linux2


## Automatic benchmark and ploting scripts

Scripts are available at:  [https://github.com/shenwei356/fakit/tree/master/benchmark](https://github.com/shenwei356/fakit/tree/master/benchmark)


## Test 1. Reverse Complement

### Commands

`revcom_biogo` ([source](https://github.com/shenwei356/fakit/blob/master/benchmark/revcom_biogo.go),
 [binary](https://github.com/shenwei356/fakit/raw/master/benchmark/revcom_biogo) ), 
 a tool written in Golang using [biogo](https://github.com/biogo/biogo) package,
 is also used for comparison of FASTA file parsing performance.

```
echo == fakit
for f in dataset_{A,B}.fa; do echo data: $f; time fakit seq -r -p $f > $f.fakit.rc; done

echo == fasta_utilities
for f in dataset_{A,B}.fa; do echo data: $f; time reverse_complement.pl $f > .$f.fautil.rc; done

echo == pyfaidx
for f in dataset_{A,B}.fa; do echo data: $f; time faidx -c -r $f > $f.pyfaidx.rc; done

echo == seqmagick
for f in dataset_{A,B}.fa; do echo data: $f; time seqmagick convert --reverse-complement $f - > $f.seqmagick.rc; done

echo == seqtk
for f in dataset_{A,B}.fa; do echo data: $f; time seqtk seq -r $f > $f.seqtk.rc; done

echo == biogo
for f in dataset_{A,B}.fa; do echo data: $f; time ./revcom_biogo $f > $f.biogo.rc; done
```

## Test 2. Extract sequencs by ID list

### ID lists

ID lists come from sampling 80% of the corresponding dataset and shuffling.

```
$ fakit sample -p 0.8 dataset_A.fa | fakit shuffle | fakit seq -n -i > ids_A.txt
$ wc -l ids_A.txt
140261 ids_A.txt
$ head -n 2 ids_A.txt
GQ103704.1.1352
FR853054.1.1478

$ fakit sample -p 0.8 dataset_B.fa | fakit shuffle | fakit seq -n -i  > ids_B.txt
wc -l ids_B.txt
4 ids_B.txt
$ cat ids_B.txt
Y
18
22
21
```

### Commands

```
echo == fakit
for g in A B; do echo data: dataset_$g.fa; time fakit grep -f ids_$g.txt dataset_$g.fa > ids_$g.txt.fakit.fa; done

echo == fasta_utilities
for g in A B; do echo data: dataset_$g.fa; time in_list.pl -files ids_$g.txt dataset_$g.fa > ids_$g.txt.fautil.fa; done

echo == seqmagick
for g in A B; do echo data: dataset_$g.fa; time seqmagick convert --include-from-file ids_$g.txt dataset_$g.fa - > ids_$g.txt.seqmagic.fa; done

echo == seqtk
for g in A B; do echo data: dataset_$g.fa; time seqtk subseq  dataset_$g.fa ids_$g.txt > ids_$g.txt.seqtk.fa; done
```

## Test 3. Sampling

### Commands

Sample by number

```
n=1000

echo == fakit
for f in dataset_{A,B}.fa; do echo data: $f; time fakit sample -n $n $f > $f.sample.fakit.fa; done

echo == fasta_utilities
for f in dataset_{A,B}.fa; do echo data: $f; time subset_fasta.pl -size $n $f > $f.sample.fautil.fa; done

echo == seqmagick
for f in dataset_{A,B}.fa; do echo data: $f; time seqmagick convert --sample $n $f - > $f.sample.seqmagick.fa; done

echo == seqtk
for f in dataset_{A,B}.fa; do echo data: $f; time seqtk sample $f $n > $f.sample.seqtk.fa; done
```


## Test 4. Remove duplicated sequences

### Dataset

Randomly extract 10% sequences from dataset_*.fa and merge back and then shuffle.

```
$ cat <(fakit sample -p 0.1 dataset_A.fa) dataset_A.fa | fakit shuffle > dataset_A_dup.fasta
$ cat <(fakit sample -p 0.2 dataset_B.fa) dataset_B.fa | fakit shuffle > dataset_B_dup.fasta
```

Numbers:

```
dataset_A_dup.fasta: 175364 + 17411 = 192775
dataset_B_dup.fasta: 6 + 2 = 8
```

Unique seqs:

```
$ fakit fa2tab dataset_A_dup.fasta  | cut -f 2 | sort | uniq | wc -l
161864
$ fakit fa2tab dataset_B_dup.fasta  | cut -f 2 | sort | uniq | wc -l
6
```

### Commands

By sequence

```
echo == fakit
for f in dataset_{A,B}_dup.fasta; do echo data: $f; time fakit rmdup -s $f > $f.rmdup.fakit.fa; done

echo == seqmagick
for f in dataset_{A,B}_dup.fasta; do echo data: $f; time seqmagick convert --deduplicate-sequences $f - > $f.rmdup.seqmagick.fa; done
```

## Test 5. Extract subsequencs by BED file

### Commands

```
echo ==  fakit
echo data: chr1.fa.gz; time fakit subseq -c 1 chr1.fa.gz --bed chr1.bed.gz > chr1.bed.gz.fakit.fa

echo ==  seqtk
echo data: chr1.fa.gz; time seqtk subseq chr1.fa.gz chr1.bed.gz > chr1.bed.gz.seqtk.fa
```

TODO: bedtools

## Result

All tests were repeated 4 times.

### Performance comparison with other tools

Fakit used all CPUs (4 for my computer) by default.

![benchmark_colorful.png](benchmark/benchmark_colorful.png)

### Speedup with multi-threads

![benchmark_colorful.png](benchmark/fakit_multi_threads/benchmark_colorful.png)

