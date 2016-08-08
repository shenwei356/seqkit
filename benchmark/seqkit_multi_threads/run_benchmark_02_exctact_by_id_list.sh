#!/bin/sh

echo Test: B\) Searching by ID list


NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for g in A B C; do
        echo read file once with cat
        if [ $g == "C" ]; then
            f=dataset_C.fq
        else
            f=dataset_$g.fa
        fi
            
        cat ids_$g.txt  $f > /dev/null
        
        echo data: $f
        memusg -t -H seqkit grep -f ids_$g.txt $f -j $i -w 0 > ids_$g.txt.seqkit.fa
        /bin/rm ids_$g.txt.seqkit.fa
    done
done
