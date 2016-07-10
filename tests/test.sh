#!/bin/bash

for f in test_*; do
    echo -n "=============================[ "
    echo -n $f
    echo    " ]============================="
    ./$f;
done
