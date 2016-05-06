# Benchmark


## Softwares

1. [fakit](https://github.com/shenwei356/fakit). (Go).
   Version [v0.2.3](https://github.com/shenwei356/fakit/releases/tag/v0.2.3).
1. [fasta_utilities](https://github.com/jimhester/fasta_utilities). (Perl).
   Version [3dcc0bc](https://github.com/jimhester/fasta_utilities/tree/3dcc0bc6bf1e97839476221c26984b1789482579).
   Lots of dependencies to install.
1. [fastx_toolkit](http://hannonlab.cshl.edu/fastx_toolkit/). (Perl).
   Version [0.0.13](http://hannonlab.cshl.edu/fastx_toolkit/fastx_toolkit_0.0.13_binaries_Linux_2.6_amd64.tar.bz2).
   Can't handle multi-line FASTA files.
1. [seqmagick](http://seqmagick.readthedocs.io/en/latest/index.html#installation). (Python).
   Version 0.6.1
1. [seqtk](https://github.com/lh3/seqtk). (C).
   Version [1.1-r92-dirty](https://github.com/lh3/seqtk/tree/fb85aad4ce1fc7b3d4543623418a1ae88fe1cea6).

Not used:

1. [pyfaidx](https://github.com/mdshw5/pyfaidx). (Python).
   Version [0.4.7.1](https://pypi.python.org/packages/source/p/pyfaidx/pyfaidx-0.4.7.1.tar.gz#md5=f33604a3550c2fa115ac7d33b952127d). *Not used, because it exhausted my memory (10G) when computing reverse-complement on a 5GB fasta file of 250 bp.*

A Python script [memusg](https://github.com/shenwei356/memusg) was used
to compute running time and peak memory usage of a process.

## Features

Features         | fakit    | fasta_utilities | fastx_toolkit | pyfaidx | seqmagick | seqtk
:--------------- | :------: | :-------------: | :-----------: | :-----: | :-------: | :----
Cross-platform   | Yes      | Partly          | Partly        | Yes     | Yes       | Yes
Mutli-line FASTA | Yes      | Yes             | --            | Yes     | Yes       | Yes
Read FASTQ       | Yes      | Yes             | Yes           | --      | Yes       | Yes
Mutli-line FASTQ | Yes      | Yes             | --            | --      | Yes       | Yes
Validate bases   | Yes      | --              | Yes           | Yes     | --        | --
Recognize RNA    | Yes      | Yes             | --            | --      | Yes       | Yes
Read STDIN       | Yes      | Yes             | Yes           | --      | Yes       | Yes
Read gzip        | Yes      | Yes             | --            | --      | Yes       | Yes
Write gzip       | Yes      | --              | --            | --      | Yes       | --
Search by motifs | Yes      | Yes             | --            | --      | Yes       | --
Sample seqs      | Yes      | --              | --            | --      | Yes       | Yes
Subseq           | Yes      | Yes             | --            | Yes     | Yes       | Yes
Deduplicate seqs | Yes      | --              | --            | --      | Partly    | --
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
Rename head      | Yes      | Yes             | --            | --      | Yes       | Yes


## Datasets

All test data is available here: [fakit-benchmark-data.tar.gz](http://bioinf.shenwei.me/fakit-benchmark-data.tar.gz)  (1.7G)

### dataset_A.fa - large number of short sequences

Dataset A is reference genomes DNA sequences of gastrointestinal tract from
[NIH Human Microbiome Project](http://hmpdacc.org/):
[`Gastrointestinal_tract.nuc.fsa`](http://downloads.hmpdacc.org/data/reference_genomes/body_sites/Gastrointestinal_tract.nuc.fsa) (FASTA format, ~2.7G).

### dataset_B.fa - small number of large sequences

Dataset B is Human genome from [ensembl](http://uswest.ensembl.org/info/data/ftp/index.html).

- Genome DNA:  [`Homo_sapiens.GRCh38.dna_sm.primary_assembly.fa.gz`](ftp://ftp.ensembl.org/pub/release-84/fasta/homo_sapiens/dna/Homo_sapiens.GRCh38.dna_sm.primary_assembly.fa.gz) (Gzipped FASTA file, ~900M)
- GTF file:  [`Homo_sapiens.GRCh38.84.gtf.gz`](ftp://ftp.ensembl.org/pub/release-84/gtf/homo_sapiens/Homo_sapiens.GRCh38.84.gtf.gz) (~44M)
- BED file: `Homo_sapiens.GRCh38.84.bed.gz` was converted from `Homo_sapiens.GRCh38.84.gtf.gz` by  [`gtf2bed`](http://bedops.readthedocs.org/en/latest/content/reference/file-management/conversion/gtf2bed.html?highlight=gtf2bed)  with command

        $ zcat Homo_sapiens.GRCh38.84.gtf.gz | gtf2bed --do-not-sort | gzip -c > Homo_sapiens.GRCh38.84.bed.gz

Summary

    $ fakit stat *.fa
    file           seq_format   seq_type   num_seqs   min_len        avg_len       max_len
    dataset_A.fa   FASTA        DNA          67,748        56       41,442.5     5,976,145
    dataset_B.fa   FASTA        DNA             194       970   15,978,096.5   248,956,422

### Sequence ID list

Parts of sequences IDs was sampled and shuffled from original data.
They were used in test of extracting sequences by ID list.

Commands:

    $ fakit sample -p 0.3 dataset_A.fa | fakit seq --name --only-id | shuf > ids_A.txt
    $ fakit sample -p 0.3 dataset_B.fa | fakit seq --name --only-id | shuf > ids_B.txt

Numbers:

    $ wc -l ids_*
    20138 ids_A.txt
       58 ids_B.txt

### BED file

Only BED data of chromosome 19 was used in test of subsequence with BED file:

    $ zcat Homo_sapiens.GRCh38.84.bed.gz | grep -E "^19" | gzip -c > chr19.bed.gz


## Platform

PC:

- CPU: Intel Core i5-3320M @ 2.60GHz, two cores/4 threads
- RAM: DDR3 1600MHz, 12GB
- SSD: SAMSUNG 850 EVO 250G, SATA-3
- OS: Fedora 23 (Scientific KDE spin),  Kernal: 4.4.7-300.fc23.x86_64

Softwares:

- Perl: perl 5, version 22, subversion 1 (v5.22.1) built for x86_64-linux-thread-multi
- Python: Python 2.7.10 (default, Sep  8 2015, 17:20:17) [GCC 5.1.1 20150618 (Red Hat 5.1.1-4)] on linux2

## Tests

Automatic benchmark and plotting scripts are available at:  [https://github.com/shenwei356/fakit/tree/master/benchmark](https://github.com/shenwei356/fakit/tree/master/benchmark).

All tests were repeated 3 times ( ~20 min for one time),
and average time and peak memory ware used for plotting.

All data were readed once before tests began to minimize the influence of page cache.

### Test 1. Reverse Complement

Output sequences of all Softwares were not wrapped to fixed length.

`revcom_biogo` ([source](https://github.com/shenwei356/fakit/blob/master/benchmark/revcom_biogo.go),
 [binary](https://github.com/shenwei356/fakit/blob/master/benchmark/revcom_biogo?raw=true) ),
 a tool written in Golang using [biogo](https://github.com/biogo/biogo) package,
 was also used for comparison of FASTA file parsing performance.

[Commands](https://github.com/shenwei356/fakit/blob/master/benchmark/run_benchmark_01_revcom.sh)

### Test 2. Extracting sequences by ID list

[Commands](https://github.com/shenwei356/fakit/blob/master/benchmark/run_benchmark_02_exctact_by_id_list.sh)

### Test 3. Sampling by number

Note that different softwares have different sampling strategies,
the peak memory may depends on size of sampled sequences.

[Commands](https://github.com/shenwei356/fakit/blob/master/benchmark/run_benchmark_03_sampling.sh)

### Test 4. Removing duplicates by sequence content

[Commands](https://github.com/shenwei356/fakit/blob/master/benchmark/run_benchmark_04_remove_duplicated_seqs_by_seq.sh)

### Test 5. Subsequence with BED file

[Commands](https://github.com/shenwei356/fakit/blob/master/benchmark/run_benchmark_05_subseq_with_bed.sh)

## Results

![benchmark-5tests.csv.png](benchmark/benchmark.5tests.csv.png)

### Test of multiple threads:

From the results, 2 threads/CPU is enough, so the default threads of fakit is 2.

![benchmark-5tests.csv.png](benchmark/fakit_multi_threads/benchmark.5tests.csv.png)

### Tests on different file sizes

Files are generated by replicating Human genome chr1 for N times.

![benchmark.fakit.files_size.csv.png](benchmark/fakit_file_size/benchmark.fakit.files_size.csv.png)

<div id="disqus_thread"></div>
<script>
/**
* RECOMMENDED CONFIGURATION VARIABLES: EDIT AND UNCOMMENT THE SECTION BELOW TO INSERT DYNAMIC VALUES FROM YOUR PLATFORM OR CMS.
* LEARN WHY DEFINING THESE VARIABLES IS IMPORTANT: https://disqus.com/admin/universalcode/#configuration-variables
*/
/*
var disqus_config = function () {
this.page.url = PAGE_URL; // Replace PAGE_URL with your page's canonical URL variable
this.page.identifier = PAGE_IDENTIFIER; // Replace PAGE_IDENTIFIER with your page's unique identifier variable
};
*/
(function() { // DON'T EDIT BELOW THIS LINE
var d = document, s = d.createElement('script');

s.src = '//fastakit.disqus.com/embed.js';

s.setAttribute('data-timestamp', +new Date());
(d.head || d.body).appendChild(s);
})();
</script>
<noscript>Please enable JavaScript to view the <a href="https://disqus.com/?ref_noscript" rel="nofollow">comments powered by Disqus.</a></noscript>
