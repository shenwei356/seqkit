#!/usr/bin/env sh

gox;

dir=binaries
mkdir -p $dir;
rm -rf $dir/$f;

for f in fakit_*; do
    mkdir -p $dir/$f;
    mv $f $dir/$f;
    cd $dir/$f;
    brename -s '_[^\.]+';
    tar -zcf $f.tar.gz fakit*;
    mv *.tar.gz ../;
    cd ..;
    rm -rf $f;
    cd ..;
done;
