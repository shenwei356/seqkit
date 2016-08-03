#!/bin/sh

echo Test: E\) Subsequence with BED file
echo Output sequences of all apps are not wrapped to fixed length.


echo read file once with cat
cat dataset_B.fa > /dev/null
zat chr19.bed.gz  > /dev/null


NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo ==  $i
    echo data: dataset_B.fa;
    if [[ ! -f dataset_B.fa.seqkit.fai ]]; then seqkit faidx dataset_B.fa --id-regexp "^(.+)$" -o dataset_B.fa.seqkit.fai; fi;
    memusg -t -H seqkit subseq dataset_B.fa --bed chr19.bed.gz -j $i > chr19.bed.gz.seqkit.fa;
    /bin/rm chr19.bed.gz.seqkit.fa;
done

