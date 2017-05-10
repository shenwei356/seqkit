- Increasing speed of reading `.gz` file by utilizing `gzip` (1.3X),
    it would be much faster if you installed `pigz` (2X).
- ***Fixing colorful output in Windows***
- `seqkit locate`: add flag `--gtf` and `--bed` to output GTF/BED6 format,
    so the result can be used in `seqkit subseq`.
- `seqkit subseq`: fix bug of `--bed`, add checking coordinate.
