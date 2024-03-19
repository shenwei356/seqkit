#!/bin/bash

test -e ssshtest || wget -q https://raw.githubusercontent.com/ryanlayer/ssshtest/master/ssshtest

. ssshtest
set -e

cd seqkit; go build; cd ..;
app=./seqkit/seqkit

set +e

which csvtk || (git clone --depth 1 https://github.com/shenwei356/csvtk; cd ./csvtk/csvtk; go get -v ... ; go build)
CSVTK=csvtk
which csvtk || CSVTK=./csvtk/csvtk/csvtk; true

STOP_ON_FAIL=1

md5sum () { 
	openssl dgst -sha256 $1  | cut -d $' ' -f 2; 
}

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
assert_equal $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1) $($app fx2tab $file | cut -f 1,2 | $app tab2fx -w 0 | md5sum | cut -d" " -f 1)

READS_FQ=tests/pcs109_5k.fq
NANO_FQ_TSV=tests/pcs109_5k_fq_NanoPlot.tsv

float_gt(){
	CODE=$(awk 'BEGIN {PREC="double"; print ("'$1'" >= "'$2'")}')
	return $CODE
}

fun () {
    echo -e "Len\tQual" > seqkit.tsv
    $app fx2tab -q -l $READS_FQ | cut -f 4,5 >> seqkit.tsv
    paste seqkit.tsv $NANO_FQ_TSV > joint.tsv
    $CSVTK corr -t -f Len,lengths joint.tsv 2> corr_len.tsv
    $CSVTK corr -t -f Qual,quals joint.tsv 2> corr_qual.tsv
}
run fx2tab_qual_len fun
RL=$(cut -f 3 corr_len.tsv)
echo Length correlation: $RL
float_gt $RL 0.99
assert_equal $? 1

RL=$(cut -f 3 corr_len.tsv)
echo Length correlation: $RL
float_gt $RL 0.99
assert_equal $? 1

RQ=$(cut -f 3 corr_len.tsv)
echo Qual correlation: $RQ
float_gt $RQ 0.99
assert_equal $? 1
rm seqkit.tsv corr_len.tsv corr_qual.tsv

# ------------------------------------------------------------
#                       grep
# ------------------------------------------------------------

file=tests/hairpin.fa

# by regexp
run grep_by_regexp $app grep -r -p "^hsa" $file
assert_equal $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1) $($app fx2tab $file | grep -E "^hsa" | $app tab2fx | md5sum | cut -d" " -f 1)

# by list

$app fx2tab -n $file -i | cut -f 1 > list
run grep_by_list_all $app grep -f list $file
assert_equal $(cat $STDOUT_FILE | md5sum | cut -d" " -f 1) $(md5sum $file | cut -d" " -f 1)
rm list

cat $file | $app head -n 100 | $app seq -n -i > list
run grep_by_list_head100 $app grep -f list $file
assert_equal $($app fx2tab $STDOUT_FILE | wc -l) 100
rm list

echo -en "Homo\nMus\n" > list
run grep_by_regexp_list $app grep -r -n -f list $file
assert_equal $($app fx2tab $STDOUT_FILE | wc -l) $($app seq -n $file | grep -E "Homo|Mus" | wc -l)
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

#-------------------------------------------------------------
#                       bam
#-------------------------------------------------------------
BAM=tests/pcs109_5k.bam
PRIM_BAM=tests/pcs109_5k_prim.bam
SPLICE_BAM=tests/pcs109_5k_spliced.bam
PRIM_NANOPLOT=tests/pcs109_5k_prim_bam_NanoPplot.tsv
PRIM_WUB=tests/pcs109_5k_bam_alignment_length.tsv
WUB_CLIP=tests/pcs109_5k_bam_soft_clips_tab.tsv
PCS_FQ=tests/pcs109_5k.fq

float_gt(){
CODE=$(awk 'BEGIN {PREC="double"; print ("'$1'" >= "'$2'")}')
return $CODE
}

# accuracy
fun(){
    $app bam -f Read,Acc $PRIM_BAM 2> seqkit_acc.tsv
    paste seqkit_acc.tsv $PRIM_NANOPLOT > joint.tsv
    $CSVTK corr -t -f Acc,percentIdentity joint.tsv 2> corr.tsv
}
run bam_acc fun
R=$(cut -f 3 corr.tsv)
echo Correlation: $R
float_gt $R 0.99
assert_equal $? 1
rm corr.tsv seqkit_acc.tsv joint.tsv

