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
        memusg -t -H fakit sample -2 -n $n  $f -j $i -w 0 > $f.sample.fakit.fa;
        # fakit stat $f.sample.fakit.fa;
        /bin/rm $f.sample.fakit.fa;
    done

    for f in dataset_B.fa; do
        echo data: $f;
        memusg -t -H fakit sample -2 -n $n2 $f -j $i -w 0 > $f.sample.fakit.fa;
        # fakit stat $f.sample.fakit.fa;
        /bin/rm $f.sample.fakit.fa;
    done

done
