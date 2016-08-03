#!/bin/sh

echo Test: D\) Removing duplicates by seq
echo Output sequences of all apps are not wrapped to fixed length.

NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for f in dataset_{A,B}.fa; do
        echo read file once with cat
        cat $f > /dev/null
        
        echo data: $f;
        memusg -t -H seqkit rmdup -s -m $f -j $i -w 0 > $f.rmdup.seqkit.fa;
        /bin/rm $f.rmdup.seqkit.fa;
    done
done
