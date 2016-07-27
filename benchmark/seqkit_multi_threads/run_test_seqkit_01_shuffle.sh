#!/bin/sh

echo Test: Shuffling 

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > t; /bin/rm t; done


NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    echo recreate FASTA index file
    for f in dataset_{A,B}.fa; do
        if [[ -f $f.seqkit.fai ]]; then
            /bin/rm $f.seqkit.fai
            # seqkit faidx $f --id-regexp "^(.+)$" -o $f.seqkit.fai;
        fi;
    done

    for f in dataset_{A,B}.fa; do
        echo data: $f;
        memusg -t -H seqkit shuffle -2 $f -w 0 > $f.seqkit.shuffle;
        # seqkit stat $f.seqkit.rc;
        /bin/rm $f.seqkit.shuffle;
    done
done



