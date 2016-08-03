#!/bin/sh


echo Test: A\) Reverse complement
echo Output sequences of all apps are not wrapped to fixed length.


NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for f in dataset_{A,B}.fa; do
        echo read file once with cat
        cat $f > /dev/null
        
        echo data: $f
        memusg -t -H seqkit seq -r -p $f -w 0 -j $i > $f.seqkit.rc
        /bin/rm $f.seqkit.rc
    done
done

