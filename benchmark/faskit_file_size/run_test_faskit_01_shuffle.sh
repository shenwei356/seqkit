#!/bin/sh

echo Test: C\) Shuffling 

for i in 1 2 4 8 16 32; do 
    echo == ${i}X
    f=${i}X.fa
    echo recreate FASTA index file
    if [[ -f $f.faskit.fai ]]; then
        /bin/rm $f.faskit.fai
        # faskit faidx $f --id-regexp "^(.+)$" -o $f.faskit.fai;
    fi;
    
    echo data: $f;
    memusg -t -H faskit shuffle -2 $f -w 0 > $f.faskit.shuffle;
    # faskit stat $f.faskit.rc;
    /bin/rm $f.faskit.shuffle;

done



