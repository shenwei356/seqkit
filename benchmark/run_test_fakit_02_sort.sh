#!/bin/sh

echo Test: Sorting by length

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done

echo delete old FASTA index file
for f in dataset_{A,B}.fa; do
    if [[ -f $f.fakit.fai ]]; then
        /bin/rm $f.fakit.fai
        # fakit faidx $f --id-regexp "^(.+)$" -o $f.fakit.fai;
    fi;
done


echo == fakit
for f in dataset_{A,B}.fa; do
    echo data: $f;
    memusg -t -H fakit sort -l -2 $f > $f.fakit.sort;
    # fakit stat $f.fakit.rc;
    /bin/rm $f.fakit.sort;
done
