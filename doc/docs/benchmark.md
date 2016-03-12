# Benchmark

## Softwares

## Datasets

Original datasets included:

- [SILVA_123_SSURef_tax_silva.fasta.gz](http://www.arb-silva.de/fileadmin/silva_databases/current/Exports/SILVA_123_SSURef_tax_silva.fasta.gz)
- [hs_ref_GRCh38.p2_*.mfa.gz](ftp://ftp.ncbi.nlm.nih.gov/refseq/H_sapiens/H_sapiens/Assembled_chromosomes/seq/)

They are so large, so only subsets are used.

1. `dataset_A`. Sampling by proption of 0.1 for `SILVA_123_SSURef_tax_silva.fasta.gz`

        fakit sample SILVA_123_SSURef_tax_silva.fasta.gz -p 0.1 -o dataset_A.fa.gz

2. `dataset_B`. Merging chr18,19,20,21,22,Y to a single file

        zcat hs_ref_GRCh38.p2_chr{18,19,20,21,22,Y}.mfa.gz | pigz -c > dataset_B.fa.gz


 file                   | type  |  num_seqs   |     min_len |  avg_len    |  max_len
------------------------|------ |-------------|-------------|-------------|----------       
dataset_A.fa.gz (48.5M) | RNA   |    175364   |  900        | 1419.6      |  3725
dataset_B.fa.gz (90.6M) | DNA   |    6        | 46709983    | 59698489.0  |  80373285
