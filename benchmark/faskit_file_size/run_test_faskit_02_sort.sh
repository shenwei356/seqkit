#!/bin/sh

echo Test: D\) Sorting by length

for i in 1 2 4 8 16 32; do 
    echo == ${i}X
    f=${i}X.fa
    echo delete old FASTA index file
    if [[ -f $f.faskit.fai ]]; then
        /bin/rm $f.faskit.fai
        # faskit faidx $f --id-regexp "^(.+)$" -o $f.faskit.fai;
    fi;

    echo data: $f;
    memusg -t -H faskit sort -l -2 $f -w 0 > $f.faskit.sort;
    # faskit stat $f.faskit.rc;
    /bin/rm $f.faskit.sort;
done








