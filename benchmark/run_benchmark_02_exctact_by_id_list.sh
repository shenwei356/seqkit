#!/bin/sh

echo Test: B\) Searching by ID list

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done
for g in A B; do echo data: ids_$g.txt; cat ids_$g.txt > /dev/null; done

echo == fakit
for g in A B; do
    echo data: dataset_$g.fa;
    memusg -t -H fakit grep -f ids_$g.txt dataset_$g.fa > ids_$g.txt.fakit.fa;
    # fakit stat ids_$g.txt.fakit.fa;
    /bin/rm ids_$g.txt.fakit.fa
done

echo == fasta_utilities
for g in A B; do
    echo data: dataset_$g.fa;
    memusg -t -H in_list.pl -files ids_$g.txt dataset_$g.fa > ids_$g.txt.fautil.fa;
    # fakit stat ids_$g.txt.fautil.fa;
    /bin/rm ids_$g.txt.fautil.fa;
done

echo == seqmagick
for g in A B; do
    echo data: dataset_$g.fa;
    memusg -t -H seqmagick convert --include-from-file ids_$g.txt dataset_$g.fa - > ids_$g.txt.seqmagic.fa;
    # fakit stat ids_$g.txt.seqmagic.fa;
    /bin/rm ids_$g.txt.seqmagic.fa;
done

echo == seqtk
for g in A B; do
    echo data: dataset_$g.fa;
    memusg -t -H seqtk subseq dataset_$g.fa ids_$g.txt > ids_$g.txt.seqtk.fa;
    # fakit stat ids_$g.txt.seqtk.fa;
    /bin/rm ids_$g.txt.seqtk.fa;
done
