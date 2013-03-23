#!/usr/bin/perl -w
use strict;
use Test::More;

require 'grepp';

my $file = 'test.txt';

my $lines = Lines->new( { file => $file, prev_size => 3 });
my @methods = ('next', 'prev', 'current', 'lines_group', 'string_group');
can_ok($lines, @methods);

is( $lines->prev,      undef, 'prev is undef');
like( $lines->current, qr/1/, 'current is 1');
like( $lines->next,    qr/2/, 'next is 2');
like( $lines->current, qr/2/, 'current is 2');
like( $lines->prev,    qr/1/, 'prev is 1');
is( $lines->prev(1),   undef, 'prev(1) is undef');
like( $lines->next,    qr/3/, 'next is 3');
like( $lines->current, qr/3/, 'current is 3');
like( $lines->prev,    qr/2/, 'prev is 2');
like( $lines->prev(1), qr/1/, 'prev(1) is 1');
is( $lines->prev(2),   undef, 'prev(2) is undef');
like( $lines->next,    qr/4/, 'next is 4');
like( $lines->next,    qr/5/, 'next is 5');
like( $lines->next,    qr/6/, 'next is 6');
like( $lines->prev,    qr/5/, 'prev is 5');
like( $lines->prev(1), qr/4/, 'prev(1) is 4');
like( $lines->prev(2), qr/3/, 'prev(2) is 3');
is( $lines->prev(3),   undef, 'prev(3) is undef');
is( $lines->prev(4),   undef, 'prev(4) is undef');
is( $lines->prev(5),   undef, 'prev(5) is undef');
is( $lines->lines_group, "3\n4\n5\n6\n", 'lines_group');
is( $lines->string_group, "3456", 'string_group');

## vim set ft:perl
