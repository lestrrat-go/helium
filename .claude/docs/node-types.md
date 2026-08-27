# Node Types

## Base Structs

### `docnode` — base for all nodes with tree linkage

```
Fields:
  name       string      # node name (or sentinel: "(document)", "(text)", "(CDATA)", "(comment)")
  etype      ElementType # node type enum
  firstChild Node        # linked list of children
  lastChild  Node
  parent     Node
  next       Node        # sibling links
  prev       Node
  doc        *Document   # owning document
  line       int         # source line number
```

Methods: `FirstChild()`, `LastChild()`, `NextSibling()`, `PrevSibling()`, `Parent()`, `Content()` (aggregates children), `Type()`, `Name()`, `LocalName()`, `Line()`, `OwnerDocument()`

### `node` — extends docnode with content and namespaces

```
Fields:
  docnode                  # embedded
  content    []byte        # text content (Text, Comment, CDATA, Entity)
  properties *Attribute    # linked list of attributes (Element only)
  ns         *Namespace    # active namespace
  nsDefs     []*Namespace  # all namespace declarations on this element
```

Methods: `DeclareNamespace(prefix, uri)`, `SetActiveNamespace(prefix, uri)`, `Namespace()`, `Namespaces()`, `Prefix()`, `URI()`, `Name()` (qualifies with prefix: `prefix:localname`)

## Namespace Storage (critical)

Element has THREE namespace-related fields:
- **`ns *Namespace`** — active namespace qualifying the element name
- **`nsDefs []*Namespace`** — all `xmlns` declarations on this element
- **`properties *Attribute`** — attributes (some may be xmlns, stored separately in nsDefs)

`Namespace` is NOT a tree node — lightweight struct: `{etype, href, prefix, context}`. No parent/child/sibling links.

`NamespaceNodeWrapper` wraps Namespace for XPath: adds docnode linkage. `Name()` = prefix, `Content()` = URI. Read-only (AddChild/AddSibling are no-ops).

## Attribute Storage

Attributes are a **linked list via next/prev** on the Element, NOT children:
- `Element.properties` → first Attribute
- `Attribute.NextAttribute()` → next in list
- Attribute VALUE stored as children Text/EntityRef nodes of the Attribute itself
- `Attribute.Value()` aggregates child content as string

## Node Builders (colon rule)

`Document.CreateElement(name) → (*Element, error)` and `Document.CreateAttribute(name, value, ns) →
(*Attribute, error)` REJECT a colon in `name` (the single name field holds a bare local name; a colon there is
an unbound prefix that would serialize as namespace-ill-formed XML). Supply a namespaced name through the `NS`
siblings instead: `Document.CreateElementNS(localname, ns) → (*Element, error)` rejects a colon in
`localname`, allocates the element via `CreateElement`, then installs `ns` as the active namespace with
`SetNamespace` (so `Name()` renders `prefix:localname` and the cached qname is invalidated; if `ns` is
slab-backed by another document `SetNamespace` marks that source escaped — Cross-Document Slab Safety below);
`Element.SetAttributeNS(localname, value, ns)` is the attribute counterpart. A nil receiver allocates a
standalone (heap, non-slab) element/attribute. Neither builder checks the rest of the XML Name/NCName grammar.
The parser/copy code never passes a colon — it splits a QName into prefix+local and sets the namespace via
`SetActiveNamespace`/`SetNamespace` — so the rejection only guards hand-built trees.

## ElementType Enum (21 values)

```
ElementNode(1) AttributeNode(2) TextNode(3) CDATASectionNode(4) EntityRefNode(5)
EntityNode(6) ProcessingInstructionNode(7) CommentNode(8) DocumentNode(9)
DocumentTypeNode(10) DocumentFragNode(11) NotationNode(12) HTMLDocumentNode(13)
DTDNode(14) ElementDeclNode(15) AttributeDeclNode(16) EntityDeclNode(17)
NamespaceDeclNode(18) XIncludeStartNode(19) XIncludeEndNode(20) NamespaceNode(21)
```

## Node Types Summary

| Type | Struct | Base | Children | Content | Siblings | Special Fields |
|------|--------|------|----------|---------|----------|----------------|
| Document | `Document` | docnode | ✓ | — | ✗ | version, encoding, standalone, url, properties, intSubset, extSubset, ids map |
| Element | `Element` | node | ✓ | via children | ✓ | properties (Attribute linked list), ns, nsDefs, contentHasReference (parser flag: a reference appeared in content; validity-only, invisible to serialization/C14N/XPath/copy) |
| Attribute | `Attribute` | docnode | ✓ (text/entityref for value) | via children | ✓ (linked list) | ns, atype, defaultAttr, syntheticBase (parser-injected external-entity xml:base) |
| Text | `Text` | node | ✗ (merges) | ✓ content | ✓ | Adjacent text nodes auto-merge. fromCharRef (parser flag: some content came from a character reference; OR-merged on text merge; validity-only, invisible to serialization/C14N/XPath/copy) |
| CDATASection | `CDATASection` | node | ✗ | ✓ content | ✓ | — |
| Comment | `Comment` | node | ✗ | ✓ content | ✓ | — |
| PI | `ProcessingInstruction` | docnode | ✗ | data field | ✓ | target, data (Name() returns target). AddChild/AppendText route text into `data`; non-text children rejected |
| EntityRef | `EntityRef` | node | ✓ (if expanded) | ✓ (if resolved) | ✓ | References Entity by name |
| Entity | `Entity` | node | ✓ (parsed) | ✓ content | ✓ | entityType, externalID, systemID, uri, checked, expanding, expandedSize |
| DTD | `DTD` | docnode | ✓ (decls) | — | ✓ | attributes/attrsByElem/elements/entities/pentities/notations maps, attrDecls slice, externalID, systemID |
| ElementDecl | `ElementDecl` | docnode | — | — | ✓ | decltype, content (grammar tree), attributes, prefix |
| AttributeDecl | `AttributeDecl` | docnode | — | — | ✓ | atype, def, defvalue, tree (enumeration), prefix, elem, external (declared in external subset/PE) |
| Notation | `Notation` | docnode | — | — | ✓ | publicID, systemID |
| Namespace | `Namespace` | — | ✗ | ✗ | ✗ | href, prefix, context (lightweight, no tree linkage) |
| NamespaceNodeWrapper | `NamespaceNodeWrapper` | docnode | ✗ | ns.URI() | ✗ | ns pointer (XPath only, read-only) |

