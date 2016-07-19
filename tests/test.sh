#!/bin/bash

test -e ssshtest || wget -q https://raw.githubusercontent.com/ryanlayer/ssshtest/master/ssshtest

. ssshtest

set -o nounset

STOP_ON_FAIL=1

# ------------------------------------------------------------

file="hairpin.fa"

# ------------------------------------------------------------



# ------------------------------------------------------------
#                                 seq 
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

# ------------------------------------------------------------
#                                 head 
# ------------------------------------------------------------

run head fakit head -n 10 $file
assert_equal 10 $(grep -c ">" $STDOUT_FILE)


# ------------------------------------------------------------
#                                 subseq
# ------------------------------------------------------------

seq=">seq\nacgtnACGTN"

# by region
fun () {    
    echo -en $seq | fakit subseq -r 1:1 | fakit seq -s -w 0
}
run subseq_region fun
assert_equal a $(cat $STDOUT_FILE)

fun () {    
    echo -en $seq | fakit subseq -r 1:-1 | fakit seq -s -w 0
}
run subseq_region fun
assert_equal acgtnACGTN $(cat $STDOUT_FILE) 

fun () {    
    echo -en $seq | fakit subseq -r 3:5 | fakit seq -s -w 0
}
run subseq_region fun
assert_equal gtn $(cat $STDOUT_FILE) 

fun () {    
    echo -en $seq | fakit subseq -r -5:-3 | fakit seq -s -w 0
}
run subseq_region fun
assert_equal ACG $(cat $STDOUT_FILE) 

fun () {    
    echo -en $seq | fakit subseq -r -1:-1 | fakit seq -s -w 0
}
run subseq_region fun
assert_equal N $(cat $STDOUT_FILE)

# ------------------------------------------------------------
# gtf
# seq=">seq\nacgtnACGTN"
gtf="seq\ttest\tCDS\t4\t6\t.\t+\t.\tgene_id \"A\"; transcript_id \"A\"\nseq\ttest\tCDS\t4\t6\t.\t-\t.\tgene_id \"B\"; transcript_id \"B\"\n"

fun () {    
    echo -en $seq | fakit subseq --gtf <(echo -ne $gtf) | fakit seq -s -w 0 | paste -sd"+"
}
run subseq_gtf fun
assert_equal "tnA+Tna" $(cat $STDOUT_FILE)

fun () {    
    echo -en $seq | fakit subseq --gtf <(echo -ne $gtf) -u 3 -d 2 | fakit seq -s -w 0 | paste -sd"+"
}
run subseq_gtf fun
assert_equal "acgtnACG+ACGTnacg" $(cat $STDOUT_FILE)

fun () {    
    echo -en $seq | fakit subseq --gtf <(echo -ne $gtf) -u 3 -f | fakit seq -s -w 0 | paste -sd"+"
}
run subseq_gtf fun
assert_equal "acg+ACG" $(cat $STDOUT_FILE)


# ------------------------------------------------------------
#                                 sliding
# ------------------------------------------------------------
fun () {    
    echo -en $seq | fakit sliding -W 5 -s 5 | fakit seq -s -w 0 | paste -sd"+"
}
run subseq_sliding fun
assert_equal "acgtn+ACGTN" $(cat $STDOUT_FILE)


# ------------------------------------------------------------
#                            fq2fa, fx2tab, tab2fx
# ------------------------------------------------------------
file="hairpin.fa"
fun () {    
    fakit fx2tab $file | fakit tab2fx
}
run fx2tab_tab2fx fun
assert_equal $(md5sum $STDOUT_FILE | cut -d" " -f 1) $(fakit seq $file > tmp; md5sum tmp | cut -d" " -f 1; rm tmp)


file="reads_1.fq.gz"
run fq2fa fakit fq2fa $file
assert_equal $(md5sum $STDOUT_FILE | cut -d" " -f 1) $(fakit fx2tab $file | cut -f 1,2 | fakit tab2fx > tmp; md5sum tmp | cut -d" " -f 1; rm tmp)

# ------------------------------------------------------------
#                       grep
# ------------------------------------------------------------
file="hairpin.fa"
# by regexp
run grep_by_regexp fakit grep -r -p "^hsa" $file 
assert_equal $(md5sum $STDOUT_FILE | cut -d" " -f 1) $(fakit fx2tab $file | grep -E "^hsa" | fakit tab2fx > tmp; md5sum tmp | cut -d" " -f 1; rm tmp)

# by list
fakit fx2tab -n hairpin.fa -i | cut -f 1 > list
run grep_by_list fakit grep -f list $file
assert_equal $(md5sum $STDOUT_FILE | cut -d" " -f 1) $(md5sum $file | cut -d" " -f 1)
rm list

# ------------------------------------------------------------
#                       locate
# ------------------------------------------------------------