# MeanQual
fun(){
    $app bam -f Read,MeanQual $PRIM_BAM 2> seqkit.tsv
    paste seqkit.tsv $PRIM_NANOPLOT > joint.tsv
    $CSVTK corr -t -f MeanQual,quals joint.tsv 2> corr.tsv
}
run bam_mean_qual fun
R=$(cut -f 3 corr.tsv)
echo Correlation: $R
float_gt $R 0.99
assert_equal $? 1
rm corr.tsv seqkit.tsv joint.tsv

# MapQual
fun(){
    $app bam -f Read,MapQual $PRIM_BAM 2> seqkit.tsv
    paste seqkit.tsv $PRIM_NANOPLOT > joint.tsv
    $CSVTK corr -t -f MapQual,mapQ joint.tsv 2> corr.tsv
}
run bam_map_qual fun
R=$(cut -f 3 corr.tsv)
echo Correlation: $R
float_gt $R 0.99
assert_equal $? 1
rm corr.tsv seqkit.tsv joint.tsv

# ReadLen
fun(){
    $app bam -f Read,ReadLen $PRIM_BAM 2> seqkit.tsv
    paste seqkit.tsv $PRIM_NANOPLOT > joint.tsv
    $CSVTK corr -t -f ReadLen,lengths joint.tsv 2> corr.tsv
}
run bam_read_len fun
R=$(cut -f 3 corr.tsv)
echo Correlation: $R
float_gt $R 0.99
assert_equal $? 1
rm corr.tsv seqkit.tsv joint.tsv

# ReadAln
fun(){
    $app bam -f Read,ReadAln $PRIM_BAM 2> seqkit.tsv
    paste seqkit.tsv $PRIM_NANOPLOT > joint.tsv
    $CSVTK corr -t -f ReadAln,aligned_lengths joint.tsv 2> corr.tsv
}
run bam_read_aln fun
R=$(cut -f 3 corr.tsv)
echo Correlation: $R
float_gt $R 0.99
assert_equal $? 1
rm corr.tsv seqkit.tsv joint.tsv

# LeftClip
fun(){
    $app bam -f Read,LeftClip $PRIM_BAM 2> seqkit.tsv
    paste seqkit.tsv $WUB_CLIP > joint.tsv
    head -1 joint.tsv > TMP
    grep "\+" joint.tsv >> TMP
    mv TMP joint.tsv
    $CSVTK corr -t -f LeftClip,ClipStart joint.tsv 2> corr.tsv
}
run bam_left_clip fun
R=$(cut -f 3 corr.tsv)
echo Correlation: $R
float_gt $R 0.99
assert_equal $? 1
rm corr.tsv seqkit.tsv joint.tsv

# RightClip
fun(){
    $app bam -f Read,RightClip $PRIM_BAM 2> seqkit.tsv
    paste seqkit.tsv $WUB_CLIP > joint.tsv
    head -1 joint.tsv > TMP
    grep "\+" joint.tsv >> TMP
    mv TMP joint.tsv
    $CSVTK corr -t -f RightClip,ClipEnd joint.tsv 2> corr.tsv
}
run bam_right_clip fun
R=$(cut -f 3 corr.tsv)
echo Correlation: $R
float_gt $R 0.99
assert_equal $? 1
rm corr.tsv seqkit.tsv joint.tsv

