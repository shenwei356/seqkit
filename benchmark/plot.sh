#!/bin/sh

./plot.R -i benchmark.5tests.tsv
./plot.R -i benchmark.5tests.tsv -d dataset_C.fq -o benchmark.5tests.tsv.C.png --width 8 --height 4 --lx 0.87 --ly 0.5

# ./plot.R -i benchmark.seqkit.tsv --width 8 --height 3 --lx 0.75 --ly 0.3
