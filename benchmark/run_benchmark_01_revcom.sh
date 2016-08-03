#!/bin/sh

echo Test: A\) Reverse complement
echo Output sequences of all apps are not wrapped to fixed length.

function check() {
    seqkit stat $1
    md5sum $1
    /bin/rm $1
}


for f in dataset_{A,B}.fa; do
    echo read file once with cat
    cat $f > /dev/null
    
    
    echo == seqkit
    echo data: $f
    out=$f.seqkit.rc
    memusg -t -H seqkit seq -r -p $f -w 0 > $out
    check $out
    
    
    echo == fasta_utilities    
    echo data: $f
    out=$f.fautil.rc
    memusg -t -H reverse_complement.pl $f > $out
    check $out
    
    
    echo == seqmagick
    echo data: $f
    out=$f.seqmagick.rc
    memusg -t -H seqmagick convert --line-wrap 0 --reverse-complement $f - > $out
    check $out
    
    
    echo == seqtk
    echo data: $f
    out=$f.seqtk.rc
    memusg -t -H seqtk seq -r $f > $out
    check $out
    
    
    echo == biogo
    echo data: $f
    out=$f.biogo.rc
    memusg -t -H ./revcom_biogo $f > $out
    check $out
    
done
