#!/bin/sh

echo Test: Sample

n=1000

NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for f in dataset_{A,B}.fa; do echo data: $f; time fakit -j $i sample -n $n $f > $f.sample.fakit.fa; done
done

rm dataset_{A,B}.fa.sample.*.fa
