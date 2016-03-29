#!/bin/sh

echo  Test: Revcom

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done

NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for f in dataset_{A,B}.fa; do echo data: $f; time fakit -j $i seq -r -p $f > $f.fakit.rc; done
done


rm dataset_*.rc
