v0.5.3-dev

-  `seqkit grep`: fix bug when using `seqkit grep -r -f patternfile`:
    all records will be retrived due to failt to discarding the blank pattern (`""`). #11
