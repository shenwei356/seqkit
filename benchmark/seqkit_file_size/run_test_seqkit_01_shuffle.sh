#!/bin/sh

echo Test: C\) Shuffling 

for i in 1 2 4 8 16 32; do 
    echo == ${i}X
    f=${i}X.fa
    echo recreate FASTA index file
    if [[ -f $f.seqkit.fai ]]; then
        /bin/rm $f.seqkit.fai
        # seqkit faidx $f --id-regexp "^(.+)$" -o $f.seqkit.fai;
    fi;
    
    echo data: $f;
    memusg -t -H seqkit shuffle -2 $f -w 0 > $f.seqkit.shuffle;
    # seqkit stat $f.seqkit.rc;
    /bin/rm $f.seqkit.shuffle;

done



