# Download

`fakit` is implemented in [Golang](https://golang.org/) programming language,
 executable binary files **for most popular operating system** are freely available
  in [release](https://github.com/shenwei356/fakit/releases) page.

## Current Version

- [fakit v0.1.4](https://github.com/shenwei356/fakit/releases/tag/v0.1.4)
    - add subcommand `sort`
    - improve subcommand `subseq`: supporting of getting subsequences by GTF and BED files
    - change name format of `sliding` result
    - prettier output of `stat`

## Installation

Just [download](https://github.com/shenwei356/fakit/releases) executable file
 of your operating system and rename it to `fakit.exe` (Windows) or
 `fakit` (other operating systems) for convenience,
 and then run it in command-line interface, no dependencies,
 without complicated compilation process.

You can also add the directory of the executable file to environment variable
`PATH`, so you can run `fakit` anywhere.

1. For windows, the simplest way is copy it to `C:\WINDOWS\system32`.

2. For Linux, type:

        chmod a+x /PATH/OF/FASTCOV/fakit
        echo export PATH=\$PATH:/PATH/OF/FASTCOV >> ~/.bashrc

    or simply copy it to `/usr/local/bin`

## Previous Versions

- [fakit v0.1.3.1](https://github.com/shenwei356/fakit/releases/tag/v0.1.3.1)
    - Performance improvement by reducing time of cleaning spaces
    - Document update
- [fakit v0.1.3](https://github.com/shenwei356/fakit/releases/tag/v0.1.3)
    - **Further performance improvement**
    - Rename sub command `extract` to `grep`
    - Change default value of flag `--threads` back CPU number of current device,
      change default value of flag `--chunk-size` back 10000 sequences.
    - Update benchmark
- [fakit v0.1.2](https://github.com/shenwei356/fakit/releases/tag/v0.1.2)
    - Add flag `--dna2rna` and `--rna2dna` to subcommand `seq`.
- [fakit v0.1.1](https://github.com/shenwei356/fakit/releases/tag/v0.1.1)
    - **5.5X speedup of FASTA file parsing** by avoid using regular expression to remove spaces ([detail](https://github.com/shenwei356/bio/commit/2457b877cf1b8d79d05adb1a8952f2dff9046eaf) ) and using slice indexing instead of map to validate letters ([detail](https://github.com/shenwei356/bio/commit/0f5912f6a3c6d737faacf9212f62d11c94e5044a))
    - Change default value of global flag `-- thread` to 1. Since most of the subcommands are I/O intensive,  For computation intensive jobs, like extract and locate, you may set a bigger value.
    - Change default value of global flag `--chunk-size` to 100.
    - Add subcommand `stat`
    - Fix bug of failing to automatically detect alphabet when only one record in file.
- [fakit v0.1](https://github.com/shenwei356/fakit/releases/tag/v0.1)
    - first release of fakit
