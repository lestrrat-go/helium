# Multi-phase `ds:Reference` transform pipeline

Status: implemented.

## Processing model

`transform_pipeline.go` executes every `ds:Transform` in document order over a
tagged value:

```text
transformValue = nodeSetValue | octets
```

The next transform's contract decides whether the executor converts the current
value:

- octets → node-set: parse with the configured reference parser;
- node-set → octets: inclusive Canonical XML 1.0 without comments;
- final node-set → digest octets: the same inclusive C14N 1.0 conversion.

No conversion runs when the value already has the required kind. Repeated kind
changes are valid:

```text
octets → XPath → C14N 1.1 → XSLT → XPath → C14N 1.1 → XSLT
```

This implements the [XML Signature Second Edition Reference Processing
Model](https://www.w3.org/TR/2008/REC-xmldsig-core-20080610/#sec-ReferenceProcessingModel)
and its [ordered Transforms
rule](https://www.w3.org/TR/2008/REC-xmldsig-core-20080610/#sec-Transforms).

## Transform contracts

| Algorithm | Required input | Output |
|---|---|---|
| C14N 1.0/1.1, exclusive C14N | node-set | octets |
| Base64 decode | octets | octets |
| XPath filter (`REC-xpath-19991116`) | node-set | node-set |
| Enveloped signature | node-set | node-set |
| XSLT (`REC-xslt-19991116`) | octets | octets |

Base64 has one algorithm-specific input rule from XMLDSig §6.6.2. When its
current value is a node-set, it concatenates the remaining text-node
string-values in document order and decodes those bytes. It never applies the
generic node-set → C14N conversion. Untouched initial subtrees and node-sets
materialized by an earlier transform use the same text-node selection, excluding
comments and processing instructions.

## Entry adapters

All transform consumers call `executeTransformPipeline`:

| Entry | Starting value | Adapter |
|---|---|---|
| Same-document Reference | lazy node-set | `applyReferenceTransforms` |
| General-XPointer Reference | lazy node-set | `applyReferenceTransforms` |
| External Reference | resolver octets | `externalReferenceDigestInput` |
| Same-document signing | lazy node-set | `signReferenceOctets` |
| External signing | resolver octets | `externalReferenceDigestInput` |
| Manifest inner Reference | same Reference adapters | `canonicalizeReference` |
| Transformed same-document RetrievalMethod | lazy subtree node-set | `processRetrievalMethod` |
| Transformed external RetrievalMethod | resolver octets | `externalReferenceDigestInput` |

A transform-free same-document RetrievalMethod stays outside the executor. It
passes the resolved element directly to `interpretRetrievalElement`, including
recursive RetrievalMethod handling.

## Value model

`transformValue.kind` is always explicit. Empty node-sets and empty octet streams
remain distinct states. Constructors populate exactly one payload.

`nodeSetValue` carries:

- owning `*helium.Document`;
- explicit `[]helium.Node` membership when materialized;
- `materialized` state;
- whether it remains the original Reference selection;
- the Reference URI's comment-membership rule;
- a lazy `referenceNodeSetOrigin` when no transform needs explicit membership.

The lazy origin records the resolved document/target, whole-document selection,
comment policy, containing Signature, optional detached/enveloping root, and a
pending enveloped exclusion.

## Lazy same-document node-sets

An initial same-document node-set stays lazy until XPath, node-set Base64, or an
ordered enveloped operation needs explicit membership. Direct canonicalization
uses the established specialized helper:

- whole document → `canonicalize`;
- attached subtree → `canonicalizeSubtree`;
- enveloped selection → `canonicalizeEnveloped`;
- detached/enveloping subtree → `canonicalizeDetachedSubtree`.

This preserves namespace inheritance, `xml:*` inheritance, comment membership,
detached subtree restoration, and caller-DOM immutability.

`materializeNodeSet` uses `collectDocumentNodes` or `collectSubtreeNodes`, removes
comments excluded by the Reference URI form, applies a pending enveloped
exclusion, clears the lazy origin, and marks the set materialized even when it is
empty.

A document parsed from octets is materialized immediately with
`collectDocumentNodes`. Its node-set includes parsed comments and carries no
original Reference or Signature identity.

## Static validation

`validateTransformSteps` scans the complete list before execution. It checks:

- supported algorithm URI;
- required XPath expression and successful XPath compilation;
- required XSLT stylesheet;
- usable non-nil/non-typed-nil XSLT transformer;
- sign-side XPath/XSLT capability;
- enveloped-transform entry and node-identity requirements.

Validation tracks value kinds only to determine when original Signature identity
is lost. It never rejects a list merely because it changes kinds repeatedly.
This guarantees a later static error is reported before an earlier injected XSLT
transform runs.

Signing and execution share this validator. External verification also runs it
against octet input before URI joining or resolver invocation. Base64 is
supported for signing through the exported `Transform` interface. XPath and XSLT
remain rejected on signing because `ReferenceConfig` cannot carry or emit their
required child content.

## Execution

`executeTransformPipeline`:

1. validates the complete list;
2. checks `ctx.Err()` before each transform;
3. applies Base64's node-set text conversion when needed;
4. performs a generic kind conversion only when required;
5. executes the declared transform;
6. replaces the current value with the transform result;
7. checks `ctx.Err()` before finalization;
8. canonicalizes a final node-set with inclusive C14N 1.0.

Earlier intermediate documents and byte slices are not retained by the
executor.

## Algorithm behavior

### Canonicalization

Explicit C14N uses the declared algorithm and inclusive namespace prefixes. An
octet input is parsed first. The initial same-document selection applies
`effectiveC14NMethod` for URI-form comment membership. A reparsed node-set uses
the comments actually present in its parsed bytes.

### Base64

Octet input is decoded directly with `internal/xmlbase64`. Node-set input uses
the XMLDSig text-node rule. Invalid data wraps `ErrInvalidSignature`. Decoded
octets may feed any later transform.

### XPath

XPath materializes the current node-set and calls `applyXPathFilter` once for
that step. Namespace bindings, `defaultXPathOpLimit`, and the bounded
`newDSigXPathEvaluator` remain unchanged.

`here()` uses the bearing `ds:XPath` node only while it belongs to the current
node-set's document. After an octet result is reparsed into another document,
`here()` is registered without a node and fails with `ErrHereUnavailable`.

### Enveloped signature

The transform is valid only on the original same-document Reference selection.
On a lazy set it records a pending exclusion so a following C14N can use
`canonicalizeEnveloped`. On a materialized original set it removes the containing
Signature and descendants by node identity.

External References, RetrievalMethods, and any node-set reparsed after an octet
boundary lack the required parent-document identity and fail with
`ErrUnsupportedTransform`.

### XSLT

XSLT receives the current pipeline octets. A node-set is implicitly converted
with inclusive C14N 1.0 first. Each declared XSLT step invokes the configured
`XSLTTransformer`; multiple calls per Reference are valid. Output bytes remain
unchanged unless a later transform requires parsing or canonicalization.

XSLT stays opt-in and verify-only. Nil and typed-nil transformers fail closed.
The transformer owns compute, memory, output, I/O, URI, and XXE policy.

## Parser and resource policy

Every generic octets → node-set transition uses `runtime.parser`, supplied by
`Signer.ReferenceParser` or `Verifier.ReferenceParser`. The default
`helium.NewParser()` blocks XXE, filesystem access, DTD/entity substitution, and
network access while retaining parser limits.

Existing bounds remain active:

- `resolveReferenceOctets` caps resolver results at `maxReferenceBytes`;
- parser limits apply independently to every reparse;
- `defaultXPathOpLimit` applies to every XPath step;
- `verifyBudget` still caps Reference/KeyInfo work;
- RetrievalMethod depth, visited-URI, entry, and decoded-byte limits remain;
- XSLT output policy remains the injected transformer's responsibility.

No default intermediate-octet cap is imposed.

## Error behavior

- Unknown algorithms, invalid parameters, absent XSLT capability, invalid
  enveloped placement, and intermediate parse failures →
  `ErrUnsupportedTransform`.
- Invalid Base64 → `ErrInvalidSignature`.
- First parse of externally resolved input required as XML →
  `ErrReferenceNotFound`.
- Context cancellation/deadline → context error unchanged.
- XSLT transformer error → unchanged.

New transform errors identify the step index and algorithm. Intermediate parse
errors identify both the producing and consuming steps. Top-level sign/verify
paths retain `ReferenceError` / `VerificationError` wrapping.

## Coverage and DOM guarantees

Reference URI resolution runs once before transform execution. Duplicate-ID and
general-XPointer single-apex checks therefore remain authoritative. Reparses never
resolve the Reference URI again, and an ID produced by XSLT cannot redirect
`VerifiedReference.Element` or coverage attribution.

Enveloped canonicalization deep-copies the document. Detached canonicalization
restores temporary moves on every exit. XPath builds new node slices, and reparses
allocate separate documents. The executor does not mutate the caller's DOM.

## Test coverage

Internal tests cover:

- repeated C14N, Base64, and XSLT steps;
- node-set/octet conversions in both directions;
- XSLT → XPath and Base64 → XPath reparsing;
- final implicit C14N;
- complete-list algorithm, parameter, and XPath compilation validation before
  transformer invocation;
- external transform validation before resolver invocation;
- external versus intermediate parse error classes;
- `here()` after reparse;
- both XPath/enveloped declaration orders;
- empty node-set and empty-octet states;
- cancellation between steps;
- Base64 sign/verify symmetry;
- transformed RetrievalMethod multi-phase execution;
- nil and typed-nil XSLT handling;
- W3C `defCan-1`, `defCan-2`, and `defCan-3` digest/signature verification;
- all existing xmldsig1 interop and regression vectors.

## Future policy

An intermediate/cumulative byte limit requires a separate public option; adding
a hard default here would reject transform lists that are otherwise valid. A
typed Base64 constructor may be added without changing executor semantics.
XPath/XSLT signing requires immutable parameter-bearing transform types plus
serialization support and, for XSLT, a signer-side transformer seam.
