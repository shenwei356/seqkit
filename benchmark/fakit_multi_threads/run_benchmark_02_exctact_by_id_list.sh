#!/bin/sh

echo Test: Search


NCPUs=$(grep -c processor /proc/cpuinfo)
for i in $(seq 1 $NCPUs); do 
    echo == $i
    for g in A B; do echo data: dataset_$g.fa; time fakit -j $i grep -f ids_$g.txt dataset_$g.fa > ids_$g.txt.fakit.fa; done
done


rm ids_*.fa
