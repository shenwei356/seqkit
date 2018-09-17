# Usage and Examples

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
## Table of Contents

- [Technical details and guides for use](#technical-details-and-guides-for-use)
- [seqkit](#seqkit)

**Sequence and subsequence**

- [seq](#seq)
- [subseq](#subseq)
- [sliding](#sliding)
- [stats](#stats)
- [faidx](#faidx)

**Format conversion**

- [fq2fa](#fq2fa)
- [fx2tab & tab2fx](#fx2tab--tab2fx)
- [convert](#convert)

**Searching**

- [grep](#grep)
- [locate](#locate)

**Set operations**

- [head](#head)
- [range](#range)
- [sample](#sample)
- [rmdup](#rmdup)
- [duplicate](#duplicate)
- [common](#common)
- [split](#split)
- [split2](#split2)

**Edit**

- [replace](#replace)
- [rename](#rename)
- [restart](#restart)
- [concat](#concat)

**Ordering**

- [shuffle](#shuffle)
- [sort](#sort)

**Misc**

- [genautocomplete](#genautocomplete)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->


## Technical details and guides for use

### FASTA/Q format parsing

SeqKit uses author's lightweight and high-performance bioinformatics packages
[bio](https://github.com/shenwei356/bio) for FASTA/Q parsing,
which has [high performance](https://github.com/shenwei356/bio#fastaq-parsing)
close to the
famous C lib [klib](https://github.com/attractivechaos/klib/) ([kseq.h](https://github.com/attractivechaos/klib/blob/master/kseq.h)).

![](https://github.com/shenwei356/bio/raw/master/benchmark/benchmark.tsv.png)

<s>Seqkit calls `pigz` (much faster than `gzip`) or `gzip` to decompress .gz file if they are available.
So please **install [pigz](http://zlib.net/pigz/) to gain better parsing performance for gzipped data**.</s>
Seqkit does not call `pigz` or `gzip` any more since v0.8.1,
Because it does not always increase the speed.
But you can still utilize `pigz` or `gzip` by `pigz -d -c seqs.fq.gz | seqkit xxx`.

Seqkit uses package [pgzip](https://github.com/klauspost/pgzip) to write gzip file,
which is very fast (**10X of `gzip`, 4X of `pigz`**) and the gzip file would be slighty larger.

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

|   FASTA header                                                |     ID                                            |
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

ATTENTION: the `.seqkit.fai` file created by SeqKit is a little different from `.fai` file
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

## seqkit

```
SeqKit -- a cross-platform and ultrafast toolkit for FASTA/Q file manipulation

Version: 0.9.0-dev3

Author: Wei Shen <shenwei356@gmail.com>

Documents  : http://bioinf.shenwei.me/seqkit
Source code: https://github.com/shenwei356/seqkit
Please cite: https://doi.org/10.1371/journal.pone.0163962

Usage:
  seqkit [command]

Available Commands:
  common          find common sequences of multiple files by id/name/sequence
  concat          concatenate sequences with same ID from multiple files
  convert         convert FASTQ quality encoding between Sanger, Solexa and Illumina
  duplicate       duplicate sequences N times
  faidx           create FASTA index file and extract subsequence
  fq2fa           convert FASTQ to FASTA
  fx2tab          convert FASTA/Q to tabular format (with length/GC content/GC skew)
  genautocomplete generate shell autocompletion script
  grep            search sequences by ID/name/sequence/sequence motifs, mismatch allowed
  head            print first N FASTA/Q records
  help            Help about any command
  locate          locate subsequences/motifs, mismatch allowed
  range           print FASTA/Q records in a range (start:end)
  rename          rename duplicated IDs
  replace         replace name/sequence by regular expression
  restart         reset start position for circular genome
  rmdup           remove duplicated sequences by id/name/sequence
  sample          sample sequences by number or proportion
  seq             transform sequences (revserse, complement, extract ID...)
  shuffle         shuffle sequences
  sliding         sliding sequences, circular genome supported
  sort            sort sequences by id/name/sequence/length
  split           split sequences into files by id/seq region/size/parts (mainly for FASTA)
  split2          split sequences into files by size/parts (FASTA, PE/SE FASTQ)
  stats           simple statistics of FASTA/Q files
  subseq          get subsequences by region/gtf/bed, including flanking sequences
  tab2fx          convert tabular format to FASTA/Q format
  version         print version information and check for update

Flags:
      --alphabet-guess-seq-length int   length of sequence prefix of the first FASTA record based on which seqkit guesses the sequence type (0 for whole seq) (default 10000)
  -h, --help                            help for seqkit
      --id-ncbi                         FASTA head is NCBI-style, e.g. >gi|110645304|ref|NC_002516.2| Pseud...
      --id-regexp string                regular expression for parsing ID (default "^([^\\s]+)\\s?")
  -w, --line-width int                  line width when outputing FASTA format (0 for no wrap) (default 60)
  -o, --out-file string                 out file ("-" for stdout, suffix .gz for gzipped out) (default "-")
      --quiet                           be quiet and do not show extra information
  -t, --seq-type string                 sequence type (dna|rna|protein|unlimit|auto) (for auto, it automatically detect by the first sequence) (default "auto")
  -j, --threads int                     number of CPUs. (default value: 1 for single-CPU PC, 2 for others) (default 2)

Use "seqkit [command] --help" for more information about a command.

```

### Datasets

Datasets from [The miRBase Sequence Database -- Release 21](ftp://mirbase.org/pub/mirbase/21/)

- [`hairpin.fa.gz`](ftp://mirbase.org/pub/mirbase/21/hairpin.fa.gz)
- [`mature.fa.gz`](ftp://mirbase.org/pub/mirbase/21/mature.fa.gz)
- [`miRNA.diff.gz`](ftp://mirbase.org/pub/mirbase/21/miRNA.diff.gz)

Human genome from [ensembl](http://uswest.ensembl.org/info/data/ftp/index.html)
(For `seqkit subseq`)

- [`Homo_sapiens.GRCh38.dna_sm.primary_assembly.fa.gz`](ftp://ftp.ensembl.org/pub/release-84/fasta/homo_sapiens/dna/Homo_sapiens.GRCh38.dna_sm.primary_assembly.fa.gz)
- [`Homo_sapiens.GRCh38.84.gtf.gz`](ftp://ftp.ensembl.org/pub/release-84/gtf/homo_sapiens/Homo_sapiens.GRCh38.84.gtf.gz)
- `Homo_sapiens.GRCh38.84.bed.gz` is converted from `Homo_sapiens.GRCh38.84.gtf.gz`
by [`gtf2bed`](http://bedops.readthedocs.org/en/latest/content/reference/file-management/conversion/gtf2bed.html?highlight=gtf2bed)
with command

        zcat Homo_sapiens.GRCh38.84.gtf.gz | gtf2bed --do-not-sort | gzip -c > Homo_sapiens.GRCh38.84.bed.gz

Only DNA and gtf/bed data of Chr1 were used:

- `chr1.fa.gz`

        seqkit grep -p 1 Homo_sapiens.GRCh38.dna_sm.primary_assembly.fa.gz -o chr1.fa.gz

- `chr1.gtf.gz`

        zcat Homo_sapiens.GRCh38.84.gtf.gz | grep -w '^1' | gzip -c > chr1.gtf.gz

- `chr1.bed.gz`

        zcat Homo_sapiens.GRCh38.84.bed.gz | grep -w '^1' | gzip -c > chr1.bed.gz


## seq

Usage

```
transform sequences (revserse, complement, extract ID...)

Usage:
  seqkit seq [flags]

Flags:
  -p, --complement                complement sequence (blank for Protein sequence)
      --dna2rna                   DNA to RNA
  -G, --gap-letters string        gap letters (default "- .")
  -h, --help                      help for seq
  -l, --lower-case                print sequences in lower case
  -M, --max-len int               only print sequences shorter than the maximum length (-1 for no limit) (default -1)
  -m, --min-len int               only print sequences longer than the minimum length (-1 for no limit) (default -1)
  -n, --name                      only print names
  -i, --only-id                   print ID instead of full head
  -q, --qual                      only print qualities
  -g, --remove-gaps               remove gaps
  -r, --reverse                   reverse sequence
      --rna2dna                   RNA to DNA
  -s, --seq                       only print sequences
  -u, --upper-case                print sequences in upper case
  -v, --validate-seq              validate bases according to the alphabet
  -V, --validate-seq-length int   length of sequence to validate (0 for whole seq) (default 10000)

```

Examples

1. Read and print

    - From file:

            $ seqkit seq hairpin.fa.gz
            >cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop
            UACACUGUGGAUCCGGUGAGGUAGUAGGUUGUAUAGUUUGGAAUAUUACCACCGGUGAAC
            UAUGCAAUUUUCUACCUUACCGGAGACAGAACUCUUCGA

            $ seqkit seq read_1.fq.gz
            @HWI-D00523:240:HF3WGBCXX:1:1101:2574:2226 1:N:0:CTGTAG
            TGAGGAATATTGGTCAATGGGCGCGAGCCTGAACCAGCCAAGTAGCGTGAAGGATGACTGCCCTACGGG
            +
            HIHIIIIIHIIHGHHIHHIIIIIIIIIIIIIIIHHIIIIIHHIHIIIIIGIHIIIIHHHHHHGHIHIII

    - From stdin:

            zcat hairpin.fa.gz | seqkit seq


1. Sequence types

    - By default, `seqkit seq` automatically detect the sequence type

            $ echo -e ">seq\nacgtryswkmbdhvACGTRYSWKMBDHV" | seqkit stats
            file  format  type  num_seqs  sum_len  min_len  avg_len  max_len
            -     FASTA   DNA          1       28       28       28       28

            $ echo -e ">seq\nACGUN ACGUN" | seqkit stats
            file  format  type  num_seqs  sum_len  min_len  avg_len  max_len
            -     FASTA   RNA          1       11       11       11       11

            $ echo -e ">seq\nabcdefghijklmnpqrstvwyz" | seqkit stats
            file  format  type     num_seqs  sum_len  min_len  avg_len  max_len
            -     FASTA   Protein         1       23       23       23       23

            $ echo -e "@read\nACTGCN\n+\n@IICCG" | seqkit stats
            file  format  type  num_seqs  sum_len  min_len  avg_len  max_len
            -     FASTQ   DNA          1        6        6        6        6

    - You can also set sequence type by flag `-t` (`--seq-type`).
      But this only take effect on subcommands `seq` and `locate`.

            $ echo -e ">seq\nabcdefghijklmnpqrstvwyz" | seqkit seq -t dna
            [INFO] when flag -t (--seq-type) given, flag -v (--validate-seq) is automatically switched on
            [ERRO] error when parsing seq: seq (invalid DNAredundant letter: e)


1. Only print names

    - Full name:

            $ seqkit seq hairpin.fa.gz -n
            cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop
            cel-lin-4 MI0000002 Caenorhabditis elegans lin-4 stem-loop
            cel-mir-1 MI0000003 Caenorhabditis elegans miR-1 stem-loop

    - Only ID:

            $ seqkit seq hairpin.fa.gz -n -i
            cel-let-7
            cel-lin-4
            cel-mir-1

    - Custom ID region by regular expression (this could be applied to all subcommands):

            $ seqkit seq hairpin.fa.gz -n -i --id-regexp "^[^\s]+\s([^\s]+)\s"
            MI0000001
            MI0000002
            MI0000003

1. Only print seq (global flag `-w` defines the output line width, 0 for no wrap)

        $ seqkit seq hairpin.fa.gz -s -w 0
        UACACUGUGGAUCCGGUGAGGUAGUAGGUUGUAUAGUUUGGAAUAUUACCACCGGUGAACUAUGCAAUUUUCUACCUUACCGGAGACAGAACUCUUCGA
        AUGCUUCCGGCCUGUUCCCUGAGACCUCAAGUGUGAGUGUACUAUUGAUGCUUCACACCUGGGCUCUCCGGGUACCAGGACGGUUUGAGCAGAU
        AAAGUGACCGUACCGAGCUGCAUACUUCCUUACAUGCCCAUACUAUAUCAUAAAUGGAUAUGGAAUGUAAAGAAGUAUGUAGAACGGGGUGGUAGU

1. Convert multi-line FASTQ to 4-line FASTQ

        $ seqkit seq reads_1.fq.gz -w 0

1. Reverse comlement sequence

        $ seqkit seq hairpin.fa.gz -r -p
        >cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop
        UCGAAGAGUUCUGUCUCCGGUAAGGUAGAAAAUUGCAUAGUUCACCGGUGGUAAUAUUCC
        AAACUAUACAACCUACUACCUCACCGGAUCCACAGUGUA

1. Remove gaps and to lower/upper case

        $ echo -e ">seq\nACGT-ACTGC-ACC" | seqkit seq -g -u
        >seq
        ACGTACTGCACC

1. RNA to DNA

        $ echo -e ">seq\nUCAUAUGCUUGUCUCAAAGAUUA" | seqkit seq --rna2dna
        >seq
        TCATATGCTTGTCTCAAAGATTA

1. Filter by sequence length

        $ cat hairpin.fa | seqkit seq | seqkit stats
        file  format  type  num_seqs    sum_len  min_len  avg_len  max_len
        -     FASTA   RNA     28,645  2,949,871       39      103    2,354

        $ cat hairpin.fa | seqkit seq -m 100 | seqkit stats
        file  format  type  num_seqs    sum_len  min_len  avg_len  max_len
        -     FASTA   RNA     10,975  1,565,486      100    142.6    2,354

        $ cat hairpin.fa | seqkit seq -m 100 -M 1000 | seqkit stats
        file  format  type  num_seqs    sum_len  min_len  avg_len  max_len
        -     FASTA   RNA     10,972  1,560,270      100    142.2      938


## subseq

Usage

```
get subsequences by region/gtf/bed, including flanking sequences.

Recommendation: use plain FASTA file, so seqkit could utilize FASTA index.

The definition of region is 1-based and with some custom design.

Examples:

 1-based index    1 2 3 4 5 6 7 8 9 10
negative index    0-9-8-7-6-5-4-3-2-1
           seq    A C G T N a c g t n
           1:1    A
           2:4      C G T
         -4:-2                c g t
         -4:-1                c g t n
         -1:-1                      n
          2:-2      C G T N a c g t
          1:-1    A C G T N a c g t n
          1:12    A C G T N a c g t n
        -12:-1    A C G T N a c g t n

Usage:
  seqkit subseq [flags]

Flags:
      --bed string        by BED file
      --chr value         select limited sequence with sequence IDs (multiple value supported, case ignored) (default [])
  -d, --down-stream int   down stream length
      --feature value     select limited feature types (multiple value supported, case ignored, only works with GTF) (default [])
      --gtf string        by GTF (version 2.2) file
      --gtf-tag string        output this tag as sequence comment (default "gene_id")
  -f, --only-flank        only return up/down stream sequence
  -r, --region string     by region. e.g 1:12 for first 12 bases, -12:-1 for last 12 bases, 13:-1 for cutting first 12 bases. type "seqkit subseq -h" for more examples
  -u, --up-stream int     up stream length

```

Examples

***Recommendation: use plain FASTA file, so seqkit could utilize FASTA index.***

1. First 12 bases

        $ zcat hairpin.fa.gz | seqkit subseq -r 1:12

1. Last 12 bases

        $ zcat hairpin.fa.gz | seqkit subseq -r -12:-1

1. Subsequences without first and last 12 bases

        $ zcat hairpin.fa.gz | seqkit subseq -r 13:-13

1. Get subsequence by GTF file

        $ cat t.fa
        >seq
        actgACTGactgn

        $ cat t.gtf
        seq     test    CDS     5       8       .       .       .       gene_id "A"; transcript_id "";
        seq     test    CDS     5       8       .       -       .       gene_id "B"; transcript_id "";

        $ seqkit subseq --gtf t.gtf t.fa
        >seq_5:8:. A
        ACTG
        >seq_5:8:- B
        CAGT

    Human genome example:

    ***AVOID loading all data from Homo_sapiens.GRCh38.84.gtf.gz,
    the uncompressed data are so big and may exhaust your RAM.***

    We could specify chromesomes and features.

        $ seqkit subseq --gtf Homo_sapiens.GRCh38.84.gtf.gz --chr 1 --feature cds  hsa.fa > chr1.gtf.cds.fa

        $ seqkit stats chr1.gtf.cds.fa
        file             format  type  num_seqs    sum_len  min_len  avg_len  max_len
        chr1.gtf.cds.fa  FASTA   DNA     65,012  9,842,274        1    151.4   12,045

1. Get CDS and 3bp up-stream sequences

        $ seqkit subseq --gtf t.gtf t.fa -u 3
        >seq_5:8:._us:3 A
        ctgACTG
        >seq_5:8:-_us:3 B
        agtCAGT

1. Get 3bp up-stream sequences of CDS, not including CDS

        $ seqkit subseq --gtf t.gtf t.fa -u 3 -f
        >seq_5:8:._usf:3 A
        ctg
        >seq_5:8:-_usf:3 B
        agt

1. Get subsequences by BED file.

    ***AVOID loading all data from Homo_sapiens.GRCh38.84.gtf.gz,
    the uncompressed data are so big and may exhaust your RAM.***

        $  seqkit subseq --bed Homo_sapiens.GRCh38.84.bed.gz --chr 1 hsa.fa >  chr1.bed.gz.fa

    We may need to remove duplicated sequences

        $ seqkit subseq --bed Homo_sapiens.GRCh38.84.bed.gz --chr 1 hsa.fa | seqkit rmdup > chr1.bed.rmdup.fa
        [INFO] 141060 duplicated records removed

    Summary:

        $ seqkit stats chr1.gz.*.gz
        file               seq_format   seq_type   num_seqs   min_len   avg_len     max_len
        chr1.gz.fa         FASTA        DNA         231,974         1   3,089.5   1,551,957
        chr1.gz.rmdup.fa   FASTA        DNA          90,914         1   6,455.8   1,551,957


## sliding

Usage

```
sliding sequences, circular genome supported

Usage:
  seqkit sliding [flags]

Flags:
  -C, --circular-genome   circular genome.
  -g, --greedy            greedy mode, i.e., exporting last subsequences even shorter than windows size
  -s, --step int          step size
  -W, --window int        window size

```

Examples

1. General use

        $ echo -e ">seq\nACGTacgtNN" | seqkit sliding -s 3 -W 6
        >seq_sliding:1-6
        ACGTac
        >seq_sliding:4-9
        TacgtN

1. Greedy mode

        $ echo -e ">seq\nACGTacgtNN" | seqkit sliding -s 3 -W 6 -g
        >seq_sliding:1-6
        ACGTac
        >seq_sliding:4-9
        TacgtN
        >seq_sliding:7-12
        gtNN
        >seq_sliding:10-15
        N

2. Circular genome

        $ echo -e ">seq\nACGTacgtNN" | seqkit sliding -s 3 -W 6 -C
        >seq_sliding:1-6
        ACGTac
        >seq_sliding:4-9
        TacgtN
        >seq_sliding:7-2
        gtNNAC
        >seq_sliding:10-5
        NACGTa

3. Generate GC content for ploting

        $ zcat hairpin.fa.gz | seqkit fx2tab | head -n 1 | seqkit tab2fx | seqkit sliding -s 5 -W 30 | seqkit fx2tab -n -g
        cel-let-7_sliding:1-30          50.00
        cel-let-7_sliding:6-35          46.67
        cel-let-7_sliding:11-40         43.33
        cel-let-7_sliding:16-45         36.67
        cel-let-7_sliding:21-50         33.33
        cel-let-7_sliding:26-55         40.00
        ...

## stats

Usage

```
simple statistics of FASTA/Q files

Tips:
  1. For lots of small files (especially on SDD), use big value of '-j' to
     parallelize counting.

Usage:
  seqkit stats [flags]

Aliases:
  stats, stat

Flags:
  -a, --all                  all statistics, including quartiles of seq length, sum_gap, N50
  -G, --gap-letters string   gap letters (default "- .")
  -h, --help                 help for stats
  -e, --skip-err             skip error, only show warning message
  -T, --tabular              output in machine-friendly tabular format
```

Eexamples

1. General use

        $ seqkit stats *.f{a,q}.gz
        file           format  type  num_seqs    sum_len  min_len  avg_len  max_len
        hairpin.fa.gz  FASTA   RNA     28,645  2,949,871       39      103    2,354
        mature.fa.gz   FASTA   RNA     35,828    781,222       15     21.8       34
        reads_1.fq.gz  FASTQ   DNA      2,500    567,516      226      227      229
        reads_2.fq.gz  FASTQ   DNA      2,500    560,002      223      224      225

1. Machine-friendly tabular format

        $ seqkit stats *.f{a,q}.gz -T
        file    format  type    num_seqs        sum_len min_len avg_len max_len
        hairpin.fa.gz   FASTA   RNA     28645   2949871 39      103.0   2354
        mature.fa.gz    FASTA   RNA     35828   781222  15      21.8    34
        Illimina1.8.fq.gz       FASTQ   DNA     10000   1500000 150     150.0   150
        reads_1.fq.gz   FASTQ   DNA     2500    567516  226     227.0   229
        reads_2.fq.gz   FASTQ   DNA     2500    560002  223     224.0   225

        # So you can process the result with tools like [csvtk](http://bioinf.shenwei.me/csvtk)

        $ seqkit stats *.f{a,q}.gz -T | csvtk pretty -t
        file                format   type   num_seqs   sum_len   min_len   avg_len   max_len
        hairpin.fa.gz       FASTA    RNA    28645      2949871   39        103.0     2354
        mature.fa.gz        FASTA    RNA    35828      781222    15        21.8      34
        Illimina1.8.fq.gz   FASTQ    DNA    10000      1500000   150       150.0     150
        reads_1.fq.gz       FASTQ    DNA    2500       567516    226       227.0     229
        reads_2.fq.gz       FASTQ    DNA    2500       560002    223       224.0     225

        # To markdown

        $ seqkit stats *.f{a,q}.gz -T | csvtk csv2md -t
        file             |format|type|num_seqs|sum_len|min_len|avg_len|max_len
        :----------------|:-----|:---|:-------|:------|:------|:------|:------
        hairpin.fa.gz    |FASTA |RNA |28645   |2949871|39     |103.0  |2354
        mature.fa.gz     |FASTA |RNA |35828   |781222 |15     |21.8   |34
        Illimina1.8.fq.gz|FASTQ |DNA |10000   |1500000|150    |150.0  |150
        reads_1.fq.gz    |FASTQ |DNA |2500    |567516 |226    |227.0  |229
        reads_2.fq.gz    |FASTQ |DNA |2500    |560002 |223    |224.0  |225

    file             |format|type|num_seqs|sum_len|min_len|avg_len|max_len
    :----------------|:-----|:---|:-------|:------|:------|:------|:------
    hairpin.fa.gz    |FASTA |RNA |28645   |2949871|39     |103.0  |2354
    mature.fa.gz     |FASTA |RNA |35828   |781222 |15     |21.8   |34
    Illimina1.8.fq.gz|FASTQ |DNA |10000   |1500000|150    |150.0  |150
    reads_1.fq.gz    |FASTQ |DNA |2500    |567516 |226    |227.0  |229
    reads_2.fq.gz    |FASTQ |DNA |2500    |560002 |223    |224.0  |225


1. Extra information

        $ seqkit stats *.f{a,q}.gz -a
        file               format  type  num_seqs    sum_len  min_len  avg_len  max_len   Q1   Q2   Q3  sum_gap  N50
        hairpin.fa.gz      FASTA   RNA     28,645  2,949,871       39      103    2,354   76   91  111        0  101
        mature.fa.gz       FASTA   RNA     35,828    781,222       15     21.8       34   21   22   22        0   22
        Illimina1.8.fq.gz  FASTQ   DNA     10,000  1,500,000      150      150      150  150  150  150        0  150
        reads_1.fq.gz      FASTQ   DNA      2,500    567,516      226      227      229  227  227  227        0  227
        reads_2.fq.gz      FASTQ   DNA      2,500    560,002      223      224      225  224  224  224        0  224

1. Parallelize counting files, it's much faster for lots of small files, especially for files on SSD

        seqkit stats -j 10 refseq/virual/*.fna.gz

1. Skip error

        $ seqkit stats tests/*
        [ERRO] tests/hairpin.fa.fai: fastx: invalid FASTA/Q format

        $ seqkit stats tests/* -e
        [WARN] tests/hairpin.fa.fai: fastx: invalid FASTA/Q format
        [WARN] tests/hairpin.fa.seqkit.fai: fastx: invalid FASTA/Q format
        [WARN] tests/miRNA.diff.gz: fastx: invalid FASTA/Q format
        [WARN] tests/test.sh: fastx: invalid FASTA/Q format
        file                     format  type  num_seqs    sum_len  min_len  avg_len  max_len
        tests/contigs.fa         FASTA   DNA          9         54        2        6       10
        tests/hairpin.fa         FASTA   RNA     28,645  2,949,871       39      103    2,354
        tests/Illimina1.5.fq     FASTQ   DNA          1        100      100      100      100
        tests/Illimina1.8.fq.gz  FASTQ   DNA     10,000  1,500,000      150      150      150
        tests/hairpin.fa.gz      FASTA   RNA     28,645  2,949,871       39      103    2,354
        tests/reads_1.fq.gz      FASTQ   DNA      2,500    567,516      226      227      229
        tests/mature.fa.gz       FASTA   RNA     35,828    781,222       15     21.8       34
        tests/reads_2.fq.gz      FASTQ   DNA      2,500    560,002      223      224      225

## faidx

Usage

```
create FASTA index file and extract subsequence

This command is similar with "samtools faidx" but has some extra features:

  1. output full header line with flag -f
  2. support regular expression as sequence ID with flag -r

Usage:
  seqkit faidx [flags] <fasta-file> [regions...]

Flags:
  -f, --full-head     print full header line instead of just ID. New fasta index file ending with .seqkit.fai will be created
  -h, --help          help for faidx
  -i, --ignore-case   ignore case
  -r, --use-regexp    IDs are regular expression. But subseq region is not suppored here.

```

Example

1. common usage

        $ seqkit faidx tests/hairpin.fa hsa-let-7a-1 hsa-let-7a-2
        >hsa-let-7a-1
        UGGGAUGAGGUAGUAGGUUGUAUAGUUUUAGGGUCACACCCACCACUGGGAGAUAACUAU
        ACAAUCUACUGUCUUUCCUA
        >hsa-let-7a-2
        AGGUUGAGGUAGUAGGUUGUAUAGUUUAGAAUUACAUCAAGGGAGAUAACUGUACAGCCU
        CCUAGCUUUCCU

2. output full header

        $ seqkit faidx tests/hairpin.fa hsa-let-7a-1 hsa-let-7a-2 -f
        >hsa-let-7a-1 MI0000060 Homo sapiens let-7a-1 stem-loop
        UGGGAUGAGGUAGUAGGUUGUAUAGUUUUAGGGUCACACCCACCACUGGGAGAUAACUAU
        ACAAUCUACUGUCUUUCCUA
        >hsa-let-7a-2 MI0000061 Homo sapiens let-7a-2 stem-loop
        AGGUUGAGGUAGUAGGUUGUAUAGUUUAGAAUUACAUCAAGGGAGAUAACUGUACAGCCU
        CCUAGCUUUCCU

3. extract subsequence of specific region

        $ seqkit faidx tests/hairpin.fa hsa-let-7a-1:1-10
        >hsa-let-7a-1:1-10
        UGGGAUGAGG

        $ seqkit faidx tests/hairpin.fa hsa-let-7a-1:-10--1
        >hsa-let-7a-1:-10--1
        GUCUUUCCUA

        $ seqkit faidx tests/hairpin.fa hsa-let-7a-1:1
        >hsa-let-7a-1:1-1
        U

4. use regular expression

        $ seqkit faidx tests/hairpin.fa hsa -r | seqkit stats
        file  format  type  num_seqs  sum_len  min_len  avg_len  max_len
        -     FASTA   RNA      1,881  154,002       41     81.9      180

## fq2fa

Usage

```
convert FASTQ to FASTA

Usage:
  seqkit fq2fa [flags]

```

Examples

    seqkit fq2fa reads_1.fq.gz -o reads1_.fa.gz


## fx2tab & tab2fx

Usage (fx2tab)

```
convert FASTA/Q to tabular format, and provide various information,
like sequence length, GC content/GC skew.

Usage:
  seqkit fx2tab [flags]

Flags:
  -B, --base-content value   print base content. (case ignored, multiple values supported) e.g. -B AT -B N (default [])
  -g, --gc                   print GC content
  -G, --gc-skew              print GC-Skew
  -H, --header-line          print header line
  -l, --length               print sequence length
  -n, --name                 only print names (no sequences and qualities)
  -i, --only-id              print ID instead of full head

```

Usage (tab2fx)

```
convert tabular format (first two/three columns) to FASTA/Q format

Usage:
  seqkit tab2fx [flags]

Flags:
  -p, --comment-line-prefix value   comment line prefix (default [#,//])


```

Examples

1. Default output

        $ seqkit fx2tab hairpin.fa.gz | head -n 2
        cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop      UACACUGUGGAUCCGGUGAGGUAGUAGGUUGUAUAGUUUGGAAUAUUACCACCGGUGAACUAUGCAAUUUUCUACCUUACCGGAGACAGAACUCUUCGA
        cel-lin-4 MI0000002 Caenorhabditis elegans lin-4 stem-loop      AUGCUUCCGGCCUGUUCCCUGAGACCUCAAGUGUGAGUGUACUAUUGAUGCUUCACACCUGGGCUCUCCGGGUACCAGGACGGUUUGAGCAGAU


1. Print sequence length, GC content, and only print names (no sequences),
we could also print title line by flag `-T`.

        $ seqkit fx2tab hairpin.fa.gz -l -g -n -i -H | head -n 4 | csvtk -t -C '&' pretty
        #name       seq   qual   length   GC
        cel-let-7                99       43.43
        cel-lin-4                94       54.26
        cel-mir-1                96       40.62

1. Use fx2tab and tab2fx in pipe

        $ zcat hairpin.fa.gz | seqkit fx2tab | seqkit tab2fx

        $ zcat reads_1.fq.gz | seqkit fx2tab | seqkit tab2fx

1. Sort sequences by length (use `seqkit sort -l`)

        $ zcat hairpin.fa.gz | seqkit fx2tab -l | sort -t"`echo -e '\t'`" -n -k4,4 | seqkit tab2fx
        >cin-mir-4129 MI0015684 Ciona intestinalis miR-4129 stem-loop
        UUCGUUAUUGGAAGACCUUAGUCCGUUAAUAAAGGCAUC
        >mmu-mir-7228 MI0023723 Mus musculus miR-7228 stem-loop
        UGGCGACCUGAACAGAUGUCGCAGUGUUCGGUCUCCAGU
        >cin-mir-4103 MI0015657 Ciona intestinalis miR-4103 stem-loop
        ACCACGGGUCUGUGACGUAGCAGCGCUGCGGGUCCGCUGU

        $ seqkit sort -l hairpin.fa.gz

    Sorting or filtering by GC (or other base by -flag `-B`) content could also achieved in similar way.

1. Get first 1000 sequences (use `seqkit head -n 1000`)

        $ seqkit fx2tab hairpin.fa.gz | head -n 1000 | seqkit tab2fx

        $ seqkit fx2tab reads_1.fq.gz | head -n 1000 | seqkit tab2fx

**Extension**

After converting FASTA to tabular format with `seqkit fx2tab`,
it could be handled with CSV/TSV tools,
 e.g. [csvtk](https://github.com/shenwei356/csvtkt), a cross-platform, efficient and practical CSV/TSV toolkit

- `csvtk grep` could be used to filter sequences (similar with `seqkit grep`)
- `csvtk inter`
computates intersection of multiple files. It could achieve similar function
as `seqkit common -n` along with shell.
- `csvtk join` joins multiple CSV/TSV files by multiple IDs.
- [csv_melt](https://github.com/shenwei356/datakit/blob/master/csv_melt)
provides melt function, could be used in preparation of data for ploting.

## convert

Usage

```
convert FASTQ quality encoding between Sanger, Solexa and Illumina

Usage:
  seqkit convert [flags]

Flags:
  -d, --dry-run                         dry run
  -f, --force                           for Illumina-1.8+ -> Sanger, truncate scores > 40 to 40
      --from string                     source quality encoding. if not given, we'll guess it
  -h, --help                            help for convert
  -n, --nrecords int                    number of records for guessing quality encoding (default 1000)
  -N, --thresh-B-in-n-most-common int   threshold of 'B' in top N most common quality for guessing Illumina 1.5. (default 4)
  -F, --thresh-illumina1.5-frac float   threshold of faction of Illumina 1.5 in the leading N records (default 0.1)
      --to string                       target quality encoding (default "Sanger")
```

Examples:

Note that `seqkit convert` always output sequences.

The test dataset contains score 41 (`J`):

```
$ seqkit head -n 1 tests/Illimina1.8.fq.gz
@ST-E00493:56:H33MFALXX:4:1101:23439:1379 1:N:0:NACAACCA
NCGTGGAAAGACGCTAAGATTGTGATGTGCTTCCCTGACGATTACAACTGGCGTAAGGACGTTTTGCCTACCTATAAGGCTAACCGTAAGGGTTCTCGCAAGCCTGTAGGTTACAAGAGGTTCGTAGCCGAAGTGATGGCTGACTCACGG
+
#AAAFAAJFFFJJJ<JJJJJFFFJFJJJJJFJJAJJJFJJFJFJJJJFAFJ<JA<FFJ7FJJFJJAAJJJJ<JJJJJJJFJJJAJJJJJFJJ77<JJJJ-F7A-FJFFJJJJJJ<FFJ-<7FJJJFJJ)A7)7AA<7--)<-7F-A7FA<
```

By default, nothing changes when converting Illumina 1.8 to Sanger. A warning message show that source and target quality encoding match.

```
$ seqkit convert tests/Illimina1.8.fq.gz  | seqkit head -n 1
[INFO] possible quality encodings: [Illumina-1.8+]
[INFO] guessed quality encoding: Illumina-1.8+
[INFO] converting Illumina-1.8+ -> Sanger
[WARN] source and target quality encoding match.
@ST-E00493:56:H33MFALXX:4:1101:23439:1379 1:N:0:NACAACCA
NCGTGGAAAGACGCTAAGATTGTGATGTGCTTCCCTGACGATTACAACTGGCGTAAGGACGTTTTGCCTACCTATAAGGCTAACCGTAAGGGTTCTCGCAAGCCTGTAGGTTACAAGAGGTTCGTAGCCGAAGTGATGGCTGACTCACGG
+
#AAAFAAJFFFJJJ<JJJJJFFFJFJJJJJFJJAJJJFJJFJFJJJJFAFJ<JA<FFJ7FJJFJJAAJJJJ<JJJJJJJFJJJAJJJJJFJJ77<JJJJ-F7A-FJFFJJJJJJ<FFJ-<7FJJJFJJ)A7)7AA<7--)<-7F-A7FA<
```

When switching flag `--force` on,  `J` (41) was converted to `I` (40).

```
$ seqkit convert tests/Illimina1.8.fq.gz -f | seqkit head -n 1
[INFO] possible quality encodings: [Illumina-1.8+]
[INFO] guessed quality encoding: Illumina-1.8+
[INFO] converting Illumina-1.8+ -> Sanger
@ST-E00493:56:H33MFALXX:4:1101:23439:1379 1:N:0:NACAACCA
NCGTGGAAAGACGCTAAGATTGTGATGTGCTTCCCTGACGATTACAACTGGCGTAAGGACGTTTTGCCTACCTATAAGGCTAACCGTAAGGGTTCTCGCAAGCCTGTAGGTTACAAGAGGTTCGTAGCCGAAGTGATGGCTGACTCACGG
+
#AAAFAAIFFFIII<IIIIIFFFIFIIIIIFIIAIIIFIIFIFIIIIFAFI<IA<FFI7FIIFIIAAIIII<IIIIIIIFIIIAIIIIIFII77<IIII-F7A-FIFFIIIIII<FFI-<7FIIIFII)A7)7AA<7--)<-7F-A7FA<
```

Other cases:

To Illumina-1.5.

```
$ seqkit convert tests/Illimina1.8.fq.gz --to Illumina-1.5+ | seqkit head -n 1
[INFO] possible quality encodings: [Illumina-1.8+]
[INFO] guessed quality encoding: Illumina-1.8+
[INFO] converting Illumina-1.8+ -> Illumina-1.5+
@ST-E00493:56:H33MFALXX:4:1101:23439:1379 1:N:0:NACAACCA
NCGTGGAAAGACGCTAAGATTGTGATGTGCTTCCCTGACGATTACAACTGGCGTAAGGACGTTTTGCCTACCTATAAGGCTAACCGTAAGGGTTCTCGCAAGCCTGTAGGTTACAAGAGGTTCGTAGCCGAAGTGATGGCTGACTCACGG
+
B```e``ieeeiii[iiiiieeeieiiiiieii`iiieiieieiiiie`ei[i`[eeiVeiieii``iiii[iiiiiiieiii`iiiiieiiVV[iiiiLeV`Leieeiiiiii[eeiL[VeiiieiiH`VHV``[VLLH[LVeL`Ve`[
```

To Illumina-1.5 and back to Sanger.

```
$ seqkit convert tests/Illimina1.8.fq.gz --to Illumina-1.5+ | seqkit convert | seqkit head -n 1
[INFO] possible quality encodings: [Illumina-1.8+]
[INFO] guessed quality encoding: Illumina-1.8+
[INFO] converting Illumina-1.8+ -> Illumina-1.5+
[INFO] possible quality encodings: [Illumina-1.5+]
[INFO] guessed quality encoding: Illumina-1.5+
[INFO] converting Illumina-1.5+ -> Sanger
@ST-E00493:56:H33MFALXX:4:1101:23439:1379 1:N:0:NACAACCA
NCGTGGAAAGACGCTAAGATTGTGATGTGCTTCCCTGACGATTACAACTGGCGTAAGGACGTTTTGCCTACCTATAAGGCTAACCGTAAGGGTTCTCGCAAGCCTGTAGGTTACAAGAGGTTCGTAGCCGAAGTGATGGCTGACTCACGG
+
!AAAFAAJFFFJJJ<JJJJJFFFJFJJJJJFJJAJJJFJJFJFJJJJFAFJ<JA<FFJ7FJJFJJAAJJJJ<JJJJJJJFJJJAJJJJJFJJ77<JJJJ-F7A-FJFFJJJJJJ<FFJ-<7FJJJFJJ)A7)7AA<7--)<-7F-A7FA<
```

Checking encoding

```
$ seqkit convert tests/Illimina1.8.fq.gz --from Solexa
[INFO] converting Solexa -> Sanger
[ERRO] seq: invalid Solexa quality
```
Real Illumina 1.5+ data

```
$ seqkit seq tests/Illimina1.5.fq
@HWI-EAS209_0006_FC706VJ:5:58:5894:21141#ATCACG/1
TTAATTGGTAAATAAATCTCCTAATAGCTTAGATNTTACCTTNNNNNNNNNNTAGTTTCTTGAGATTTGTTGGGGGAGACATTTTTGTGATTGCCTTGAT
+
efcfffffcfeefffcffffffddf`feed]`]_Ba_^__[YBBBBBBBBBBRTT\]][]dddd`ddd^dddadd^BBBBBBBBBBBBBBBBBBBBBBBB

$ seqkit convert tests/Illimina1.5.fq | seqkit head -n 1
[INFO] possible quality encodings: [Illumina-1.5+]
[INFO] guessed quality encoding: Illumina-1.5+
[INFO] converting Illumina-1.5+ -> Sanger
@HWI-EAS209_0006_FC706VJ:5:58:5894:21141#ATCACG/1
TTAATTGGTAAATAAATCTCCTAATAGCTTAGATNTTACCTTNNNNNNNNNNTAGTTTCTTGAGATTTGTTGGGGGAGACATTTTTGTGATTGCCTTGAT
+
FGDGGGGGDGFFGGGDGGGGGGEEGAGFFE>A>@!B@?@@<:!!!!!!!!!!355=>><>EEEEAEEE?EEEBEE?!!!!!!!!!!!!!!!!!!!!!!!!
```


## grep

Usage

```
search sequences by pattern(s) of name or sequence motifs

Attentions:
    1. Unlike POSIX/GNU grep, we compare the pattern to the whole target
       (ID/full header) by default. Please switch "-r/--use-regexp" on
       for partly matching.
    2. While when searching by sequences, it's partly matching. And mismatch
       is allowed using flag "-m/--max-mismatch".
    3. The order of sequences in result is consistent with that in original
       file, not the order of the query patterns.

You can specify the sequence region for searching with flag -R (--region).
The definition of region is 1-based and with some custom design.

Examples:

 1-based index    1 2 3 4 5 6 7 8 9 10
negative index    0-9-8-7-6-5-4-3-2-1
           seq    A C G T N a c g t n
           1:1    A
           2:4      C G T
         -4:-2                c g t
         -4:-1                c g t n
         -1:-1                      n
          2:-2      C G T N a c g t
          1:-1    A C G T N a c g t n
          1:12    A C G T N a c g t n
        -12:-1    A C G T N a c g t n

Usage:
  seqkit grep [flags]

Flags:
  -n, --by-name               match by full name instead of just id
  -s, --by-seq                match by seq
  -d, --degenerate            pattern/motif contains degenerate base
      --delete-matched        delete matched pattern to speedup
  -i, --ignore-case           ignore case
  -v, --invert-match          invert the sense of matching, to select non-matching records
  -m, --max-mismatch int      max mismatch when matching by seq (experimental, costs too much RAM for large genome, 8G for 50Kb sequence)
  -p, --pattern stringSlice   search pattern (multiple values supported)
  -f, --pattern-file string   pattern file
  -R, --region string         specify sequence region for searching. e.g 1:12 for first 12 bases, -12:-1 for last 12 bases
  -r, --use-regexp            patterns are regular expression

```

Examples

1. Extract human hairpins (i.e. sequences with name starting with `hsa`)

        $ zcat hairpin.fa.gz | seqkit grep -r -p ^hsa
        >hsa-let-7a-1 MI0000060 Homo sapiens let-7a-1 stem-loop
        UGGGAUGAGGUAGUAGGUUGUAUAGUUUUAGGGUCACACCCACCACUGGGAGAUAACUAU
        ACAAUCUACUGUCUUUCCUA
        >hsa-let-7a-2 MI0000061 Homo sapiens let-7a-2 stem-loop
        AGGUUGAGGUAGUAGGUUGUAUAGUUUAGAAUUACAUCAAGGGAGAUAACUGUACAGCCU
        CCUAGCUUUCCU

1. Remove human and mice hairpins.

        $ zcat hairpin.fa.gz | seqkit grep -r -p ^hsa -p ^mmu -v

1. Extract new entries by information from miRNA.diff.gz

    1. Get IDs of new entries.

            $ zcat miRNA.diff.gz | grep ^# -v | grep NEW | cut -f 2 > list
            $ more list
            cfa-mir-486
            cfa-mir-339-1
            pmi-let-7


    2. Extract by ID list file

            $ zcat hairpin.fa.gz | seqkit grep -f list > new.fa

1  Extract sequences containing AGGCG

        $ cat hairpin.fa.gz | seqkit grep -s -i -p aggcg


1  Extract sequences containing AGGCG (allow mismatch, **only for short (<50kb) sequences now**)

        $ time cat hairpin.fa.gz | seqkit grep -s -i -p aggcg | seqkit stats
        file  format  type  num_seqs  sum_len  min_len  avg_len  max_len
        -     FASTA   RNA      1,181  145,789       49    123.4    2,354

        real    0m0.070s
        user    0m0.107s
        sys     0m0.017s

        $ time cat hairpin.fa.gz | seqkit grep -s -i -p aggcg -m 1 | seqkit stats
        file  format  type  num_seqs    sum_len  min_len  avg_len  max_len
        -     FASTA   RNA     17,168  1,881,005       39    109.6    2,354

        real    0m4.655s
        user    0m4.753s

1. Extract sequences starting with AGGCG

        $ zcat hairpin.fa.gz | seqkit grep -s -r -i -p ^aggcg

1. Extract sequences with TTSAA (AgsI digest site) in SEQUENCE. Base S stands for C or G.

        $ zcat hairpin.fa.gz | seqkit grep -s -d -i -p TTSAA

    It's equal to but simpler than:

        $ zcat hairpin.fa.gz | seqkit grep -s -r -i -p TT[CG]AA

1. Specify sequence regions for searching. e.g., leading 30 bases.

        $ seqkit grep -s -R 1:30 -i -r -p GCTGG

## locate

Usage

```
locate subsequences/motifs, mismatch allowed

Motifs could be EITHER plain sequence containing "ACTGN" OR regular
expression like "A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)" for ORFs.
Degenerate bases like "RYMM.." are also supported by flag -d.

By default, motifs are treated as regular expression.
When flag -d given, regular expression may be wrong.
For example: "\w" will be wrongly converted to "\[AT]".

Usage:
  seqkit locate [flags]

Flags:
      --bed                       output in BED6 format
  -d, --degenerate                pattern/motif contains degenerate base
      --gtf                       output in GTF format
  -h, --help                      help for locate
  -i, --ignore-case               ignore case
  -m, --max-mismatch int          max mismatch when matching by seq (experimental, costs too much RAM for large genome, 8G for 50Kb sequence)
  -G, --non-greedy                non-greedy mode, faster but may miss motifs overlaping with others
  -P, --only-positive-strand      only search on positive strand
  -p, --pattern strings           pattern/motif (multiple values supported. Attention: use double quotation marks for patterns containing comma, e.g., -p '"A{2,}"')
  -f, --pattern-file string       pattern/motif file (FASTA format)
  -V, --validate-seq-length int   length of sequence to validate (0 for whole seq) (default 10000)
```

Examples

1. Locating subsequences (mismatch allowed)

        $ cat t.fa
        >seq
        agctggagctacc

        $ cat t.fa | seqkit locate -p agc
        seqID   patternName     pattern strand  start   end     matched
        seq     agc     agc     +       1       3       agc
        seq     agc     agc     +       7       9       agc
        seq     agc     agc     -       8       10      agc
        seq     agc     agc     -       2       4       agc

        $ cat t.fa | seqkit locate -p agc -m 1
        seqID   patternName     pattern strand  start   end     matched
        seq     agc     agc     +       1       3       agc
        seq     agc     agc     +       7       9       agc
        seq     agc     agc     +       11      13      acc
        seq     agc     agc     -       8       10      agc
        seq     agc     agc     -       2       4       agc


        $ cat t.fa | seqkit locate -p agc -m 2
        seqID   patternName     pattern strand  start   end     matched
        seq     agc     agc     +       1       3       agc
        seq     agc     agc     +       4       6       tgg
        seq     agc     agc     +       5       7       gga
        seq     agc     agc     +       7       9       agc
        seq     agc     agc     +       10      12      tac
        seq     agc     agc     +       11      13      acc
        seq     agc     agc     -       11      13      ggt
        seq     agc     agc     -       8       10      agc
        seq     agc     agc     -       6       8       ctc
        seq     agc     agc     -       5       7       tcc
        seq     agc     agc     -       2       4       agc

1. Locate ORFs.

        $ zcat hairpin.fa.gz | seqkit locate -i -p "A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)"
        seqID   patternName     pattern strand  start   end     matched
        cel-lin-4       A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)        A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)        +  136      AUGCUUCCGGCCUGUUCCCUGAGACCUCAAGUGUGA
        cel-mir-1       A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)        A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)        +  54       95      AUGGAUAUGGAAUGUAAAGAAGUAUGUAGAACGGGGUGGUAG
        cel-mir-1       A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)        A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)        -  43       51      AUGAUAUAG

1. Locate Motif.

        $ zcat hairpin.fa.gz | seqkit locate -i -d -p AUGGACUN
        seqID         patternName   pattern    strand   start   end   matched
        cel-mir-58a   AUGGACUN      AUGGACUN   +        81      88    AUGGACUG
        ath-MIR163    AUGGACUN      AUGGACUN   -        122     129   AUGGACUC

    Notice that `seqkit grep` only searches in positive strand, but `seqkit loate` could recognize both strand.


1. Output in `GTF` or `BED6` format, which you can use in `seqkit subseq`

        $ zcat hairpin.fa.gz | seqkit locate -i -d -p AUGGACUN --bed
        cel-mir-58a     80      88      AUGGACUN        0       +
        ath-MIR163      121     129     AUGGACUN        0       -

        $ zcat hairpin.fa.gz | seqkit locate -i -d -p AUGGACUN --gtf
        cel-mir-58a     SeqKit  location        81      88      0       +       .       gene_id "AUGGACUN";
        ath-MIR163      SeqKit  location        122     129     0       -       .       gene_id "AUGGACUN";

1. greedy mode (default)

         $ echo -e '>seq\nACGACGACGA' | seqkit locate -p ACGA | csvtk -t pretty
         seqID   patternName   pattern   strand   start   end   matched
         seq     ACGA          ACGA      +        1       4     ACGA
         seq     ACGA          ACGA      +        4       7     ACGA
         seq     ACGA          ACGA      +        7       10    ACGA

1. non-greedy mode (`-G`)

        $ echo -e '>seq\nACGACGACGA' | seqkit locate -p ACGA -G | csvtk -t pretty
        seqID   patternName   pattern   strand   start   end   matched
        seq     ACGA          ACGA      +        1       4     ACGA
        seq     ACGA          ACGA      +        7       10    ACGA

## duplicate

Usage

```
duplicate sequences N times

You may need "seqkit rename" to make the the sequence IDs unique.

Usage:
  seqkit duplicate [flags]

Aliases:
  duplicate, dup

Flags:
  -h, --help        help for duplicate
  -n, --times int   duplication number (default 1)

```

Examples

1. Data

        $ cat tests/hairpin.fa | seqkit head -n 1
        >cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop
        UACACUGUGGAUCCGGUGAGGUAGUAGGUUGUAUAGUUUGGAAUAUUACCACCGGUGAAC
        UAUGCAAUUUUCUACCUUACCGGAGACAGAACUCUUCGA

1. Duplicate 2 times

        $ cat tests/hairpin.fa | seqkit head -n 1 | seqkit duplicate -n 2
        >cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop
        UACACUGUGGAUCCGGUGAGGUAGUAGGUUGUAUAGUUUGGAAUAUUACCACCGGUGAAC
        UAUGCAAUUUUCUACCUUACCGGAGACAGAACUCUUCGA
        >cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop
        UACACUGUGGAUCCGGUGAGGUAGUAGGUUGUAUAGUUUGGAAUAUUACCACCGGUGAAC
        UAUGCAAUUUUCUACCUUACCGGAGACAGAACUCUUCGA

1. use `seqkit rename` to make the the sequence IDs unique.

        $ cat tests/hairpin.fa | seqkit head -n 1 | seqkit duplicate -n 2 | seqkit rename
        >cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop
        UACACUGUGGAUCCGGUGAGGUAGUAGGUUGUAUAGUUUGGAAUAUUACCACCGGUGAAC
        UAUGCAAUUUUCUACCUUACCGGAGACAGAACUCUUCGA
        >cel-let-7_2 cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop
        UACACUGUGGAUCCGGUGAGGUAGUAGGUUGUAUAGUUUGGAAUAUUACCACCGGUGAAC
        UAUGCAAUUUUCUACCUUACCGGAGACAGAACUCUUCGA

## rmdup

Usage

```
remove duplicated sequences by id/name/sequence

Usage:
  seqkit rmdup [flags]

Flags:
    -n, --by-name                by full name instead of just id
    -s, --by-seq                 by seq
    -D, --dup-num-file string    file to save number and list of duplicated seqs
    -d, --dup-seqs-file string   file to save duplicated seqs
    -i, --ignore-case            ignore case
    -m, --md5                    use MD5 instead of original seqs to reduce memory usage when comparing by seqs

```

Examples

Similar to `common`.

1. General use

        $ zcat hairpin.fa.gz | seqkit rmdup -s -o clean.fa.gz
        [INFO] 2226 duplicated records removed

        $ zcat reads_1.fq.gz | seqkit rmdup -s -o clean.fa.gz
        [INFO] 1086 duplicated records removed

1. Save duplicated sequences to file

        $ zcat hairpin.fa.gz | seqkit rmdup -s -i -m -o clean.fa.gz -d duplicated.fa.gz -D duplicated.detail.txt

        $ cat duplicated.detail.txt   # here is not the entire list
        3	hsa-mir-424, mml-mir-424, ppy-mir-424
        3	hsa-mir-342, mml-mir-342, ppy-mir-342
        2	ngi-mir-932, nlo-mir-932
        2	ssc-mir-9784-1, ssc-mir-9784-2

## common

Usage

```
find common sequences of multiple files by id/name/sequence

Note:
    1. 'seqkit common' is designed to support 2 and MORE files.
    2. For 2 files, 'seqkit grep' is much faster and consumes lesser memory:
         seqkit grep -f <(seqkit seq -n -i small.fq.gz) big.fq.gz # by seq ID
         seqkit grep -s -f <(seqkit seq -s small.fq.gz) big.fq.gz # by seq
    3. Some records in one file may have same sequences/IDs. They will ALL be
       retrieved if the sequence/ID was shared in multiple files.
       So the records number may be larger than that of the smallest file.

Usage:
  seqkit common [flags]

Flags:
  -n, --by-name       match by full name instead of just id
  -s, --by-seq        match by sequence
  -h, --help          help for common
  -i, --ignore-case   ignore case
  -m, --md5           use MD5 instead of original seqs to reduce memory usage when comparing by seqs

```

Examples

1. By ID (default)

        seqkit common file*.fa -o common.fasta

1. By full name

        seqkit common file*.fa -n -o common.fasta

1. By sequence

        seqkit common file*.fa -s -i -o common.fasta

1. By sequence (***for large sequences***)

        seqkit common file*.fa -s -i -o common.fasta --md5


## split

Usage

```
split sequences into files by name ID, subsequence of given region,
part size or number of parts.

Please use "seqkit split2" for paired- and single-end FASTQ.

The definition of region is 1-based and with some custom design.

Examples:

 1-based index    1 2 3 4 5 6 7 8 9 10
negative index    0-9-8-7-6-5-4-3-2-1
           seq    A C G T N a c g t n
           1:1    A
           2:4      C G T
         -4:-2                c g t
         -4:-1                c g t n
         -1:-1                      n
          2:-2      C G T N a c g t
          1:-1    A C G T N a c g t n
          1:12    A C G T N a c g t n
        -12:-1    A C G T N a c g t n

Usage:
  seqkit split [flags]

Flags:
  -i, --by-id              split squences according to sequence ID
  -p, --by-part int        split sequences into N parts
  -r, --by-region string   split squences according to subsequence of given region. e.g 1:12 for first 12 bases, -12:-1 for last 12 bases. type "seqkit split -h" for more examples
  -s, --by-size int        split sequences into multi parts with N sequences
  -d, --dry-run            dry run, just print message and no files will be created.
  -f, --force              overwrite output directory
  -h, --help               help for split
  -k, --keep-temp          keep tempory FASTA and .fai file when using 2-pass mode
  -m, --md5                use MD5 instead of region sequence in output file when using flag -r (--by-region)
  -O, --out-dir string     output directory (default value is $infile.split)
  -2, --two-pass           two-pass mode read files twice to lower memory usage. (only for FASTA format)

```

Examples

1. Split sequences into parts with at most 10000 sequences

        $ seqkit split hairpin.fa.gz -s 10000
        [INFO] split into 10000 seqs per file
        [INFO] write 10000 sequences to file: hairpin.fa.part_001.gz
        [INFO] write 10000 sequences to file: hairpin.fa.part_002.gz
        [INFO] write 8645 sequences to file: hairpin.fa.part_003.gz

1. Split sequences into 4 parts

        $ seqkit split hairpin.fa.gz -p 4
        [INFO] split into 4 parts
        [INFO] read sequences ...
        [INFO] read 28645 sequences
        [INFO] write 7162 sequences to file: hairpin.fa.part_001.gz
        [INFO] write 7162 sequences to file: hairpin.fa.part_002.gz
        [INFO] write 7162 sequences to file: hairpin.fa.part_003.gz
        [INFO] write 7159 sequences to file: hairpin.fa.part_004.gz


    ***To reduce memory usage when spliting big file, we should alwasy use flag `--two-pass`***

        $ seqkit split hairpin.fa.gz -p 4 -2
        [INFO] split into 4 parts
        [INFO] read and write sequences to tempory file: hairpin.fa.gz.fa ...
        [INFO] create and read FASTA index ...
        [INFO] read sequence IDs from FASTA index ...
        [INFO] 28645 sequences loaded
        [INFO] write 7162 sequences to file: hairpin.part_001.fa.gz
        [INFO] write 7162 sequences to file: hairpin.part_002.fa.gz
        [INFO] write 7162 sequences to file: hairpin.part_003.fa.gz
        [INFO] write 7159 sequences to file: hairpin.part_004.fa.gz

1. Split sequences by species. i.e. by custom IDs (first three letters)

        $ seqkit split hairpin.fa.gz -i --id-regexp "^([\w]+)\-" -2
        [INFO] split by ID. idRegexp: ^([\w]+)\-
        [INFO] read and write sequences to tempory file: hairpin.fa.gz.fa ...
        [INFO] create and read FASTA index ...
        [INFO] create FASTA index for hairpin.fa.gz.fa
        [INFO] read sequence IDs from FASTA index ...
        [INFO] 28645 sequences loaded
        [INFO] write 48 sequences to file: hairpin.id_cca.fa.gz
        [INFO] write 3 sequences to file: hairpin.id_hci.fa.gz
        [INFO] write 106 sequences to file: hairpin.id_str.fa.gz
        [INFO] write 1 sequences to file: hairpin.id_bkv.fa.gz
        ...

1. Split sequences by sequence region (for example, sequence barcode)

        $ seqkit split hairpin.fa.gz -r 1:3 -2
        [INFO] split by region: 1:3
        [INFO] read and write sequences to tempory file: hairpin.fa.gz.fa ...
        [INFO] read sequence IDs and sequence region from FASTA file ...
        [INFO] create and read FASTA index ...
        [INFO] write 463 sequences to file: hairpin.region_1:3_AUG.fa.gz
        [INFO] write 349 sequences to file: hairpin.region_1:3_ACU.fa.gz
        [INFO] write 311 sequences to file: hairpin.region_1:3_CGG.fa.gz

    **If region is too long, we could use falg `--md5`**,
    i.e. use MD5 instead of region sequence in output file.

    Sequence suffix could be defined as `-r -12:-1`

## split2

Usage

```
split sequences into files by part size or number of parts

This command supports FASTA and paired- or single-end FASTQ with low memory
occupation and fast speed.

The file extensions of output are automatically detected and created
according to the input files.

Usage:
  seqkit split2 [flags]

Flags:
  -p, --by-part int      split sequences into N parts
  -s, --by-size int      split sequences into multi parts with N sequences
  -f, --force            overwrite output directory
  -h, --help             help for split2
  -O, --out-dir string   output directory (default value is $infile.split)
  -1, --read1 string     read1 file
  -2, --read2 string     read2 file
```

Examples

1. Split sequences into parts with at most 10000 sequences

        $ seqkit split2 hairpin.fa.gz -s 10000 -f
        [INFO] split into 10000 seqs per file
        [INFO] write 10000 sequences to file: hairpin.fa.part_001.gz
        [INFO] write 10000 sequences to file: hairpin.fa.part_002.gz
        [INFO] write 8645 sequences to file: hairpin.fa.part_003.gz

1. Split sequences into 4 parts

        $ seqkit split hairpin.fa.gz -p 4 -f
        [INFO] split into 4 parts
        [INFO] read sequences ...
        [INFO] read 28645 sequences
        [INFO] write 7162 sequences to file: hairpin.fa.gz.split/hairpin.part_001.fa.gz
        [INFO] write 7162 sequences to file: hairpin.fa.gz.split/hairpin.part_002.fa.gz
        [INFO] write 7162 sequences to file: hairpin.fa.gz.split/hairpin.part_003.fa.gz
        [INFO] write 7159 sequences to file: hairpin.fa.gz.split/hairpin.part_004.fa.gz

1. For FASTQ files (paired-end)

        $ seqkit split2 -1 reads_1.fq.gz -2 reads_2.fq.gz -p 2 -O out -f
        [INFO] split seqs from reads_1.fq.gz and reads_2.fq.gz
        [INFO] split into 2 parts
        [INFO] write 1250 sequences to file: out/reads_2.part_001.fq.gz
        [INFO] write 1250 sequences to file: out/reads_2.part_002.fq.gz
        [INFO] write 1250 sequences to file: out/reads_1.part_001.fq.gz
        [INFO] write 1250 sequences to file: out/reads_1.part_002.fq.gz

1. For FASTA files (single-end)

        $ seqkit split2 -1 reads_1.fq.gz reads_2.fq.gz -p 2 -O out -f
        [INFO] flag -1/--read1 given, ignore: reads_2.fq.gz
        [INFO] split seqs from reads_1.fq.gz
        [INFO] split into 2 parts
        [INFO] write 1250 sequences to file: out/reads_1.part_001.fq.gz
        [INFO] write 1250 sequences to file: out/reads_1.part_002.fq.gz


        $ seqkit split2 reads_1.fq.gz -p 2 -O out -f
        [INFO] split seqs from reads_1.fq.gz
        [INFO] split into 2 parts
        [INFO] write 1250 sequences to file: out/reads_1.part_001.fq.gz
        [INFO] write 1250 sequences to file: out/reads_1.part_002.fq.gz


## sample

Usage

```
sample sequences by number or proportion.

Usage:
  seqkit sample [flags]

Flags:
  -n, --number int         sample by number (result may not exactly match)
  -p, --proportion float   sample by proportion
  -s, --rand-seed int      rand seed (default 11)
  -2, --two-pass           2-pass mode read files twice to lower memory usage. Not allowed when reading from stdin

```

Examples

1. Sample by proportion

        $ zcat hairpin.fa.gz | seqkit sample -p 0.1 -o sample.fa.gz
        [INFO] sample by proportion
        [INFO] 2814 sequences outputed

1. Sample by number

        $ zcat hairpin.fa.gz | seqkit sample -n 1000 -o sample.fa.gz
        [INFO] sample by number
        [INFO] 949 sequences outputed

    949 != 1000 ??? see [Effect of random seed on results of `seqkit sample`](http:bioinf.shenwei.me/seqkit/note/#effect-of-random-seed-on-results-of-seqkit-sample)

    ***To reduce memory usage when spliting big file, we could use flag `--two-pass`***

    ***We can also use `seqkit sample -p` followed with `seqkit head -n`:***

        $ zcat hairpin.fa.gz | seqkit sample -p 0.1 | seqkit head -n 1000 -o sample.fa.gz

1. Set rand seed to reproduce the result

        $ zcat hairpin.fa.gz | seqkit sample -p 0.1 -s 11

1. Most of the time, we could shuffle after sampling

        $ zcat hairpin.fa.gz | seqkit sample -p 0.1 | seqkit shuffle -o sample.fa.gz

Note that when sampling on FASTQ files, make sure using same random seed by
flag `-s` (`--rand-seed`)

## head

Usage

```
print first N FASTA/Q records

Usage:
  seqkit head [flags]

Flags:
  -n, --number int   print first N FASTA/Q records (default 10)

```

Examples

1. FASTA

        $ seqkit head -n 1 hairpin.fa.gz
        >cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop
        UACACUGUGGAUCCGGUGAGGUAGUAGGUUGUAUAGUUUGGAAUAUUACCACCGGUGAAC
        UAUGCAAUUUUCUACCUUACCGGAGACAGAACUCUUCGA

1. FASTQ

        $ seqkit head -n 1 reads_1.fq.gz
        @HWI-D00523:240:HF3WGBCXX:1:1101:2574:2226 1:N:0:CTGTAG
        TGAGGAATATTGGTCAATGGGCGCGAGCCTGAACCAGCCAAGTAGCGTGAAGGATGACTGCCCTACGGGTTGTAA
        +
        HIHIIIIIHIIHGHHIHHIIIIIIIIIIIIIIIHHIIIIIHHIHIIIIIGIHIIIIHHHHHHGHIHIIIIIIIII


## range

Usage

```
print FASTA/Q records in a range (start:end)

Usage:
  seqkit range [flags]

Flags:
  -h, --help           help for range
  -r, --range string   range. e.g., 1:12 for first 12 records (head -n 12), -12:-1 for last 12 records (tail -n 12)
```

Examples

1. leading N records (head)

        $ cat tests/hairpin.fa | seqkit head -n 100 | md5sum
        f65116af7d9298d93ba4b3d19077bbf1  -
        $ cat tests/hairpin.fa | seqkit range -r 1:100 | md5sum
        f65116af7d9298d93ba4b3d19077bbf1  -

1. last N records (tail)

        $ cat tests/hairpin.fa | seqkit range -r -100:-1 | seqkit stats
        file  format  type  num_seqs  sum_len  min_len  avg_len  max_len
        -     FASTA   RNA        100    8,656       58     86.6      172

1. Other ranges

        $ cat tests/hairpin.fa | seqkit range -r 101:150 | seqkit stats
        file  format  type  num_seqs  sum_len  min_len  avg_len  max_len
        -     FASTA   RNA         50    3,777       63     75.5       96

        $ cat tests/hairpin.fa | seqkit range -r -100:-2 | seqkit stats
        file  format  type  num_seqs  sum_len  min_len  avg_len  max_len
        -     FASTA   RNA         99    8,484       58     85.7      146


## replace

Usage

```
replace name/sequence by regular expression.

Note that the replacement supports capture variables.
e.g. $1 represents the text of the first submatch.
ATTENTION: use SINGLE quote NOT double quotes in *nix OS.

Examples: Adding space to all bases.

    seqkit replace -p "(.)" -r '$1 ' -s

Or use the \ escape character.

    seqkit replace -p "(.)" -r "\$1 " -s

more on: http://bioinf.shenwei.me/seqkit/usage/#replace

Special replacement symbols (only for replacing name not sequence):

        {nr}    Record number, starting from 1
        {kv}    Corresponding value of the key (captured variable $n) by key-value file,
                n can be specified by flag -I (--key-capt-idx) (default: 1)

Usage:
  seqkit replace [flags]

Flags:
  -s, --by-seq                 replace seq
  -h, --help                   help for replace
  -i, --ignore-case            ignore case
  -K, --keep-key               keep the key as value when no value found for the key (only for sequence name)
  -I, --key-capt-idx int       capture variable index of key (1-based) (default 1)
  -m, --key-miss-repl string   replacement for key with no corresponding value
  -k, --kv-file string         tab-delimited key-value file for replacing key with value when using "{kv}" in -r (--replacement) (only for sequence name)
      --nr-width int           minimum width for {nr} in flag -r/--replacement. e.g., formating "1" to "001" by --nr-width 3 (default 1)
  -p, --pattern string         search regular expression
  -r, --replacement string     replacement. supporting capture variables.  e.g. $1 represents the text of the first submatch. ATTENTION: for *nix OS, use SINGLE quote NOT double quotes or use the \ escape character. Record number is also supported by "{nr}".use ${1} instead of $1 when {kv} given!

```

Examples

1. Remove descriptions

        $ echo -e ">seq1 abc-123\nACGT-ACGT" | seqkit replace -p " .+"
        >seq1
        ACGT-ACGT

1. Replace "-" with "="

        $ echo -e ">seq1 abc-123\nACGT-ACGT" | seqkit replace -p "\-" -r '='
        >seq1 abc=123
        ACGT-ACGT

1. Remove gaps in sequences.

        $ echo -e ">seq1 abc-123\nACGT-ACGT" | seqkit replace -p " |-" -s
        >seq1 abc-123
        ACGTACGT

1. Add space to every base. **ATTENTION: use SINGLE quote NOT double quotes in *nix OS**

        $ echo -e ">seq1 abc-123\nACGT-ACGT" | seqkit replace -p "(.)" -r '$1 ' -s
        >seq1 abc-123
        A C G T - A C G T

1. Transpose sequence with [csvtk](https://github.com/shenwei356/csvtk)

        $ echo -e ">seq1\nACTGACGT\n>seq2\nactgccgt" | seqkit replace -p "(.)" -r     "\$1 " -s | seqkit seq -s -u | csvtk space2tab | csvtk -t transpose
        A       A
        C       C
        T       T
        G       G
        A       C
        C       C
        G       G
        T       T

1. Rename with number of record

        $ echo -e ">abc\nACTG\n>123\nATTT" |  seqkit replace -p .+ -r "seq_{nr}"
        >seq_1
        ACTG
        >seq_2
        ATTT

        $ echo -e ">abc\nACTG\n>123\nATTT" |  seqkit replace -p .+ -r "seq_{nr}" --nr-width 5
        >seq_00001
        ACTG
        >seq_00002
        ATTT

1. Replace key with value by key-value file

        $ more test.fa
        >seq1 name1
        CCCCAAAACCCCATGATCATGGATC
        >seq2 name2
        CCCCAAAACCCCATGGCATCATTCA
        >seq3 name3
        CCCCAAAACCCCATGTTGCTACTAG

        $ more alias.txt
        name0   ABC
        name1   123
        name3   Hello
        name4   World

        $ seqkit replace -p ' (.+)$' -r ' {kv}' -k alias.txt test.fa
        [INFO] read key-value file: alias.txt
        [INFO] 4 pairs of key-value loaded
        >seq1 123
        CCCCAAAACCCCATGATCATGGATC
        >seq2
        CCCCAAAACCCCATGGCATCATTCA
        >seq3 Hello
        CCCCAAAACCCCATGTTGCTACTAG

        $ seqkit replace -p ' (.+)$' -r ' {kv}' -k alias.txt test.fa --keep-key
        [INFO] read key-value file: alias.txt
        [INFO] 4 pairs of key-value loaded
        >seq1 123
        CCCCAAAACCCCATGATCATGGATC
        >seq2 name2
        CCCCAAAACCCCATGGCATCATTCA
        >seq3 Hello
        CCCCAAAACCCCATGTTGCTACTAG

1. convert fasta to genbank style


        >seq1
        TTTAAAGAGACCGGCGATTCTAGTGAAATCGAACGGGCAGGTCAATTTCCAACCAGCGAT
        GACGTAATAGATAGATACAAGGAAGTCATTTTTCTTTTAAAGGATAGAAACGGTTAATGC
        TCTTGGGACGGCGCTTTTCTGTGCATAACT
        >seq2
        AAGGATAGAAACGGTTAATGCTCTTGGGACGGCGCTTTTCTGTGCATAACTCGATGAAGC
        CCAGCAATTGCGTGTTTCTCCGGCAGGCAAAAGGTTGTCGAGAACCGGTGTCGAGGCTGT
        TTCCTTCCTGAGCGAAGCCTGGGGATGAACG

        $ cat seq.fa \
            | seqkit replace -s -p '(\w{10})' -r '$1 ' -w 66 \
            | perl -ne 'if (/^>/) {print; $n=1} \
                else {s/ \r?\n$/\n/; printf "%9d %s", $n, $_; $n+=60;}'
        >seq1
                1 TTTAAAGAGA CCGGCGATTC TAGTGAAATC GAACGGGCAG GTCAATTTCC AACCAGCGAT
               61 GACGTAATAG ATAGATACAA GGAAGTCATT TTTCTTTTAA AGGATAGAAA CGGTTAATGC
              121 TCTTGGGACG GCGCTTTTCT GTGCATAACT
        >seq2
                1 AAGGATAGAA ACGGTTAATG CTCTTGGGAC GGCGCTTTTC TGTGCATAAC TCGATGAAGC
               61 CCAGCAATTG CGTGTTTCTC CGGCAGGCAA AAGGTTGTCG AGAACCGGTG TCGAGGCTGT
              121 TTCCTTCCTG AGCGAAGCCT GGGGATGAAC G

## rename

Usage

```
rename duplicated IDs

Usage:
  seqkit rename [flags]

Flags:
  -n, --by-name   check duplication by full name instead of just id

```

Examples

```
$ echo -e ">a comment\nacgt\n>b comment of b\nACTG\n>a comment\naaaa"
>a comment
acgt
>b comment of b
ACTG
>a comment
aaaa
$ echo -e ">a comment\nacgt\n>b comment of b\nACTG\n>a comment\naaaa" | seqkit rename
>a comment
acgt
>b comment of b
ACTG
>a_2 a comment
aaaa
```

## restart

Usage

```
reset start position for circular genome

Examples

    $ echo -e ">seq\nacgtnACGTN"
    >seq
    acgtnACGTN

    $ echo -e ">seq\nacgtnACGTN" | seqkit restart -i 2
    >seq
    cgtnACGTNa

    $ echo -e ">seq\nacgtnACGTN" | seqkit restart -i -2
    >seq
    TNacgtnACG

Usage:
  seqkit restart [flags]

Flags:
  -i, --new-start int   new start position (1-base, supporting negative value counting from the end) (default 1)

```

## concat

Usage

```
concatenate sequences with same ID from multiple files

Example: concatenating leading 2 bases and last 2 bases

    $ cat t.fa
    >test
    ACCTGATGT
    >test2
    TGATAGCTACTAGGGTGTCTATCG

    $ seqkit concat <(seqkit subseq -r 1:2 t.fa) <(seqkit subseq -r -2:-1 t.fa)
    >test
    ACGT
    >test2
    TGCG

Usage:
  seqkit concat [flags]

Flags:
  -h, --help   help for concat


```


## shuffle

Usage

```
shuffle sequences.

By default, all records will be readed into memory.
For FASTA format, use flag -2 (--two-pass) to reduce memory usage. FASTQ not
supported.

Firstly, seqkit reads the sequence IDs. If the file is not plain FASTA file,
seqkit will write the sequences to tempory files, and create FASTA index.

Secondly, seqkit shuffles sequence IDs and extract sequences by FASTA index.

Usage:
  seqkit shuffle [flags]

Flags:
  -k, --keep-temp       keep tempory FASTA and .fai file when using 2-pass mode
  -s, --rand-seed int   rand seed for shuffle (default 23)
  -2, --two-pass        two-pass mode read files twice to lower memory usage. (only for FASTA format)

```

Examples

1. General use.

        $ seqkit shuffle hairpin.fa.gz > shuffled.fa
        [INFO] read sequences ...
        [INFO] 28645 sequences loaded
        [INFO] shuffle ...
        [INFO] output ...

1. ***For big genome, you'd better use two-pass mode*** so seqkit could use
   FASTA index to reduce memory usage

        $ time seqkit shuffle -2 hsa.fa > shuffle.fa
        [INFO] create and read FASTA index ...
        [INFO] create FASTA index for hsa.fa
        [INFO] read sequence IDs from FASTA index ...
        [INFO] 194 sequences loaded
        [INFO] shuffle ...
        [INFO] output ...

        real    0m35.080s
        user    0m45.521s
        sys     0m3.411s

Note that when sampling on FASTQ files, make sure using same random seed by
flag `-s` (`--rand-seed`) for read 1 and 2 files.

## sort

Usage

```
sort sequences by id/name/sequence/length.

By default, all records will be readed into memory.
For FASTA format, use flag -2 (--two-pass) to reduce memory usage. FASTQ not
supported.

Firstly, seqkit reads the sequence head and length information.
If the file is not plain FASTA file,
seqkit will write the sequences to tempory files, and create FASTA index.

Secondly, seqkit sort sequence by head and length information
and extract sequences by FASTA index.

Usage:
  seqkit sort [flags]

Flags:
  -l, --by-length               by sequence length
  -n, --by-name                 by full name instead of just id
  -s, --by-seq                  by sequence
  -i, --ignore-case             ignore case
  -k, --keep-temp               keep tempory FASTA and .fai file when using 2-pass mode
  -N, --natural-order           sort in natural order, when sorting by IDs/full name
  -r, --reverse                 reverse the result
  -L, --seq-prefix-length int   length of sequence prefix on which seqkit sorts by sequences (0 for whole sequence) (default 10000)
  -2, --two-pass                two-pass mode read files twice to lower memory usage. (only for FASTA format)

```

Examples

***For FASTA format, use flag -2 (--two-pass) to reduce memory usage***

1. sort by ID

        $ echo -e ">seq1\nACGTNcccc\n>SEQ2\nacgtnAAAA" | seqkit sort --quiet
        >SEQ2
        acgtnAAAA
        >seq1
        ACGTNcccc

1. sort by ID and in natural order

        $ echo -e ">3\na\n>1\na\n>Y\na\n>x\na\n>Mt\na\n>11\na\n>2\na\n"
        >3
        a
        >1
        a
        >Y
        a
        >x
        a
        >Mt
        a
        >11
        a
        >2
        a

        $ echo -e ">3\na\n>1\na\n>Y\na\n>x\na\n>Mt\na\n>11\na\n>2\na\n" | ./seqkit sort -N -i -2
        >1
        a
        >2
        a
        >3
        a
        >11
        a
        >Mt
        a
        >x
        a
        >Y
        a

1. sort by ID, ignoring case.

        $ echo -e ">seq1\nACGTNcccc\n>SEQ2\nacgtnAAAA" | seqkit sort --quiet -i
        >seq1
        ACGTNcccc
        >SEQ2
        acgtnAAAA

1. sort by seq, ignoring case.

        $ echo -e ">seq1\nACGTNcccc\n>SEQ2\nacgtnAAAA" | seqkit sort --quiet -s -i
        >SEQ2
        acgtnAAAA
        >seq1
        ACGTNcccc

1. sort by sequence length

        $ echo -e ">seq1\nACGTNcccc\n>SEQ2\nacgtnAAAAnnn\n>seq3\nacgt" | seqkit sort --quiet -l
        >seq3
        acgt
        >seq1
        ACGTNcccc
        >SEQ2
        acgtnAAAAnnn

## genautocomplete

Usage

```
generate shell autocompletion script

Note: The current version supports Bash only.
This should work for *nix systems with Bash installed.

Howto:

1. run: seqkit genautocomplete

2. create and edit ~/.bash_completion file if you don't have it.

        nano ~/.bash_completion

   add the following:

        for bcfile in ~/.bash_completion.d/* ; do
          . $bcfile
        done

Usage:
  seqkit genautocomplete [flags]

Flags:
      --file string   autocompletion file (default "/home/shenwei/.bash_completion.d/seqkit.sh")
  -h, --help          help for genautocomplete
      --type string   autocompletion type (currently only bash supported) (default "bash")

```

## translate

Usage

```
translate mRNA sequences into amino acid sequences (protein).

Usage:
  seqkit translate [flags]

Flags:

```

Examples

1. translate mRNA sequence

        $ seqkit translate -u ../tests/mrna-drosopholia-DQ327736.fa
        RSGRTRKAPGRRRRRQKAQPAGRRARPSKARRSPPASAPATAPPRARE

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
