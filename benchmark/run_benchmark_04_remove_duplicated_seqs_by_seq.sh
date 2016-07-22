#!/bin/sh

echo Test: D\) Removing duplicates by seq

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done


echo == seqkit
for f in dataset_{A,B}.fa; do
    echo data: $f;
    memusg -t -H seqkit rmdup -s -m $f -w 0 > $f.rmdup.seqkit.fa;
    # seqkit stat $f.rmdup.seqkit.fa;
    /bin/rm $f.rmdup.seqkit.fa;
done

echo == seqmagick
for f in dataset_{A,B}.fa; do
    echo data: $f;
    memusg -t -H seqmagick convert --line-wrap 0 --deduplicate-sequences $f - > $f.rmdup.seqmagick.fa;
    # seqkit stat $f.rmdup.seqmagick.fa;
    /bin/rm $f.rmdup.seqmagick.fa;
done
