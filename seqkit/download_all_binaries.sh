#!/bin/sh

rm seqkit_*.tar.gz
version="0.4.5"

wget https://github.com/shenwei356/seqkit/releases/download/v$version/seqkit_linux_386.tar.gz
wget https://github.com/shenwei356/seqkit/releases/download/v$version/seqkit_linux_amd64.tar.gz
wget https://github.com/shenwei356/seqkit/releases/download/v$version/seqkit_linux_arm.tar.gz
wget https://github.com/shenwei356/seqkit/releases/download/v$version/seqkit_darwin_386.tar.gz
wget https://github.com/shenwei356/seqkit/releases/download/v$version/seqkit_darwin_amd64.tar.gz
wget https://github.com/shenwei356/seqkit/releases/download/v$version/seqkit_windows_386.exe.tar.gz
wget https://github.com/shenwei356/seqkit/releases/download/v$version/seqkit_windows_amd64.exe.tar.gz
wget https://github.com/shenwei356/seqkit/releases/download/v$version/seqkit_freebsd_386.tar.gz
wget https://github.com/shenwei356/seqkit/releases/download/v$version/seqkit_freebsd_amd64.tar.gz
wget https://github.com/shenwei356/seqkit/releases/download/v$version/seqkit_freebsd_arm.tar.gz
wget https://github.com/shenwei356/seqkit/releases/download/v$version/seqkit_openbsd_386.tar.gz
wget https://github.com/shenwei356/seqkit/releases/download/v$version/seqkit_openbsd_amd64.tar.gz
