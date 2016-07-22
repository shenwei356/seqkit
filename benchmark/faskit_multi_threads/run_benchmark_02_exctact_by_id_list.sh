#!/bin/sh

echo Test: B\) Searching by ID list

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done

NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for g in A B; do
        echo data: dataset_$g.fa;
        memusg -t -H faskit grep -f ids_$g.txt dataset_$g.fa -j $i -w 0 > ids_$g.txt.faskit.fa;
        # faskit stat ids_$g.txt.faskit.fa;
        /bin/rm ids_$g.txt.faskit.fa
    done
done
