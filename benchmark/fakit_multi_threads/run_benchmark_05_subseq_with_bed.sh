#!/bin/sh

echo  Test: Subsequence with BED file

echo warm-up
cat dataset_B.fa chr19.bed.gz > /dev/null


NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo ==  $i
    echo data: dataset_B.fa;
    if [[ ! -f dataset_B.fa.fakit.fai ]]; then fakit faidx dataset_B.fa --id-regexp "^(.+)$" -o dataset_B.fa.fakit.fai; fi;
    memusg -t -H fakit subseq dataset_B.fa --bed chr19.bed.gz -j $i > chr19.bed.gz.fakit.fa;
    # fakit stat chr19.bed.gz.fakit.fa;
    /bin/rm chr19.bed.gz.fakit.fa;
done

