#!/bin/sh

echo Test: Shuffling 

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done


NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    echo recreate FASTA index file
    for f in dataset_{A,B}.fa; do
        if [[ -f $f.fakit.fai ]]; then
            /bin/rm $f.fakit.fai
            # fakit faidx $f --id-regexp "^(.+)$" -o $f.fakit.fai;
        fi;
    done

    for f in dataset_{A,B}.fa; do
        echo data: $f;
        memusg -t -H fakit shuffle -2 $f -w 0 > $f.fakit.shuffle;
        # fakit stat $f.fakit.rc;
        /bin/rm $f.fakit.shuffle;
    done
done



