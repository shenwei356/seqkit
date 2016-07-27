#!/bin/sh

echo Test: D\) Removing duplicates by seq

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > t; /bin/rm t; done

NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for f in dataset_{A,B}.fa; do
        echo data: $f;
        memusg -t -H seqkit rmdup -s -m $f -j $i -w 0 > $f.rmdup.seqkit.fa;
        # seqkit stat $f.rmdup.seqkit.fa;
        /bin/rm $f.rmdup.seqkit.fa;
    done
done
