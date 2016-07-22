#!/bin/sh

echo Test: B\) Searching by ID list

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done
for g in A B; do echo data: ids_$g.txt; cat ids_$g.txt > /dev/null; done

echo == faskit
for g in A B; do
    echo data: dataset_$g.fa;
    memusg -t -H faskit grep -f ids_$g.txt dataset_$g.fa -w 0 > ids_$g.txt.faskit.fa;
    # faskit stat ids_$g.txt.faskit.fa;
    /bin/rm ids_$g.txt.faskit.fa
done

echo == fasta_utilities
for g in A B; do
    echo data: dataset_$g.fa;
    memusg -t -H in_list.pl -files ids_$g.txt dataset_$g.fa > ids_$g.txt.fautil.fa;
    # faskit stat ids_$g.txt.fautil.fa;
    /bin/rm ids_$g.txt.fautil.fa;
done

echo == seqmagick
for g in A B; do
    echo data: dataset_$g.fa;
    memusg -t -H seqmagick convert --include-from-file ids_$g.txt dataset_$g.fa - > ids_$g.txt.seqmagic.fa;
    # faskit stat ids_$g.txt.seqmagic.fa;
    /bin/rm ids_$g.txt.seqmagic.fa;
done

echo == seqtk
for g in A B; do
    echo data: dataset_$g.fa;
    memusg -t -H seqtk subseq dataset_$g.fa ids_$g.txt > ids_$g.txt.seqtk.fa;
    # faskit stat ids_$g.txt.seqtk.fa;
    /bin/rm ids_$g.txt.seqtk.fa;
done
