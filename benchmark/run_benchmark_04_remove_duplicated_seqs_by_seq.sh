#!/bin/sh

echo Test: D\) Removing duplicates by seq
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
    out=$f.rmdup.seqkit.fa
    memusg -t -H seqkit rmdup -s -m $f -w 0 > $out
    check $out
    
    
    echo == seqmagick
    echo data: $f
    out=$f.rmdup.seqmagick.fa
    memusg -t -H seqmagick convert --line-wrap 0 --deduplicate-sequences $f - > $out 
    check $out
done