## Key Behaviors

### Text Node Consolidation
`Text.AddSibling(Text)` → content merged instead of creating sibling. Prevents whitespace node bloat. Mirrors libxml2 TEXT consolidation.

### PI Content Is A String, Not Children
A `ProcessingInstruction` stores its content in the `data` string field (mirrors libxml2's XML_PI_NODE, whose
content is the node's content string). It has NO element/text children. `AppendText` and an `AddChild` of a
Text/CDATA node append the text to `data`; `AddChild` of any other node type is rejected (so the tree cannot
be corrupted and serialization stays `<?target data?>`). The serializer reads `pi.data` directly.

### DTD Map Keys
- Elements: `name:prefix` (string)
- Attributes: `attrDeclKey{local, prefix, elem}` (struct, scoped to element)
- Entities: `name` (flat)

Attribute declarations are ALSO held in two registration-order sequences: `attrDecls []*AttributeDecl` (every
declaration in the subset) and `attrsByElem map[string][]*AttributeDecl` (the same declarations keyed by
owning element name). Registration order is declaration order — the `<!ATTLIST>` order, or the
`AddAttributeDecl` call order — and it is the order every consumer sees, so DTD validation reports attribute
diagnostics in declaration order instead of a per-run random one: the element-keyed readers
(`AttributesForElement`, `validateElementAttributes`, `checkStandaloneExternalDefaults`, `GetElementByID`)
range `attrsByElem`, and the subset-wide declaration-consistency checks (`validateDTDDeclarations`,
`validateOneIDPerElement`) range `attrDecls`. Neither ranges the `attributes` map, whose Go iteration order is
randomized per run. `AttributesForElement(elem)` returns a `slices.Clone` of the index slice, so the caller
gets a fresh copy and cannot mutate the DTD's own declarations. `registerAttribute` is the single entry point
that keeps `attributes`, `attrDecls`, and `attrsByElem` in sync; `copy_dtd.go`'s deep-copy path writes them
directly since it does not go through `registerAttribute`, and it rebuilds the two sequences from the SOURCE's
`attrDecls`, translating each source declaration through an original->copy correspondence. The child list it
walks carries the serialization order, which relinking a declaration (`AttributeDecl.AddSibling`) changes
without re-registering it, so the two orders are independent.

The QName→(name, prefix) split (`AddElementDecl`/`GetElementDesc`/`AddAttributeDecl`) splits on the FIRST
colon but treats a LEADING colon as part of the local name (mirrors libxml2 `xmlSplitQName3`): `:x` keys as
`(name=":x", prefix="")`, distinct from the unprefixed `x` (`(name="x", prefix="")`), so a leading-colon
element declaration is not a spurious redefinition of the unprefixed one (a leading colon is a legal XML 1.0
Name start character even though it is not a valid QName prefix). The attribute lookup table is keyed by an
`attrDeclKey{local, prefix, elem}` struct. A `local + ":" + prefix + ":" + elem` string would collide distinct
triples: name `"b:a"` on element `"c:d"` (local `a`, prefix `b`, elem `c:d`) and name `"c:a:b"` on element
`"d"` (local `a:b`, prefix `c`, elem `d`) both flatten to `a:b:c:d`; the struct key keeps them distinct,
mirroring the parser's `specialAttrKey`.

The `DTD.Add*` builders (`AddEntity`/`AddNotation`/`AddElementDecl`/`AddAttributeDecl`) build a declaration
from public parameters, register it in the lookup table, AND link it into the DTD child list so it serializes.
`AddAttributeDecl(elem, name, atype, def, defvalue, enumValues)` is the `<!ATTLIST>` counterpart; the
low-level table-only `registerAttribute` is unexported. Like its sibling constructors, `AddAttributeDecl`
validates only the enum parameters (`atype` must be a defined `enum.Attr*` value, `def` a defined
`enum.AttrDefault*` kind — both `ErrInvalidArgument` otherwise) and rejects a duplicate
(`ErrDuplicateDeclaration`); it splits `name` into prefix+local on the first colon exactly as `AddElementDecl`
does, and clones `enumValues` before storing (a later caller mutation cannot corrupt the stored decl). It
TRUSTS the caller for well-formed names, default values, and namespace/URI syntax — it does NOT validate them
against the parser's Name/URI grammar, character ranges, or cross-declaration validity constraints (those are
enforced by `ValidateDTD` when the document is validated), matching how
`AddNotation`/`AddEntity`/`AddElementDecl` trust the caller. `AddNotation` enforces exactly one narrow name
rule on top of that trust: it rejects a colon in the notation name (`ErrInvalidArgument`), because a notation
name is an XML NCName and the parser rejects a colon-bearing `<!NOTATION>` name (`parser_dtd_attr.go` "colons
are forbidden from notation names"), so a colon would serialize to un-reparseable output.

`DTD.RemoveElement(name, prefix) → *ElementDecl` deletes the lookup-table entry AND unlinks the `ElementDecl`
node from the DTD child list (via `unlinkNode`), so a removed declaration is no longer serialized; it returns
the removed decl (nil if none). `AddElementDecl` uses it internally to drop an `UndefinedElementType`
placeholder while completing a real declaration.

### ElementContent (content-model tree)
`ElementContent` is the binary content-model tree of an `ElementDecl` (fields all private): leaves are
`ElementContentElement` (named reference, carries name/prefix) or `ElementContentPCDATA`; interior nodes are
`ElementContentSeq` (,) or `ElementContentOr` (|), each requiring BOTH `c1` and `c2`. Every node has an
occurrence indicator (`ElementContentOnce`/`Opt`/`Mult`/`Plus`). The serializer (`writer_dtd.go`) and the
matcher (`valid.go`) dereference `c1`/`c2` for seq/choice nodes, so a structurally-incomplete node (a
seq/choice with nil children) would nil-deref. `AddElementDecl` rejects such a model up front
(`validateElementContentModel`, `valid.go`) before storing it, so no stored model can panic serialization.
Safe public composition: `Document.CreateElementContent(name, etype)` (leaf),
`Document.CreateElementContentSeq(c1, c2, occur)` / `Document.CreateElementContentChoice(c1, c2, occur)`
(validated interior nodes), and `ElementContent.SetOccurrence(occur)`.

### Document ID Lookup
`Document.GetElementByID(id)` — `idsSkip` is authoritative and checked FIRST: when `SkipIDs()` is true it
returns nil immediately, resolving NO ids regardless of the `ids` table or DTD subsets. Otherwise it is O(1)
via `ids map[string]*Element` (populated during parse), falling back to an O(n) tree walk if the map is empty.
The fallback walk consults ID-typed attribute declarations in BOTH the internal and external DTD subsets
(`intSubset`, `extSubset`). `Document.SkipIDs()`/`SetSkipIDs(bool)` read/write the `idsSkip` flag (set from
parser `SkipIDs(true)`); because it is checked before the table, `SetSkipIDs(true)` suppresses resolution even
on a doc with a populated ID table, and `SetSkipIDs(false)` restores it. Carry the flag onto derived docs so
id() semantics match the source. `Document.IDTable()` returns the document's own `ids` map (read-only; nil for
API-built docs without an interned table) so a derived doc can rebuild an equivalent table by translating each
entry's element through an original->copy correspondence. `helium.CopyDoc` does exactly this for a full
document copy: it rebuilds the copy's ID table by translating each source entry through the source->copy
element correspondence recorded during the deep copy (via the `deepCopier` onCopy hook), and carries over the
`idsSkip` flag — so a SkipIDs source yields a copy that resolves NO ids, and a parsed source's ids resolve to
the COPY's own elements (never aliasing the source map). The xsl:strip-space copy (`xslt3` copyAndStrip) does
the same from `src.IDTable()`, preserving the source's interned ID-table identity — the copy resolves ids to
elements corresponding exactly to the source's, and at O(1). Re-deriving them would fall back to the lazy O(n)
walk. The fallback walk consults ID-typed ATTLIST declarations by their raw qualified name (prefix+local), so
it correctly resolves a prefixed element's qualified ATTLIST (`<!ATTLIST a:item eid ID>`) — but rebuilding
from the source table is still preferred for identity and cost fidelity. `CopyDoc` also DEEP-COPIES the
source's external subset (via `CopyExtSubset`), so the copy's fallback walk sees the same ID-typed ATTLIST
decls as the source. `helium.CopyExtSubset(src, dst *Document)` DEEP-COPIES the source's external subset into
`dst` (independent `*DTD`; mutating one never affects the other); `CopyDTDInfo`, by contrast, copies only the
internal subset and links it into the document tree (and returns an error when `dst` already has one).

