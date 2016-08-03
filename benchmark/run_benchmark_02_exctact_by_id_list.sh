#!/bin/sh

echo Test: B\) Searching by ID list
echo Output sequences of all apps are not wrapped to fixed length.

function check() {
    seqkit stat $1
    md5sum $1
    /bin/rm $1
}


for g in A B; do
    echo read file once with cat
    cat dataset_$g.fa ids_$g.txt > /dev/null
    
    
    echo == seqkit
    echo data: dataset_$g.fa
    out=ids_$g.txt.seqkit.fa
    memusg -t -H seqkit grep -f ids_$g.txt dataset_$g.fa -w 0 > $out
    check $out
    
    
    echo == fasta_utilities
    echo data: dataset_$g.fa
    out=ids_$g.txt.fautil.fa
    memusg -t -H in_list.pl -files ids_$g.txt dataset_$g.fa > $out
    check $out
    
    
    echo == seqmagick
    echo data: dataset_$g.fa
    out=ids_$g.txt.seqmagic.fa
    memusg -t -H seqmagick convert --line-wrap 0 --include-from-file ids_$g.txt dataset_$g.fa - > $out
    check $out

    
    echo == seqtk
    echo data: dataset_$g.fa
    out=ids_$g.txt.seqtk.fa
    memusg -t -H seqtk subseq dataset_$g.fa ids_$g.txt > $out
    check $out

done
