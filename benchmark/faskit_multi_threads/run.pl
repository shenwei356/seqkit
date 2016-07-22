#!/usr/bin/env perl
use strict;
use Getopt::Long;
use File::Basename;

$0 = basename $0;

my $usage = <<USAGE;
Usage:

1. Run all tests:

perl $0 run_benchmark*.sh --outfile benchmark.5test.csv

2. Run one test:

perl $0 run_benchmark_04_remove_duplicated_seqs_by_name.sh -o benchmark.rmdup.csv

3. Custom repeate times:

perl $0 -n 3 run_benchmark_04_remove_duplicated_seqs_by_name.sh -o benchmark.rmdup.csv

USAGE

my $N = 3;    # run $N times

my $resultfile = "$0.benchmark.csv";
my $help = 0;

GetOptions( "n=i" => \$N, 'outfile|o=s' => \$resultfile, "help|h" => \$help, )
  or die $usage;
die $usage if scalar @ARGV == 0 or $help;

my @tests = @ARGV;

open my $fh_result, ">", $resultfile or die "failed to write file: $resultfile\n";
my $title_line = "test\tdataset\tapp\ttime_mean\ttime_stdev\tmem_mean\tmem_stdev\n";
print $fh_result $title_line;

for my $test (@tests) {
    print STDERR "Test: $test\n";

    my $stat = {};    # dataset->app->round
    my $stat_mem = {};
    my $t = ""; # test name
    # run $N times
    for my $n ( 1 .. $N ) {
        print STDERR "Round: $n\n";

        my $outfile = "$test.round$n.out";
        unlink $outfile if -e $outfile;

        # run
        my $cmd = "sh $test 2>&1 | tee $outfile";
        my $fail = run($cmd);
        die "failed to run:$cmd\n" if $fail;

        # stat
        my ( $app, $dataset, $time, $mem );
        open my $fh, "<", $outfile or die "failed to read file: $outfile\n";
        for my $line (<$fh>) {
            if ( $t eq "" and $line =~ /Test: (.+)/ ) {
                $t = $1;
            }
            if ( $line =~ /== (.+)/ ) {
                $app = $1;
            }
            if ( $line =~ /data: (.+)/ ) {
                $dataset = $1;
            }
            if ( $line =~ /time: (.+)/ ) {
                $time = $1;

                if ( not exists $$stat{$dataset}{$app} ) {
                    $$stat{$dataset}{$app} = [];
                }
                push @{ $$stat{$dataset}{$app} }, $time;
            }
            if ( $line =~ /rss: (\d+)/ ) {
                $mem = $1;

                if ( not exists $$stat_mem{$dataset}{$app} ) {
                    $$stat_mem{$dataset}{$app} = [];
                }
                push @{ $$stat_mem{$dataset}{$app} }, $mem;
            }
        }
        close($fh);
    }

    # benchmark result
    my $statfile = "$test.benchmark.csv";
    open my $fh, ">", $statfile or die "failed to write file: $statfile\n";

    print "\n=========[ benchmark result ]========\n";
    print $title_line;
    print $fh $title_line;
    for my $dataset ( sort keys %$stat ) {
        my $st = $$stat{$dataset};
        for my $app ( sort keys %$st ) {
            my $times = $$st{$app};
            my $mems = $$stat_mem{$dataset}{$app};
            my ( $time_mean, $time_stdev ) = mean_and_stdev($times);
            my ( $mem_mean, $mem_stdev ) = mean_and_stdev($mems);

            my $data_line = sprintf "%s\t%s\t%s\t%.2f\t%.2f\t%d\t%d\n",
                        $t, $dataset, $app, $time_mean, $time_stdev,$mem_mean, $mem_stdev;

            printf $data_line;
            printf $fh $data_line;
            printf $fh_result $data_line;
        }
    }
    close($fh);
}
close($fh_result);

sub run {
    my ($cmd) = @_;
    system($cmd);

    if ( $? == -1 ) {
        die "[ERROR] fail to run: $cmd. Command ("
          . ( split /\s+/, $cmd )[0]
          . ") not found\n";
    }
    elsif ( $? & 127 ) {
        printf STDERR "[ERROR] command died with signal %d, %s coredump\n",
          ( $? & 127 ), ( $? & 128 ) ? 'with' : 'without';
    }
    else {
        # 0, ok
    }
    return $?;
}

sub mean_and_stdev($) {
    my ($list) = @_;
    return ( 0, 0 ) if @$list == 0;
    my $sum = 0;
    $sum += $_ for @$list;
    my $sum_square = 0;
    $sum_square += $_ * $_ for @$list;
    my $mean     = $sum / @$list;
    my $variance = $sum_square / @$list - $mean * $mean;
    my $std      = sqrt $variance;
    return ( $mean, $std );
}
