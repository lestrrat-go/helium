# XSD 1.1 Implementation State — Index

Detailed feature-by-feature state of the XSD 1.1 opt-in path (`Compiler.Version(xsd.Version11)`). This is the current implementation surface, not a changelog. **Read the relevant sub-doc below before working in that area of `xsd/`.**

Version-resolution rules and the `SkipDatatypeIntegrityChecks` contract that gate everything below live in `CLAUDE.md` → "XSD — Version Toggle".

Convention across all sub-docs: **version-INDEPENDENT** rules run in both 1.0 and 1.1; all others are `Version11`-gated with the 1.0 path byte-identical to origin. Spec citations (§, cvc-*, cos-*, src-*) and W3C test IDs identify the governing rule and its conformance evidence.

XSD 1.1 is fully implemented (967/0 on the W3C suite). Sub-docs by area:

| Sub-doc | Covers |
|---------|--------|
| `xsd11-types.md` | xs:assert (complex + simpleContent), xs:assertion facet, conditional type assignment (xs:alternative), simpleContent content-type narrowing, attribute inheritance, 1.1 built-in datatypes, 1.1-only lexical forms, simple-type 1.1 edges |
| `xsd11-content-models.md` | UPA weakening, open content (xs:openContent / xs:defaultOpenContent), xs:all relaxations, wildcard notNamespace/notQName, particle-restriction relaxations & language-inclusion fallback, content-model backtracking, Wildcard EDC, derivation-block / substitution rules |
| `xsd11-identity-constraints.md` | xs:key/unique/keyref field-node classification & value canonicalization, @ref, @xpathDefaultNamespace, structural/placement rules, skip-wildcard selector scoping |
| `xsd11-doc-walks.md` | Document-wide xs:ID/IDREF/IDREFS and xs:ENTITY/ENTITIES uniqueness + referential-integrity walks |
| `xsd11-representation.md` | xs:override, xsi: attribute references, conditional inclusion (vc:), NCName/QName whitespace collapse, xs:notation, facets, and the many version-INDEPENDENT XML-representation / structural / composition checks |

Do NOT enforce 1.1-only clauses in the 1.0/default path — 1.0 must stay byte-identical to origin.
