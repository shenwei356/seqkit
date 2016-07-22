#!/bin/sh


echo Test: A\) Reverse complement

for i in 1 2 4 8 16 32; do 
    echo == ${i}X
    f=${i}X.fa
    echo data: $f;
    memusg -t -H faskit seq -r -p $f -w 0 > $f.faskit.rc;
    # faskit stat $f.faskit.rc;
    /bin/rm $f.faskit.rc;
done

