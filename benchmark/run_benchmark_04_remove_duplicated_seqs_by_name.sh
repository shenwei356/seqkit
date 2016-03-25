#!/bin/sh

echo Test: Dup_n



echo == fakit
for f in dataset_{A,B}_dup.fasta; do echo data: $f; time fakit rmdup -n $f > $f.rmdup.fakit.fa; done

echo == fasta_utilities
for f in dataset_{A,B}_dup.fasta; do echo data: $f; time unique_headers.pl $f > $f.rmdup.fautil.fa; done



echo Result summary
fakit stat dataset_{A,B}_dup.fasta.rmdup.*.fa

rm dataset_{A,B}_dup.fasta.rmdup.*.fa
