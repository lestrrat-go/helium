# Third-Party Notices

The helium library code in this repository remains licensed under the MIT
license in `LICENSE`.

This file covers third-party material used for tests, reference inputs, and
generated test code. Those materials keep their upstream license terms; they
are not relicensed by helium's MIT license.

## libxml2

Upstream project:
- `https://gitlab.gnome.org/GNOME/libxml2`

Local paths:
- `testdata/libxml2/fetch.sh`
- `testdata/libxml2/generate.sh`
- `testdata/libxml2/source/` after running the fetch script
- `testdata/libxml2-compat/`

How it is used:
- `testdata/libxml2/fetch.sh` clones libxml2 into
  `testdata/libxml2/source/`.
- `testdata/libxml2/generate.sh` copies test inputs and expected outputs from
  that checkout into the committed `testdata/libxml2-compat/` tree.
- The generator also normalizes some copied golden files for helium's Go test
  harness, for example by trimming C buffer artifacts and merging adjacent SAX
  character events.

License / notice:
- libxml2 documents that its code is released under the MIT license.
- The committed `testdata/libxml2-compat/` tree is derived from libxml2 test
  data and expected outputs. Preserve upstream libxml2 copyright and license
  notices when refreshing that data.

## W3C QT3 Test Suite

Upstream project:
- `https://github.com/w3c/qt3tests`

Local paths:
- `testdata/qt3ts/fetch.sh`
- `testdata/qt3ts/source/` after running the fetch script
- `testdata/qt3ts/testdata/`
- `tools/qt3gen/`
- `xpath3/qt3_*_gen_test.go`

How it is used:
- `testdata/qt3ts/fetch.sh` clones the upstream QT3 repository into
  `testdata/qt3ts/source/`.
- `tools/qt3gen` reads the upstream catalog and copies required context
  documents and resource files into the committed `testdata/qt3ts/testdata/`
  tree.
- `tools/qt3gen` also generates the committed `xpath3/qt3_*_gen_test.go`
  files from the upstream QT3 catalog and test definitions.

License / notice:
- W3C publishes its test suites under both the W3C test suite license and the
  W3C 3-clause BSD test suite license.
- W3C describes the 3-clause BSD test suite license as the path for software
  development, bug tracking, and other uses that do not make public
  performance or conformance claims with modified tests.
- helium uses copied and generated QT3-derived materials in that
  development-oriented way.
- Do not present modified or subsetted QT3-derived materials in this
  repository as an authoritative W3C test suite or as a basis for public W3C
  conformance claims.

## W3C XSLT 3.0 Test Suite

Upstream project:
- `https://github.com/w3c/xslt30-test`

Local paths:
- `testdata/xslt30/fetch.sh`
- `testdata/xslt30/source/` after running the fetch script
- `testdata/xslt30/testdata/`
- `tools/xslt3gen/`
- `xslt3/w3c_*_gen_test.go`

How it is used:
- `testdata/xslt30/fetch.sh` clones the upstream XSLT 3.0 test repository into
  `testdata/xslt30/source/`.
- `tools/xslt3gen` reads the upstream catalog and copies required stylesheet,
  source, and support assets into the committed `testdata/xslt30/testdata/`
  tree.
- `tools/xslt3gen` also generates the committed `xslt3/w3c_*_gen_test.go`
  files from the upstream XSLT 3.0 catalog and test definitions.

License / notice:
- W3C publishes its test suites under both the W3C test suite license and the
  W3C 3-clause BSD test suite license.
- W3C describes the 3-clause BSD test suite license as the path for software
  development, bug tracking, and other uses that do not make public
  performance or conformance claims with modified tests.
- helium uses copied and generated XSLT 3.0 test materials in that
  development-oriented way.
- Do not present modified or subsetted XSLT 3.0-derived materials in this
  repository as an authoritative W3C test suite or as a basis for public W3C
  conformance claims.

## Saxon-HE

Upstream project:
- `https://github.com/Saxonica/Saxon-HE`

Local paths:
- `testdata/saxon/fetch.sh`
- `testdata/saxon/source/` after running the fetch script

How it is used:
- `testdata/saxon/fetch.sh` clones Saxon-HE into the ignored
  `testdata/saxon/source/` directory for local reference work.
- `testdata/saxon/source/` is ignored by `.gitignore` and is not committed by
  default as part of this repository.

License / notice:
- Saxon-HE states that it is available under the Mozilla Public License 2.0.
- If a developer populates `testdata/saxon/source/`, that checkout remains
  third-party Saxon material under its upstream license terms and is not
  covered by helium's MIT license.

## Reference Sources

- libxml2 license statement:
  `https://gnome.pages.gitlab.gnome.org/libxml2/devhelp/index.html`
- W3C test suite license policy:
  `https://www.w3.org/copyright/test-suites-licenses/`
- W3C 3-clause BSD test suite license:
  `https://www.w3.org/copyright/3-clause-bsd-license-2008/`
- Saxon-HE repository and license statement:
  `https://github.com/Saxonica/Saxon-HE`
