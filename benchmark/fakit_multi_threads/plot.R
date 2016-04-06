#!/usr/bin/env Rscript
library(ggplot2)
library(scales)
library(ggthemes)
library(dplyr)

df <- read.csv("run_benchmark_00_all.pl.benchmark.csv", sep = "\t")

df$test <- factor(df$test, levels = unique(df$test), ordered = TRUE)
df$app <- factor(df$app, levels = unique(df$app), ordered = TRUE)
df$dataset <- factor(df$dataset, levels = unique(df$dataset), ordered = TRUE)

# speedup relative to CPU=1
func <- function(app, time){
  i <- app[app==1] # time of CPU==1
  return(round(time[i]/time, 1))
}
df <- df  %>%
  group_by(test, dataset) %>%
  mutate(speedup=func(app, time_mean))

  
p <-
  ggplot(df, aes(x = app, y = speedup,
                 group = dataset, fill = dataset, label=sprintf("%.1fX", speedup))) +
  geom_bar(stat = "identity", position = position_dodge(0.6), width = 0.6,  color = "black") +
  geom_text(aes(y=speedup+0.2), position = position_dodge(0.6), size=3)+
  
  scale_fill_manual(values = rev(stata_pal("s2color")(5)), labels = c("A", "B", "A_dup", "B_dup", "Chr1")) +
  
  scale_y_continuous(limits=c(0, max(df$speedup)*1.15))+
  facet_grid(test ~ .) +
  ggtitle("FASTA Manipulation Benchmark\nfakit with multiple CPUs") +
  ylab("Speedup") +
  xlab("CPUs")

p <- p +
  theme_bw() +
  theme(
    panel.border = element_blank(),
    panel.background = element_blank(),
    panel.grid.major = element_blank(),
    panel.grid.minor = element_blank(),
    axis.line.x = element_line(colour = "black", size = 0.8),
    axis.line.y = element_line(colour = "black", size = 0.8),
    axis.ticks.y = element_line(size = 0.8),
    axis.ticks.x = element_line(size = 0.8),

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
                  labels = c("A", "B", "A_dup", "B_dup", "Chr1"))
  
ggsave(p, file = "benchmark.png", width = 5, height = 6)
