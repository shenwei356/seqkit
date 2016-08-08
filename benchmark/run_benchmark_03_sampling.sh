#!/bin/sh

echo Test: C\) Sampling by number
echo Output sequences of all apps are not wrapped to fixed length.

function check() {
    seqkit stat $1
    md5sum $1
    /bin/rm $1
}


n=10000

for f in dataset_A.fa; do
    echo read file once with cat
    cat $f > /dev/null
    
    
    echo == seqkit
    echo data: $f;
    out=$f.sample.seqkit.fa
    memusg -t -H seqkit sample -2 -n $n $f -w 0 > $out
    check $out

    
    echo == seqmagick
    echo data: $f;
    out=$f.sample.seqmagick.fa
    memusg -t -H seqmagick convert --line-wrap 0 --sample $n  $f - > $out
    check $out
    
   
    echo == seqtk
    echo data: $f;
    out=$f.sample.seqtk.fa
    memusg -t -H seqtk sample -2 $f $n  > $out;
    check $out
    
done



n=20

for f in dataset_B.fa; do
    echo read file once with cat
    cat $f > /dev/null
    
    echo == seqkit
    echo data: $f;
    out=$f.sample.seqkit.fa
    memusg -t -H seqkit sample -2 -n $n $f -w 0 > $out
    check $out

    
    echo == seqmagick
    echo data: $f;
    out=$f.sample.seqmagick.fa
    memusg -t -H seqmagick convert --line-wrap 0 --sample $n  $f - > $out
    check $out
    
   
    echo == seqtk
    echo data: $f;
    out=$f.sample.seqtk.fa
    memusg -t -H seqtk sample -2 $f $n  > $out;
    check $out
    
done

n=1000000

for f in dataset_C.fq; do
    echo read file once with cat
    cat $f > /dev/null
    
    
    echo == seqkit
    echo data: $f;
    out=$f.sample.seqkit.fa
    memusg -t -H seqkit sample -2 -n $n $f -w 0 > $out
    check $out

    
    echo == seqmagick
    echo data: $f;
    out=$f.sample.seqmagick.fa
    memusg -t -H seqmagick convert --line-wrap 0 --sample $n  $f - > $out
    check $out
    
   
    echo == seqtk
    echo data: $f;
    out=$f.sample.seqtk.fa
    memusg -t -H seqtk sample -2 $f $n  > $out;
    check $out
    
done
