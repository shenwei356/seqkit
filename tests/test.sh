#!/bin/bash

test -e ssshtest || wget -q https://raw.githubusercontent.com/ryanlayer/ssshtest/master/ssshtest

. ssshtest
set -e

cd seqkit; go build; cd ..;
app=./seqkit/seqkit

set +e

STOP_ON_FAIL=1

# ------------------------------------------------------------
#                        seq
# ------------------------------------------------------------

file=tests/hairpin.fa

# seq content
run seq_content $app seq -w 60 $file
# seq number
assert_equal $(grep -c "^>" $file) $(grep -c "^>" $STDOUT_FILE)
# seq content
assert_equal $(cat $file | md5sum | cut -d" " -f 1) $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1)

# ------------------------------------------------------------

# seq type
run seq_type $app seq -t dna $file
assert_in_stderr "invalid DNAredundant letter"

fun() {
    echo -e ">seq\nabcdefghijklmnpqrstvwyz" | $app stat
}
run seq_type fun
assert_in_stdout "Protein"

fun() {
    echo -e ">seq\nACGUN ACGUN" | $app stat
}
run seq_type fun
assert_in_stdout "RNA"

fun() {
    echo -e ">seq\nacgtryswkmbdhvACGTRYSWKMBDHV" | $app stat
}
run seq_type fun
assert_in_stdout "DNA"

fun() {
    echo -e "@read\nACTGCN\n+\n@IICCG" | $app stat
}
run seq_type fun
assert_in_stdout "DNA"
assert_in_stdout "FASTQ"


# ------------------------------------------------------------

# head
run seq_head $app seq -n $file
assert_equal $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1) $(grep  "^>" $file | sed 's/^>//g' | md5sum | cut -d" " -f 1)

# id
run seq_id $app seq -n -i $file
assert_equal $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1) $(grep  "^>" $file | sed 's/^>//g' | sed 's/ .*//g' | md5sum | cut -d" " -f 1)

# seq
run seq_seq $app seq $file -s -w 0
assert_equal $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1) $($app fx2tab $file | cut -f 2 | md5sum | cut -d" " -f 1)


# ------------------------------------------------------------

file=tests/hairpin.fa

# reverse complement
fun() {
    $app head -n 1 $file | $app seq -r | $app seq -p
}
run seq_revcom fun
assert_equal $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1) $($app head -n 1 $file | $app seq -r -p | md5sum | cut -d" " -f 1)

# remove gaps
fun() {
    echo -e ">seq\nACGT-ACTGC-ACC" | $app seq -g -l
}
run seq_rmgap_lowercapse fun
assert_in_stdout "acgtactgcacc"

# rna2dna
fun() {
    echo -e ">seq\nUCAUAUGCUUGUCUCAAAGAUUA" | $app seq --rna2dna
}
run seq_rna2dna fun
assert_in_stdout "TCATATGCTTGTCTCAAAGATTA"

# ------------------------------------------------------------
#                         subseq
# ------------------------------------------------------------

testseq() {
    echo -e ">seq\nacgtnACGTN"
}

# by region
fun () {
    testseq | $app subseq -r 1:1 | $app seq -s -w 0
}
run subseq_region fun
assert_equal a $(cat $STDOUT_FILE)

fun () {
    testseq | $app subseq -r 1:-1 | $app seq -s -w 0
}
run subseq_region fun
assert_equal acgtnACGTN $(cat $STDOUT_FILE)

fun () {
    testseq | $app subseq -r 3:5 | $app seq -s -w 0
}
run subseq_region fun
assert_equal gtn $(cat $STDOUT_FILE)

fun () {
    testseq | $app subseq -r -5:-3 | $app seq -s -w 0
}
run subseq_region fun
assert_equal ACG $(cat $STDOUT_FILE)

fun () {
    testseq | $app subseq -r -1:-1 | $app seq -s -w 0
}
run subseq_region fun
assert_equal N $(cat $STDOUT_FILE)

# ------------------------------------------------------------
# gtf
# seq=">seq\nacgtnACGTN"
gtf="seq\ttest\tCDS\t4\t6\t.\t+\t.\tgene_id \"A\"; transcript_id \"A\"\nseq\ttest\tCDS\t4\t6\t.\t-\t.\tgene_id \"B\"; transcript_id \"B\"\n"

fun () {
    testseq | $app subseq --gtf <(echo -ne $gtf) | $app seq -s -w 0
}
run subseq_gtf fun
assert_equal $(echo -e "tnA\nTna" | md5sum | cut -d" " -f 1)  $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1)

fun () {
    testseq | $app subseq --gtf <(echo -ne $gtf) -u 3 -d 2 | $app seq -s -w 0
}
run subseq_gtf fun
assert_equal $(echo -e "acgtnACG\nACGTnacg" | md5sum | cut -d" " -f 1) $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1)

fun () {
    testseq | $app subseq --gtf <(echo -ne $gtf) -u 100 -d 100 | $app seq -s -w 0
}
run subseq_gtf fun
assert_equal $(echo -e "acgtnACGTN\nNACGTnacgt" | md5sum | cut -d" " -f 1) $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1)

fun () {
    testseq | $app subseq --gtf <(echo -ne $gtf) -u 3 -f | $app seq -s -w 0
}
run subseq_gtf fun
assert_equal $(echo -e "acg\nACG" | md5sum | cut -d" " -f 1) $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1)



# ------------------------------------------------------------
#                                 sliding
# ------------------------------------------------------------
testseq() {
    echo -e ">seq\nacgtnACGTN"
}
fun () {
    testseq | $app sliding -W 5 -s 5 | $app seq -s -w 0
}
run sliding fun
assert_equal $(echo -e "acgtn\nACGTN" | md5sum | cut -d" " -f 1) $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1)


