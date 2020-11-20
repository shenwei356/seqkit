# FAQ on FASTA/Q manipulations

**This page was originally written as the section [FASTA/Q manipulations](https://read.biostarhandbook.com/data/fastq-manipulation.html) of [The Biostar Handbook: A Beginner's Guide to Bioinformatics](https://read.biostarhandbook.com/) ([discussion on Biostars.org](https://www.biostars.org/p/225812/))**.

----

This page illustrates common FASTA/Q manipulations using
[SeqKit](http://bioinf.shenwei.me/seqkit/).
Some other utilities, including [csvtk](http://bioinf.shenwei.me/csvtk/) (CSV/TSV toolkit) and shell commands were also used.

Note: SeqKit seamlessly support FASTA and FASTQ formats both in their original form or
in stored in gzipped compressed format. We list FASTA or FASTQ depending on the more common
usage but you can always use it on the other type as well.


<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->


- [Example data](#example-data)
- [How to produce an overview of FASTQ files?](#how-to-produce-an-overview-of-fastq-files)
- [How to get GC content of every sequence in FASTA/Q file?](#how-to-get-gc-content-of-every-sequence-in-fastaq-file)
- [How to extract sequences subset from FASTA/Q file with name/ID list file?](#how-to-extract-sequences-subset-from-fastaq-file-with-nameid-list-file)
- [How to find FASTA/Q sequences containing degenerate bases and locate them?](#how-to-find-fastaq-sequences-containing-degenerate-bases-and-locate-them)
- [How to remove duplicated FASTA/Q records with same sequences?](#how-to-remove-duplicated-fastaq-records-with-same-sequences)
- [How to locate motif/subsequence/enzyme digest site in FASTA/Q sequence?](#how-to-locate-motifsubsequenceenzyme-digest-site-in-fastaq-sequence)
- [How to sort huge number of FASTA sequences by length?](#how-to-sort-huge-number-of-fasta-sequences-by-length)
- [How to split FASTA sequences according to information in header?](#how-to-split-fasta-sequences-according-to-information-in-header)
- [How to search and replace FASTA header with known character strings from a text file?](#how-to-search-and-replace-fasta-header-with-known-character-strings-from-a-text-file)
- [How to extract paired reads from two paired-end reads file?](#how-to-extract-paired-reads-from-two-paired-end-reads-file)
- [How to concatenate two FASTA sequences in to one?](#how-to-concatenate-two-fasta-sequences-in-to-one)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Example data

One FASTQ file (sample reads, 1M) and two FASTA files (Virus DNA and protein
sequences from NCBI RefSeq database, 60+40M) are used.

    wget http://data.biostarhandbook.com/reads/duplicated-reads.fq.gz
    wget ftp://ftp.ncbi.nih.gov/refseq/release/viral/viral.1.1.genomic.fna.gz
    wget ftp://ftp.ncbi.nih.gov/refseq/release/viral/viral.1.protein.faa.gz

## How to produce an overview of FASTQ files?

Sequence format and type are automatically detected.

    $ seqkit stat *.gz
    file                      format  type     num_seqs      sum_len  min_len   avg_len    max_len
    duplicated-reads.fq.gz    FASTQ   DNA        15,000    1,515,000      101       101        101
    viral.1.1.genomic.fna.gz  FASTA   DNA         7,048  203,325,120      200  28,848.6  2,473,870
    viral.1.protein.faa.gz    FASTA   Protein   252,611   62,024,702        5     245.5      8,960

## How to get GC content of every sequence in FASTA/Q file?

`seqkit fx2tab` converts FASTA/Q to 3-column tabular format (1th: name/ID,
2nd: sequence, 3rd: quality), and can also provides
various information in new columns, including sequence length, GC content/GC skew, alphabet.

GC content:

    $ seqkit fx2tab --name --only-id --gc viral*.fna.gz
    gi|526245010|ref|NC_021865.1|                   40.94
    gi|460042095|ref|NC_020479.1|                   41.82
    gi|526244636|ref|NC_021857.1|                   49.17

Custom bases (A, C and A+C):

    $ seqkit fx2tab -H -n -i -B a -B c -B ac viral.1.1.genomic.fna.gz
    #name   seq     qual    a       c       ac
    gi|526245010|ref|NC_021865.1|                   33.20   18.24   51.44
    gi|460042095|ref|NC_020479.1|                   32.57   19.63   52.20
    gi|526244636|ref|NC_021857.1|                   25.52   23.06   48.59

## How to extract sequences subset from FASTA/Q file with name/ID list file?

This is a frequently used manipulation. Let's create a sample ID list file,
which may also come from other way like mapping result.

    $ seqkit sample --proportion 0.001  duplicated-reads.fq.gz \
        | seqkit seq --name --only-id > id.txt

ID list file:

    $ head id.txt
    SRR1972739.2996
    SRR1972739.3044
    SRR1972739.3562

Searching by ID list file:

    $ seqkit grep --pattern-file id.txt duplicated-reads.fq.gz \
        > duplicated-reads.subset.fq.gz

## How to find FASTA/Q sequences containing degenerate bases and locate them?

`seqkit fx2tab` converts FASTA/Q to tabular format and can output the sequence
alphabet in a new column. And then text searching tools can be used to filter
the table.

    $ seqkit fx2tab -n -i -a viral.1.1.genomic.fna.gz \
        | csvtk -H -t grep -f 4 -r -i -p "[^ACGT]"
    gi|446730228|ref|NC_019782.1|                   ACGNT
    gi|557940284|ref|NC_022800.1|                   ACGKT
    gi|564292828|ref|NC_023009.1|                   ACGNT

Long-option version of the command:

    $ seqkit fx2tab --name --only-id --alphabet  viral.1.1.genomic.fna.gz \
        | csvtk --no-header-row --tabs grep --fields 4 --use-regexp --ignore-case --pattern "[^ACGT]"

You can then exclude these sequences with `seqkit grep`:

    # save the sequenece IDs.
    $ seqkit fx2tab -n -i -a viral.1.1.genomic.fna.gz \
        | csvtk -H -t grep -f 4 -r -i -p "[^ACGT]" | csvtk -H -t cut -f 1 > id2.txt

    # search and exclude.
    $ seqkit grep --pattern-file id2.txt --invert-match viral.1.1.genomic.fna.gz > clean.fa

Or locate the degenerate bases, e.g, `N` and `K`

    $ seqkit grep --pattern-file id2.txt viral.1.1.genomic.fna.gz \
        | seqkit locate --ignore-case --only-positive-strand --pattern K+ --pattern N+
    seqID   patternName     pattern strand  start   end     matched
    gi|564292828|ref|NC_023009.1|   N+      N+      +       87972   87972   N
    gi|564292828|ref|NC_023009.1|   N+      N+      +       100983  100983  N
    gi|557307918|ref|NC_022755.1|   K+      K+      +       1788    1788    K
    gi|557307918|ref|NC_022755.1|   K+      K+      +       4044    4044    K
    gi|589287065|ref|NC_023585.1|   K+      K+      +       28296   28296   K
    gi|590911929|ref|NC_023639.1|   N+      N+      +       741654  741753  NNNNNNNNNNNNNNNNNNNNNNNNNNN

## How to remove duplicated FASTA/Q records with same sequences?

    $ seqkit rmdup --by-seq --ignore-case duplicated-reads.fq.gz > duplicated-reads.uniq.fq.gz

If the FASTA/Q file is very large, please switch on flag `-m/--md5`,
which use MD5 instead of original seqs to reduce memory usage
when comparing by sequences.

You can also deduplicate according to sequence ID (default) or
full name (`--by-name`).

## How to locate motif/subsequence/enzyme digest site in FASTA/Q sequence?

Related posts:
[Question: Count and location of strings in fastq file reads](https://www.biostars.org/p/204658/),
[Question: Finding TATAWAA in sequence](https://www.biostars.org/p/221325/)
.

Assuming a list of motifs (enzyme digest sites) in FASTA format to be located:

    $ cat enzymes.fa
    >EcoRI
    GAATTC
    >MmeI
    TCCRAC
    >SacI
    GAGCTC
    >XcmI
    CCANNNNNNNNNTGG

Flag `--degenerate` is on because patterns contain degenerate bases. Command:

    $ seqkit locate --degenerate --ignore-case --pattern-file enzymes.fa viral.1.1.genomic.fna.gz

Sample output (simplified and reformated by `csvtk -t uniq -f 3 | csvtk -t pretty`)

    seqID                           patternName   pattern           strand   start   end     matched
    gi|526245010|ref|NC_021865.1|   MmeI          TCCRAC            +        1816    1821    TCCGAC
    gi|526245010|ref|NC_021865.1|   SacI          GAGCTC            +        19506   19511   GAGCTC
    gi|526245010|ref|NC_021865.1|   XcmI          CCANNNNNNNNNTGG   +        2221    2235    CCATATTTAGTGTGG

## How to sort huge number of FASTA sequences by length?

Sorting FASTA file in order of sequence size (small to large).

    $ seqkit sort --by-length viral.1.1.genomic.fna.gz > viral.1.1.genomic.sorted.fa

If the files are too big, use flag `--two-pass` which consumes lesser memory.

    $ seqkit sort --by-length --two-pass viral.1.1.genomic.fna.gz > viral.1.1.genomic.sorted.fa

You can also sort by sequence ID (default), full header (`--by-name`) or
sequence content (`--by-seq`).

## How to split FASTA sequences according to information in header?

Related posts:
[Question: extract same all similar sequences in FASTA based on the header](https://www.biostars.org/p/223937/)
.

For example, FASTA header line of `viral.1.protein.faa.gz` contain species name
in square brackets.

Overview of FASTA Headers:

    $ seqkit head -n 3 viral.1.protein.faa.gz | seqkit seq --name
    gi|526245011|ref|YP_008320337.1| terminase small subunit [Paenibacillus phage phiIBB_Pl23]
    gi|526245012|ref|YP_008320338.1| terminase large subunit [Paenibacillus phage phiIBB_Pl23]
    gi|526245013|ref|YP_008320339.1| portal protein [Paenibacillus phage phiIBB_Pl23]

`seqkit split` can split FASTA/Q files according to ID, number of parts, size
of every parts, and sequence region. In this case, we'll split according to
sequence ID (species names) which can be specified by flag `--id-regexp`.

Default ID:

    $ seqkit head -n 3 viral.1.protein.faa.gz \
        | seqkit seq --name --only-id
    gi|526245011|ref|YP_008320337.1|
    gi|526245012|ref|YP_008320338.1|
    gi|526245013|ref|YP_008320339.1|

New ID:

    $ seqkit head -n 3 viral.1.protein.faa.gz \
        | seqkit seq --name --only-id --id-regexp "\[(.+)\]"
    Paenibacillus phage phiIBB_Pl23
    Paenibacillus phage phiIBB_Pl23
    Paenibacillus phage phiIBB_Pl23

Split:

    $ seqkit split --by-id --id-regexp "\[(.+)\]" viral.1.protein.faa.gz

## How to search and replace FASTA header with known character strings from a text file?

Related posts:
[Question: Replace names in FASTA file with a known character string from a text file](https://www.biostars.org/p/221962/),
[Question: Fasta header, search and replace...?](https://www.biostars.org/p/205044/)
.

`seqKit replace` can find substrings in FASTA/Q header with regular expression
and replace them with strings or corresponding values of found substrings
provided by the tab-delimited key-value file.

For example, to unify names of protein with unknown functions, we want to
rename "hypothetical" to "putative" in lower case.
The replacing rules are listed below in tab-delimited file:

    $ cat changes.tsv
    Hypothetical    putative
    hypothetical    putative
    Putative        putative

Overview the FASTA headers containing "hypothetical":

    $ seqkit grep --by-name --use-regexp --ignore-case --pattern hypothetical viral.1.protein.faa.gz \
        | seqkit head -n 3 | seqkit seq --name
    gi|526245016|ref|YP_008320342.1| hypothetical protein IBBPl23_06 [Paenibacillus phage phiIBB_Pl23]
    gi|526245019|ref|YP_008320345.1| hypothetical protein IBBPl23_09 [Paenibacillus phage phiIBB_Pl23]
    gi|526245020|ref|YP_008320346.1| hypothetical protein IBBPl23_10 [Paenibacillus phage phiIBB_Pl23]

A regular expression, `^([^ ]+ )(\w+) `, was used to specify the key to be
replaced, which is the first word after sequence ID in this case. Note that we also
capture the ID (`^([^ ]+ )`) so we can restore it in "replacement" with
capture variable  `${1}` (robuster than `$1`).
And flag `-I/--key-capt-idx` (default: 1) is set to 2 because the key `(\w+)`
is the second captured match. Command:

    $ seqkit replace --kv-file changes.tsv --pattern "^([^ ]+ )(\w+) " \
        --replacement "\${1}{kv} " --key-capt-idx 2 --keep-key viral.1.protein.faa.gz > renamed.fa

## How to extract paired reads from two paired-end reads file?

Use [seqkit pair](https://bioinf.shenwei.me/seqkit/usage/#pair).

## How to concatenate two FASTA sequences in to one?

Related posts: [Combining two fasta sequences into one](https://www.biostars.org/p/231806/)

Data (not in same order):

    $ cat 1.fa
    >seq1
    aaaaa
    >seq2
    ccccc
    >seq3
    ggggg

    $ cat 2.fa
    >seq3
    TTTTT
    >seq2
    GGGGG
    >seq1
    CCCCC

Just one command:

    $ seqkit concat 1.fa 2.fa
    >seq1
    aaaaaCCCCC
    >seq2
    cccccGGGGG
    >seq3
    gggggTTTTT
