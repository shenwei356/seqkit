#!/bin/sh

echo Test: Shuffling 

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done


NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    echo recreate FASTA index file
    for f in dataset_{A,B}.fa; do
        if [[ -f $f.faskit.fai ]]; then
            /bin/rm $f.faskit.fai
            # faskit faidx $f --id-regexp "^(.+)$" -o $f.faskit.fai;
        fi;
    done

    for f in dataset_{A,B}.fa; do
        echo data: $f;
        memusg -t -H faskit shuffle -2 $f -w 0 > $f.faskit.shuffle;
        # faskit stat $f.faskit.rc;
        /bin/rm $f.faskit.shuffle;
    done
done



