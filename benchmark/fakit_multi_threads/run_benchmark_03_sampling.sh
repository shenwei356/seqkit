#!/bin/sh

echo Test: Sample

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done

n=1000
n2=4

NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for f in dataset_A.fa; do echo data: $f; time fakit sample -j $i -n $n  $f > $f.sample.fakit.fa; done
    for f in dataset_B.fa; do echo data: $f; time fakit sample -j $i -n $n2 $f > $f.sample.fakit.fa; done
done

rm dataset_{A,B}.fa.sample.*.fa
