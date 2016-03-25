#!/bin/sh

echo Test: Sample

n=1000

echo == fakit
for f in dataset_{A,B}.fa; do echo data: $f; time fakit sample -n $n $f > $f.sample.fakit.fa; done

echo == fasta_utilities
for f in dataset_{A,B}.fa; do echo data: $f; time subset_fasta.pl -size $n $f > $f.sample.fautil.fa; done

echo == seqmagick
for f in dataset_{A,B}.fa; do echo data: $f; time seqmagick convert --sample $n $f - > $f.sample.seqmagick.fa; done

echo == seqtk
for f in dataset_{A,B}.fa; do echo data: $f; time seqtk sample $f $n > $f.sample.seqtk.fa; done


echo Result summary
fakit stat dataset_{A,B}.fa.sample.*.fa

rm dataset_{A,B}.fa.sample.*.fa
