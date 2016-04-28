#!/bin/sh

echo Test: Removing duplicates by seq

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done


echo == fakit
for f in dataset_{A,B}.fa; do
    echo data: $f;
    memusg -t -H fakit rmdup -s -m $f > $f.rmdup.fakit.fa;
    # fakit stat $f.rmdup.fakit.fa;
    /bin/rm $f.rmdup.fakit.fa;
done

echo == seqmagick
for f in dataset_{A,B}.fa; do
    echo data: $f;
    memusg -t -H seqmagick convert --deduplicate-sequences $f - > $f.rmdup.seqmagick.fa;
    # fakit stat $f.rmdup.seqmagick.fa;
    /bin/rm $f.rmdup.seqmagick.fa;
done
