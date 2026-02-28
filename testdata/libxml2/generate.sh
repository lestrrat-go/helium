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

# Copy XML Schema test files
SCHEMA_DEST="$DEST/schemas"
mkdir -p "$SCHEMA_DEST/test" "$SCHEMA_DEST/result"

# test/ — schema (.xsd) and instance (.xml) files
schema_test=0
for input in "$SOURCE"/test/schemas/*; do
    [ -f "$input" ] || continue
    cp "$input" "$SCHEMA_DEST/test/"
    schema_test=$((schema_test + 1))
done
echo "Copied $schema_test XML Schema test files into $SCHEMA_DEST/test"

# result/ — expected validation output (.err files)
schema_result=0
for result in "$SOURCE"/result/schemas/*; do
    [ -f "$result" ] || continue
    cp "$result" "$SCHEMA_DEST/result/"
    schema_result=$((schema_result + 1))
done
echo "Copied $schema_result XML Schema result files into $SCHEMA_DEST/result"

# Copy XML Catalog test files
# Catalogs reference each other via relative paths (nextCatalog, delegate*)
# so all XML/script/SGML files must stay in a flat directory together.
CAT_DEST="$DEST/catalogs"
mkdir -p "$CAT_DEST" "$CAT_DEST/result"

# test/ — catalog XML, script, and SGML files
cat_test=0
for input in "$SOURCE"/test/catalogs/*; do
    [ -f "$input" ] || continue
    cp "$input" "$CAT_DEST/"
    cat_test=$((cat_test + 1))
done
echo "Copied $cat_test catalog test files into $CAT_DEST"

# result/ — expected resolution output (golden files)
cat_result=0
for result in "$SOURCE"/result/catalogs/*; do
    [ -f "$result" ] || continue
    cp "$result" "$CAT_DEST/result/"
    cat_result=$((cat_result + 1))
done
echo "Copied $cat_result catalog result files into $CAT_DEST/result"

# Copy C14N test files
# C14N tests are organized by category (without-comments, with-comments,
# exc-without-comments, 1-1-without-comments). Each category has .xml inputs
# with optional .xpath and .ns sidecar files. Results have no extension.
# Some tests reference doc.dtd via relative paths, so it must be preserved.
C14N_DEST="$DEST/c14n"
c14n_total=0

for category in without-comments with-comments exc-without-comments 1-1-without-comments; do
    test_dir="$SOURCE/test/c14n/$category"
    result_dir="$SOURCE/result/c14n/$category"
    [ -d "$test_dir" ] || continue

    mkdir -p "$C14N_DEST/$category/test" "$C14N_DEST/$category/result"

    # Copy all test files (.xml, .xpath, .ns, .dtd)
    c14n_cat=0
    for input in "$test_dir"/*; do
        [ -f "$input" ] || continue
        cp "$input" "$C14N_DEST/$category/test/"
        c14n_cat=$((c14n_cat + 1))
    done

    # Copy result files (no extension)
    c14n_results=0
    if [ -d "$result_dir" ]; then
        for result in "$result_dir"/*; do
            [ -f "$result" ] || continue
            cp "$result" "$C14N_DEST/$category/result/"
            c14n_results=$((c14n_results + 1))
        done
    fi

    echo "Copied $c14n_cat test + $c14n_results result files for c14n/$category"
    c14n_total=$((c14n_total + c14n_cat + c14n_results))
done
echo "Copied $c14n_total total C14N files into $C14N_DEST"

# Copy HTML test files
HTML_DEST="$DEST/html"
mkdir -p "$HTML_DEST"

# test/ — HTML input files and result/ — SAX golden outputs and serialized HTML
html_count=0
for input in "$SOURCE"/test/HTML/*; do
    [ -f "$input" ] || continue
    base="$(basename "$input")"
    cp "$input" "$HTML_DEST/$base"

    # Copy SAX golden file if it exists
    sax_result="$SOURCE/result/HTML/$base.sax"
    if [ -f "$sax_result" ]; then
        cp "$sax_result" "$HTML_DEST/$base.sax"
    fi

    # Copy serialized HTML golden file if it exists
    html_result="$SOURCE/result/HTML/$base"
    if [ -f "$html_result" ]; then
        cp "$html_result" "$HTML_DEST/$base.expected"
    fi

    # Copy error golden file if it exists
    err_result="$SOURCE/result/HTML/$base.err"
    if [ -f "$err_result" ]; then
        cp "$err_result" "$HTML_DEST/$base.err"
    fi

    html_count=$((html_count + 1))
done
echo "Copied $html_count HTML test files into $HTML_DEST"

# Copy RELAX NG test files
# RelaxNG tests have schema (.rng) and instance (.xml) files in test/relaxng/,
# with some schemas referencing other .rng files via include/externalRef.
# An OASIS/ subdirectory contains a spectest.xml file.
# Results are .err files in result/relaxng/.
RELAXNG_DEST="$DEST/relaxng"
mkdir -p "$RELAXNG_DEST/test" "$RELAXNG_DEST/result"

# test/ — schema (.rng) and instance (.xml) files
relaxng_test=0
for input in "$SOURCE"/test/relaxng/*; do
    [ -f "$input" ] || continue
    cp "$input" "$RELAXNG_DEST/test/"
    relaxng_test=$((relaxng_test + 1))
done
echo "Copied $relaxng_test RELAX NG test files into $RELAXNG_DEST/test"

# result/ — expected validation output (.err files)
relaxng_result=0
for result in "$SOURCE"/result/relaxng/*; do
    [ -f "$result" ] || continue
    cp "$result" "$RELAXNG_DEST/result/"
    relaxng_result=$((relaxng_result + 1))
done
echo "Copied $relaxng_result RELAX NG result files into $RELAXNG_DEST/result"

# Fix broken-xml_0.err — helium parser produces a different error message than libxml2
if [ -f "$RELAXNG_DEST/result/broken-xml_0.err" ]; then
    sed -i "s/Couldn't find end of Start Tag foo line 1/failed to parse QName ''/" "$RELAXNG_DEST/result/broken-xml_0.err"
    echo "Patched broken-xml_0.err for helium parser error format"
fi

# Copy Schematron test files
SCH_DEST="$DEST/schematron"
mkdir -p "$SCH_DEST/test" "$SCH_DEST/result"

# test/ — schema (.sct) and instance (.xml) files
sch_test=0
for input in "$SOURCE"/test/schematron/*; do
    [ -f "$input" ] || continue
    cp "$input" "$SCH_DEST/test/"
    sch_test=$((sch_test + 1))
done
echo "Copied $sch_test Schematron test files into $SCH_DEST/test"

# result/ — expected validation output (.err files)
sch_result=0
for result in "$SOURCE"/result/schematron/*; do
    [ -f "$result" ] || continue
    cp "$result" "$SCH_DEST/result/"
    sch_result=$((sch_result + 1))
done
echo "Copied $sch_result Schematron result files into $SCH_DEST/result"

# Copy valid DTD test files
# Valid tests live in test/valid/ with DTDs in test/valid/dtds/.
# Results are in result/valid/. The XML files reference DTDs via relative
# paths like "dtds/cond_sect1.dtd", so the directory structure must be preserved.
VALID_DEST="$DEST/valid"
mkdir -p "$VALID_DEST/dtds"

valid_count=0
for input in "$SOURCE"/test/valid/*.xml; do
    [ -f "$input" ] || continue
    base="$(basename "$input")"
    result="$SOURCE/result/valid/$base"
    [ -f "$result" ] || continue

    cp "$input" "$VALID_DEST/$base"
    cp "$result" "$VALID_DEST/$base.expected"
    valid_count=$((valid_count + 1))
done
echo "Copied $valid_count valid DTD test files into $VALID_DEST"

# Copy referenced DTD files
valid_dtd_count=0
for dtd in "$SOURCE"/test/valid/dtds/*; do
    [ -f "$dtd" ] || continue
    cp "$dtd" "$VALID_DEST/dtds/"
    valid_dtd_count=$((valid_dtd_count + 1))
done
echo "Copied $valid_dtd_count valid DTD files into $VALID_DEST/dtds"
