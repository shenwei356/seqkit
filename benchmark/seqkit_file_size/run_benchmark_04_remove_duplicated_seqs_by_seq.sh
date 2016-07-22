#!/bin/sh

echo Test: B\) Removing duplicates by seq

for i in 1 2 4 8 16 32; do 
    echo == ${i}X
    f=${i}X.fa
    echo data: $f;
    memusg -t -H seqkit rmdup -s -m $f -w 0 > $f.rmdup.seqkit.fa;
    # seqkit stat $f.rmdup.seqkit.fa;
    /bin/rm $f.rmdup.seqkit.fa;
done
