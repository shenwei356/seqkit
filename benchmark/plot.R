#!/usr/bin/env Rscript
library(ggplot2)
library(scales)
library(ggthemes)

df <- read.csv("run_benchmark_00_all.pl.benchmark.csv", sep = "\t")

df$test <- factor(df$test, levels = unique(df$test), ordered = TRUE)
df$app <- factor(df$app, levels = unique(df$app), ordered = TRUE)
df$dataset <- factor(df$dataset, levels = unique(df$dataset), ordered = TRUE)


p <-
  ggplot(df, aes(x = app, y = time_mean, 
                 ymax=time_mean+time_stdev, ymin=time_mean+time_stdev,
                 group = dataset, fill = dataset, label=time_mean)) +
  geom_bar(stat = "identity", position = position_dodge(0.7), width = 0.7,  color = "black") +
  geom_errorbar(width = 0.3, position = position_dodge(0.7), size = 0.4) +
  geom_text(aes(y=time_mean+4), position = position_dodge(1), size=3)+
  
  # geom_hline(yintercept = 0, linetype = 1, size = 1) +
  
  scale_fill_manual(values = rev(stata_pal("s2color")(5)), labels = c("A", "B", "A_dup", "B_dup", "Chr1")) +
  
  scale_y_continuous(limits=c(0, max(df$time_mean)+5))+
  facet_grid(test ~ .) +
  ggtitle("FASTA Manipulation Performance") +
  ylab("Time (s)") +
  xlab(NULL)

p <- p +
  theme_bw() +
  theme(
    panel.border = element_blank(),
    panel.background = element_blank(),
    panel.grid.major = element_blank(),
    panel.grid.minor = element_blank(),
    axis.line = element_line(colour = "black", size = 0.8),
    axis.ticks.y = element_line(size = 0.8),
    axis.ticks.x = element_line(size = 0.8),
    axis.text.x = element_text(
      angle = 20, hjust = 1, vjust = 1
    ),
    
    strip.background = element_rect(
      colour = "white", fill = "white",
      size = 0.2
    ),
    # strip.text = element_text(size=10),
    
    legend.position = "top",
    # legend.title = element_blank(),
    legend.key.size = unit(0.4, "cm"),
    
    text = element_text(
      size = 14, family = "arial", face = "bold"
    ),
    plot.title = element_text(size=15)
  )

ggsave(p, file = "benchmark_colorful.png", width = 5, height = 6)

p <-p +scale_fill_manual(values = c("grey100", "grey80", "grey60", "grey40", "grey20"),
                  labels = c("A", "B", "A_dup", "B_dup", "chr1"))
  
ggsave(p, file = "benchmark.png", width = 5, height = 6)
