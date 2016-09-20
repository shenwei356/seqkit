# Download

SeqKit is implemented in [Golang](https://golang.org/) programming language,
 executable binary files **for most popular operating systems** are freely available
  in [release](https://github.com/shenwei356/seqkit/releases) page.

## Latest Version

[SeqKit v0.3.4](https://github.com/shenwei356/seqkit/releases/tag/v0.3.4)

- bugfix: `seq` wrongly handles only the first one sequence file when multiple files given
- new feature: `fxtab` can output alphabet letters of a sequence by flag `-a` (`--alphabet`)
- new feature: new flag `-K` (`--keep-key`) for `replace`,  when replacing
with key-value file, one can choose keeping the key as value or not.

***64-bit versions are highly recommended.***

### Links

- **Linux**
    - [seqkit_linux_386.tar.gz](https://github.com/shenwei356/seqkit/releases/download/v0.3.4/seqkit_linux_386.tar.gz)
    - [seqkit_linux_amd64.tar.gz](https://github.com/shenwei356/seqkit/releases/download/v0.3.4/seqkit_linux_amd64.tar.gz)
    - [seqkit_linux_arm.tar.gz](https://github.com/shenwei356/seqkit/releases/download/v0.3.4/seqkit_linux_arm.tar.gz)
- **Mac OS X**
    - [seqkit_darwin_386.tar.gz](https://github.com/shenwei356/seqkit/releases/download/v0.3.4/seqkit_darwin_386.tar.gz)
    - [seqkit_darwin_amd64.tar.gz](https://github.com/shenwei356/seqkit/releases/download/v0.3.4/seqkit_darwin_amd64.tar.gz)
- **Windows**
    - [seqkit_windows_386.exe.tar.gz](https://github.com/shenwei356/seqkit/releases/download/v0.3.4/seqkit_windows_386.exe.tar.gz)
    - [seqkit_windows_amd64.exe.tar.gz](https://github.com/shenwei356/seqkit/releases/download/v0.3.4/seqkit_windows_amd64.exe.tar.gz)
- **FreeBSD**
    - [seqkit_freebsd_386.tar.gz](https://github.com/shenwei356/seqkit/releases/download/v0.3.4/seqkit_freebsd_386.tar.gz)
    - [seqkit_freebsd_amd64.tar.gz](https://github.com/shenwei356/seqkit/releases/download/v0.3.4/seqkit_freebsd_amd64.tar.gz)
    - [seqkit_freebsd_arm.tar.gz](https://github.com/shenwei356/seqkit/releases/download/v0.3.4/seqkit_freebsd_arm.tar.gz)
- **OpenBSD**
    - [seqkit_openbsd_386.tar.gz](https://github.com/shenwei356/seqkit/releases/download/v0.3.4/seqkit_openbsd_386.tar.gz)
    - [seqkit_openbsd_amd64.tar.gz](https://github.com/shenwei356/seqkit/releases/download/v0.3.4/seqkit_openbsd_amd64.tar.gz)

### Mirror site for Chinese user

- **Linux**
    - [seqkit_linux_386.tar.gz](http://app.shenwei.me/data/seqkit/seqkit_linux_386.tar.gz)
    - [seqkit_linux_amd64.tar.gz](http://app.shenwei.me/data/seqkit/seqkit_linux_amd64.tar.gz)
    - [seqkit_linux_arm.tar.gz](http://app.shenwei.me/data/seqkit/seqkit_linux_arm.tar.gz)
- **Mac OS X**
    - [seqkit_darwin_386.tar.gz](http://app.shenwei.me/data/seqkit/seqkit_darwin_386.tar.gz)
    - [seqkit_darwin_amd64.tar.gz](http://app.shenwei.me/data/seqkit/seqkit_darwin_amd64.tar.gz)
- **Windows**
    - [seqkit_windows_386.exe.tar.gz](http://app.shenwei.me/data/seqkit/seqkit_windows_386.exe.tar.gz)
    - [seqkit_windows_amd64.exe.tar.gz](http://app.shenwei.me/data/seqkit/seqkit_windows_amd64.exe.tar.gz)
- **FreeBSD**
    - [seqkit_freebsd_386.tar.gz](http://app.shenwei.me/data/seqkit/seqkit_freebsd_386.tar.gz)
    - [seqkit_freebsd_amd64.tar.gz](http://app.shenwei.me/data/seqkit/seqkit_freebsd_amd64.tar.gz)
    - [seqkit_freebsd_arm.tar.gz](http://app.shenwei.me/data/seqkit/seqkit_freebsd_arm.tar.gz)
- **OpenBSD**
    - [seqkit_openbsd_386.tar.gz](http://app.shenwei.me/data/seqkit/seqkit_openbsd_386.tar.gz)
    - [seqkit_openbsd_amd64.tar.gz](http://app.shenwei.me/data/seqkit/seqkit_openbsd_amd64.tar.gz)

## Installation

Just [download](https://github.com/shenwei356/seqkit/releases) compressed
executable file of your operating system,
and uncompress it with `tar -zxvf *.tar.gz` command or other tools.
And then:

1. **For Linux-like systems**
    1. If you have root privilege simply copy it to `/usr/local/bin`:

            sudo cp seqkit /usr/local/bin/

    1. Or add the directory of the executable file to environment variable
    `PATH`:

            echo export PATH=\$PATH:/PATH/OF/seqkit >> ~/.bashrc


1. **For windows**, just copy `seqkit.exe` to `C:\WINDOWS\system32`.

For Go developer, just one command:

    go get -u github.com/shenwei356/seqkit/seqkit

## Release History
- [SeqKit v0.3.3](https://github.com/shenwei356/seqkit/releases/tag/v0.3.4)
    - add feature: `seqkit fx2tab ` can print the alphabet letters of every sequence with flag `-a` (`--alphabet`)
- [SeqKit v0.3.3](https://github.com/shenwei356/seqkit/releases/tag/v0.3.3)
    - fix bug of `seqkit replace`, wrongly starting from 2 when using `{nr}`
      in `-r` (`--replacement`)
    - new feature: `seqkit replace` supports replacement symbols `{nr}` (record number)
      and `{kv}` (corresponding value of the key ($1) by key-value file)
- [SeqKit v0.3.2](https://github.com/shenwei356/seqkit/releases/tag/v0.3.2)
    - fix bug of `seqkit split`, error when target file is in a directory.
    - improve performance of `seqkit spliding` for big sequences, and output
      last part even if it's shorter than window sze,
      output of FASTQ is also supported.
- [SeqKit v0.3.1.1](https://github.com/shenwei356/seqkit/releases/tag/v0.3.1.1)
    - compile with go1.7rc5, with ***higher performance and smaller size of binary file***
- [SeqKit v0.3.1](https://github.com/shenwei356/seqkit/releases/tag/v0.3.1)
    - improve speed of `seqkit locate`
- [SeqKit v0.3.0](https://github.com/shenwei356/seqkit/releases/tag/v0.3.0)
    - use fork of github.com/brentp/xopen, using `zcat` for speedup of .gz file
      reading on \*nix systems.
    - improve speed of parsing sequence ID when creating FASTA index
    - reduce memory usage of `seqkit subseq --gtf`
    - fix bug of `seqkit subseq` when using flag `--id-ncbi`
    - fix bug of `seqkit split`, outdir error
    - fix bug of `seqkit seq -p`, last base is wrongly failed to convert when
      sequence length is odd.
    - add "sum_len" result for output of `seqkit stat`
- [seqkit v0.2.9](https://github.com/shenwei356/seqkit/releases/tag/v0.2.9)
    - fix minor bug of `seqkit split` and `seqkit shuffle`,
      header name error due to improper use of pointer
    - add option `-O (--out-dir)` to `seqkit split`
- [seqkit v0.2.8](https://github.com/shenwei356/seqkit/releases/tag/v0.2.8)
    - improve speed of parsing sequence ID, not using regular expression for default `--id-regexp`
    - improve speed of record outputing for small-size sequences
    - fix minor bug: `seqkit seq` for blank record
    - update benchmark result
- [seqkit v0.2.7](https://github.com/shenwei356/seqkit/releases/tag/v0.2.7)
    - ***reduce memory usage*** by optimize the outputing of sequences.
      detail: using [`BufferedByteSliceWrapper`](https://godoc.org/github.com/shenwei356/util/byteutil#BufferedByteSliceWrapper) to resuse bytes.Buffer.
    - ***reduce memory usage and improve speed*** by using custom buffered
     reading mechanism, instead of using standard library `bufio`,
      which is slow for large genome sequence.
    - discard strategy of "buffer" and "chunk" of FASTA/Q records,
      just parse records one by one.
    - delete global flags `-c (--chunk-size)` and `-b (--buffer-size)`.
    - add function testing scripts
- [seqkit v0.2.6](https://github.com/shenwei356/seqkit/releases/tag/v0.2.6)
    - fix bug of `seqkit subseq`: Inplace subseq method leaded to wrong result
- [seqkit v0.2.5.1](https://github.com/shenwei356/seqkit/releases/tag/v0.2.5.1)
    - fix a bug of `seqkit subseq`: chromesome name was not be converting to lower case when using `--gtf` or `--bed`
- [seqkit v0.2.5](https://github.com/shenwei356/seqkit/releases/tag/v0.2.5)
    - fix a serious bug brought in `v0.2.3`, using unsafe method to convert `string` to `[]byte`
    - add awk-like built-in variable of record number (`{NR}`) for `seqkit replace`
- [seqkit v0.2.4.1](https://github.com/shenwei356/seqkit/releases/tag/v0.2.4.1)
    - fix several bugs from library `bio`, affected situations:
        - Locating patterns in sequences by pattern FASTA file: `seqkit locate -f`
        - Reading FASTQ file with record of which the quality starts with `+`
    - add command `version`
- [seqkit v0.2.4](https://github.com/shenwei356/seqkit/releases/tag/v0.2.4)
    - add subcommand `head`
- [seqkit v0.2.3](https://github.com/shenwei356/seqkit/releases/tag/v0.2.3)
    - reduce memory occupation by avoid copy data when convert `string` to `[]byte`
    - speedup reverse-complement by avoid repeatly calling functions
- [seqkit v0.2.2](https://github.com/shenwei356/seqkit/releases/tag/v0.2.2)
    - reduce memory occupation of subcommands that use FASTA index
- [seqkit v0.2.1](https://github.com/shenwei356/seqkit/releases/tag/v0.2.1)
    - improve performance of outputing.
    - fix bug of `seqkit seq -g` for FASTA fromat
    - some other minor fix of code and docs
    - update benchmark results
- [seqkit v0.2.0](https://github.com/shenwei356/seqkit/releases/tag/v0.2.0)
    - ***reduce memory usage of writing output***
    - fix bug of `subseq`, `shuffle`, `sort` when reading from stdin
    - reduce memory usage of `faidx`
    - make validating sequences an optional option in `seq` command, it saves some time.
- [seqkit v0.1.9](https://github.com/shenwei356/seqkit/releases/tag/v0.1.9)
    - using custom FASTA index file extension: `.seqkit.fai`
    - reducing memory usage of `sample --number --two-pass`
    - ***change default CPU number to 2 for multi-cpus computer, and 1 for single-CPU computer***
- [seqkit v0.1.8](https://github.com/shenwei356/seqkit/releases/tag/v0.1.8)
    - add subcommand `rename` to rename duplicated IDs
    - add subcommand `faidx` to create FASTA index file
    - ***utilize faidx to improve performance of `subseq`***
    - *`shuffle`, `sort` and split support two-pass mode (by flag `-2`) with faidx to reduce memory usage.*
    - document update
- [seqkit v0.1.7](https://github.com/shenwei356/seqkit/releases/tag/v0.1.7)
    - ***add support for (multi-line) FASTQ format***
    - update document, add technical details
    - rename subcommands `fa2tab` and `tab2fa` to `fx2tab` and `tab2fx`
    - add subcommand `fq2fa`
    - add column "seq_format" to `stat`
    - add global flag `-b` (`--bufer-size`)
    - little change of flag in `subseq` and some other commands
- [seqkit v0.1.6](https://github.com/shenwei356/seqkit/releases/tag/v0.1.6)
    - add subcommand `replace`
- [seqkit v0.1.5.2](https://github.com/shenwei356/seqkit/releases/tag/v0.1.5.2)
    - fix bug of `grep`, when not using flag `-r`, flag `-i` will not take effect.
- [seqkit v0.1.5.1](https://github.com/shenwei356/seqkit/releases/tag/v0.1.5.1)
    - fix result of `seqkit sample -n`
    - fix benchmark script
- [seqkit v0.1.5](https://github.com/shenwei356/seqkit/releases/tag/v0.1.5)
    - add global flag `--id-ncbi`
    - add flag `-d` (`--dup-seqs-file`) and `-D` (`--dup-num-file`) for subcommand `rmdup`
    - make using MD5 as an optional flag `-m` (`--md5`) in subcommand `rmdup` and `common`
    - fix file name suffix of `seqkit split` result
    - minor modification of `sliding` output
- [seqkit v0.1.4.1](https://github.com/shenwei356/seqkit/releases/tag/v0.1.4.1)
    - change alignment of `stat` output
    - preciser CPUs number control
- [seqkit v0.1.4](https://github.com/shenwei356/seqkit/releases/tag/v0.1.4)
    - add subcommand `sort`
    - improve subcommand `subseq`: supporting of getting subsequences by GTF and BED files
    - change name format of `sliding` result
    - prettier output of `stat`
- [seqkit v0.1.3.1](https://github.com/shenwei356/seqkit/releases/tag/v0.1.3.1)
    - Performance improvement by reducing time of cleaning spaces
    - Document update
- [seqkit v0.1.3](https://github.com/shenwei356/seqkit/releases/tag/v0.1.3)
    - **Further performance improvement**
    - Rename sub command `extract` to `grep`
    - Change default value of flag `--threads` back CPU number of current device,
      change default value of flag `--chunk-size` back 10000 sequences.
    - Update benchmark
- [seqkit v0.1.2](https://github.com/shenwei356/seqkit/releases/tag/v0.1.2)
    - Add flag `--dna2rna` and `--rna2dna` to subcommand `seq`.
- [seqkit v0.1.1](https://github.com/shenwei356/seqkit/releases/tag/v0.1.1)
    - **5.5X speedup of FASTA file parsing** by avoid using regular expression to remove spaces ([detail](https://github.com/shenwei356/bio/commit/2457b877cf1b8d79d05adb1a8952f2dff9046eaf) ) and using slice indexing instead of map to validate letters ([detail](https://github.com/shenwei356/bio/commit/0f5912f6a3c6d737faacf9212f62d11c94e5044a))
    - Change default value of global flag `-- thread` to 1. Since most of the subcommands are I/O intensive,  For computation intensive jobs, like extract and locate, you may set a bigger value.
    - Change default value of global flag `--chunk-size` to 100.
    - Add subcommand `stat`
    - Fix bug of failing to automatically detect alphabet when only one record in file.
- [seqkit v0.1](https://github.com/shenwei356/seqkit/releases/tag/v0.1)
    - first release of seqkit

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
