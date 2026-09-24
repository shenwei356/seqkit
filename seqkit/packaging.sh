#!/usr/bin/env sh

set -eu

name=seqkit
output_dir=binaries
targets="
linux/amd64
linux/arm64
darwin/amd64
darwin/arm64
windows/amd64
windows/arm64
openbsd/amd64
openbsd/arm64
freebsd/amd64
freebsd/arm64
"

staging_dir=$(mktemp -d "${TMPDIR:-/tmp}/$name-packaging.XXXXXX")
trap 'rm -rf "$staging_dir"' EXIT HUP INT TERM

mkdir -p "$output_dir"

for target in $targets; do
    goos=${target%/*}
    goarch=${target#*/}
    extension=
    if [ "$goos" = windows ]; then
        extension=.exe
    fi

    binary="${name}${extension}"
    archive="${name}_${goos}_${goarch}${extension}.tar.gz"

    printf 'Building %s/%s\n' "$goos" "$goarch"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
        go build -trimpath -tags netgo \
        -ldflags "-extldflags '-static' -w -s" \
        -o "$staging_dir/$binary" .

    tar -C "$staging_dir" -zcf "$output_dir/$archive" "$binary"
    (
        cd "$output_dir"
        md5sum "$archive" > "$archive.md5.txt"
    )
    rm -f "$staging_dir/$binary"
done
