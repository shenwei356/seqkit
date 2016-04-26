#!/bin/sh

echo Test: Searching by ID list

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done

NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for g in A B; do
        echo data: dataset_$g.fa;
        memusg -t -H fakit grep -f ids_$g.txt dataset_$g.fa -j $i > ids_$g.txt.fakit.fa;
        # fakit stat ids_$g.txt.fakit.fa;
        /bin/rm ids_$g.txt.fakit.fa
    done
done
