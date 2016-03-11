# Usage and Examples

## fakit

Usage

```
fakit -- FASTA kit

Version: 0.1.1

Author: Wei Shen <shenwei356(at)gmail.com>

Documents  : http://shenwei356.github.io/fakit
Source code: https://github.com/shenwei356/fakit

Usage:
  fakit [command]

Available Commands:
  common      find common sequences of multiple files by id/name/sequence
  extract     extract sequences by pattern(s) of name or sequence motifs
        covert FASTA to tabular format, and provide various information
  locate      locate subsequences/motifs
  rmdup       remove duplicated sequences by id/name/sequence
  sample      sample sequences by number or proportion
  seq         transform sequence (revserse, complement, extract ID...)
  shuffle     shuffle sequences
  sliding     sliding sequences, circle genome supported
  split       split sequences into files by id/seq region/size/parts
  stat        simple statistics of FASTA files
  subseq      get subsequence by region
  tab2fa      covert tabular format to FASTA format

Flags:
  -c, --chunk-size int     chunk size (attention: unit is FASTA records not lines) (default 1000)
      --id-regexp string   regular expression for parsing ID (default "^([^\\s]+)\\s?")
  -w, --line-width int     line width (0 for no wrap) (default 60)
  -o, --out-file string    out file ("-" for stdout, suffix .gz for gzipped out) (default "-")
      --quiet              be quiet and do not show extra information
  -t, --seq-type string    sequence type (dna|rna|protein|unlimit|auto) (for auto, it automatically detect by the first sequence) (default "auto")
  -j, --threads int        number of CPUs (default value depends on your device) (default 4)

Use "fakit [command] --help" for more information about a command.
```

### Datasets

