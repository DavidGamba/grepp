#!/usr/bin/perl -w
use strict;
use Test::More;

require 'grepp';

my $file = 'test.txt';

my $ob = Lines->new( { file => $file, prev_size => 3 });
my @methods = ('next', 'prev', 'current', 'line_group');
can_ok($ob, @methods);

like( $ob->current, qr/1/, 'current is 1');
like( $ob->next,    qr/2/, 'next is 2');
like( $ob->current, qr/2/, 'current is 2');
like( $ob->prev,    qr/1/, 'prev is 1');
is( $ob->prev(1),   undef, 'prev(1) is undef');
like( $ob->next,    qr/3/, 'next is 3');
like( $ob->current, qr/3/, 'current is 3');
like( $ob->prev,    qr/2/, 'prev is 2');
like( $ob->prev(1), qr/1/, 'prev(1) is 1');
is( $ob->prev(2),   undef, 'prev(2) is undef');
like( $ob->next,    qr/4/, 'next is 4');
like( $ob->next,    qr/5/, 'next is 5');
like( $ob->next,    qr/6/, 'next is 6');
like( $ob->prev,    qr/5/, 'prev is 5');
like( $ob->prev(1), qr/4/, 'prev(1) is 4');
like( $ob->prev(2), qr/3/, 'prev(2) is 3');
is( $ob->prev(3),   undef, 'prev(3) is undef');
is( $ob->prev(4),   undef, 'prev(4) is undef');
is( $ob->prev(5),   undef, 'prev(5) is undef');

## vim set ft:perl
