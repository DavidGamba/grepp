#!/usr/bin/perl -w
use strict;
use Test::More tests => 38;
use Data::Dumper;

require 'grepp';

my $file = 'test.txt';

print "### Testing Lines class\n";
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
like( $lines->prev(0),    qr/5/, 'prev is 5');
like( $lines->prev(1), qr/4/, 'prev(1) is 4');
like( $lines->prev(2), qr/3/, 'prev(2) is 3');
is( $lines->prev(3),   undef, 'prev(3) is undef');
is( $lines->prev(4),   undef, 'prev(4) is undef');
is( $lines->prev(5),   undef, 'prev(5) is undef');
is( $lines->lines_group, "3\n4\n5\n6\n", 'lines_group');
is( $lines->string_group, "3 4 5 6", 'string_group');
is( $lines->lines_group, "3\n4\n5\n6\n", 'lines_group');
is( $lines->string_group, "3 4 5 6", 'string_group');

# Advance to the next section of the test.txt file
while($lines->prev(2) !~ /Lorem/) {
    $lines->next;
}

print "### Testing Match class\n";
my $match = Match->new( { string => $lines->string_group, regex => '[lt]or' });
@methods = ('match');
can_ok($match, @methods);
my ($match_found, @match_array) = $match->match;
is( $match_found, 1, 'match found');
my @expected = (
    { 'no_match' => 'Lorem ipsum doLor sit amet, consectetur adipiscing elit. Integer congue, nisl eget luctus pharetra, ' },
    { 'match'    => 'lor' },
    { 'no_match' => 'em ipsum portti' },
    { 'match'    => 'tor' },
    {   'no_match' =>
            ' urna, sed pretium eros arcu id justo. Integer a purus ut urna interdum elementum. Phasellus lobortis adipiscing vulputate. Pellentesque vel nunc nibh. Proin in velit ante. Nulla venenatis'
    }
);
is_deeply(\@match_array, \@expected, 'case sensitive matching');

$match = undef;
$match = Match->new( { string => $lines->string_group, regex => '[lt]or', case_insensitive => 1 });
($match_found, @match_array) = $match->match;
is( $match_found, 1, 'match found');
@expected = (
    { 'match' => 'Lor' },
    { 'no_match' => 'em ipsum do' },
    { 'match'    => 'Lor' },
    {   'no_match' =>
            ' sit amet, consectetur adipiscing elit. Integer congue, nisl eget luctus pharetra, '
    },
    { 'match'    => 'lor' },
    { 'no_match' => 'em ipsum portti' },
    { 'match'    => 'tor' },
    {   'no_match' =>
            ' urna, sed pretium eros arcu id justo. Integer a purus ut urna interdum elementum. Phasellus lobortis adipiscing vulputate. Pellentesque vel nunc nibh. Proin in velit ante. Nulla venenatis'
    }
);
is_deeply(\@match_array, \@expected, 'case insensitive matching');