Datasets are from [The miRBase Sequence Database -- Release 21](ftp://mirbase.org/pub/mirbase/21/)

- [hairpin.fa.gz](ftp://mirbase.org/pub/mirbase/21/hairpin.fa.gz)
- [mature.fa.gz](ftp://mirbase.org/pub/mirbase/21/mature.fa.gz)
- [miRNA.diff.gz](ftp://mirbase.org/pub/mirbase/21/miRNA.diff.gz)


## seq

Usage

```
transform sequence (revserse, complement, extract ID...)

Usage:
  fakit seq [flags]

Flags:
  -p, --complement          complement sequence (blank for Protein sequence)
  -G, --gap-letter string   gap letters (default "-")
  -l, --lower-case          print sequences in lower case
  -n, --name                only print names
  -i, --only-id             print ID instead of full head
  -g, --remove-gaps         remove gaps
  -r, --reverse             reverse sequence)
  -s, --seq                 only print sequences
  -u, --upper-case          print sequences in upper case
```

Examples

1. Read and print

    - From file:

            $ fakit seq hairpin.fa.gz
            >cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop
            UACACUGUGGAUCCGGUGAGGUAGUAGGUUGUAUAGUUUGGAAUAUUACCACCGGUGAAC
            UAUGCAAUUUUCUACCUUACCGGAGACAGAACUCUUCGA

    - From stdin:

            zcat hairpin.fa.gz | fakit seq

1. Only print names

    - Full name:

            $ fakit seq hairpin.fa.gz -n
            cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop
            cel-lin-4 MI0000002 Caenorhabditis elegans lin-4 stem-loop
            cel-mir-1 MI0000003 Caenorhabditis elegans miR-1 stem-loop

    - Only ID:

            $ fakit seq hairpin.fa.gz -n -i
            cel-let-7
            cel-lin-4
            cel-mir-1

    - Custom ID region by regular expression (this could be applied to all subcommands):

            $ fakit seq hairpin.fa.gz -n -i --id-regexp "^[^\s]+\s([^\s]+)\s"
            MI0000001
            MI0000002
            MI0000003

1. Only print seq (global flag -w defines the output line width, 0 for no wrap)

        $ fakit seq hairpin.fa.gz -s -w 0
        UACACUGUGGAUCCGGUGAGGUAGUAGGUUGUAUAGUUUGGAAUAUUACCACCGGUGAACUAUGCAAUUUUCUACCUUACCGGAGACAGAACUCUUCGA
        AUGCUUCCGGCCUGUUCCCUGAGACCUCAAGUGUGAGUGUACUAUUGAUGCUUCACACCUGGGCUCUCCGGGUACCAGGACGGUUUGAGCAGAU
        AAAGUGACCGUACCGAGCUGCAUACUUCCUUACAUGCCCAUACUAUAUCAUAAAUGGAUAUGGAAUGUAAAGAAGUAUGUAGAACGGGGUGGUAGU

1. Reverse comlement sequence

        $ fakit seq hairpin.fa.gz -r -p
        >cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop
        UCGAAGAGUUCUGUCUCCGGUAAGGUAGAAAAUUGCAUAGUUCACCGGUGGUAAUAUUCC
        AAACUAUACAACCUACUACCUCACCGGAUCCACAGUGUA

1. Remove gaps and to lower/upper case

        $ echo -e ">seq\nACGT-ACTGC-ACC" | fakit seq -i -g
        >seq
        ACGTACTGCACC

## subseq

Usage

```
get subsequence by region.

The definition of region is 1-based and with some custom design.

Examples:

 0-based index    0 1 2 3 4 5 6 7 8 9
 1-based index    1 2 3 4 5 6 7 8 9 10
negative index    0-9-8-7-6-5-4-3-2-1
           seq    A C G T N a c g t n
           1:1    A
           2:4        G T N
         -4:-2                c g t
         -4:-1                c g t n
         -1:-1                      n
          2:-2      C G T N a c g t
          1:-1    A C G T N a c g t n

Usage:
  fakit subseq [flags]

Flags:
  -r, --region string   subsequence of given region. e.g 1:12 for first 12 bases, -12:-1 for last 12 bases, 13:-1 for cutting first 12 bases. type "fakit subseq -h" for more examples
```

Examples

1. first 12 bases

        $ zcat hairpin.fa.gz | fakit subseq -r 1:12

1. last 12 bases

        $ zcat hairpin.fa.gz | fakit subseq -r -12:-1

1. subsequence without first and last 12 bases

        $ zcat hairpin.fa.gz | fakit subseq -r 13:-13

## stat

Usage

```
simple statistics of FASTA files

Usage:
  fakit stat [flags]
```

Eexamples

1. General use

        $ fakit stat *.fa.gz
        file    type    num_seqs        min_len avg_len max_len
        hairpin.fa.gz   RNA     28645   39      103.0   2354
        mature.fa.gz    RNA     35828   15      21.8    34

## fa2tab & fa2tab

Usage (fa2tab)

```
covert FASTA to tabular format, and provide various information,
like sequence length, GC content/GC skew.

Usage:
  fakit fa2tab [flags]

Flags:
  -b, --base-content value   print base content. (case ignored, multiple values supported) e.g. -b AT -b N (default [])
  -g, --gc                   print GC content
  -G, --gc-skew              print GC-Skew
  -l, --length               print sequence length
  -n, --name                 only print names (no sequences)
  -i, --only-id              print ID instead of full head
  -T, --title                print title line

```

Usage (tab2fa)

```
covert tabular format (first two columns) to FASTA format

Usage:
  fakit tab2fa [flags]

Flags:
  -p, --comment-line-prefix value   comment line prefix (default [#,//])
```

Examples

1. Default output

        $ fakit fa2tab hairpin.fa.gz
        cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop      UACACUGUGGAUCCGGUGAGGUAGUAGGUUGUAUAGUUUGGAAUAUUACCACCGGUGAACUAUGCAAUUUUCUACCUUACCGGAGACAGAACUCUUCGA
        cel-lin-4 MI0000002 Caenorhabditis elegans lin-4 stem-loop      AUGCUUCCGGCCUGUUCCCUGAGACCUCAAGUGUGAGUGUACUAUUGAUGCUUCACACCUGGGCUCUCCGGGUACCAGGACGGUUUGAGCAGAU
        cel-mir-1 MI0000003 Caenorhabditis elegans miR-1 stem-loop      AAAGUGACCGUACCGAGCUGCAUACUUCCUUACAUGCCCAUACUAUAUCAUAAAUGGAUAUGGAAUGUAAAGAAGUAUGUAGAACGGGGUGGUAGU

1. Print sequence length, GC content, and only print names (no sequences),
we could also print title line by flag `-T`.

        $ fakit fa2tab hairpin.fa.gz -i -l -g -n -T
        # name  seq     length  GC
        cel-let-7               99      43.43
        cel-lin-4               94      54.26
        cel-mir-1               96      40.62


1. Use fa2tab and tab2fa in pipe

        $ zcat hairpin.fa.gz | fakit fa2tab | fakit tab2fa

1. Sort sequences by length

        $ zcat hairpin.fa.gz | fakit fa2tab -l | sort -t"`echo -e '\t'`" -n -k3,3 | fakit tab2fa
        >cin-mir-4129 MI0015684 Ciona intestinalis miR-4129 stem-loop
        UUCGUUAUUGGAAGACCUUAGUCCGUUAAUAAAGGCAUC
        >mmu-mir-7228 MI0023723 Mus musculus miR-7228 stem-loop
        UGGCGACCUGAACAGAUGUCGCAGUGUUCGGUCUCCAGU
        >cin-mir-4103 MI0015657 Ciona intestinalis miR-4103 stem-loop
        ACCACGGGUCUGUGACGUAGCAGCGCUGCGGGUCCGCUGU

    Sorting or filtering by GC (or other base by -flag `-b`) content could also achieved in similar way.

1. Get first 1000 sequence

        $ zcat hairpin.fa.gz | fakit fa2tab | head -n 1000 | fakit tab2fa

## extract

Usage

```
extract sequences by pattern(s) of name or sequence motifs

Usage:
  fakit extract [flags]

Flags:
  -n, --by-name               match by full name instead of just id
  -s, --by-seq                match by seq
  -d, --degenerate            pattern/motif contains degenerate base
      --delete-matched        delete matched pattern to speedup
  -i, --ignore-case           ignore case
  -v, --invert-match          invert the sense of matching, to select non-matching records
  -p, --pattern value         search pattern (multiple values supported) (default [])
  -f, --pattern-file string   pattern file
  -r, --use-regexp            patterns are regular expression
```

Examples

1. Extract human hairpins (i.e. sequences with name starting with `hsa`)

        $ zcat hairpin.fa.gz | fakit extract -r -p ^hsa
        >hsa-let-7a-1 MI0000060 Homo sapiens let-7a-1 stem-loop
        UGGGAUGAGGUAGUAGGUUGUAUAGUUUUAGGGUCACACCCACCACUGGGAGAUAACUAU
        ACAAUCUACUGUCUUUCCUA
        >hsa-let-7a-2 MI0000061 Homo sapiens let-7a-2 stem-loop
        AGGUUGAGGUAGUAGGUUGUAUAGUUUAGAAUUACAUCAAGGGAGAUAACUGUACAGCCU
        CCUAGCUUUCCU

1. Remove human and mice hairpins.

        $ zcat hairpin.fa.gz | fakit extract -r -p ^hsa -p ^mmu -v

1. Extract new entries by information from miRNA.diff.gz

    1. Get IDs of new entries.

            $ zcat miRNA.diff.gz | grep ^# -v | grep NEW | cut -f 2 > list
            $ more list
            cfa-mir-486
            cfa-mir-339-1
            pmi-let-7


    2. Extract by ID list file

            $ zcat hairpin.fa.gz | fakit extract -f list > new.fa

1. Extract sequences starting with AGGCG

        $ zcat hairpin.fa.gz | fakit extract -s -r -i -p ^aggcg

1. Extract sequences with TTSAA (AgsI digest site) in SEQUENCE. Base S stands for C or G.

        $ zcat hairpin.fa.gz | fakit extract -s -d -i -p TTSAA

    It's equal to but simpler than:

        $ zcat hairpin.fa.gz | fakit extract -s -r -i -p TT[CG]AA

## common

Usage

```
find common sequences of multiple files by id/name/sequence

Usage:
  fakit common [flags]

Flags:
  -n, --by-name       match by full name instead of just id
  -s, --by-seq        match by sequence
  -i, --ignore-case   ignore case
```

Examples

1. By ID (default)

        fakit common file*.fa > common.fasta

2. By full name

        fakit common file*.fa -n > common.fasta

2. By sequence

        fakit common file*.fa -s > common.fasta

## rmdup

Usage

```
remove duplicated sequences by id/name/sequence

Usage:
  fakit rmdup [flags]

Flags:
  -n, --by-name       by full name instead of just id
  -s, --by-seq        by seq
  -i, --ignore-case   ignore case
```

Examples

Similar to `common`.

1. General use

        $ zcat hairpin.fa.gz | fakit rmdup -s -o clean.fa.gz
        [INFO] 2226 duplicated records removed


## split

Usage

```
split sequences into files by name ID, subsequence of given region,
part size or number of parts.

The definition of region is 1-based and with some custom design.

Examples:

 0-based index    0 1 2 3 4 5 6 7 8 9
 1-based index    1 2 3 4 5 6 7 8 9 10
negative index    0-9-8-7-6-5-4-3-2-1
           seq    A C G T N a c g t n
           1:1    A
           2:4        G T N
         -4:-2                c g t
         -4:-1                c g t n
         -1:-1                      n
          2:-2      C G T N a c g t
          1:-1    A C G T N a c g t n

Usage:
  fakit split [flags]

Flags:
  -i, --by-id              split squences according to sequence ID
  -p, --by-part int        split squences into N parts
  -r, --by-region string   split squences according to subsequence of given region. e.g 1:12 for first 12 bases, -12:-1 for last 12 bases. type "fakit split -h" for more example
  -s, --by-size int        split squences into multi parts with N sequences
  -d, --dry-run            dry run, just print message and no files will be created.
  -m, --md5                use MD5 instead of region sequence in output file when using flag -r (--by-region)
  -2, --two-pass           2-pass mode read files twice to lower memory usage. Not allowed when reading from stdin
```

Examples

1. Split sequences into parts with at most 10000 sequences

        $ fakit split hairpin.fa.gz -s 10000
        [INFO] split into 10000 seqs per file
        [INFO] write 10000 sequences to file: hairpin.fa.part_001.gz
        [INFO] write 10000 sequences to file: hairpin.fa.part_002.gz
        [INFO] write 8645 sequences to file: hairpin.fa.part_003.gz

1. Split sequences into 4 parts

        $ fakit split hairpin.fa.gz -p 4
        [INFO] split into 4 parts
        [INFO] read sequences ...
        [INFO] read 28645 sequences
        [INFO] write 7162 sequences to file: hairpin.fa.part_001.gz
        [INFO] write 7162 sequences to file: hairpin.fa.part_002.gz
        [INFO] write 7162 sequences to file: hairpin.fa.part_003.gz
        [INFO] write 7159 sequences to file: hairpin.fa.part_004.gz


    To reduce memory usage when spliting big file, we could use flag `--two-pass`

        $ fakit split hairpin.fa.gz -p 4 -2
        [INFO] split into 4 parts
        [INFO] first pass: get seq number
        [INFO] seq number: 28645
        [INFO] second pass: read and split
        [INFO] write 7162 sequences to file: hairpin.fa.part_001.gz
        [INFO] write 7162 sequences to file: hairpin.fa.part_002.gz
        [INFO] write 7162 sequences to file: hairpin.fa.part_003.gz
        [INFO] write 7159 sequences to file: hairpin.fa.part_004.gz

1. Split sequences by species. i.e. by custom IDs (first three letters)

        $ fakit split hairpin.fa.gz -i --id-regexp "^([\w]+)\-"
        [INFO] split by ID. idRegexp: ^([\w]+)\-
        [INFO] read sequences ...
        [INFO] read 28645 sequences
        [INFO] write 97 sequences to file: hairpin.fa.id_asu.gz
        [INFO] write 267 sequences to file: hairpin.fa.id_chi.gz
        [INFO] write 296 sequences to file: hairpin.fa.id_gra.gz
        ...

1. Split sequences by sequence region (for example, sequence barcode)

        $ fakit split hairpin.fa.gz -r 1:12
        [INFO] split by region: ^([^\s]+)\s?
        [INFO] read sequences ...
        [INFO] read 28645 sequences
        [INFO] write 1 sequences to file: hairpin.fa.region_1:12_UGUUUGCUCAGC.gz
        [INFO] write 1 sequences to file: hairpin.fa.region_1:12_GAAGAAGAAGAC.gz
        [INFO] write 4 sequences to file: hairpin.fa.region_1:12_UGAGUGUAGUGC.gz

    If region is too long, we could use falg `-m`, i.e. use MD5 instead of region sequence in output file.

    Sequence suffix could be defined as `-r -12:-1`

## sample

Usage

```
sample sequences by number or proportion.

Usage:
  fakit sample [flags]

Flags:
  -n, --number int         sample by number (result may not exactly match)
  -p, --proportion float   sample by proportion
  -s, --rand-seed int      rand seed for shuffle (default 11)
  -2, --two-pass           2-pass mode read files twice to lower memory usage. Not allowed when reading from stdin
```

Examples

1. Sample by number

        $ zcat hairpin.fa.gz | fakit sample -n 1000 -o sample.fa.gz
        [INFO] sample by number
        [INFO] 949 sequences outputed

    To reduce memory usage when spliting big file, we could use flag `--two-pass`

1. Sample by proportion

        $ zcat hairpin.fa.gz | fakit sample -p 0.1 -o sample.fa.gz
        [INFO] sample by proportion
        [INFO] 2814 sequences outputed

1. Set rand seed to reproduce the result

        $ zcat hairpin.fa.gz | fakit sample -p 0.1 -s 11

1. Most of the time, we could shuffle after sampling

        $ zcat hairpin.fa.gz | fakit sample -p 0.1 | fakit shuffle -o sample.fa.gz

## shuffle

Usage

```
shuffle sequences

Usage:
  fakit shuffle [flags]

Flags:
  -s, --rand-seed int   rand seed for shuffle (default 23)
```

Examples

1. General use.

        $ zcat hairpin.fa.gz | fakit shuffle -o shuffled.fa.gz
        [INFO] read sequences ...
        [INFO] 28645 sequences loaded
        [INFO] shuffle ...
        [INFO] output ...

## locate

Usage

```
locate subsequences/motifs

Motifs could be EITHER plain sequence containing "ACTGN" OR regular
expression like "A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)" for ORFs.
Degenerate bases like "RYMM.." are also supported by flag -d.

In default, motifs are treated as regular expression.
When flag -d given, regular expression may be wrong.
For example: "\w" will be wrongly converted to "\[AT]".

Usage:
  fakit locate [flags]

Flags:
  -d, --degenerate             pattern/motif contains degenerate base
  -i, --ignore-case            ignore case
  -P, --only-positive-strand   only search at positive strand
  -p, --pattern value          search pattern/motif (multiple values supported) (default [])
  -f, --pattern-file string    pattern/motif file (FASTA format)
```

Examples

1. Locate ORFs.

        $ zcat hairpin.fa.gz | fakit locate -i -p "A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)"
        seqID   patternName     pattern strand  start   end     matched
        cel-lin-4       A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)        A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)        1  136      AUGCUUCCGGCCUGUUCCCUGAGACCUCAAGUGUGA
        cel-mir-1       A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)        A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)        1  54       95      AUGGAUAUGGAAUGUAAAGAAGUAUGUAGAACGGGGUGGUAG
        cel-mir-1       A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)        A[TU]G(?:.{3})+?[TU](?:AG|AA|GA)        -1 43       51      AUGAUAUAG

