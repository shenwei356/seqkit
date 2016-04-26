#!/bin/sh

echo Test: Sampling by number

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done

n=10000
n2=20

echo == fakit
for f in dataset_A.fa; do
    echo data: $f;
    memusg -t -H fakit sample -2 -n $n  $f > $f.sample.fakit.fa;
    # fakit stat $f.sample.fakit.fa;
    /bin/rm $f.sample.fakit.fa;
done

for f in dataset_B.fa; do
    echo data: $f;
    memusg -t -H fakit sample -2 -n $n2 $f > $f.sample.fakit.fa;
    # fakit stat $f.sample.fakit.fa;
    /bin/rm $f.sample.fakit.fa;
done

echo == seqmagick
for f in dataset_A.fa; do
    echo data: $f; memusg -t -H seqmagick convert --sample $n  $f - > $f.sample.seqmagick.fa;
    # fakit stat $f.sample.seqmagick.fa;
    /bin/rm $f.sample.seqmagick.fa;
done

for f in dataset_B.fa; do
    echo data: $f; memusg -t -H seqmagick convert --sample $n2 $f - > $f.sample.seqmagick.fa;
    # fakit stat $f.sample.seqmagick.fa;
    /bin/rm $f.sample.seqmagick.fa;
done

echo == seqtk
for f in dataset_A.fa; do
    echo data: $f; memusg -t -H seqtk sample $f $n  > $f.sample.seqtk.fa;
    # fakit stat $f.sample.seqtk.fa;
    /bin/rm $f.sample.seqtk.fa;
done

for f in dataset_B.fa; do
    echo data: $f; memusg -t -H seqtk sample $f $n2 > $f.sample.seqtk.fa;
    # fakit stat $f.sample.seqtk.fa;
    /bin/rm $f.sample.seqtk.fa;
done
