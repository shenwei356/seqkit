#!/bin/sh

echo Test: D\) Sorting by length

for i in 1 2 4 8 16 32; do 
    echo == ${i}X
    f=${i}X.fa
    echo delete old FASTA index file
    if [[ -f $f.seqkit.fai ]]; then
        /bin/rm $f.seqkit.fai
        # seqkit faidx $f --id-regexp "^(.+)$" -o $f.seqkit.fai;
    fi;

    echo data: $f;
    memusg -t -H seqkit sort -l -2 $f -w 0 > $f.seqkit.sort;
    # seqkit stat $f.seqkit.rc;
    /bin/rm $f.seqkit.sort;
done








