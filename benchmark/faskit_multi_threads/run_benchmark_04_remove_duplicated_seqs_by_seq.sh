#!/bin/sh

echo Test: D\) Removing duplicates by seq

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done

NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for f in dataset_{A,B}.fa; do
        echo data: $f;
        memusg -t -H faskit rmdup -s -m $f -j $i -w 0 > $f.rmdup.faskit.fa;
        # faskit stat $f.rmdup.faskit.fa;
        /bin/rm $f.rmdup.faskit.fa;
    done
done
