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
