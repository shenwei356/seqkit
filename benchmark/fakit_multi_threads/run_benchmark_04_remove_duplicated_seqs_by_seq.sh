#!/bin/sh

echo Test: Removing duplicates by seq

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done

NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for f in dataset_{A,B}.fa; do
        echo data: $f;
        memusg -t -H fakit rmdup -s -m $f -j $i > $f.rmdup.fakit.fa;
        # fakit stat $f.rmdup.fakit.fa;
        /bin/rm $f.rmdup.fakit.fa;
    done
done
