#!/bin/sh

echo Test: Dup_s


NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for f in dataset_{A,B}_dup.fasta; do echo data: $f; time fakit -j $i rmdup -s $f > $f.rmdup.fakit.fa; done
done


rm dataset_{A,B}_dup.fasta.rmdup.*.fa
