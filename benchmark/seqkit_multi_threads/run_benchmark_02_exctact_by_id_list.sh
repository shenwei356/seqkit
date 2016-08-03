#!/bin/sh

echo Test: B\) Searching by ID list


NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for g in A B; do
        echo read file once with cat
        cat ids_$g.txt  dataset_$g.fa > /dev/null
        
        echo data: dataset_$g.fa
        memusg -t -H seqkit grep -f ids_$g.txt dataset_$g.fa -j $i -w 0 > ids_$g.txt.seqkit.fa
        /bin/rm ids_$g.txt.seqkit.fa
    done
done
