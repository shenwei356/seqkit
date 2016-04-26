# Benchmark

Datasets and results are described at [http://shenwei356.github.io/fakit/benchmark](http://shenwei356.github.io/fakit/benchmark)

**The benchmark needs be performed in Linux-like operating systems.**

## Install softwares

Softwares

1. [fakit](https://github.com/shenwei356/fakit). (Go).
   Version [v0.1.9](https://github.com/shenwei356/fakit/releases/tag/v0.1.9).
1. [fasta_utilities](https://github.com/jimhester/fasta_utilities). (Perl).
   Version [3dcc0bc](https://github.com/jimhester/fasta_utilities/tree/3dcc0bc6bf1e97839476221c26984b1789482579).
   Lots of dependencies to install_.
1. [fastx_toolkit](http://hannonlab.cshl.edu/fastx_toolkit/). (Perl).
   Version [0.0.13](http://hannonlab.cshl.edu/fastx_toolkit/fastx_toolkit_0.0.13_binaries_Linux_2.6_amd64.tar.bz2).
   Can't handle multi-line FASTA files_.
1. [seqmagick](http://seqmagick.readthedocs.org/en/latest/index.html). (Python).
   Version 0.6.1
1. [seqtk](https://github.com/lh3/seqtk). (C).
   Version [1.0-r82-dirty](https://github.com/lh3/seqtk/commit/4feb6e81444ab6bc44139dd3a125068f81ae4ad8).

Not used:

1. [pyfaidx](https://github.com/mdshw5/pyfaidx). (Python).
   Version [0.4.7.1](https://pypi.python.org/packages/source/p/pyfaidx/pyfaidx-0.4.7.1.tar.gz#md5=f33604a3550c2fa115ac7d33b952127d). *Not used, because it 

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

## Data preparation

[http://blog.shenwei.me/fakit/benchmark/#datasets](http://blog.shenwei.me/fakit/benchmark/#datasets)

## Run tests

```
Usage:

1. Run all tests:

perl run_benchmark_00_all.pl run_*.sh

2. Run one test:

perl run_benchmark_00_all.pl run_benchmark_04_remove_duplicated_seqs_by_name.sh

3. Custom repeating times:

perl run_benchmark_00_all.pl -n 5 run_benchmark_04_remove_duplicated_seqs_by_name.sh
```

## Plot result

Before this, you need to run

    perl run_benchmark_00_all.pl run_*.sh

R libraries `dplyr`, `ggplot2`, `scales`, `ggthemes`, `ggrepel` are needed.

Run:

    ./plot.R
