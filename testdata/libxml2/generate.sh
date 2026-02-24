#!/bin/bash
#
# Copies relevant XML test inputs and expected outputs from a libxml2 clone
# into testdata/libxml2-compat/ for use by Go integration tests.
#
# Usage:
#   bash testdata/libxml2/generate.sh
#
# Prerequisites:
#   bash testdata/libxml2/fetch.sh   (clone libxml2 first)
#
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SOURCE="$SCRIPT_DIR/source"
DEST="$REPO_ROOT/testdata/libxml2-compat"

if [ ! -d "$SOURCE/test" ]; then
    echo "Error: libxml2 source not found at $SOURCE"
    echo "Run: bash testdata/libxml2/fetch.sh"
    exit 1
fi

rm -rf "$DEST"
mkdir -p "$DEST"

count=0

# Iterate over files (not directories) in test/
for input in "$SOURCE"/test/*; do
    [ -f "$input" ] || continue

    base="$(basename "$input")"

    # The result file has the same basename, sitting directly in result/
    result="$SOURCE/result/$base"
    [ -f "$result" ] || continue

    # Copy input and expected output
    cp "$input" "$DEST/$base"
    cp "$result" "$DEST/$base.expected"
    count=$((count + 1))
done

echo "Copied $count test cases into $DEST"

# Copy SAX2 golden files
sax2count=0
for input in "$DEST"/*; do
    [ -f "$input" ] || continue
    base="$(basename "$input")"

    # Skip .expected and .err files
    case "$base" in
        *.expected|*.err) continue ;;
    esac

    # The SAX2 result file is result/<base>.sax2
    sax2result="$SOURCE/result/$base.sax2"
    [ -f "$sax2result" ] || continue

    cp "$sax2result" "$DEST/$base.sax2.expected"
    sax2count=$((sax2count + 1))
done

echo "Copied $sax2count SAX2 golden files into $DEST"

# Fix C buffer artifacts in SAX2 golden files.
# xmllint's C code uses %.4s to print attribute values, which reads 4 bytes
# from a raw pointer regardless of string length, leaking adjacent buffer
# contents. Normalize by trimming displayed value to the reported length.
fixed=0
for f in "$DEST"/*.sax2.expected; do
    [ -f "$f" ] || continue
    if perl -0777 -i -pe '
        s{=\047(.{1,4})\.\.\.\047(, )(\d+)}{
            my($d,$s,$n)=($1,$2,$3);
            $d = substr($d,0,$n) if length($d) > $n;
            "=\047${d}...\047$s$n"
        }gse
    ' "$f"; then
        fixed=$((fixed + 1))
    fi
done
echo "Normalized $fixed SAX2 golden files (C buffer truncation fix)"

# Merge consecutive SAX.characters() events in SAX2 golden files.
# libxml2 splits character data at internal buffer boundaries (4000 bytes for
# native UTF-8, ~300 bytes for transcoded input). The SAX spec allows arbitrary
# splitting, so we normalize by merging adjacent characters() events into one.
merged=0
for f in "$DEST"/*.sax2.expected; do
    [ -f "$f" ] || continue
    perl -0777 -i -e '
        my $content = <>;
        my @events;
        my $cur = "";
        for my $line (split /(?<=\n)/, $content) {
            if ($line =~ /^SAX\./ && $cur ne "") {
                push @events, $cur;
                $cur = $line;
            } else {
                $cur .= $line;
            }
        }
        push @events, $cur if $cur ne "";

        my @out;
        my ($merged_data, $merged_len) = ("", 0);
        for my $ev (@events) {
            if ($ev =~ /^SAX\.characters\((.*), (\d+)\)\n$/s) {
                $merged_data .= $1;
                $merged_len += $2;
            } else {
                if ($merged_len > 0) {
                    push @out, "SAX.characters($merged_data, $merged_len)\n";
                    ($merged_data, $merged_len) = ("", 0);
                }
                push @out, $ev;
            }
        }
        if ($merged_len > 0) {
            push @out, "SAX.characters($merged_data, $merged_len)\n";
        }
        print @out;
    ' "$f"
    merged=$((merged + 1))
done
echo "Merged consecutive SAX.characters() events in $merged SAX2 golden files"

# Copy XPath test files
mkdir -p "$DEST/xpath/expr" "$DEST/xpath/tests" "$DEST/xpath/docs"

# Expression tests (no document context)
xpath_expr_count=0
for input in "$SOURCE"/test/XPath/expr/*; do
    [ -f "$input" ] || continue
    base="$(basename "$input")"
    result="$SOURCE/result/XPath/expr/$base"
    [ -f "$result" ] || continue
    cp "$input" "$DEST/xpath/expr/$base"
    cp "$result" "$DEST/xpath/expr/$base.expected"
    xpath_expr_count=$((xpath_expr_count + 1))
done
echo "Copied $xpath_expr_count XPath expression tests into $DEST/xpath/expr"

# Document-based tests
xpath_tests_count=0
for input in "$SOURCE"/test/XPath/tests/*; do
    [ -f "$input" ] || continue
    base="$(basename "$input")"
    result="$SOURCE/result/XPath/tests/$base"
    [ -f "$result" ] || continue
    cp "$input" "$DEST/xpath/tests/$base"
    cp "$result" "$DEST/xpath/tests/$base.expected"
    xpath_tests_count=$((xpath_tests_count + 1))
done
echo "Copied $xpath_tests_count XPath document-based tests into $DEST/xpath/tests"

# XML documents used by document-based tests
xpath_docs_count=0
for input in "$SOURCE"/test/XPath/docs/*; do
    [ -f "$input" ] || continue
    base="$(basename "$input")"
    cp "$input" "$DEST/xpath/docs/$base"
    xpath_docs_count=$((xpath_docs_count + 1))
done
echo "Copied $xpath_docs_count XPath test documents into $DEST/xpath/docs"

# Copy XInclude test files
# XInclude tests need directory structure preserved because included files use
# relative paths like "../ents/something.xml".
XINC_DEST="$DEST/xinclude"
mkdir -p "$XINC_DEST/docs" "$XINC_DEST/ents" "$XINC_DEST/result" "$XINC_DEST/without-reader"

# docs/ — test inputs
xinc_docs=0
for input in "$SOURCE"/test/XInclude/docs/*; do
    [ -f "$input" ] || continue
    cp "$input" "$XINC_DEST/docs/"
    xinc_docs=$((xinc_docs + 1))
done
echo "Copied $xinc_docs XInclude doc inputs into $XINC_DEST/docs"

# ents/ — included entities/fragments
xinc_ents=0
for input in "$SOURCE"/test/XInclude/ents/*; do
    [ -f "$input" ] || continue
    cp "$input" "$XINC_DEST/ents/"
    xinc_ents=$((xinc_ents + 1))
done
echo "Copied $xinc_ents XInclude entity files into $XINC_DEST/ents"

# without-reader/ — additional test inputs
xinc_wr=0
for input in "$SOURCE"/test/XInclude/without-reader/*; do
    [ -f "$input" ] || continue
    cp "$input" "$XINC_DEST/without-reader/"
    xinc_wr=$((xinc_wr + 1))
done
echo "Copied $xinc_wr XInclude without-reader inputs into $XINC_DEST/without-reader"

# result/ — expected outputs (flat directory covers both docs/ and without-reader/)
xinc_results=0
for result in "$SOURCE"/result/XInclude/*; do
    [ -f "$result" ] || continue
    cp "$result" "$XINC_DEST/result/"
    xinc_results=$((xinc_results + 1))
done
echo "Copied $xinc_results XInclude result files into $XINC_DEST/result"
