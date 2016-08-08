#!/bin/sh

echo Test: C\) Sampling by number

n=10000
n2=20
n3=1000000

NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for f in dataset_A.fa; do
        echo read file once with cat
        cat $f > /dev/null
        echo data: $f
        memusg -t -H seqkit sample -2 -n $n  $f -j $i -w 0 > $f.sample.seqkit.fa
        /bin/rm $f.sample.seqkit.fa
    done

    for f in dataset_B.fa; do
        echo read file once with cat
        cat $f > /dev/null
        echo data: $f
        memusg -t -H seqkit sample -2 -n $n2 $f -j $i -w 0 > $f.sample.seqkit.fa
        /bin/rm $f.sample.seqkit.fa
    done
    
    for f in dataset_C.fq; do
        echo read file once with cat
        cat $f > /dev/null
        echo data: $f
        memusg -t -H seqkit sample -2 -n $n3 $f -j $i -w 0 > $f.sample.seqkit.fa
        /bin/rm $f.sample.seqkit.fa
    done

done
