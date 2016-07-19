#!/bin/sh

echo Test: E\) Subsequence with BED file

echo warm-up
cat dataset_B.fa chr19.bed.gz > /dev/null

echo ==  fakit
echo data: dataset_B.fa;
if [[ ! -f dataset_B.fa.fakit.fai ]]; then fakit faidx dataset_B.fa --id-regexp "^(.+)$" -o dataset_B.fa.fakit.fai; fi;
memusg -t -H fakit subseq dataset_B.fa --bed chr19.bed.gz -w 0 > chr19.bed.gz.fakit.fa;
# fakit stat chr19.bed.gz.fakit.fa;
/bin/rm chr19.bed.gz.fakit.fa;

echo ==  seqtk
echo data: dataset_B.fa;
memusg -t -H seqtk subseq dataset_B.fa chr19.bed.gz > chr19.bed.gz.seqtk.fa
# fakit stat chr19.bed.gz.seqtk.fa;
/bin/rm chr19.bed.gz.seqtk.fa;
