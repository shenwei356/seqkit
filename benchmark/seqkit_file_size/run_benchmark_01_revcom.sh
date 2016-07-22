#!/bin/sh


echo Test: A\) Reverse complement

for i in 1 2 4 8 16 32; do 
    echo == ${i}X
    f=${i}X.fa
    echo data: $f;
    memusg -t -H seqkit seq -r -p $f -w 0 > $f.seqkit.rc;
    # seqkit stat $f.seqkit.rc;
    /bin/rm $f.seqkit.rc;
done

