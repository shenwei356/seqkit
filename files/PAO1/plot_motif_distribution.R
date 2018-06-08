#!/usr/bin/env Rscript
library(ggplot2)
library(plyr)
library(ggthemes)
library(scales)
# library(som)
library(Biostrings)

motifs <- readDNAStringSet("motifs.fa")
motif.names = names(motifs)

df <- read.csv("PAO1.fasta.sliding.fa.motifs.tsv2", sep = "\t")

df <- ddply(df, .(seqID, pattern), summarise, N = length(start))


df$pattern <- factor(df$pattern, levels = motif.names, ordered = TRUE)

df2 <- 
  ddply(
    df, .(pattern), summarise, 
    mean=mean(N),
    min=min(N),
    max=max(N)
  )


p <-
  ggplot(df) +
  geom_line(aes(
    x = seqID, y = N, color = pattern, group = pattern
  )) +
  geom_hline(data=df2, aes(group = pattern, yintercept=min), linetype=1, size=0.2) + 
  geom_hline(data=df2, aes(group = pattern, yintercept=max), linetype=1, size=0.2) + 
  geom_hline(data=df2, aes(group = pattern, yintercept=mean), linetype=2, size=0.2) +  # mean
  
  scale_x_continuous(breaks = seq(0, max(df$seqID), by = 1000000),
                     labels = comma) +
  facet_grid(pattern ~ .) +
  scale_color_stata() +
  ylab("Counts") +
  xlab("Position(bp)") +
  ggtitle("Motif Distribution")

p <- p +
  theme_bw() +
  theme(
    panel.border = element_blank(),
    panel.background = element_blank(),
    panel.grid.major = element_blank(),
    panel.grid.minor = element_blank(),
    axis.line = element_line(colour = "black"),
    axis.text.y = element_text(size=7),
    legend.key = element_blank(),
    legend.title = element_blank(),
    legend.position = "bottom",
    legend.text = element_text(size = 8),
    # strip.background = element_rect(
    #   colour = "white", fill = "white",
    #   size = 0.2
    # )
    strip.background = element_blank(),
    strip.text.x = element_blank(),
    strip.text.y = element_blank()
  )


ggsave(p, file = "motif_distribution.png", width = 10, height = 5)