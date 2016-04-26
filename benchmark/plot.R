#!/usr/bin/env Rscript
library(ggplot2)
library(dplyr)
library(scales)
library(ggthemes)
library(ggrepel)


w <- 5 # image width
h <- 5

df <- read.csv("run_benchmark_00_all.pl.benchmark.csv", sep = "\t")

# sort
df$test <- factor(df$test, levels = unique(df$test), ordered = TRUE)
df$app <- factor(df$app, levels = unique(df$app), ordered = TRUE)
df$dataset <-
  factor(df$dataset, levels = unique(df$dataset), ordered = TRUE)

# rename dataset
re <- function(x) {
  if (is.character(x) | is.factor(x)) {
    x <- gsub("dataset_","",x)
    x <- gsub("\\.fa","",x)
  }
  return(x)
}
df <- as.data.frame(lapply(df, re))


# one picture for one test
for (test1 in unique(df$test)) {
  df2 <- filter(df, test == test1)
  # humanize mem unit
  max_mem <- max(df2$mem_mean)
  unit <- "KB"
  if (max_mem > 1024 * 1024) {
    df2 <- df2 %>% mutate(mem_mean2 = mem_mean / 1024 / 1024)
    unit <- "GB"
  } else if (max_mem > 1024) {
    df2 <- df2 %>% mutate(mem_mean2 = mem_mean / 1024)
    unit <- "MB"
  } else {
    df2 <- df2 %>% mutate(mem_mean2 = mem_mean / 1)
    unit <- "KB"
  }
  
  p <-
    ggplot(df2, aes(
      x = mem_mean2, y = time_mean,
      color = app, shape = dataset, label = app
    )) +
    
    geom_point(size = 2.5) +
    geom_hline(aes(yintercept = time_mean, color=app), size = 0.2, alpha = 0.4) +
    geom_vline(aes(xintercept = mem_mean2, color=app), size = 0.2, alpha = 0.4) +
    geom_text_repel() +
    scale_color_wsj() +
    ylim(0, max(df2$time_mean)) +    
    xlim(0, max(df2$mem_mean2)) +
    
    # ggtitle(paste("FASTA/Q Manipulation Performance\n", test1, sep = "")) +
    ggtitle(test1) +
    ylab("Time (s)") +
    xlab(paste("Peak Memory (", unit, ")", sep = ""))
  
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
      #     axis.text.x = element_text(
      #       angle = 20, hjust = 1, vjust = 1
      #     ),
      
      strip.background = element_rect(
        colour = "white", fill = "white",
        size = 0.2
      ),
      
      legend.position = "top",
      legend.key.size = unit(0.4, "cm"),
      legend.margin = unit(0.0, "cm"),
      legend.key = element_blank(),
      
      text = element_text(
        size = 14, family = "arial", face = "bold"
      ),
      plot.title = element_text(size = 15)
    ) +
    guides(color = FALSE)

  ggsave(
    p, file = paste("benchmark-",  gsub(" ", "-", tolower(test1)), ".png",sep = ""), width = w, height = h
  )
  
  #   p <- p + scale_color_manual(values = rep("black", length(df$app)))
  #
  #   ggsave(
  #     p, file = paste("benchmark-", gsub(" ", "-", tolower(test1)), ".grey.png", sep = ""), width = w, height = h
  #   )
  
}