1. Locate Motif.

        $ zcat hairpin.fa.gz | fakit locate -i -p UUS
        seqID   patternName     pattern strand  start   end     matched
        bna-MIR396a     UUS     UUS     -1      105     107     UUS
        bna-MIR396a     UUS     UUS     -1      89      91      UUS

    Notice that `fakit extract` only searches in positive strand, but `fakit loate` could recogize both strand

## sliding

Usage

```
sliding sequences, circle genome supported

Usage:
  fakit sliding [flags]

Flags:
  -C, --circle-genome   circle genome
  -s, --step int        step size
  -W, --window int      window size
```

Examples

1. General use

        $ echo -e ">seq\nACGTacgtNN" | fakit sliding -s 3 -W 6
        >seq sliding:1-6
        ACGTac
        >seq sliding:4-9
        TacgtN

2. Circle genome

        $ echo -e ">seq\nACGTacgtNN" | fakit sliding -s 3 -W 6 -C
        >seq sliding:1-6
        ACGTac
        >seq sliding:4-9
        TacgtN
        >seq sliding:7-2
        gtNNAC
        >seq sliding:10-5
        NACGTa

3. Generate GC content for ploting

        $ zcat hairpin.fa.gz | fakit fa2tab | head -n 1 | fakit tab2fa | fakit sliding -s 5 -W 30 | fakit fa2tab  -n -g
        cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop sliding:1-30         50.00
        cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop sliding:6-35         46.67
        cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop sliding:11-40                43.33
        cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop sliding:16-45                36.67
        cel-let-7 MI0000001 Caenorhabditis elegans let-7 stem-loop sliding:21-50                33.33
        ...
