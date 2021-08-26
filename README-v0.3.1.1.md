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
all major operating systems, including Windows, Linux, and macOS, and can
be directly used without any dependencies or pre-configurations.
SeqKit demonstrates competitive performance in execution time and memory
usage compared to similar tools. The efficiency and usability of SeqKit
enable researchers to rapidly accomplish common FASTA/Q file manipulations.

### Features comparison

|Categories          |Features               |seqkit  |fasta_utilities|fastx_toolkit|pyfaidx|seqmagick|seqtk
|:-------------------|:----------------------|:------:|:-------------:|:-----------:|:-----:|:-------:|:---:
|**Formats support** |Multi-line FASTA       |Yes     |Yes            |--           |Yes    |Yes      |Yes
|                    |FASTQ                  |Yes     |Yes            |Yes          |--     |Yes      |Yes
|                    |Multi-line  FASTQ      |Yes     |Yes            |--           |--     |Yes      |Yes
|                    |Validating sequences   |Yes     |--             |Yes          |Yes    |--       |--
|                    |Supporting RNA         |Yes     |Yes            |--           |--     |Yes      |Yes
|**Functions**       |Searching by motifs    |Yes     |Yes            |--           |--     |Yes      |--
|                    |Sampling               |Yes     |--             |--           |--     |Yes      |Yes
|                    |Extracting sub-sequence|Yes     |Yes            |--           |Yes    |Yes      |Yes
|                    |Removing duplicates    |Yes     |--             |--           |--     |Partly   |--
|                    |Splitting              |Yes     |Yes            |--           |Partly |--       |--
|                    |Splitting by seq       |Yes     |--             |Yes          |Yes    |--       |--
|                    |Shuffling              |Yes     |--             |--           |--     |--       |--
|                    |Sorting                |Yes     |Yes            |--           |--     |Yes      |--
|                    |Locating motifs        |Yes     |--             |--           |--     |--       |--
|                    |Common sequences       |Yes     |--             |--           |--     |--       |--
|                    |Cleaning bases         |Yes     |Yes            |Yes          |Yes    |--       |--
|                    |Transcription          |Yes     |Yes            |Yes          |Yes    |Yes      |Yes
|                    |Translation            |Yes     |Yes            |Yes          |Yes    |Yes      |--
|                    |Filtering by size      |Yes     |Yes            |--           |Yes    |Yes      |--
|                    |Renaming header        |Yes     |Yes            |--           |--     |Yes      |Yes
|**Other features**  |Cross-platform         |Yes     |Partly         |Partly       |Yes    |Yes      |Yes
|                    |Reading STDIN          |Yes     |Yes            |Yes          |--     |Yes      |Yes
|                    |Reading gzipped file   |Yes     |Yes            |--           |--     |Yes      |Yes
|                    |Writing gzip file      |Yes     |--             |--           |--     |Yes      |--

**Note 1**: See [version information](http://bioinf.shenwei.me/seqkit/benchmark/#softwares) of the softwares.

**Note 2**: See [usage](http://bioinf.shenwei.me/seqkit/usage/) for detailed options of seqkit.
 
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
