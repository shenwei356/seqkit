#!/bin/sh

echo Test: C\) Sampling by number

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done

n=10000
n2=20

NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for f in dataset_A.fa; do
        echo data: $f;
        memusg -t -H faskit sample -2 -n $n  $f -j $i -w 0 > $f.sample.faskit.fa;
        # faskit stat $f.sample.faskit.fa;
        /bin/rm $f.sample.faskit.fa;
    done

    for f in dataset_B.fa; do
        echo data: $f;
        memusg -t -H faskit sample -2 -n $n2 $f -j $i -w 0 > $f.sample.faskit.fa;
        # faskit stat $f.sample.faskit.fa;
        /bin/rm $f.sample.faskit.fa;
    done

done
