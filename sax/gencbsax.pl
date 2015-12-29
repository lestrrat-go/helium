#!perl
use strict;

# This script generates the default SAX2 handler, which is
# just a collection of callbacks

open my $fh, '<', "interface.go" or die;
# my $content = do { local $/; <$fh> };

while (my $ln = <$fh>) {
    if ($ln =~ /^\/\/ SAX functions/) {
        last;
    }
}

my %handler_returns;
my %handler_args;
my @handler_funcs;
while (my $ln = <$fh>) {
    if ($ln =~ /^type (.+)Func func\(([^)]+)\) (\([^\)]+\)|.+)$/) {
        push @handler_funcs, $1;
        $handler_args{$1} = $2;
        $handler_returns{$1} = $3;
    }
}

open my $out, '>', 'sax2.go' or die;

my $klass = "SAX2";
print $out <<EOM;
package sax

import "errors"

// ErrHandlerUnspecified is returned when there is no Handler
// registered for that particular event callback. This is not
// a fatal error per se, and can be ignored if the implementation
// chooses to do so.
var ErrHandlerUnspecified = errors.New("handler unspecified")

// SAX2Handler is an interface for anything that can satisfy
// helium's expected SAX2 API
type SAX2Handler interface {
EOM

foreach my $func (@handler_funcs) {
    my $args = $handler_args{$func};
    my $ret  = $handler_returns{$func};
    print $out "\t$func($args) $ret\n";
}

print $out <<EOM;
}

// $klass is the callback based SAX2 handler.
type $klass struct {
EOM

foreach my $func (@handler_funcs) {
    print $out "\t${func}Handler ${func}Func\n";
}

print $out <<EOM;
}

// New creates a new instance of $klass. All callbacks are
// uninitialized.
func New() *${klass} {
\treturn &${klass}{}
}

EOM

foreach my $func (@handler_funcs) {
    my $args = $handler_args{$func};
    my $ret  = $handler_returns{$func};
    my $no_handler_ret  = 
        join ", ",
            map { $_ eq "error" ? "ErrHandlerUnspecified" : $_ eq "bool" ? "false" : "nil" }
            split /\s*,\s*/, $ret =~ s{\(([^\)]+)\)}{$1}r;
    my $bare_args = join ", ", map { (split /\s+/, $_)[0] } split /\s*,\s*/, $args;
    print $out <<EOM
func (s $klass) $func($args) $ret {
\tif h := s.${func}Handler; h != nil {
\t\treturn h($bare_args)
\t}
\treturn $no_handler_ret;
}

EOM
}