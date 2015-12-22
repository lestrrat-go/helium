#!perl

# This script generates the default SAX2 handler, which is
# just a collection of callbacks

open my $fh, '<', "interface.go" or die;
my $content = do { local $/; <$fh> };

my @methods; # SAX2 methods
my @functypes; # BlahFunc definitions
my @fields; # fields in SAX2 struct

sub extract {
    my ($iface, $methods) = @_;
    $iface =~ s/^\s+//;
    $iface =~ s/\s+$//;
    for my $line (split /\n/, $methods) {
        $line =~ s{//.*}{};
        $line =~ s/^\s+//;
        $line =~ s/\s+$//;

        next unless $line =~ /\S+/;
        $line =~ /^([^\s\(]+)\s*\(([^\)]+)\)/;
        my $name = $1;
        my $args = $2;

        # split the args, strip the typ
        my @args = map { s/\s*\S+$//; $_ } split(/\s*,\s*/,$args);

        push @methods, <<EOM;
// ${name} satisfies the ${iface} interface
func (s *SAX2) $line {
\tif h := s.${name}Handler; h != nil {
\t\treturn h(@{[ join ", ", @args ]})
\t}
\treturn nil
}

EOM
        push @functypes, "// ${name}Func defines the function type for SAX2.${name}Handler\ntype ${name}Func func($args) error";
        push @fields, "${name}Handler ${name}Func";
    }
}

while ($content =~ /^type (\w+Handler) interface {([^}]+)}/msg) {
    extract($1, $2)
}
if ($content =~ /^type (EntityResolver) interface {([^}]+)}/msg) {
    extract($1, $2)
}
if ($content =~ /^type (Extensions) interface {([^}]+)}/msg) {
    extract($1, $2)
}

open my $out, '>', 'callback.go' or die;

print $out "package sax\n\n";

foreach my $func (sort { $a cmp $b } @functypes) {
    print $out $func, "\n\n";
}

print $out "type SAX2 struct {\n";
foreach my $field (sort { $a cmp $b } @fields) {
    print $out "\t$field\n"
}
print $out "}\n";

print $out <<EOM;
func New() *SAX2 {
\treturn &SAX2{}
}

EOM

foreach my $method (sort { $a cmp $b } @methods) {
    print $out "$method";
}