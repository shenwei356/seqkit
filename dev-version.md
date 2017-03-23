## v0.5.2-dev2

- `seqkit stats -a` and `seqkit seq -g -G`: change default gap letters from '- ' to '- .'
- `seqkit subseq`: fix bug of range overflow when using `-d/--down-stream`
    or `-u/--up-stream` for retieving subseq using BED (`--beb`) or GTF (`--gtf`) file.
- `seqkit locate`: add flag `-G/--non-greedy`, non-greedy mode,
 faster but may miss motifs overlaping with others. Example:

    - greedy mode (default)

             $ echo -e '>seq\nACGACGACGA' | seqkit locate -p ACGA | csvtk -t pretty
             seqID   patternName   pattern   strand   start   end   matched
             seq     ACGA          ACGA      +        1       4     ACGA
             seq     ACGA          ACGA      +        4       7     ACGA
             seq     ACGA          ACGA      +        7       10    ACGA

    - non-greedy mode (`-G`)

            $ echo -e '>seq\nACGACGACGA' | seqkit locate -p ACGA -G | csvtk -t pretty
            seqID   patternName   pattern   strand   start   end   matched
            seq     ACGA          ACGA      +        1       4     ACGA
            seq     ACGA          ACGA      +        7       10    ACGA
