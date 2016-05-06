#!/bin/sh

echo Test: D\) Sorting by length

for i in 1 2 4 8 16 32; do 
    echo == ${i}X
    f=${i}X.fa
    echo delete old FASTA index file
    if [[ -f $f.fakit.fai ]]; then
        /bin/rm $f.fakit.fai
        # fakit faidx $f --id-regexp "^(.+)$" -o $f.fakit.fai;
    fi;

    echo data: $f;
    memusg -t -H fakit sort -l -2 $f > $f.fakit.sort;
    # fakit stat $f.fakit.rc;
    /bin/rm $f.fakit.sort;
done








