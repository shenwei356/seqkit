#!/usr/bin/env sh

gox;

dir=binaries
mkdir -p $dir;
rm -rf $dir/$f;

for f in seqkit_*; do
    mkdir -p $dir/$f;
    mv $f $dir/$f;
    cd $dir/$f;
    mv $f $(echo $f | perl -pe 's/_[^\.]+//g');
    tar -zcf $f.tar.gz seqkit*;
    mv *.tar.gz ../;
    cd ..;
    rm -rf $f;
    cd ..;
done;
