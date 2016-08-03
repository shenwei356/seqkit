#!/bin/sh

echo Test: E\) Subsequence with BED file
echo Output sequences of all apps are not wrapped to fixed length.


function check() {
    seqkit stat $1
    md5sum $1
    /bin/rm $1
}


echo read file once with cat
cat dataset_B.fa  > /dev/null
zcat chr19.bed.gz > /dev/null

echo == seqkit
echo data: dataset_B.fa
if [[ ! -f dataset_B.fa.seqkit.fai ]]; then seqkit faidx dataset_B.fa --id-regexp "^(.+)$" -o dataset_B.fa.seqkit.fai; fi
memusg -t -H seqkit subseq dataset_B.fa --bed chr19.bed.gz -w 0 > chr19.bed.gz.seqkit.fa
check chr19.bed.gz.seqkit.fa

echo == seqtk
echo data: dataset_B.fa
memusg -t -H seqtk subseq dataset_B.fa chr19.bed.gz > chr19.bed.gz.seqtk.fa
check chr19.bed.gz.seqtk.fa