### Document Encoding (defaulted vs raw)
`Document.Encoding()` returns `"utf8"` when the `encoding` field is empty (no encoding recorded);
`Document.RawEncoding()` returns the field verbatim (empty = none recorded). The XML serializer (`writer.go`)
emits `encoding="..."` ONLY when the raw encoding is non-empty, so a faithful document copy must propagate
`RawEncoding()` — using `Encoding()` would make the copy serialize a spurious `encoding="utf8"` that was not
recorded. `CopyDoc` reads the raw field directly (same package); cross-package copiers (e.g. the
xsl:strip-space copy in `xslt3`) use `RawEncoding()`. `Version()` and `Standalone()` already return raw,
unsynthesized values.

### Content() Default
`docnode.Content()` walks children and concatenates (returns a fresh buffer). Overridden by Text, CDATA,
Comment, PI, EntityRef. It has a POINTER receiver (`*docnode`) so the receiver is the real owning node — every
`Node` is a pointer (the sealed `baseDocNode()` interface method is itself pointer-receiver), so this changes
nothing for callers. The aggregation runs through the private `aggregateOwnedContent` helper, which advances
between children with the OWNED-BOUNDARY rule (`nextOwnedChild`): a foreign child — an entity reference's
shared Entity child, owned by the DTD, whose sibling pointers belong to the DTD declaration list — ends the
aggregation instead of spilling into another list's siblings, and a per-list seen set terminates a cyclic
sibling pointer. The recursion into a container child's subtree carries an ACTIVE-PATH set (the container
docnodes currently being aggregated, receiver inclusive): a child already on that path is a back-edge and is
skipped, so a pure child-pointer cycle (`element -> element -> element`, NOT routed through an Entity's
terminating stored-text `Content()`) terminates instead of recursing forever. A leaf child
(Text/Comment/CDATA/PI/Entity/NamespaceNodeWrapper — `aggregatesOwnContent` returns false) is self-contained
and called directly; every other node type recurses under the guard. The active-path set is not a global
visited set, so a shared DAG node reached on a different path is re-aggregated per occurrence.

The text-bearing leaves (Text, Comment, CDATASection) store content in an internal mutable `content []byte`.
Their exported `Content()` returns a **defensive copy** (`bytes.Clone`) so a caller mutating the result cannot
corrupt the DOM. Internal read-only hot paths (serializers in `writer.go`/`writer_xhtml.go`) use the
package-level `rawContent(Node)` helper — backed by an unexported `rawContent()` method on each of those three
leaf types — to get the raw slice without the copy. The `rawContentNode` interface gates the no-copy path; for
any other node `rawContent` falls back to `Content()`. PI/EntityRef/Entity/NamespaceNodeWrapper already
returned string-derived copies and are unaffected.

### Predefined Entities
5 unexported singletons: `entityLT`, `entityGT`, `entityAmpersand`, `entityApostrophe`, `entityQuote` (resolved by name through `resolvePredefinedEntity`). Type `InternalPredefinedEntity`. Cannot be redeclared.

### NamespaceDeclNode Special Case
Skipped in `setTreeDoc()` — sentinel type rarely instantiated.

