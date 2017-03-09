#!/usr/bin/env sh

gox -os="windows darwin linux" -arch="386 amd64";

dir=binaries
mkdir -p $dir;
rm -rf $dir/$f;

for f in seqkit_*; do
    mkdir -p $dir/$f;
    mv $f $dir/$f;
    cd $dir/$f;
    brename -s '_[^\.]+';
    tar -zcf $f.tar.gz seqkit*;
    mv *.tar.gz ../;
    cd ..;
    rm -rf $f;
    cd ..;
done;
