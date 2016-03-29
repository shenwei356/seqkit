#!/bin/sh

echo  Test: Subseq

echo warm-up
zcat chr1.fa.gz chr1.bed.gz > /dev/null

echo ==  fakit
echo data: chr1.fa.gz; time fakit subseq -c 1 chr1.fa.gz --bed chr1.bed.gz > chr1.bed.gz.fakit.fa

echo ==  seqtk
echo data: chr1.fa.gz; time seqtk subseq chr1.fa.gz chr1.bed.gz > chr1.bed.gz.seqtk.fa



echo ==  result summary
fakit stat chr1.bed.gz.*.fa

rm chr1.bed.gz.*.fa