# bundling with -N
fun(){
    rm -fr tests/bundler_test tests/bundler_merged.bam
    $app bam -N -1 $SPLICE_BAM -o tests/bundler_test
    ($app bam -s $SPLICE_BAM 2>&1) | cut -f 1,8 | sed '1d' > tests/bundler_stats_bulk.tsv
    ($app bam -s tests/bundler_test/*.bam 2>&1) \
    | cut -f 1,8 | sed '1d' | $CSVTK -H -t summary -w 0 -g 1 -f "2:sum" > tests/bundler_stats_merged.tsv
}
run bam_bundler fun
cmp tests/bundler_stats_merged.tsv tests/bundler_stats_bulk.tsv
assert_equal $? 0
rm -fr tests/bundler_test tests/bundler_stats_merged.tsv tests/bundler_stats_bulk.tsv 

# ------------------------------------------------------------
#                       fish
# ------------------------------------------------------------

# Regression test for fish
fun(){
    Q1="GTTGTTATGGAGGATACTTTCCTACCGTGACAAGAAAGTTGT"
    Q2="GCCAGTAGACAAGTTTCTCCATCTCCGGCCTTTT"
    Q3="CAGTATGCTTCGTTTCAATTTCGGGTTTGGAGTGTTTG"
    Q4="TTTTATCAAAAGAAAAAAAAGAAGATAGAGCGACAGGCAAGTCACAAAGACACCGACAACTTTCTTGTCATC"
    head -n 40 $PCS_FQ > TMP.fq
    $app fish -q 40 -g -F "$Q1,$Q2,$Q3,$Q4" TMP.fq 2> seqkit_fish.tsv
    rm TMP.fq
}
run bam_fish_regression fun
cmp seqkit_fish.tsv tests/pcs109_5k_fish_regression.tsv
assert_exit_code 0
rm seqkit_fish.tsv

# ------------------------------------------------------------
#                       sana
# ------------------------------------------------------------

# sana/fastq test for IDs in separator
fun(){
	$app  sana tests/sana_sep_id.fq > tests/sana_output.fq
}
run sana_fastq_sep_id fun
cmp tests/sana_output.fq <(tail -4 tests/sana_sep_id.fq)
assert_equal $? 0
rm -f tests/sana_output.fq

# Regression test for sana/fasta
fun(){
	awk '{print ">" $1 "\n" $2}' tests/scat_test.tsv > tests/sana_test_input.fas
	$app sana -i fasta tests/sana_test_input.fas > tests/sana_output.fas
}
run sana_fasta_regression fun
cmp tests/sana_output.fas tests/sana_ground.fas
assert_equal $? 0
rm -f tests/sana_output.fas tests/sana_test_input.fas

# Regression test for sana/fastq
fun(){
	awk '{print "@" $1 "\n" $2 "\n+\n" $3}' tests/scat_test.tsv > tests/sana_test_input.fq
	# $app  sana -i fastq tests/sana_test_input.fq > tests/sana_output.fq
	$app  sana tests/sana_test_input.fq > tests/sana_output.fq
}
run sana_fastq_regression_empty_line fun
cmp tests/sana_output.fq tests/sana_ground.fq
assert_equal $? 0
rm -f tests/sana_output.fq tests/sana_test_input.fq

# Regression test for sana/fasta empty file issue
fun(){
	$app  sana -j 2 -i fasta tests/empty.fx
}
run sana_fasta_regression_empty_line fun
assert_equal $? 0


# Regression test for sana/fastq empty file issue
fun(){
	$app  sana -j 2 -i fastq tests/empty.fx
}
run sana_fastq_regression fun
assert_equal $? 0


# ------------------------------------------------------------
#                       scat
# ------------------------------------------------------------

# Regression test for scat/fasta
fun(){
	BASE=tests/scat_test_fasta
	rm -fr $BASE 
	rm -f tests/scat_test_all.fas tests/scat_output.fas
        mkdir -p $BASE
	($app scat -j 4 -i fasta $BASE > tests/scat_output.fas)&
	SCAT_PID=$!
        BAK=$IFS
        IFS=$'\n'
	SIZE=2
        for i in `seq 0 $SIZE`;
        do
                for j in `seq 0 $SIZE`;
                do
			D=$BASE/$RANDOM/$RANDOM
			mkdir -p $D
                        F=$D/${RANDOM}.fas
                        for l in `cat tests/scat_test.tsv`;
                        do
                                PRE=".${RANDOM}.${i}.${j}"
                                echo -n $l | awk -v pre="$PRE" '{print ">" $1 pre  "\n" $2}' - >> $F
                                echo -n $l | awk -v pre="$PRE" '{print ">" $1 pre  "\n" $2}' - >> tests/scat_test_all.fas
				sync;
                        done;
		done;
        done;
        IFS=$BAK
	sync; sleep 0.5
	$app sana -j 4 -i fasta tests/scat_test_all.fas | $app sort -n -j 1 - > tests/sorted_scat_test_all.fas
	kill -s INT $SCAT_PID
	sync; sleep 0.5
	wait $SCAT_PID
	sync; 
	$app scat -f -j 4 -i fasta $BASE | $app sort -n -j 1 - > tests/sorted_scat_find.fas
	$app sort -n -j 1 tests/scat_output.fas > tests/sorted_scat_output.fas
	rm -fr $BASE
	rm -f tests/scat_test_all.fas tests/scat_output.fas
}

run scat_fasta fun
cmp tests/sorted_scat_output.fas tests/sorted_scat_test_all.fas
assert_equal $? 0
cmp tests/sorted_scat_find.fas tests/sorted_scat_test_all.fas
assert_equal $? 0
rm -f tests/sorted_scat_output.fas tests/sorted_scat_test_all.fas tests/sorted_scat_find.fas

# Regression test for scat/fastq
fun(){
	BASE=tests/scat_test_fastq
	rm -fr $BASE 
	rm -f tests/scat_test_all.fq tests/scat_output.fq
        mkdir -p $BASE
	($app scat -j 4 -i fastq $BASE > tests/scat_output.fq)&
	SCAT_PID=$!
        BAK=$IFS
        IFS=$'\n'
	SIZE=2
        for i in `seq 0 $SIZE`;
        do
                for j in `seq 0 $SIZE`;
                do
                        D=$BASE/$RANDOM/$RANDOM
                        mkdir -p $D
                        F=$D/${RANDOM}.fq
                        for l in `cat tests/scat_test.tsv`;
                        do
                                PRE=".${RANDOM}.${i}.${j}"
                                echo -n $l | awk -v pre="$PRE" '{print "@" $1 pre  "\n" $2 "\n+\n" $3}' - >> $F
                                echo -n $l | awk -v pre="$PRE" '{print "@" $1 pre  "\n" $2 "\n+\n" $3}' - >> tests/scat_test_all.fq
                        done;
		done;
        done;
        IFS=$BAK
	sync; sleep 0.5
	$app sana -j 4 -i fastq tests/scat_test_all.fq > tests/scat_test_all_sana.fq
	kill -s INT $SCAT_PID
	wait $SCAT_PID
	sync; sleep 0.5
	$app scat -f -j 4 -i fastq $BASE | $app sort -n -j 1 - > tests/sorted_scat_find.fq
	$app sort -n -j 1 tests/scat_test_all_sana.fq > tests/sorted_scat_test_all.fq
	$app sort -n -j 1 tests/scat_output.fq > tests/sorted_scat_output.fq
	rm -fr $BASE
	rm -f tests/scat_test_all.fq tests/scat_output.fq
}

run scat_fastq fun
cmp tests/sorted_scat_output.fq tests/sorted_scat_test_all.fq
assert_equal $? 0
cmp tests/sorted_scat_find.fq tests/sorted_scat_test_all.fq
assert_equal $? 0
rm -f tests/sorted_scat_output.fq tests/sorted_scat_test_all.fq tests/sorted_scat_find.fq tests/scat_test_all_sana.fq

# ------------------------------------------------------------
#                       faidx
# ------------------------------------------------------------

file=tests/hairpin.fa
idFile=tests/t.ids
outFile=tests/t.fa
$app sample -n 10 $file| $app seq -i -n | shuf > $idFile

fun(){
    $app faidx $file $(paste -s -d ' ' $idFile) > $outFile
}
run faidx_id fun
assert_equal $($app grep -f $idFile $file | $app seq -i | $app sort | md5sum | cut -d" " -f 1) $(cat $outFile | $app sort | md5sum | cut -d" " -f 1)
rm $outFile

fun(){
    $app faidx $file $(paste -s -d ' ' $idFile) -f > $outFile
}
run faidx_full_head fun
assert_equal $($app grep -f $idFile $file | $app sort | md5sum | cut -d" " -f 1) $(cat $outFile | $app sort | md5sum | cut -d" " -f 1)
rm $outFile


ref=$(head -n 1 $idFile)
fun(){
    $app faidx $file "${ref}:5--5" -f > $outFile
}
run faidx_region fun
assert_equal $($app grep -p $ref $file | $app subseq -r 5:-5 | $app seq -s -w 0) $(cat $outFile | $app seq -s -w 0)
rm $idFile $outFile
