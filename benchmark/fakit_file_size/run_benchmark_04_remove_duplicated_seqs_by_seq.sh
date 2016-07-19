#!/bin/sh

echo Test: B\) Removing duplicates by seq

for i in 1 2 4 8 16 32; do 
    echo == ${i}X
    f=${i}X.fa
    echo data: $f;
    memusg -t -H fakit rmdup -s -m $f -w 0 > $f.rmdup.fakit.fa;
    # fakit stat $f.rmdup.fakit.fa;
    /bin/rm $f.rmdup.fakit.fa;
done
