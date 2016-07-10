#!/bin/bash

test -e ssshtest || wget -q https://raw.githubusercontent.com/ryanlayer/ssshtest/master/ssshtest

. ssshtest

set -o nounset

STOP_ON_FAIL=1

# ------------------------------------------------------------

file="hairpin.fa"

# ------------------------------------------------------------

# seq content
run seq_content fakit seq -w 60 $file
# seq number
assert_equal $(grep -c "^>" $file) $(grep -c "^>" $STDOUT_FILE)
# seq content
assert_equal $(md5sum $file | cut -d" " -f 1) $(md5sum $STDOUT_FILE | cut -d" " -f 1)

# ------------------------------------------------------------

# seq type
run seq_type fakit seq -t dna $file
assert_in_stderr "invalid DNAredundant letter"

fun() {
    echo -e ">seq\nabcdefghijklmnpqrstvwyz" | fakit stat
}
run seq_type fun
assert_in_stdout "Protein"

fun() {
    echo -e ">seq\nACGUN ACGUN" | fakit stat
}
run seq_type fun
assert_in_stdout "RNA"

fun() { 
    echo -e ">seq\nacgtryswkmbdhvACGTRYSWKMBDHV" | fakit stat 
}
run seq_type fun
assert_in_stdout "DNA"

fun() {
    echo -e "@read\nACTGCN\n+\n@IICCG" | fakit stat
}
run seq_type fun
assert_in_stdout "DNA"
assert_in_stdout "FASTQ"


# ------------------------------------------------------------

# head
run seq_head fakit seq -n $file 
assert_equal $(md5sum $STDOUT_FILE | cut -d" " -f 1) $(grep  "^>" $file | sed -s 's/^>//g' > tmp; md5sum tmp | cut -d" " -f 1; rm tmp)

# id
run seq_id fakit seq -n -i $file 
assert_equal $(md5sum $STDOUT_FILE | cut -d" " -f 1) $(grep  "^>" $file | sed -s 's/^>//g' | sed -s 's/ .*//g' > tmp; md5sum tmp | cut -d" " -f 1; rm tmp)

# seq

file="reads_1.fq.gz"
run seq_seq fakit seq $file -s -w 0
assert_equal $(md5sum $STDOUT_FILE | cut -d" " -f 1) $(fakit fx2tab $file | cut -f 2 > tmp; md5sum tmp | cut -d" " -f 1; rm tmp)


# ------------------------------------------------------------

file="hairpin.fa"

# reverse complement
fun() {
    fakit head -n 1 $file | fakit seq -r | fakit seq -p
}
run seq_revcom fun
assert_equal $(md5sum $STDOUT_FILE | cut -d" " -f 1) $(fakit head -n 1 $file | fakit seq -r -p > tmp; md5sum tmp | cut -d" " -f 1; rm tmp)

# remove gaps
fun() {
    echo -e ">seq\nACGT-ACTGC-ACC" | fakit seq -g -l
}
run seq_rmgap_lowercapse fun
assert_in_stdout "acgtactgcacc"

# rna2dna
fun() {
    echo -e ">seq\nUCAUAUGCUUGUCUCAAAGAUUA" | fakit seq --rna2dna
}
run seq_rna2dna fun
assert_in_stdout "TCATATGCTTGTCTCAAAGATTA"