$match = undef;
$match = Match->new( { string => $lines->lines_group, regex => '[lt]or', case_insensitive => 1 });
($match_found, @match_array) = $match->match;
is( $match_found, 1, 'match found');
@expected = (
    { 'match' => 'Lor' },
    { 'no_match' => 'em ipsum do' },
    { 'match'    => 'Lor' },
    {   'no_match' =>
            ' sit amet, consectetur adipiscing elit. Integer congue, nisl
' },
    {   'no_match' => 'eget luctus pharetra, ' },
    { 'match'    => 'lor' },
    { 'no_match' => 'em ipsum portti' },
    { 'match'    => 'tor' },
    {   'no_match' => ' urna, sed pretium eros arcu id
' },
    {   'no_match' => 'justo. Integer a purus ut urna interdum elementum. Phasellus lobortis adipiscing
' },
    {   'no_match' => 'vulputate. Pellentesque vel nunc nibh. Proin in velit ante. Nulla venenatis
' }
);
is_deeply(\@match_array, \@expected, 'case insensitive multiline matching');

# Advance to the next section of the test.txt file
while($lines->prev(2) !~ /^\s*Pellentesque/) {
    $lines->next;
}

$match = undef;
$match = Match->new( { string => $lines->lines_group, regex => 'u[ei]', case_insensitive => 1 });
($match_found, @match_array) = $match->match;
is( $match_found, 1, 'match found');
@expected = (
    { 'no_match' => 'Pellentesq' },
    { 'match' => 'ue' },
    { 'no_match' => ' et libero nisl, nec pos' },
    { 'match' => 'ue' },
    { 'no_match' => 're turpis. Aliquam erat volutpat. Maecenas
' },
    { 'no_match' => '    enim eros, hendrerit non ullamcorper aliquam, commodo q' },
    { 'match' => 'ui' },
    { 'no_match' => 's erat. Cras a nunc
' },
    { 'no_match' => 'q' },
    { 'match' => 'ui' },
    { 'no_match' => 's mi ornare tincidunt eget eget risus. D' },
    { 'match' => 'ui' },
    { 'no_match' => 's ac volutpat enim. Etiam nibh
' },
    { 'no_match' => '      lacus, tristiq' },
    { 'match' => 'ue' },
    { 'no_match' => ' sed molestie in, molestie ac tellus. Integer dolor metus,
' }
);
is_deeply(\@match_array, \@expected, 'remove intial spacing');

$match = undef;
$match = Match->new( { string => $lines->lines_group, regex => 'u[ei]', case_insensitive => 1, initial_spacing => 1 });
($match_found, @match_array) = $match->match;
is( $match_found, 1, 'match found');
@expected = (
    { 'no_match' => '  Pellentesq' },
    { 'match' => 'ue' },
    { 'no_match' => ' et libero nisl, nec pos' },
    { 'match' => 'ue' },
    { 'no_match' => 're turpis. Aliquam erat volutpat. Maecenas
' },
    { 'no_match' => '    enim eros, hendrerit non ullamcorper aliquam, commodo q' },
    { 'match' => 'ui' },
    { 'no_match' => 's erat. Cras a nunc
' },
    { 'no_match' => 'q' },
    { 'match' => 'ui' },
    { 'no_match' => 's mi ornare tincidunt eget eget risus. D' },
    { 'match' => 'ui' },
    { 'no_match' => 's ac volutpat enim. Etiam nibh
' },
    { 'no_match' => '      lacus, tristiq' },
    { 'match' => 'ue' },
    { 'no_match' => ' sed molestie in, molestie ac tellus. Integer dolor metus,
' }
);
is_deeply(\@match_array, \@expected, 'keep intial spacing');

$match = undef;
$match = Match->new( { string => $lines->lines_group, regex => '(ue|ui)', case_insensitive => 1 });
($match_found, @match_array) = $match->match;
is( $match_found, 1, 'match found');
@expected = (
    { 'no_match' => 'Pellentesq' },
    { 'match' => 'ue' },
    { 'no_match' => ' et libero nisl, nec pos' },
    { 'match' => 'ue' },
    { 'no_match' => 're turpis. Aliquam erat volutpat. Maecenas
' },
    { 'no_match' => '    enim eros, hendrerit non ullamcorper aliquam, commodo q' },
    { 'match' => 'ui' },
    { 'no_match' => 's erat. Cras a nunc
' },
    { 'no_match' => 'q' },
    { 'match' => 'ui' },
    { 'no_match' => 's mi ornare tincidunt eget eget risus. D' },
    { 'match' => 'ui' },
    { 'no_match' => 's ac volutpat enim. Etiam nibh
' },
    { 'no_match' => '      lacus, tristiq' },
    { 'match' => 'ue' },
    { 'no_match' => ' sed molestie in, molestie ac tellus. Integer dolor metus,
' }
);
is_deeply(\@match_array, \@expected, 'Allow capturing groups in given regex: "(ue|ui)"');
## vim set ft:perl
