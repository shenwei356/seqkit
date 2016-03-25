#!/bin/sh

echo  Test: Revcom



echo == fakit
for f in dataset_{A,B}.fa; do echo data: $f; time fakit seq -r -p $f > $f.fakit.rc; done

echo == fasta_utilities
for f in dataset_{A,B}.fa; do echo data: $f; time reverse_complement.pl $f > .$f.fautil.rc; done

echo == pyfaidx
for f in dataset_{A,B}.fa; do echo data: $f; time faidx -c -r $f > $f.pyfaidx.rc; done

echo == seqmagick
for f in dataset_{A,B}.fa; do echo data: $f; time seqmagick convert --reverse-complement $f - > $f.seqmagick.rc; done

echo == seqtk
for f in dataset_{A,B}.fa; do echo data: $f; time seqtk seq -r $f > $f.seqtk.rc; done

echo == biogo
for f in dataset_{A,B}.fa; do echo data: $f; time ./revcom_biogo $f > $f.biogo.rc; done



echo Result summary
fakit stat dataset_*.rc

rm dataset_*.rc