# ------------------------------------------------------------
#                            fq2fa, fx2tab, tab2fx
# ------------------------------------------------------------

file=tests/hairpin.fa

fun () {
    $app fx2tab $file | $app tab2fx
}
run fx2tab_tab2fx fun
assert_equal $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1) $($app seq $file | md5sum | cut -d" " -f 1)


file=tests/reads_1.fq.gz
run fq2fa $app fq2fa $file
assert_equal $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1) $($app fx2tab $file | cut -f 1,2 | $app tab2fx | md5sum | cut -d" " -f 1)

# ------------------------------------------------------------
#                       grep
# ------------------------------------------------------------

file=tests/hairpin.fa

# by regexp
run grep_by_regexp $app grep -r -p "^hsa" $file
assert_equal $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1) $($app fx2tab $file | grep -E "^hsa" | $app tab2fx | md5sum | cut -d" " -f 1)

# by list
$app fx2tab -n $file -i | cut -f 1 > list
run grep_by_list $app grep -f list $file
assert_equal $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1) $(md5sum $file | cut -d" " -f 1)
rm list

# ------------------------------------------------------------
#                       locate
# ------------------------------------------------------------


# ------------------------------------------------------------
#                       rmdup
# ------------------------------------------------------------
testseq() {
    echo -e ">seq\nacgtnACGTN"
}
repeated_seq() {
    for i in $(seq 10); do
        testseq
    done
}
fun() {
    repeated_seq | $app rmdup
}
run rmdup fun
assert_in_stderr "9 duplicated records removed"
assert_equal $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1) $(testseq | md5sum | cut -d" " -f 1)

fun() {
    repeated_seq | $app rmdup -s
}
run "rmdup -s" fun
assert_in_stderr "9 duplicated records removed"
assert_equal $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1) $(testseq | md5sum | cut -d" " -f 1)

# ------------------------------------------------------------
#                       common
# ------------------------------------------------------------

file=tests/hairpin.fa

$app rmdup $file > t.1
$app sample t.1 -p 0.1 > t.2
fun() {
    $app common t.1 t.2 > t.c
}
run common fun
assert_equal $(cat t.c | $app stat -a | md5sum | cut -d" " -f 1) $(cat t.2 | $app stat -a | md5sum | cut -d" " -f 1)
rm t.*


# ------------------------------------------------------------
#                       split
# ------------------------------------------------------------

file=tests/hairpin.fa

testseq() {
    cat $file | $app head -n 100 | $app rmdup
}
fun() {
    testseq | $app split -i -f
}
run split fun
assert_equal $(ls stdin.split/* | wc -l  | tail -n 1) $(testseq | $app seq -n -i | wc -l | tail -n 1)
assert_equal $(cat stdin.split/* | $app stat -a | md5sum | cut -d" " -f 1) $(testseq | $app stat -a | md5sum | cut -d" " -f 1)
rm -r stdin.split

# ------------------------------------------------------------
#                       sample
# ------------------------------------------------------------
file=tests/hairpin.fa
assert_equal $(cat $file | $app sample -p 0.1 | $app stat -a | md5sum | cut -d" " -f 1) $(cat $file | $app sample -p 0.1 | $app stat -a | md5sum | cut -d" " -f 1)


# ------------------------------------------------------------
#                       head
# ------------------------------------------------------------

run head $app head -n 10 $file
assert_equal 10 $(grep -c ">" $STDOUT_FILE)


# ------------------------------------------------------------
#                       replace
# ------------------------------------------------------------
testseq() {
    echo -e ">seq\nacgtnACGTN"
}
assert_equal $(testseq | $app replace -p e -r n | $app seq -n -i) snq

# ------------------------------------------------------------
#                       rename
# ------------------------------------------------------------

testseq() {
    echo -e ">seq\na\n>seq\nc"
}
assert_equal $(testseq | $app rename | $app seq -n -i | tail -n 1)  seq_2



# ------------------------------------------------------------
#                       restart
# ------------------------------------------------------------
testseq() {
    echo -e ">seq\nacgtnACGTN"
}
fun(){
    testseq | $app restart -i 6
}
run restart fun
assert_equal $(cat $STDOUT_FILE | $app seq -s) "ACGTNacgtn"

fun(){
    testseq | $app restart -i -5
}
run restart2 fun
assert_equal $(cat $STDOUT_FILE | $app seq -s) "ACGTNacgtn"



# ------------------------------------------------------------
#                       shuffle and sort
# ------------------------------------------------------------

file=tests/hairpin.fa

fun(){
    $app seq $file > t.shu.0
    $app shuffle -s 1 $file > t.shu.1
    $app shuffle -s 1 $file > t.shu.2
    $app sort -l $file > t.sort.l
    $app sort -n $file > t.sort.n
    $app sort -s $file > t.sort.s
}
run shuffle fun
assert_equal $(cat t.shu.1 | $app stat -a | md5sum | cut -d" " -f 1) $(cat t.shu.0 | $app stat -a | md5sum | cut -d" " -f 1)
assert_equal $(cat t.shu.1 | md5sum | cut -d" " -f 1) $(cat t.shu.2 | md5sum | cut -d" " -f 1)
rm t.shu.*

assert_equal $(cat $file | $app stat -a | md5sum | cut -d" " -f 1) $(cat t.sort.l | $app stat -a | md5sum | cut -d" " -f 1)
assert_equal $(cat $file | $app stat -a | md5sum | cut -d" " -f 1) $(cat t.sort.n | $app stat -a | md5sum | cut -d" " -f 1)
assert_equal $(cat $file | $app stat -a | md5sum | cut -d" " -f 1) $(cat t.sort.s | $app stat -a | md5sum | cut -d" " -f 1)
rm t.sort.*
