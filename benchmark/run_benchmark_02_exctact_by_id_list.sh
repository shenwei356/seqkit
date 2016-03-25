#!/bin/sh

echo Test: Search



echo == fakit
for g in A B; do echo data: dataset_$g.fa; time fakit grep -f ids_$g.txt dataset_$g.fa > ids_$g.txt.fakit.fa; done

echo == fasta_utilities
for g in A B; do echo data: dataset_$g.fa; time in_list.pl -files ids_$g.txt dataset_$g.fa > ids_$g.txt.fautil.fa; done

echo == seqmagick
for g in A B; do echo data: dataset_$g.fa; time seqmagick convert --include-from-file ids_$g.txt dataset_$g.fa - > ids_$g.txt.seqmagic.fa; done

echo == seqtk
for g in A B; do echo data: dataset_$g.fa; time seqtk subseq  dataset_$g.fa ids_$g.txt > ids_$g.txt.seqtk.fa; done



echo Result summary
fakit stat ids_*.fa

rm ids_*.fa
