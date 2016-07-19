#!/bin/sh

echo Test: D\) Removing duplicates by seq

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done

NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for f in dataset_{A,B}.fa; do
        echo data: $f;
        memusg -t -H fakit rmdup -s -m $f -j $i -w 0 > $f.rmdup.fakit.fa;
        # fakit stat $f.rmdup.fakit.fa;
        /bin/rm $f.rmdup.fakit.fa;
    done
done
