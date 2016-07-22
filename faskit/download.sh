#!/bin/sh

rm faskit_*.tar.gz
version="0.2.8"

wget https://github.com/shenwei356/faskit/releases/download/v$version/faskit_linux_386.tar.gz
wget https://github.com/shenwei356/faskit/releases/download/v$version/faskit_linux_amd64.tar.gz
wget https://github.com/shenwei356/faskit/releases/download/v$version/faskit_linux_arm.tar.gz
wget https://github.com/shenwei356/faskit/releases/download/v$version/faskit_darwin_386.tar.gz
wget https://github.com/shenwei356/faskit/releases/download/v$version/faskit_darwin_amd64.tar.gz
wget https://github.com/shenwei356/faskit/releases/download/v$version/faskit_windows_386.exe.tar.gz
wget https://github.com/shenwei356/faskit/releases/download/v$version/faskit_windows_amd64.exe.tar.gz
wget https://github.com/shenwei356/faskit/releases/download/v$version/faskit_freebsd_386.tar.gz
wget https://github.com/shenwei356/faskit/releases/download/v$version/faskit_freebsd_amd64.tar.gz
wget https://github.com/shenwei356/faskit/releases/download/v$version/faskit_freebsd_arm.tar.gz
wget https://github.com/shenwei356/faskit/releases/download/v$version/faskit_openbsd_386.tar.gz
wget https://github.com/shenwei356/faskit/releases/download/v$version/faskit_openbsd_amd64.tar.gz
