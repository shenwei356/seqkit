#!/bin/sh

echo Test: Reverse complement
echo Output sequences of all apps are not wrapped to fixed length.

echo warm-up
for f in dataset_{A,B}.fa; do echo data: $f; cat $f > /dev/null; done

echo == fakit
for f in dataset_{A,B}.fa; do
    echo data: $f;
    memusg -t -H fakit seq -r -p $f -w 0 > $f.fakit.rc;
    # fakit stat $f.fakit.rc;
    /bin/rm $f.fakit.rc;
done

echo == fasta_utilities
for f in dataset_{A,B}.fa; do
    echo data: $f;
    memusg -t -H reverse_complement.pl $f > $f.fautil.rc;
    # fakit stat $f.fautil.rc;
    /bin/rm $f.fautil.rc;
done

# out of memory
# echo == pyfaidx
# for f in dataset_{A,B}.fa; do
#     echo data: $f;
#     memusg -t -H faidx -c -r $f > $f.pyfaidx.rc;
#     # fakit stat $f.pyfaidx.rc;
#     /bin/rm $f.pyfaidx.rc;
# done

echo == seqmagick
for f in dataset_{A,B}.fa; do
    echo data: $f;
    memusg -t -H seqmagick convert --line-wrap 0 --reverse-complement $f - > $f.seqmagick.rc;
    # fakit stat $f.seqmagick.rc;
    /bin/rm $f.seqmagick.rc;
done

echo == seqtk
for f in dataset_{A,B}.fa;
    do echo data: $f;
    memusg -t -H seqtk seq -r $f > $f.seqtk.rc;
    # fakit stat $f.seqtk.rc;
    /bin/rm $f.seqtk.rc;
done

echo == biogo
for f in dataset_{A,B}.fa; do
    echo data: $f;
    memusg -t -H ./revcom_biogo $f > $f.biogo.rc;
    # fakit stat $f.biogo.rc;
    /bin/rm $f.biogo.rc;
done
