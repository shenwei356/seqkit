# Benchmark

Datasets and results are described at [http://shenwei356.github.io/fakit/benchmark](http://shenwei356.github.io/fakit/benchmark)

***The benchmark needs be performed in Linux-like operating systems.***

## Install softwares

Softwares

1. [fakit](https://github.com/shenwei356/fakit). (Go).
   Version [v0.2.1](https://github.com/shenwei356/fakit/releases/tag/v0.2.1).
1. [fasta_utilities](https://github.com/jimhester/fasta_utilities). (Perl).
   Version [3dcc0bc](https://github.com/jimhester/fasta_utilities/tree/3dcc0bc6bf1e97839476221c26984b1789482579).
   Lots of dependencies to install_.
1. [fastx_toolkit](http://hannonlab.cshl.edu/fastx_toolkit/). (Perl).
   Version [0.0.13](http://hannonlab.cshl.edu/fastx_toolkit/fastx_toolkit_0.0.13_binaries_Linux_2.6_amd64.tar.bz2).
   Can't handle multi-line FASTA files_.
1. [seqmagick](http://seqmagick.readthedocs.io/en/latest/index.html#installation). (Python).
   Version 0.6.1
1. [seqtk](https://github.com/lh3/seqtk). (C).
   Version [1.1-r92-dirty](https://github.com/lh3/seqtk/tree/fb85aad4ce1fc7b3d4543623418a1ae88fe1cea6).


A Python script [memusg](https://github.com/shenwei356/memusg) was used
   to computate running time and peak memory usage of a process.

**Attention**: the `fasta_utilities` uses Perl module `Term-ProgressBar`
which makes it failed to run when using benchmark script `run_benchmark_00_all.pl`.
Please change the source code of ProgressBar.pm (for me, the path is
/usr/share/perl5/vendor_perl/Term/ProgressBar.pm). Add the code below after line `535`:

    $config{bar_width} = 1 if $config{bar_width} < 1;

The edited code is

    } else {
      $config{bar_width}  = $target;
      $config{bar_width} = 1 if $config{bar_width} < 1;   # new line
      die "configured bar_width $config{bar_width} < 1"
      if $config{bar_width} < 1;
    }

## Clone this repository

    git clone https://github.com/shenwei356/fakit
    cd fakit/benchmark

## Data preparation

[http://shenwei356.github.io/fakit/benchmark/#datasets](http://shenwei356.github.io/fakit/benchmark/#datasets)

Or download all test data [fakit-benchmark-data.tar.gz](http://bioinf.shenwei.me/fakit-benchmark-data.tar.gz)
 (1.7G) and uncompress it, and then move them into directory `fakit/benchmark`

    wget ***
    tar -zxvf fakit-benchmark-data.tar.gz
    mv fakit-benchmark-data/* fakit/benchmark

## Run tests

A Perl scripts
[`run.pl`](https://github.com/shenwei356/fakit/blob/master/benchmark/run_benchmark_00_all.pl)
is used to automatically running tests and generate data for plotting.

```
$ perl run.pl -h
Usage:

1. Run all tests:

perl run.pl run_benchmark*.sh --outfile benchmark.5test.csv

2. Run one test:

perl run.pl run_benchmark_04_remove_duplicated_seqs_by_name.sh -o benchmark.rmdup.csv

3. Custom repeate times:

perl run.pl -n 3 run_benchmark_04_remove_duplicated_seqs_by_name.sh -o benchmark.rmdup.csv
```

To compare performance between different softwares, run:

    ./run.pl run_benchmark*.sh -n 3 -o benchmark.5tests.csv

It costed ~50min for me.

To test performance of other functions in fakit, run:

    ./run.pl run_test*.sh -n 1 -o benchmark.fakit.csv

## Plot result

R libraries `dplyr`, `ggplot2`, `scales`, `ggthemes`, `ggrepel` are needed.

Plot for result of the five tests:

    ./plot.R -i benchmark.5tests.csv

Plot for result of the tests of other functions in fakit:

    ./plot.R -i benchmark.fakit.csv --width 5 --height 3
