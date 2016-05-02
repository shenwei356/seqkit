#!/bin/sh


echo Test: Reverse complement

for i in 1 2 4 8 16 32; do 
    echo == ${i}X
    f=$i.fa
    echo data: $f;
    memusg -t -H fakit seq -r -p $f -w 0 > $f.fakit.rc;
    # fakit stat $f.fakit.rc;
    /bin/rm $f.fakit.rc;
done

