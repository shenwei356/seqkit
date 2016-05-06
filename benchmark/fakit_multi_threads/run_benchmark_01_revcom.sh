#!/bin/sh


echo Test: A\) Reverse complement

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done

NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for f in dataset_{A,B}.fa; do
        echo data: $f;
        memusg -t -H fakit seq -r -p $f -w 0 -j $i > $f.fakit.rc;
        # fakit stat $f.fakit.rc;
        /bin/rm $f.fakit.rc;
    done
done

