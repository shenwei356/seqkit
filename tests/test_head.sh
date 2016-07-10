#!/bin/bash

test -e ssshtest || wget -q https://raw.githubusercontent.com/ryanlayer/ssshtest/master/ssshtest

. ssshtest

set -o nounset

STOP_ON_FAIL=1

# ------------------------------------------------------------
file="hairpin.fa"


run head fakit head -n 10 $file
assert_equal 10 $(grep -c ">" $STDOUT_FILE)
