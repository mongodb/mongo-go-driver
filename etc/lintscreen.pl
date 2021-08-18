#!/usr/bin/env perl
use v5.10;
use strict;
use warnings;
use utf8;
use open qw/:std :utf8/;
use Getopt::Long qw/GetOptions/;
use List::Util qw/first/;

my $update;
GetOptions ("update|u" => \$update) or die("Error in command line arguments\n");

my @allowlist;

my ($allowlist_file) = shift @ARGV;
if ( length $allowlist_file && -r $allowlist_file ) {
    open my $fh, "<", $allowlist_file or die "${allowlist_file}: $!";
    @allowlist = map {
        my ($file,$msg) = m{^([^:]+):\d+:\d+: (.*)};
        qr/\Q$file\E:\d+:\d+: \Q$msg\E/;
    } <$fh>;
}

my $out_fh = \*STDOUT;
if ( $update ) {
    open $out_fh, ">>", $allowlist_file;
}

my $exit_code = 0;
while (my $line = <STDIN>) {
    next if first { $line =~ $_ } @allowlist;
    $exit_code = 1;
    print {$out_fh} $line;
}

exit ($update ? 0 : $exit_code);
