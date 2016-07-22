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
run seq_content seqkit seq -w 60 $file
# seq number
assert_equal $(grep -c "^>" $file) $(grep -c "^>" $STDOUT_FILE)
# seq content
assert_equal $(md5sum $file | cut -d" " -f 1) $(md5sum $STDOUT_FILE | cut -d" " -f 1)

# ------------------------------------------------------------

# seq type
run seq_type seqkit seq -t dna $file
assert_in_stderr "invalid DNAredundant letter"

fun() {
    echo -e ">seq\nabcdefghijklmnpqrstvwyz" | seqkit stat
}
run seq_type fun
assert_in_stdout "Protein"

fun() {
    echo -e ">seq\nACGUN ACGUN" | seqkit stat
}
run seq_type fun
assert_in_stdout "RNA"

fun() { 
    echo -e ">seq\nacgtryswkmbdhvACGTRYSWKMBDHV" | seqkit stat 
}
run seq_type fun
assert_in_stdout "DNA"

fun() {
    echo -e "@read\nACTGCN\n+\n@IICCG" | seqkit stat
}
run seq_type fun
assert_in_stdout "DNA"
assert_in_stdout "FASTQ"


# ------------------------------------------------------------

# head
run seq_head seqkit seq -n $file 
assert_equal $(md5sum $STDOUT_FILE | cut -d" " -f 1) $(grep  "^>" $file | sed -s 's/^>//g' > tmp; md5sum tmp | cut -d" " -f 1; rm tmp)

# id
run seq_id seqkit seq -n -i $file 
assert_equal $(md5sum $STDOUT_FILE | cut -d" " -f 1) $(grep  "^>" $file | sed -s 's/^>//g' | sed -s 's/ .*//g' > tmp; md5sum tmp | cut -d" " -f 1; rm tmp)

# seq

file="reads_1.fq.gz"
run seq_seq seqkit seq $file -s -w 0
assert_equal $(md5sum $STDOUT_FILE | cut -d" " -f 1) $(seqkit fx2tab $file | cut -f 2 > tmp; md5sum tmp | cut -d" " -f 1; rm tmp)


# ------------------------------------------------------------

file="hairpin.fa"

# reverse complement
fun() {
    seqkit head -n 1 $file | seqkit seq -r | seqkit seq -p
}
run seq_revcom fun
assert_equal $(md5sum $STDOUT_FILE | cut -d" " -f 1) $(seqkit head -n 1 $file | seqkit seq -r -p > tmp; md5sum tmp | cut -d" " -f 1; rm tmp)

# remove gaps
fun() {
    echo -e ">seq\nACGT-ACTGC-ACC" | seqkit seq -g -l
}
run seq_rmgap_lowercapse fun
assert_in_stdout "acgtactgcacc"

# rna2dna
fun() {
    echo -e ">seq\nUCAUAUGCUUGUCUCAAAGAUUA" | seqkit seq --rna2dna
}
run seq_rna2dna fun
assert_in_stdout "TCATATGCTTGTCTCAAAGATTA"

# ------------------------------------------------------------
#                                 head 
# ------------------------------------------------------------

run head seqkit head -n 10 $file
assert_equal 10 $(grep -c ">" $STDOUT_FILE)


# ------------------------------------------------------------
#                                 subseq
# ------------------------------------------------------------

seq=">seq\nacgtnACGTN"

# by region
fun () {    
    echo -en $seq | seqkit subseq -r 1:1 | seqkit seq -s -w 0
}
run subseq_region fun
assert_equal a $(cat $STDOUT_FILE)

fun () {    
    echo -en $seq | seqkit subseq -r 1:-1 | seqkit seq -s -w 0
}
run subseq_region fun
assert_equal acgtnACGTN $(cat $STDOUT_FILE) 

fun () {    
    echo -en $seq | seqkit subseq -r 3:5 | seqkit seq -s -w 0
}
run subseq_region fun
assert_equal gtn $(cat $STDOUT_FILE) 

fun () {    
    echo -en $seq | seqkit subseq -r -5:-3 | seqkit seq -s -w 0
}
run subseq_region fun
assert_equal ACG $(cat $STDOUT_FILE) 

fun () {    
    echo -en $seq | seqkit subseq -r -1:-1 | seqkit seq -s -w 0
}
run subseq_region fun
assert_equal N $(cat $STDOUT_FILE)

# ------------------------------------------------------------
# gtf
# seq=">seq\nacgtnACGTN"
gtf="seq\ttest\tCDS\t4\t6\t.\t+\t.\tgene_id \"A\"; transcript_id \"A\"\nseq\ttest\tCDS\t4\t6\t.\t-\t.\tgene_id \"B\"; transcript_id \"B\"\n"

fun () {    
    echo -en $seq | seqkit subseq --gtf <(echo -ne $gtf) | seqkit seq -s -w 0 | paste -sd"+"
}
run subseq_gtf fun
assert_equal "tnA+Tna" $(cat $STDOUT_FILE)

fun () {    
    echo -en $seq | seqkit subseq --gtf <(echo -ne $gtf) -u 3 -d 2 | seqkit seq -s -w 0 | paste -sd"+"
}
run subseq_gtf fun
assert_equal "acgtnACG+ACGTnacg" $(cat $STDOUT_FILE)

fun () {    
    echo -en $seq | seqkit subseq --gtf <(echo -ne $gtf) -u 3 -f | seqkit seq -s -w 0 | paste -sd"+"
}
run subseq_gtf fun
assert_equal "acg+ACG" $(cat $STDOUT_FILE)


# ------------------------------------------------------------
#                                 sliding
# ------------------------------------------------------------
fun () {    
    echo -en $seq | seqkit sliding -W 5 -s 5 | seqkit seq -s -w 0 | paste -sd"+"
}
run subseq_sliding fun
assert_equal "acgtn+ACGTN" $(cat $STDOUT_FILE)


# ------------------------------------------------------------
#                            fq2fa, fx2tab, tab2fx
# ------------------------------------------------------------
file="hairpin.fa"
fun () {    
    seqkit fx2tab $file | seqkit tab2fx
}
run fx2tab_tab2fx fun
assert_equal $(md5sum $STDOUT_FILE | cut -d" " -f 1) $(seqkit seq $file > tmp; md5sum tmp | cut -d" " -f 1; rm tmp)


file="reads_1.fq.gz"
run fq2fa seqkit fq2fa $file
assert_equal $(md5sum $STDOUT_FILE | cut -d" " -f 1) $(seqkit fx2tab $file | cut -f 1,2 | seqkit tab2fx > tmp; md5sum tmp | cut -d" " -f 1; rm tmp)

# ------------------------------------------------------------
#                       grep
# ------------------------------------------------------------
file="hairpin.fa"
# by regexp
run grep_by_regexp seqkit grep -r -p "^hsa" $file 
assert_equal $(md5sum $STDOUT_FILE | cut -d" " -f 1) $(seqkit fx2tab $file | grep -E "^hsa" | seqkit tab2fx > tmp; md5sum tmp | cut -d" " -f 1; rm tmp)

# by list
seqkit fx2tab -n hairpin.fa -i | cut -f 1 > list
run grep_by_list seqkit grep -f list $file
assert_equal $(md5sum $STDOUT_FILE | cut -d" " -f 1) $(md5sum $file | cut -d" " -f 1)
rm list

# ------------------------------------------------------------
#                       locate
# ------------------------------------------------------------


