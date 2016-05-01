#!/bin/sh

echo Test: Removing duplicates by seq content

for i in 1 2 4 8 16 32; do 
    echo == $i
    f=$i.fa
    echo data: $f;
    memusg -t -H fakit rmdup -s -m $f -j $i > $f.rmdup.fakit.fa;
    # fakit stat $f.rmdup.fakit.fa;
    /bin/rm $f.rmdup.fakit.fa;
done
