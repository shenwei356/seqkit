#!/bin/sh

echo Test: E\) Subsequence with BED file

echo warm-up
cat dataset_B.fa chr19.bed.gz > t; /bin/rm t;

echo ==  seqkit
echo data: dataset_B.fa;
if [[ ! -f dataset_B.fa.seqkit.fai ]]; then seqkit faidx dataset_B.fa --id-regexp "^(.+)$" -o dataset_B.fa.seqkit.fai; fi;
memusg -t -H seqkit subseq dataset_B.fa --bed chr19.bed.gz -w 0 > chr19.bed.gz.seqkit.fa;

seqkit stat chr19.bed.gz.seqkit.fa;
md5sum chr19.bed.gz.seqkit.fa;
/bin/rm chr19.bed.gz.seqkit.fa;

echo ==  seqtk
echo data: dataset_B.fa;
memusg -t -H seqtk subseq dataset_B.fa chr19.bed.gz > chr19.bed.gz.seqtk.fa

seqkit stat chr19.bed.gz.seqtk.fa;
md5sum chr19.bed.gz.seqtk.fa;
/bin/rm chr19.bed.gz.seqtk.fa;
