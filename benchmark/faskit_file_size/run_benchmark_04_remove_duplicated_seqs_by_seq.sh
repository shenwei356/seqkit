#!/bin/sh

echo Test: B\) Removing duplicates by seq

for i in 1 2 4 8 16 32; do 
    echo == ${i}X
    f=${i}X.fa
    echo data: $f;
    memusg -t -H faskit rmdup -s -m $f -w 0 > $f.rmdup.faskit.fa;
    # faskit stat $f.rmdup.faskit.fa;
    /bin/rm $f.rmdup.faskit.fa;
done
