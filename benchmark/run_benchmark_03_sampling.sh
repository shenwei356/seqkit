#!/bin/sh

echo Test: C\) Sampling by number

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done

n=10000
n2=20

echo == faskit
for f in dataset_A.fa; do
    echo data: $f;
    memusg -t -H faskit sample -2 -n $n $f -w 0 > $f.sample.faskit.fa;
    # faskit stat $f.sample.faskit.fa;
    /bin/rm $f.sample.faskit.fa;
done

for f in dataset_B.fa; do
    echo data: $f;
    memusg -t -H faskit sample -2 -n $n2 $f -w 0 > $f.sample.faskit.fa;
    # faskit stat $f.sample.faskit.fa;
    /bin/rm $f.sample.faskit.fa;
done

echo == seqmagick
for f in dataset_A.fa; do
    echo data: $f; memusg -t -H seqmagick convert --line-wrap 0 --sample $n  $f - > $f.sample.seqmagick.fa;
    # faskit stat $f.sample.seqmagick.fa;
    /bin/rm $f.sample.seqmagick.fa;
done

for f in dataset_B.fa; do
    echo data: $f; memusg -t -H seqmagick convert --line-wrap 0 --sample $n2 $f - > $f.sample.seqmagick.fa;
    # faskit stat $f.sample.seqmagick.fa;
    /bin/rm $f.sample.seqmagick.fa;
done

echo == seqtk
for f in dataset_A.fa; do
    echo data: $f; memusg -t -H seqtk sample $f $n  > $f.sample.seqtk.fa;
    # faskit stat $f.sample.seqtk.fa;
    /bin/rm $f.sample.seqtk.fa;
done

for f in dataset_B.fa; do
    echo data: $f; memusg -t -H seqtk sample $f $n2 > $f.sample.seqtk.fa;
    # faskit stat $f.sample.seqtk.fa;
    /bin/rm $f.sample.seqtk.fa;
done