### Tree Operations
- `addChild(parent, child)` — append to end of children. Attribute-aware, BEFORE the generic child-splice: an
  `*Attribute` operand is never an ordinary child — on an `*Element` parent it is routed into the properties
  list via `addProperty` (running the preflight first for cross-document escape marking and auto-unlink from
  any prior parent/property chain, then replacing a same-named attribute in place, mirroring libxml2
  `xmlAddChild`), so `elem.AddChild(attr)` serializes as an attribute and never appears in `Children()`; on
  ANY other parent (Document, Text, Comment, PI, EntityRef, Attribute, …) an attribute has no valid placement
  and is rejected with a `%w`-wrapped `ErrInvalidOperation`. A non-attribute child takes the ordinary
  child-list path (a `*Document` accepts multiple element children — it is an XDM document node, not a
  well-formed-document constraint; single-root enforcement lives in `SetDocumentElement`, mirroring libxml2's
  `xmlDocSetRootElement`).
- `addSibling(node, sibling)` — append to end of siblings. The tree an append leaves behind is always the one
  the plain `NextSibling()` walk from the anchor produces, INCLUDING every `parent.lastChild` write, which is
  unconditional exactly as it is in that walk. What changes is how the append point is REACHED: the generic
  child-list branch resolves it from `parent.lastChild` without walking whenever `tailJumpTarget` (`node.go`)
  can prove that record is the very node the walk would have found, and otherwise walks. The proof needs two
  facts. (1) The ANCHOR is a member of the chain `parent` owns — `chainMember` (`node.go`) answers in a pointer
  comparison when the anchor is `parent.firstChild`, and otherwise walks `prev` to the head of the anchor's own
  chain and compares it against `parent.firstChild`. (An anchor that is `parent.lastChild` never reaches
  `chainMember`: `tailJumpTarget` has already declined, because the anchor and the recorded tail are the same
  node.) That walk is bounded by the anchor's distance BEHIND it, never by the chain ahead of it, so it can
  never cost more than the `NextSibling()` walk it replaces; it carries a `siblingCycleGuard` so a corrupt
  `prev` chain terminates instead of spinning, and it crosses only RECIPROCAL `prev` edges (`reciprocalPrev` —
  the `prev` node points forward at the node again), so a one-way edge cannot carry it out of the anchor's own
  chain. There is no raw `prev` setter, but a one-way `prev` edge survives whenever a node is spliced out of a
  chain from the FRONT, so the check is not hypothetical. (2) `parent.lastChild` is the final node of that same
  chain. NO local read can establish (2): two trees can have pointer-identical neighborhoods around `parent` and
  `lastChild` and differ only in a `next` pointer an unbounded distance forward from `firstChild`. It holds
  instead as an INVARIANT of the guarded paths, each of which moves `lastChild` only to a node it has just
  linked onto the chain, so what is checked is that this document holds no node claiming a parent it is not a
  child of: `Document.offChainClaims` (`document.go`). `noteOrphanedChildClaim` records the one such claim the
  GUARDED paths create. A parent holding a `firstChild` with NO `lastChild` — the shape
  `Document.stringToNodeList` leaves behind on an entity referenced from an attribute value — no longer reaches
  the empty-parent branch of `addChild`/`appendFastChild`, because `resolveOwnedTail` walks that child list and
  returns its true tail, so the append joins the list instead of overwriting `firstChild`. An append through
  that detached child then moves the parent's recorded tail off the child list. Recording the claim at the
  moment it is created is what keeps the later append byte-identical to the walk. Each claim is recorded on the
  document owning the parent whose chain is at stake and on the document owning the detached child, each
  resolved through `owningDocument` so a `*Document` names ITSELF (a document node's own `doc` pointer is nil
  because it IS the owner). Once set it stays set, and appends for that document walk — which is what every
  other tree operation does unconditionally. The record FOLLOWS THE TREE, not only the document it was first
  made on, because a subtree can change owning document afterwards, and a claim made on a still-DETACHED subtree
  has no document to be recorded on at all. `adoptOffChainClaims` (`node.go`) carries it across every change of
  owner — `docnode.SetOwnerDocument`, `setTreeDoc`'s attribute write, `setListDoc`'s direct write, and
  `noteCrossDocumentEscape` — and the unattributable case lands on the package-level `unownedOffChainClaim` (an
  `atomic.Bool`), which a document inherits when it adopts a subtree that arrives owning no document. That
  global is the ONE package-wide part; the flag itself stays per-DOCUMENT, because one claim must not slow every
  other document in the process. `TestAdoptOffChainClaimWithoutOwner` (`node_sibling_internal_test.go`) pins the
  unowned case. A `*Document` parent is NOT declined by type. Its owning document is read through
  `owningDocument`, so it names itself, and a document's child list is an ordinary sibling chain that an append
  walks exactly like any other. A document is claimed off-chain by `CopyExtSubset`, which gives the copied
  EXTERNAL subset the destination document as its parent and then leaves it reachable only through `ExtSubset`,
  never from the child list, so an append through that subset records a tail off the child list. That is a
  CONDITION, not a type, and `Document.offChainChildClaim` (`document.go`, set by `CopyExtSubset`) records it
  separately from `offChainClaims` so it declines only for the `*Document` parent handed the claimant, not for
  every parent in that document. (`CreateInternalSubset` also gives a `DTD` the document as its parent, but it
  splices that `DTD` into the child list, so it creates no claim.) `TestTailJumpTargetDocumentParent`
  (`node_sibling_internal_test.go`) pins all three outcomes. `tailJumpTarget` then makes three cheap
  confirmations that the record is usable (`tail != anchor`, `tail.next == nil`, `tail.parent` is this parent);
  they cannot fail on a document holding no recorded claim, and are kept because a tree may span documents
  (`noteCrossDocumentEscape`) while the record does not, so an uncovered corner degrades to the walk instead of
  splicing onto the wrong node. They also keep the stale-`lastChild` repair path `addChild`/`appendFastChild`
  depend on: they call `AddSibling` on `parent.lastChild` precisely when that node's `next` is non-nil, so the
  resolution declines and the walk finds and repairs the true tail. WHAT THE O(1) RESOLUTION IS WORTH DEPENDS ON
  THE ANCHOR: appending through `parent.firstChild`, or at the true tail (where the anchor's `next` is nil and
  `addSibling` links directly without consulting `tailJumpTarget` at all), is O(1) per call and LINEAR over N
  appends (105µs vs 49.5ms at N=4000, a ~470x win), while appending through a MIDDLE anchor stays QUADRATIC —
  `chainMember`'s `prev` walk grows with the chain — and gains a ~3.8x constant factor instead (26.5ms vs 99.6ms
  at N=4000). `BenchmarkAddSibling` (`node_sibling_bench_test.go`) measures five arms: `tail`, `nontail` (=
  first child) and `middle` on an `*Element` parent, plus `docnontail` and `docmiddle` on a `*Document` parent,
  which track the element arms to within noise. The attribute-chain branch (an anchor genuinely reachable from
  its owning `*Element`'s `properties` chain) resolves its tail by walking, bounded by the attribute count,
  because the `properties` membership test already costs that walk. A node can claim a parent without being a
  member of that parent's child list — the detached-child shape above, `CopyExtSubset`'s copied external subset,
  or a package-private `unsafeSetParent` write. An append through such a node records its own result as the
  parent's `lastChild`, moving that record off the child list. All three append entry points then land at the
  end of the REACHABLE children: `AddSibling` walks from its anchor, and `AddChild`, `appendFastChild` and
  `appendCopiedChild` reach the same node through `resolveOwnedTail` (`node.go`), which returns nil when
  `parent.firstChild` is nil whatever `lastChild` records, trusts the record only when that node claims this
  parent, has no `next`, AND the owning document holds no off-chain claim (the same signal `tailJumpTarget`
  declines on, read through `holdsOffChainChildClaim`), and otherwise walks the child list under a
  `siblingCycleGuard` to the last node still claiming the parent. That is what keeps an append in the same place
  whichever entry point the caller used, and it is why a stale record is repaired rather than followed: trusting
  `lastChild` alone loses the reachable list when `firstChild` is nil (the `CopyExtSubset` shape) and discards
  it when `lastChild` is nil (the `stringToNodeList` shape). `TestAddChildResolvesAnOwnedTail`
  (`node_owned_tail_test.go`) pins all three reachable shapes, and `TestAddSiblingOffChainParentClaim`,
  `TestAddSiblingCopiedExternalSubsetClaim` and `TestAddSiblingCorruptShapesMatchWalk` (`node_sibling_test.go`)
  pin the sibling side; every expectation in the last of those holds for the walk-only implementation too, which
  is what makes it a differential check. The raw setters record NOTHING, so a tree corrupted through them is
  outside this agreement — they already document the tree as inconsistent afterwards, and no importer and no
  production path can reach them.
- `replaceNode(old, new)` — swap in same position. Attribute-aware: replacing an `Attribute` updates the
  owning `Element.properties` head/chain (NOT `firstChild`/`lastChild`), and an attribute may only be replaced
  by attribute node(s) (non-attribute replacement is rejected)
- `appendFastChild(parent, child)` (`tree_fastpath.go`) — package-private no-preflight append: links child as
  last child WITHOUT the cycle-guard / duplicate-attr checks. The caller guarantees an acyclic, dup-free child
  (deep copies, freshly-built trees). Ordinary code uses `AddChild`. It backs the parser's fast SAX path
  in-package; `xslt3`'s strip-space copier reaches it through the `internal/nodelink` hook `AppendFastChild`,
  and the external `helium_test` package through `UnsafeAppendChildForTesting` in `export_test.go`.
- `appendCopiedChild(parent, child)` (`copy_deep.go`) — the deep-copy core's own no-preflight link, used by
  `copyChildren` in place of `AddChild`. It mirrors `addChild`'s linking rules exactly, INCLUDING the
  adjacent-Text merge and the `pdn.lastChild` correction that follows a merge, but skips `addChild`'s
  `wouldCreateCycle` preflight. This is what makes `CopyNode`/`CopyDoc` linear instead of quadratic in tree
  depth: bottom-up child-linking through `AddChild` runs `childReaches` over each already-built child subtree
  as it is attached, so linking a deep chain one level at a time costs the sum of all subtree sizes. Skipping
  the preflight is sound here — never elsewhere — because every node `copyNode` returns is freshly allocated
  in `dc.dst` and has never been linked anywhere: `CreateCharRef` (the `EntityRefNode` branch) creates a
  childless `EntityRef` with no shared-Entity foreign link, unlike `CreateReference`, so a copied subtree can
  never carry the one production foreign-link source `wouldCreateCycle` exists to catch. `appendCopiedChild`
  is distinct from `appendFastChild` — it is not used outside the copier — because `appendFastChild` also
  backs `xslt3`'s strip-space copier, and widening its linking rule to add a merge check would change behavior
  for that unrelated caller too.
- `DeclareNamespace(prefix, uri)` / `AddNamespaceDecl(ns)` — do NOT themselves add a second declaration for a
  prefix in `nsDefs`, and NEVER touch the node's active namespace (`n.ns`) or expanded name.
  `DeclareNamespace` allocates a fresh `*Namespace`; `AddNamespaceDecl` attaches the caller's existing object
  (no alloc), so a tree-builder can reuse one Namespace as both a declaration and an element's active ns. Both
  apply the same rule keyed on the prefix. The CONFLICT test runs FIRST and is ELEMENT-scoped: on a
  non-element node (Text, Comment, CDATASection, …) `n.ns` is never serialized, so there is never a conflict —
  the node skips straight to the dedup step below (never rejecting; `AddNamespaceDecl` installs on case 1/3
  and keeps the existing slot on a same-URI no-op). On an element node, whether or not an nsDefs entry exists:
  if the prefix is in use by `n.ns` or a NON-empty-prefix attribute's `ns` at a URI DIFFERENT from the
  requested URI, that is a genuine conflict and the tree is unchanged (both `DeclareNamespace` and
  `AddNamespaceDecl` return the same `%w`-wrapped `ErrInvalidOperation`, matchable with `errors.Is`). This
  covers both the fresh-append path (`SetActiveNamespace("p",X)` then `DeclareNamespace("p",Y)` with `Y≠X` —
  no nsDefs entry yet, but declaring `p→Y` while the reconciler synthesizes `p→X` would emit two `xmlns:p`)
  and the existing-entry path. An EMPTY-prefix attribute never uses the default namespace (an unprefixed
  attribute name is in no namespace) and the serializer skips its namespace (`reconcileOne` returns early for
  `prefix==""` && !isElement), so it is NOT counted as a conflict for the empty/default prefix — a real
  default-namespace element name via `n.ns` still counts. A use at the SAME URI is not a conflict (the
  reconciler finds the prefix already bound to that URI and synthesizes nothing). When there is no conflict:
  (1) no existing entry → append; (2) existing entry, same URI → no-op; (3) existing entry, different URI →
  COLLAPSE (replace the single slot in place: `DeclareNamespace` installs a FRESH `*Namespace`,
  `AddNamespaceDecl` installs the CALLER's object; the old slot object is left unmutated, so an aliasing
  `n.ns`/`attr.ns` or a caller-retained handle is unaffected). When `AddNamespaceDecl` RETAINS the caller's
  object (case 1 append and case 3 collapse) and that object is slab-backed by a different document than the
  node's owner, the source document is marked `slabEscaped` (Cross-Document Slab Safety below), mirroring a
  cross-document node move. A caller that rebinds an in-use prefix must clear the use itself (reassign `n.ns`
  via `SetActiveNamespace`/`SetNamespace` and any prefixed attribute); `RemoveNamespaceByPrefix` alone drops
  only the nsDefs entry, not the use, so a rebind still rejects while the use remains. These methods do NOT
  reconcile a conflict introduced AFTER declaration by `SetActiveNamespace`/`SetNamespace` (which set the
  active namespace independently); guaranteeing at most one `xmlns:prefix` per element ACROSS all mutators is
  a serializer-level concern, outside these mutators' scope. The serializer enforces it: `writer.go`
  `dropConflictingActiveNS` drops any `nsDefs` declaration for the ACTIVE namespace's prefix at a different
  URI (the general form of the default-namespace `xmlns=""` precedent — the active binding qualifies the
  element name, so it wins), and `reconcileNamespaces` carries a per-element `emitted` prefix set so no prefix
  is declared twice on one start tag even when two sources (name vs attribute) bind it to different URIs — the
  first occupant wins (the element's own declarations, then its name, before attributes). Both are no-ops for
  parsed documents (a real parse never binds one prefix to two URIs on an element), so no golden output
  changes. Residual limitation: when the element NAME needs prefix `p` at URI Y and an ATTRIBUTE on the same
  element needs `p` at a different URI X, one prefix cannot serve both; the writer keeps the output
  well-formed by letting the NAME win (emits `xmlns:p="Y"` once, suppresses the attribute's `X`), which
  mis-binds the attribute. Synthesizing a fresh prefix to preserve both faithfully would need
  `xmlReconciliateNs`-style prefix generation, out of scope here. `AddNamespaceDecl(nil)` returns `ErrNilNode`
  and leaves `nsDefs` unchanged (a nil declaration is rejected, not appended); the dedup scan still skips any
  nil `nsDefs` slot injected by an in-package field write.
- `UnlinkNode(n)` — detach a `MutableNode` from parent and siblings (delegates to the internal `unlinkNode(Node)`)
- `unlinkNode(n)` — internal detach that works for ANY sealed node via `baseDocNode()`, including
  non-`MutableNode` nodes like `NamespaceNodeWrapper`. Attribute-aware: an `Attribute` under an `*Element` is
  detached via `spliceOutAttribute`, repairing `Element.properties`

The `MutableNode` interface exposes ONLY guarded/whole-value mutation (`AddChild`, `AddSibling`, `Replace`,
`AppendText`, `SetLine`, `SetOwnerDocument`, `SetTreeDoc`). Raw single-pointer linkage that updates just one
of `parent`/`next` — with none of the cycle detection, auto-unlinking, or reciprocal back-pointer maintenance
— is NOT on the interface and NOT a method on the node types. Two package-private functions in `node.go`
provide it, both writing `n.baseDocNode().<field>` directly: `unsafeSetNextSibling(n Node, next Node)` and
`unsafeSetParent(n Node, parent Node)`. Neither is exported. The external `helium_test` package reaches them
through `UnsafeSetNextSiblingForTesting` / `UnsafeSetParentForTesting` in `export_test.go` (test builds only),
and the cycle-guard tests in `xsd` and `xmldsig1` reach `unsafeSetNextSibling` through the `internal/nodelink`
hooks `CorruptSelfNextSibling` (writes `n.next = n`) and `CorruptTypedNilNextSibling` (writes `n.next` = a
typed-nil `*Element`), which exist for those corrupt-tree fixtures and nothing else. There is no
`prev`-pointer setter. Both can leave the tree inconsistent or cyclic; they exist for low-level construction
and for tests that must build a deliberately corrupt tree to exercise the traversal cycle guards. Ordinary
code uses `AddChild`/`AddSibling`/`Replace`/`UnlinkNode`. In-package tree builders write the docnode fields
directly (e.g. `attr.parent = n`). Neither setter records anything: `addSibling`'s O(1) append-point
resolution (see `addSibling` above) is not promised to match its walk on a tree corrupted through them.

`Document.SetDocumentElement(root MutableNode)` requires `root` to be an element: a non-element kind
(Text/Comment/DTD/NamespaceDecl/…) is rejected with `ErrInvalidOperation` and the document is left untouched,
a nil (or typed-nil) `root` returns `ErrNilNode`, and a nil receiver returns `ErrNilNode` (not a silent
success). An existing document element is replaced via `Replace`; otherwise the new root is appended via
`AddChild`, so the cycle/self preflight still runs before linking.

All three insertion paths share `wouldCreateCycle(parent, cur)`: they reject inserting a node into itself or
into one of its own descendants (which would put an ancestor below itself). The guard has TWO parts: an
ANCESTOR walk (parent's parent chain, inclusive, looking for cur) that catches every cycle at O(depth) when
parent/child links are consistent; and, ONLY when cur has children, a CHILD-pointer reachability search
(`childReaches`, an ITERATIVE stack-based DFS with a per-call visited set) that catches a cycle formed through
a FOREIGN child link. A foreign link is a child whose own parent pointer points elsewhere — an entity
reference's child is the shared Entity node, whose parent stays the DTD — so `ent.AddChild(ref)` where ref's
child is ent forms a child-pointer cycle the ancestor walk cannot see. `childReaches` enumerates each node's
OWN children via `nextOwnedSibling` (a foreign child's `next` belongs to another list, so it is not followed
as a sibling) and descends fully (it does NOT skip foreign subtrees, since a live insertion parent CAN
legitimately lie inside a shared entity expansion). It is iterative (no goroutine-stack overflow on a deep
tree) and has NO depth cap — the popped-node visited set alone bounds it (each node visited once), so it is
SOUND at any depth: a depth cap would fail OPEN and admit a cycle deeper than the cap. The popped-node visited
set (`childReachesVisited`) starts as a fixed 64-entry array scanned linearly — no allocation — and promotes
itself to a map once a call pops more than 64 distinct nodes, from which point the map is authoritative; the
64-entry cap selects a DATA STRUCTURE only, never a search cutoff, so a call past the cap keeps searching
exactly as before, backed by a map. The INNER sibling enumeration is bounded by `siblingCycleGuard` (the same
allocation-free Brent's-algorithm guard `Children`/`ChildElements`/`Descendants` use, described below) rather
than a per-call sibling-seen set: a seen set would stop at the exact first repeat, while `siblingCycleGuard`
stops within a small multiple of the cycle length, so on a corrupt sibling list a few nodes may be pushed onto
the stack more than once — harmless, since the popped-node visited set deduplicates the extra pushes on pop,
and termination is unconditional either way. The childless-cur fast path (the parser hot path appends fresh
leaves) skips the search entirely, so plain parsing is unaffected; it only runs for a subtree/entity-ref
insertion. addChild/addSibling auto-unlink an already-linked incoming node before relinking so it never lives
in two places; rejection leaves the tree untouched. The shared guard + auto-unlink is factored into
`addChildPreflight`/`addSiblingPreflight`. Leaf `AddChild`/`AddSibling` overrides that take a content-merge
fast path (Text, Comment, and ProcessingInstruction — whose `AddChild` merges a Text/CDATA operand's content
into the PI data string) run the matching preflight BEFORE merging, so
`txt.AddChild(txt)`/`comment.AddChild(comment)` are rejected instead of doubling content, and an
already-linked incoming leaf (including a Text/CDATA node merged into a PI) is unlinked from its old parent
before its content is merged. These overrides also reject a nil or typed-nil operand with `ErrNilNode` before
any method call on the operand, since a typed nil reaching the type switch / merge path would panic.

The read-only traversal APIs (`Walk`, `Children`, `ChildElements`, `Descendants`, and the aggregating
`Content()`) share the same two safety properties, so a hand-built or foreign-linked graph cannot make them
wander or loop. (1) OWNED BOUNDARY: they advance between siblings via `nextOwnedChild(owner, child)` (returns
`child.NextSibling()` only when `child.Parent()` is `owner`), so a foreign child — an entity reference's
shared Entity child, owned by the DTD — ends that child list instead of spilling into the DTD's unrelated
declaration siblings. (2) CYCLE SAFETY: `Walk` carries the set of nodes currently ON the DFS stack (the active
path, O(depth)) and returns the `ErrWalkCycle` sentinel when it would descend into a node already on that path
(a child-pointer back-edge, e.g. an Entity whose child links back to its reference); it also carries a
PER-FRAME `seenChildren` set that returns `ErrWalkCycle` when a child repeats within one sibling list — this
covers BOTH a one-node self-loop (`child.next == child`) and a longer sibling cycle (`a -> b -> a`), which the
active-path set misses because each child is popped before its next sibling is examined. `nextWalkSibling`
does NOT special-case the self-loop: it lets the duplicate flow back so `seenChildren` reports `ErrWalkCycle`.
Special-casing it would silently terminate and report a corrupt one-node cycle as fully traversed.
`Descendants` mirrors this with an active-path set threaded through its recursion (visits a back-edge node
once, does not descend through it) plus a per-list sibling guard; `Children`/`ChildElements` use that same
per-list sibling guard. That guard is `siblingCycleGuard` (`iter.go`), Brent's cycle detection over the
sibling chain: three words of state and no allocation, so a well-formed list — every list the parser builds —
pays nothing for a check that fires only on a corrupt graph, where a seen set costs a map plus one insert per
child on EVERY list. The trade is where the walk stops: a seen set stops at the exact first repeat, Brent
stops once the chasing pointer catches its checkpoint, within a small multiple of the cycle length, so a
cyclic list can yield some of its nodes more than once before terminating. Termination is unconditional either
way, which is the property the iterators promise. `Walk` keeps its per-frame `seenChildren` set because it
must REPORT the cycle as `ErrWalkCycle`, not merely survive it. None of them uses a GLOBAL visited set, so a
shared DAG node reached on two different paths (e.g. `&e;&e;` — two references to one Entity) is still visited
on EACH occurrence; only same-path back-edges are cut. On an acyclic, parent-consistent tree behavior is
byte-identical to a naive descent — `Walk` returns nil, so its error return is a non-nil `ErrWalkCycle` ONLY
on a corrupt (cyclic) tree the guarded insertion API refuses to build. Production `Walk` callers with an
error/validity channel propagate a non-nil return (a cyclic document is not valid / a failed serialization
check); the few callers with no channel (a best-effort id lookup, a result-tree mutation over a freshly-built
acyclic tree) document why the error is safely ignored. `Walk` is the exported cycle-DETECTION primitive: only
it carries an error channel. The `iter.Seq` range-over-func iterators (`Children`, `ChildElements`,
`Descendants`) cannot report an error to the range loop, so on a detected cycle they terminate and yield the
PARTIAL set gathered up to that point with no signal; their doc comments point a caller who needs to detect
(not merely survive) a cycle at `Walk`/`ErrWalkCycle`. No separate validation helper exists or is needed — a
`Walk` with a no-op visitor already returns `ErrWalkCycle` on a corrupt tree.

Every internal traversal that a corrupt (cyclic) tree could otherwise stall on routes through a cycle-safe
scan, never a raw `NextSibling()` walk, so a hang cannot precede the `Walk`-based cycle guards in validation —
with one exception: `addSibling`'s fallback walk (above) is a raw, unbounded `NextSibling()` walk, so calling
`AddSibling` through a node whose sibling list is cyclic hangs forever. The document root-element scans
(`Document.DocumentElement`/`SetDocumentElement`/`CreateInternalSubset`) and the deep-copy (`copyChildren`,
`copyDTDChildren`) and serializer child descents iterate the SOURCE through `Children` (bounding a corrupt
source sibling list); `copyChildren` links each copied child into the destination via `appendCopiedChild`
(above), not `Children`; `setListDoc` (the `SetTreeDoc` sibling walker) and the serializer's attribute-chain
walk each carry a per-list seen guard (the latter also terminates a non-`*Attribute` successor that would
otherwise leave the cursor unadvanced). The element attribute-lookup hot paths
(`addProperty`/`HasAttribute`/`Attributes`/`ForEachAttribute`) traverse the `properties` chain with a plain
`NextAttribute` loop and NO guard: that chain is built exclusively through the guarded property-splice /
`AddSibling` paths (which reject self/cycle insertion and install no foreign link), so a well-formed chain is
a short, self-owned, acyclic list.

The auto-unlink and `replaceNode`'s splice operate through `unlinkNode`/`baseDocNode()` links, never
per-pointer setters. A non-`MutableNode` operand (e.g. a public `NamespaceNodeWrapper`, which embeds `docnode`
directly) is therefore detached and spliced safely: the preflights perform the unlink through those links
(leaving no stale old-parent links) and `replaceNode` splices without a `MutableNode` force-cast (which would
panic on such an operand). `setListDoc(Node, doc)` (the `SetTreeDoc` sibling walker) likewise accepts any
`Node`: a non-`MutableNode` sibling has its `doc` set directly via `baseDocNode()` instead of a `MutableNode`
force-cast, so `SetTreeDoc` over a tree containing a `NamespaceNodeWrapper` does not panic.

`addChild`/`addSibling`/`replaceNode` reject a nil or typed-nil operand (every replacement operand is checked)
with `ErrNilNode` BEFORE any `baseDocNode()` dereference, so the call returns an error and leaves the tree
untouched instead of panicking. The same typed-nil guard (`isNilNode`) covers the public read helpers so a
typed-nil node — notably the `*Element` `Document.DocumentElement()` returns for a rootless document, wrapped
in a non-nil `Node` interface — never panics: `AsNode` returns `(zero, false)` (never `(nil, true)`);
`Children`/`ChildElements`/`Descendants` yield nothing; `Walk`, `CopyNode`, and `Parser.ParseInNodeContext`
return `ErrNilNode`; `UnlinkNode` is a no-op.

The guarded mutation ops return matchable sentinels (`errors.Is`): a `wouldCreateCycle` rejection in
`addChild`/`addSibling`/`replaceNode` is `ErrCyclicNode` (wrapped via `%w`); an empty `Replace()` is
`ErrInvalidOperation` (matching `Document.Replace` — use `UnlinkNode` to delete a node); and the structural
rejections (a non-attribute sibling/replacement of a property attribute, duplicate replacement operands) wrap
`ErrInvalidOperation`. An `addChild` `*Attribute` operand on a non-element parent also wraps
`ErrInvalidOperation`. A leaf-node `AddChild` (`Text`, `Comment`, `CDATASection`, `ProcessingInstruction`)
that rejects a non-mergeable operand — anything but the same-kind content merge (`ProcessingInstruction`
merges a Text/CDATA operand into its `data`) — likewise wraps `ErrInvalidOperation` with a descriptive `cannot
add a <kind> as a child of a <kind> node` message, so the leaf rejections match `errors.Is` uniformly with the
shared-path ones. A leaf self-add (`Text`, `Comment`, `ProcessingInstruction`, `EntityRef`) is `ErrCyclicNode`
instead — `Text`/`Comment`/`EntityRef` catch it via the shared cycle guard, and `ProcessingInstruction`
detects it by identity (direct pointer comparison of the operand against the receiver — never dereferencing
the receiver, so a typed-nil PI receiver still gets the plain rejection error instead of a panic) before its
type rejection so `pi.AddChild(pi)` matches every other leaf self-add. An ancestor operand is NOT a self-add:
`pi.AddChild(parentElement)` (or the owning document) wraps `ErrInvalidOperation` with the shared shape,
matching the other strict leaves.

### Cross-Document Slab Safety

`Document` allocates its high-frequency nodes (`Element`, `Text`, `Namespace`, `Attribute`) and parsed
text-content bytes from per-document SLAB allocators backed by process-global `sync.Pool`s (`document.go`).
`Document.Free()` returns those chunks to the pools for reuse by a later parse. A node's struct and content
bytes therefore physically live in its owning document's slab, so recycling a chunk that a still-live node
references would let a subsequent parse overwrite that node.

The insertion paths (`addChildPreflight`/`addSiblingPreflight`/`replaceNode`, via `noteCrossDocumentEscape`)
permit linking a node into a DIFFERENT document than the one that owns it — XInclude merges an included
subtree into the main document, and xslt3 moves result-tree nodes into result documents — but they mark the
SOURCE document's `slabEscaped` flag when they do. `AddNamespaceDecl` and `SetNamespace` apply the same guard
for the namespace slab (via `noteCrossDocumentNamespaceEscape`): `AddNamespaceDecl` when it RETAINS the
caller's `*Namespace` (case 1 append, case 3 collapse), and `SetNamespace` when it installs the caller's
`*Namespace` as the node's active namespace (`CreateElementNS` reaches this path), mark the source document
escaped when that Namespace is slab-backed by a document other than the receiver's owner (`Namespace.context`
non-nil and != the node's doc), so a later `Free` cannot recycle the namespace chunk out from under the
retained reference. A same-document or heap-allocated Namespace, and `AddNamespaceDecl`'s non-retaining cases
(a same-URI no-op, a declined conflict), mark nothing. `SetActiveNamespace` creates its Namespace from the
node's own document, so it is always same-document and marks nothing. `Free` is a no-op on a document whose
`slabEscaped` is set: its chunks are not returned to the pool, and GC reclaims them once the moved nodes are
no longer referenced. A node with a nil owner (created with a nil `Document` receiver — heap-allocated, not
slab-backed) triggers no marking. A same-document reparent does not mark the flag, so the common
parse-then-`Free` fast path still recycles. Insertion does NOT rewrite the moved node's `OwnerDocument`; use
`CopyNode(src, targetDoc)` to import a subtree whose storage is owned by the destination document.
