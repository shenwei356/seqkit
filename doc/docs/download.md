# Download

`faskit` is implemented in [Golang](https://golang.org/) programming language,
 executable binary files **for most popular operating system** are freely available
  in [release](https://github.com/shenwei356/faskit/releases) page.

## Latest Version

[faskit v0.2.8](https://github.com/shenwei356/faskit/releases/tag/v0.2.8)

#### Links

- **Linux**
    - [faskit_linux_386.tar.gz](https://github.com/shenwei356/faskit/releases/download/v0.2.8/faskit_linux_386.tar.gz)
    - [faskit_linux_amd64.tar.gz](https://github.com/shenwei356/faskit/releases/download/v0.2.8/faskit_linux_amd64.tar.gz)
    - [faskit_linux_arm.tar.gz](https://github.com/shenwei356/faskit/releases/download/v0.2.8/faskit_linux_arm.tar.gz)
- **Mac OS X**
    - [faskit_darwin_386.tar.gz](https://github.com/shenwei356/faskit/releases/download/v0.2.8/faskit_darwin_386.tar.gz)
    - [faskit_darwin_amd64.tar.gz](https://github.com/shenwei356/faskit/releases/download/v0.2.8/faskit_darwin_amd64.tar.gz)
- **Windows**
    - [faskit_windows_386.exe.tar.gz](https://github.com/shenwei356/faskit/releases/download/v0.2.8/faskit_windows_386.exe.tar.gz)
    - [faskit_windows_amd64.exe.tar.gz](https://github.com/shenwei356/faskit/releases/download/v0.2.8/faskit_windows_amd64.exe.tar.gz)
- **FreeBSD**
    - [faskit_freebsd_386.tar.gz](https://github.com/shenwei356/faskit/releases/download/v0.2.8/faskit_freebsd_386.tar.gz)
    - [faskit_freebsd_amd64.tar.gz](https://github.com/shenwei356/faskit/releases/download/v0.2.8/faskit_freebsd_amd64.tar.gz)
    - [faskit_freebsd_arm.tar.gz](https://github.com/shenwei356/faskit/releases/download/v0.2.8/faskit_freebsd_arm.tar.gz)
- **OpenBSD**
    - [faskit_openbsd_386.tar.gz](https://github.com/shenwei356/faskit/releases/download/v0.2.8/faskit_openbsd_386.tar.gz)
    - [faskit_openbsd_amd64.tar.gz](https://github.com/shenwei356/faskit/releases/download/v0.2.8/faskit_openbsd_amd64.tar.gz)

#### Mirror site for Chinese user

- **Linux**
    - [faskit_linux_386.tar.gz](http://app.shenwei.me/data/faskit_linux_386.tar.gz)
    - [faskit_linux_amd64.tar.gz](http://app.shenwei.me/data/faskit_linux_amd64.tar.gz)
    - [faskit_linux_arm.tar.gz](http://app.shenwei.me/data/faskit_linux_arm.tar.gz)
- **Mac OS X**
    - [faskit_darwin_386.tar.gz](http://app.shenwei.me/data/faskit_darwin_386.tar.gz)
    - [faskit_darwin_amd64.tar.gz](http://app.shenwei.me/data/faskit_darwin_amd64.tar.gz)
- **Windows**
    - [faskit_windows_386.exe.tar.gz](http://app.shenwei.me/data/faskit_windows_386.exe.tar.gz)
    - [faskit_windows_amd64.exe.tar.gz](http://app.shenwei.me/data/faskit_windows_amd64.exe.tar.gz)
- **FreeBSD**
    - [faskit_freebsd_386.tar.gz](http://app.shenwei.me/data/faskit_freebsd_386.tar.gz)
    - [faskit_freebsd_amd64.tar.gz](http://app.shenwei.me/data/faskit_freebsd_amd64.tar.gz)
    - [faskit_freebsd_arm.tar.gz](http://app.shenwei.me/data/faskit_freebsd_arm.tar.gz)
- **OpenBSD**
    - [faskit_openbsd_386.tar.gz](http://app.shenwei.me/data/faskit_openbsd_386.tar.gz)
    - [faskit_openbsd_amd64.tar.gz](http://app.shenwei.me/data/faskit_openbsd_amd64.tar.gz)

## Installation

Just [download](https://github.com/shenwei356/faskit/releases) compressed
executable file of your operating system, and uncompress it with `tar -zxvf *.tar.gz` command.

You can add the directory of the executable file to environment variable
`PATH`, so you can run `faskit` anywhere.

1. For windows, the simplest way is copy it to `C:\WINDOWS\system32`.

2. For Linux, type:

        chmod a+x /PATH/OF/FAKIT/faskit
        echo export PATH=\$PATH:/PATH/OF/faskit >> ~/.bashrc

    or simply copy it to `/usr/local/bin`

For Go developer, just one command:

    go get -u github.com/shenwei356/faskit/faskit

## Release History

- [faskit v0.2.8](https://github.com/shenwei356/faskit/releases/tag/v0.2.8)
    - improve speed of parsing sequence ID, not using regular expression for default `--id-regexp`
    - improve speed of record outputing for small-size sequences
    - fix minor bug: `faskit seq` for blank record
    - update benchmark result
- [faskit v0.2.7](https://github.com/shenwei356/faskit/releases/tag/v0.2.7)
    - ***reduce memory usage*** by optimize the outputing of sequences.
      detail: using [`BufferedByteSliceWrapper`](https://godoc.org/github.com/shenwei356/util/byteutil#BufferedByteSliceWrapper) to resuse bytes.Buffer.
    - ***reduce memory usage and improve speed*** by using custom buffered
     reading mechanism, instead of using standard library `bufio`,
      which is slow for large genome sequence.
    - discard strategy of "buffer" and "chunk" of FASTA/Q records,
      just parse records one by one.
    - delete global flags `-c (--chunk-size)` and `-b (--buffer-size)`.
    - add function testing scripts
- [faskit v0.2.6](https://github.com/shenwei356/faskit/releases/tag/v0.2.6)
    - fix bug of `faskit subseq`: Inplace subseq method leaded to wrong result
- [faskit v0.2.5.1](https://github.com/shenwei356/faskit/releases/tag/v0.2.5.1)
    - fix a bug of `faskit subseq`: chromesome name was not be converting to lower case when using `--gtf` or `--bed`
- [faskit v0.2.5](https://github.com/shenwei356/faskit/releases/tag/v0.2.5)
    - fix a serious bug brought in `v0.2.3`, using unsafe method to convert `string` to `[]byte`
    - add awk-like built-in variable of record number (`{NR}`) for `faskit replace`
- [faskit v0.2.4.1](https://github.com/shenwei356/faskit/releases/tag/v0.2.4.1)
    - fix several bugs from library `bio`, affected situations:
        - Locating patterns in sequences by pattern FASTA file: `faskit locate -f`
        - Reading FASTQ file with record of which the quality starts with `+`
    - add command `version`
- [faskit v0.2.4](https://github.com/shenwei356/faskit/releases/tag/v0.2.4)
    - add subcommand `head`
- [faskit v0.2.3](https://github.com/shenwei356/faskit/releases/tag/v0.2.3)
    - reduce memory occupation by avoid copy data when convert `string` to `[]byte`
    - speedup reverse-complement by avoid repeatly calling functions
- [faskit v0.2.2](https://github.com/shenwei356/faskit/releases/tag/v0.2.2)
    - reduce memory occupation of subcommands that use FASTA index
- [faskit v0.2.1](https://github.com/shenwei356/faskit/releases/tag/v0.2.1)
    - improve performance of outputing.
    - fix bug of `faskit seq -g` for FASTA fromat
    - some other minor fix of code and docs
    - update benchmark results
- [faskit v0.2.0](https://github.com/shenwei356/faskit/releases/tag/v0.2.0)
    - ***reduce memory usage of writing output***
    - fix bug of `subseq`, `shuffle`, `sort` when reading from stdin
    - reduce memory usage of `faidx`
    - make validating sequences an optional option in `seq` command, it saves some time.
- [faskit v0.1.9](https://github.com/shenwei356/faskit/releases/tag/v0.1.9)
    - using custom FASTA index file extension: `.faskit.fai`
    - reducing memory usage of `sample --number --two-pass`
    - ***change default CPU number to 2 for multi-cpus computer, and 1 for single-CPU computer***
- [faskit v0.1.8](https://github.com/shenwei356/faskit/releases/tag/v0.1.8)
    - add subcommand `rename` to rename duplicated IDs
    - add subcommand `faidx` to create FASTA index file
    - ***utilize faidx to improve performance of `subseq`***
    - *`shuffle`, `sort` and split support two-pass mode (by flag `-2`) with faidx to reduce memory usage.*
    - document update
- [faskit v0.1.7](https://github.com/shenwei356/faskit/releases/tag/v0.1.7)
    - ***add support for (multi-line) FASTQ format***
    - update document, add technical details
    - rename subcommands `fa2tab` and `tab2fa` to `fx2tab` and `tab2fx`
    - add subcommand `fq2fa`
    - add column "seq_format" to `stat`
    - add global flag `-b` (`--bufer-size`)
    - little change of flag in `subseq` and some other commands
- [faskit v0.1.6](https://github.com/shenwei356/faskit/releases/tag/v0.1.6)
    - add subcommand `replace`
- [faskit v0.1.5.2](https://github.com/shenwei356/faskit/releases/tag/v0.1.5.2)
    - fix bug of `grep`, when not using flag `-r`, flag `-i` will not take effect.
- [faskit v0.1.5.1](https://github.com/shenwei356/faskit/releases/tag/v0.1.5.1)
    - fix result of `faskit sample -n`
    - fix benchmark script
- [faskit v0.1.5](https://github.com/shenwei356/faskit/releases/tag/v0.1.5)
    - add global flag `--id-ncbi`
    - add flag `-d` (`--dup-seqs-file`) and `-D` (`--dup-num-file`) for subcommand `rmdup`
    - make using MD5 as an optional flag `-m` (`--md5`) in subcommand `rmdup` and `common`
    - fix file name suffix of `faskit split` result
    - minor modification of `sliding` output
- [faskit v0.1.4.1](https://github.com/shenwei356/faskit/releases/tag/v0.1.4.1)
    - change alignment of `stat` output
    - preciser CPUs number control
- [faskit v0.1.4](https://github.com/shenwei356/faskit/releases/tag/v0.1.4)
    - add subcommand `sort`
    - improve subcommand `subseq`: supporting of getting subsequences by GTF and BED files
    - change name format of `sliding` result
    - prettier output of `stat`
- [faskit v0.1.3.1](https://github.com/shenwei356/faskit/releases/tag/v0.1.3.1)
    - Performance improvement by reducing time of cleaning spaces
    - Document update
- [faskit v0.1.3](https://github.com/shenwei356/faskit/releases/tag/v0.1.3)
    - **Further performance improvement**
    - Rename sub command `extract` to `grep`
    - Change default value of flag `--threads` back CPU number of current device,
      change default value of flag `--chunk-size` back 10000 sequences.
    - Update benchmark
- [faskit v0.1.2](https://github.com/shenwei356/faskit/releases/tag/v0.1.2)
    - Add flag `--dna2rna` and `--rna2dna` to subcommand `seq`.
- [faskit v0.1.1](https://github.com/shenwei356/faskit/releases/tag/v0.1.1)
    - **5.5X speedup of FASTA file parsing** by avoid using regular expression to remove spaces ([detail](https://github.com/shenwei356/bio/commit/2457b877cf1b8d79d05adb1a8952f2dff9046eaf) ) and using slice indexing instead of map to validate letters ([detail](https://github.com/shenwei356/bio/commit/0f5912f6a3c6d737faacf9212f62d11c94e5044a))
    - Change default value of global flag `-- thread` to 1. Since most of the subcommands are I/O intensive,  For computation intensive jobs, like extract and locate, you may set a bigger value.
    - Change default value of global flag `--chunk-size` to 100.
    - Add subcommand `stat`
    - Fix bug of failing to automatically detect alphabet when only one record in file.
- [faskit v0.1](https://github.com/shenwei356/faskit/releases/tag/v0.1)
    - first release of faskit

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
