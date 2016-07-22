#!/bin/sh

cat 1X.fa 1X.fa > 2X.fa
cat 2X.fa 2X.fa > 4X.fa
cat 4X.fa 4X.fa > 8X.fa
cat 8X.fa 8X.fa > 16X.fa
cat 16X.fa 16X.fa > 32X.fa

for f in *X.fa; do seqkit rename $f > $f.re; mv $f.re $f; done

