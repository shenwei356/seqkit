#!/bin/sh


echo Test: A\) Reverse complement

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done

NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for f in dataset_{A,B}.fa; do
        echo data: $f;
        memusg -t -H faskit seq -r -p $f -w 0 -j $i > $f.faskit.rc;
        # faskit stat $f.faskit.rc;
        /bin/rm $f.faskit.rc;
    done
done

