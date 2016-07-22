#!/bin/sh

echo Test: A\) Reverse complement
echo Output sequences of all apps are not wrapped to fixed length.

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done

echo == faskit
for f in dataset_{A,B}.fa; do
    echo data: $f;
    memusg -t -H faskit seq -r -p $f -w 0 > $f.faskit.rc;
    # faskit stat $f.faskit.rc;
    /bin/rm $f.faskit.rc;
done

echo == fasta_utilities
for f in dataset_{A,B}.fa; do
    echo data: $f;
    memusg -t -H reverse_complement.pl $f > $f.fautil.rc;
    # faskit stat $f.fautil.rc;
    /bin/rm $f.fautil.rc;
done

# too high memory usage and too slow
# echo == pyfaidx
# for f in dataset_{A,B}.fa; do
#     echo data: $f;
#     memusg -t -H faidx -c -r $f > $f.pyfaidx.rc;
#     # faskit stat $f.pyfaidx.rc;
#     /bin/rm $f.pyfaidx.rc;
# done

echo == seqmagick
for f in dataset_{A,B}.fa; do
    echo data: $f;
    memusg -t -H seqmagick convert --line-wrap 0 --reverse-complement $f - > $f.seqmagick.rc;
    # faskit stat $f.seqmagick.rc;
    /bin/rm $f.seqmagick.rc;
done

echo == seqtk
for f in dataset_{A,B}.fa;
    do echo data: $f;
    memusg -t -H seqtk seq -r $f > $f.seqtk.rc;
    # faskit stat $f.seqtk.rc;
    /bin/rm $f.seqtk.rc;
done

echo == biogo
for f in dataset_{A,B}.fa; do
    echo data: $f;
    memusg -t -H ./revcom_biogo $f > $f.biogo.rc;
    # faskit stat $f.biogo.rc;
    /bin/rm $f.biogo.rc;
done
