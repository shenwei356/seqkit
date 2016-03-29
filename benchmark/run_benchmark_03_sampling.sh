#!/bin/sh

echo Test: Sample

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done

n=1000
n2=4

echo == fakit
for f in dataset_A.fa; do echo data: $f; time fakit sample -n $n  $f > $f.sample.fakit.fa; done
for f in dataset_B.fa; do echo data: $f; time fakit sample -n $n2 $f > $f.sample.fakit.fa; done

echo == seqmagick
for f in dataset_A.fa; do echo data: $f; time seqmagick convert --sample $n  $f - > $f.sample.seqmagick.fa; done
for f in dataset_B.fa; do echo data: $f; time seqmagick convert --sample $n2 $f - > $f.sample.seqmagick.fa; done

echo == seqtk
for f in dataset_A.fa; do echo data: $f; time seqtk sample $f $n  > $f.sample.seqtk.fa; done
for f in dataset_B.fa; do echo data: $f; time seqtk sample $f $n2 > $f.sample.seqtk.fa; done


echo Result summary
fakit stat dataset_{A,B}.fa.sample.*.fa

rm dataset_{A,B}.fa.sample.*.fa
