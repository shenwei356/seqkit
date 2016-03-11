# Extention

## Tabular FASTA format

After converting FASTA to tabular format with `fakit fa2tab`,
it could be handled with CSV/TSV tools,
 e.g. [datakit](https://github.com/shenwei356/datakit) (CSV/TSV file manipulation and more)

- [csv_grep](https://github.com/shenwei356/datakit/tree/master/csv_grep.go)
(go version) or [csv_grep.py](https://github.com/shenwei356/datakit/blob/master/csv_grep.py)
(python version), could be used to filter sequences (similar with `fakit extract`)
- [intersection](https://github.com/shenwei356/datakit/blob/master/intersection)
computates intersection of multiple files. It could achieve similar function
as `fakit common -n` along with shell.
- [csv_join](https://github.com/shenwei356/datakit/blob/master/csv_join) joins multiple CSV/TSV files by multiple IDs.
- [csv_melt](https://github.com/shenwei356/datakit/blob/master/csv_melt)
provides melt function, could be used in preparation of data for ploting.
