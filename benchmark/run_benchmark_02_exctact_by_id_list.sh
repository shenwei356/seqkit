#!/bin/sh

echo Test: B\) Searching by ID list

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done
for g in A B; do echo data: ids_$g.txt; cat ids_$g.txt > /dev/null; done

echo == seqkit
for g in A B; do
    echo data: dataset_$g.fa;
    memusg -t -H seqkit grep -f ids_$g.txt dataset_$g.fa -w 0 > ids_$g.txt.seqkit.fa;
    # seqkit stat ids_$g.txt.seqkit.fa;
    /bin/rm ids_$g.txt.seqkit.fa
done

echo == fasta_utilities
for g in A B; do
    echo data: dataset_$g.fa;
    memusg -t -H in_list.pl -files ids_$g.txt dataset_$g.fa > ids_$g.txt.fautil.fa;
    # seqkit stat ids_$g.txt.fautil.fa;
    /bin/rm ids_$g.txt.fautil.fa;
done

echo == seqmagick
for g in A B; do
    echo data: dataset_$g.fa;
    memusg -t -H seqmagick convert --include-from-file ids_$g.txt dataset_$g.fa - > ids_$g.txt.seqmagic.fa;
    # seqkit stat ids_$g.txt.seqmagic.fa;
    /bin/rm ids_$g.txt.seqmagic.fa;
done

echo == seqtk
for g in A B; do
    echo data: dataset_$g.fa;
    memusg -t -H seqtk subseq dataset_$g.fa ids_$g.txt > ids_$g.txt.seqtk.fa;
    # seqkit stat ids_$g.txt.seqtk.fa;
    /bin/rm ids_$g.txt.seqtk.fa;
done
