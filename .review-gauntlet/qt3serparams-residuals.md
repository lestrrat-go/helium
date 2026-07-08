# fn:serialize params — deferred residuals

## normalization-form is applied to the whole serialized string, not scoped to text/attribute character-expansion

**Location:** `xpath3/functions_serialize.go` — `fnSerialize` (~line 171,
`applySerializeNormalization(result, dispatchOpts)`), which runs Unicode
normalization over the FULL serialized markup string produced by the xml / xhtml
/ html output methods.

**Spec rule (XSLT/XQuery Serialization 3.1):** Unicode normalization is one step
of the **character-expansion phase**, which is scoped to text and attribute nodes
only:

> "Character expansion is concerned with the representation of characters
> appearing in text and attribute nodes in the sequence. For each text and
> attribute node, the following rules are applied in sequence." … "Apply Unicode
> Normalization if requested by the normalization-form parameter."

and

> "The values of attribute nodes and text nodes in the reconstructed tree may be
> different from those in the result tree, due to the effects of URI expansion,
> character mapping and Unicode Normalization in the character expansion phase of
> serialization."

So normalization must NOT touch element/attribute **names**, comment or PI
markup, the DOCTYPE, or the XML declaration. Normalizing the whole output string
can wrongly re-spell an element/attribute name that contains decomposed Unicode
when `normalization-form` is `NFC`/`NFD`/`NFKC`/`NFKD` (e.g. a tag named with a
base letter + combining accent would be composed/decomposed by the normalizer).
The reviewer's reading is CONFIRMED correct against the qtspecs mirror.

**Why deferred (large / risky restructure):** The spec-correct location for
normalization is the writer's per-text/per-attribute character-expansion funnel.
helium uses TWO independent core serializers, each with its own escaping funnel
used across the whole library (xslt3, CLI, c14n, …):

- xml/xhtml → `helium.Writer` (`writer.go` / `writer_escape.go` `escapeText` +
  `escapeAttrValue`, also `writer_xhtml.go`, `writer_dtd.go`).
- html → `html.Writer` (separate package, separate escaping).

A correct fix requires threading a normalization form into BOTH writers' text and
attribute-value escape paths (not comment/PI/name/markup paths), adding a
`golang.org/x/text/unicode/norm` dependency to those low-level packages, and
re-working the existing char-map/normalization ordering (currently handled at the
xpath3 top level via the SPUA sentinel trick in `withCharMapSentinels` /
`expandCharMapSentinels`, which protects char-map replacements from the whole-
string normalization pass — that interaction would have to move into the funnel
too). That is a cross-package restructure of the serialization pipeline with high
blast radius on code paths shared far beyond fn:serialize.

**Impact / risk of current behavior:** Practically negligible — it only misfires
when `normalization-form` is set to a non-`none` form AND an element name,
attribute name, comment, PI, or doctype contains characters that the requested
form would alter. No W3C fn-serialize target case exercises this. Text-method
output (whole output is a text node) and json string content are already correct
under whole-string normalization; adaptive/json ignore the parameter.
