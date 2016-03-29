#!/bin/sh

echo  Test: Subseq

echo warm-up
zcat chr1.fa.gz chr1.bed.gz > /dev/null


NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo ==  $i
    echo data: chr1.fa.gz; time fakit -j $i subseq -c 1 chr1.fa.gz --bed chr1.bed.gz > chr1.bed.gz.fakit.fa
done


rm chr1.bed.gz.*.fa
