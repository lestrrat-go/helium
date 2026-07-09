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
| Element | `Element` | node | ✓ | via children | ✓ | properties (Attribute linked list), ns, nsDefs |
| Attribute | `Attribute` | docnode | ✓ (text/entityref for value) | via children | ✓ (linked list) | ns, atype, defaultAttr, syntheticBase (parser-injected external-entity xml:base) |
| Text | `Text` | node | ✗ (merges) | ✓ content | ✓ | Adjacent text nodes auto-merge |
| CDATASection | `CDATASection` | node | ✗ | ✓ content | ✓ | — |
| Comment | `Comment` | node | ✗ | ✓ content | ✓ | — |
| PI | `ProcessingInstruction` | docnode | ✗ | data field | ✓ | target, data (Name() returns target). AddChild/AppendText route text into `data`; non-text children rejected |
| EntityRef | `EntityRef` | node | ✓ (if expanded) | ✓ (if resolved) | ✓ | References Entity by name |
| Entity | `Entity` | node | ✓ (parsed) | ✓ content | ✓ | entityType, externalID, systemID, uri, checked, expanding, expandedSize |
| DTD | `DTD` | docnode | ✓ (decls) | — | ✓ | attributes/elements/entities/pentities/notations maps, externalID, systemID |
| ElementDecl | `ElementDecl` | docnode | — | — | ✓ | decltype, content (grammar tree), attributes, prefix |
| AttributeDecl | `AttributeDecl` | docnode | — | — | ✓ | atype, def, defvalue, tree (enumeration), prefix, elem, external (declared in external subset/PE) |
| Notation | `Notation` | docnode | — | — | ✓ | publicID, systemID |
| Namespace | `Namespace` | — | ✗ | ✗ | ✗ | href, prefix, context (lightweight, no tree linkage) |
| NamespaceNodeWrapper | `NamespaceNodeWrapper` | docnode | ✗ | ns.URI() | ✗ | ns pointer (XPath only, read-only) |

## Key Behaviors

### Text Node Consolidation
`Text.AddSibling(Text)` → content merged instead of creating sibling. Prevents whitespace node bloat. Mirrors libxml2 TEXT consolidation.

### PI Content Is A String, Not Children
A `ProcessingInstruction` stores its content in the `data` string field (mirrors libxml2's XML_PI_NODE, whose content is the node's content string). It has NO element/text children. `AppendText` and an `AddChild` of a Text/CDATA node append the text to `data`; `AddChild` of any other node type is rejected (so the tree cannot be corrupted and serialization stays `<?target data?>`). The serializer reads `pi.data` directly.

### DTD Map Keys
- Elements: `name:prefix`
- Attributes: `name:prefix:elem` (scoped to element)
- Entities: `name` (flat)

### Document ID Lookup
`Document.GetElementByID(id)` — `idsSkip` is authoritative and checked FIRST: when `SkipIDs()` is true it returns nil immediately, resolving NO ids regardless of the `ids` table or DTD subsets. Otherwise it is O(1) via `ids map[string]*Element` (populated during parse), falling back to an O(n) tree walk if the map is empty. The fallback walk consults ID-typed attribute declarations in BOTH the internal and external DTD subsets (`intSubset`, `extSubset`). `Document.SkipIDs()`/`SetSkipIDs(bool)` read/write the `idsSkip` flag (set from parser `SkipIDs(true)`); because it is checked before the table, `SetSkipIDs(true)` suppresses resolution even on a doc with a populated ID table, and `SetSkipIDs(false)` restores it. Carry the flag onto derived docs so id() semantics match the source. `Document.IDTable()` returns the document's own `ids` map (read-only; nil for API-built docs without an interned table) so a derived doc can rebuild an equivalent table by translating each entry's element through an original->copy correspondence. The xsl:strip-space copy (`xslt3` copyAndStrip) does exactly this: it rebuilds the copy's ID table from `src.IDTable()`, which preserves the source's interned ID-table identity — the copy resolves ids to elements corresponding exactly to the source's (and at O(1)) rather than re-deriving them through the lazy O(n) fallback walk. The fallback walk consults ID-typed ATTLIST declarations by their raw qualified name (prefix+local), so it correctly resolves a prefixed element's qualified ATTLIST (`<!ATTLIST a:item eid ID>`) — but rebuilding from the source table is still preferred for identity and cost fidelity. The copy still carries over `extSubset` for the API-built/no-interned-table case, where both source and copy fall back to the walk: `helium.CopyExtSubset(src, dst *Document)` DEEP-COPIES the source's external subset into `dst` (independent `*DTD`; mutating one never affects the other), unlike `CopyDTDInfo`/`CopyDoc` which copy only the internal subset and link it into the document tree.

### Document Encoding (synthesized vs raw)
`Document.Encoding()` synthesizes `"utf8"` when the `encoding` field is empty (source omitted an encoding declaration); `Document.RawEncoding()` returns the field verbatim (empty = no declaration). The XML serializer (`writer.go`) emits `encoding="..."` ONLY when the raw encoding is non-empty, so a faithful document copy must propagate `RawEncoding()` — using `Encoding()` would make the copy serialize a spurious `encoding="utf8"` the source never had. `CopyDoc` reads the raw field directly (same package); cross-package copiers (e.g. the xsl:strip-space copy in `xslt3`) use `RawEncoding()`. `Version()` and `Standalone()` already return raw, unsynthesized values.

### Content() Default
`docnode.Content()` walks children and concatenates (returns a fresh buffer). Overridden by Text, CDATA, Comment, PI, EntityRef. It has a POINTER receiver (`*docnode`) so the receiver is the real owning node — every `Node` is a pointer (the sealed `baseDocNode()` interface method is itself pointer-receiver), so this changes nothing for callers. The aggregation runs through the private `aggregateOwnedContent` helper, which advances between children with the OWNED-BOUNDARY rule (`nextOwnedChild`): a foreign child — an entity reference's shared Entity child, owned by the DTD, whose sibling pointers belong to the DTD declaration list — ends the aggregation instead of spilling into another list's siblings, and a per-list seen set terminates a cyclic sibling pointer. The recursion into a container child's subtree carries an ACTIVE-PATH set (the container docnodes currently being aggregated, receiver inclusive): a child already on that path is a back-edge and is skipped, so a pure child-pointer cycle (`element -> element -> element`, NOT routed through an Entity's terminating stored-text `Content()`) terminates instead of recursing forever. A leaf child (Text/Comment/CDATA/PI/Entity/NamespaceNodeWrapper — `aggregatesOwnContent` returns false) is self-contained and called directly; every other node type recurses under the guard. The active-path set is not a global visited set, so a shared DAG node reached on a different path is re-aggregated per occurrence.

The text-bearing leaves (Text, Comment, CDATASection) store content in an internal mutable `content []byte`. Their exported `Content()` returns a **defensive copy** (`bytes.Clone`) so a caller mutating the result cannot corrupt the DOM. Internal read-only hot paths (serializers in `writer.go`/`writer_xhtml.go`) use the package-level `rawContent(Node)` helper — backed by an unexported `rawContent()` method on each of those three leaf types — to get the raw slice without the copy. The `rawContentNode` interface gates the no-copy path; for any other node `rawContent` falls back to `Content()`. PI/EntityRef/Entity/NamespaceNodeWrapper already returned string-derived copies and are unaffected.

### Predefined Entities
5 singletons: `EntityLT`, `EntityGT`, `EntityAmpersand`, `EntityApostrophe`, `EntityQuote`. Type `InternalPredefinedEntity`. Cannot be redeclared.

### NamespaceDeclNode Special Case
Skipped in `setTreeDoc()` — sentinel type rarely instantiated.

### Tree Operations
- `addChild(parent, child)` — append to end of children
- `addSibling(node, sibling)` — append to end of siblings
- `replaceNode(old, new)` — swap in same position. Attribute-aware: replacing an `Attribute` updates the owning `Element.properties` head/chain (NOT `firstChild`/`lastChild`), and an attribute may only be replaced by attribute node(s) (non-attribute replacement is rejected)
- `AppendChildFast(parent, child)` — exported no-preflight append (wraps internal `appendFastChild`): links child as last child WITHOUT the cycle-guard / duplicate-attr checks. Only for freshly-built, provably-acyclic, dup-free trees (deep copies). Prefer `AddChild` otherwise.
- `AddNamespaceDecl(ns)` — append an EXISTING `*Namespace` to a node's `nsDefs` without allocating (unlike `DeclareNamespace`, which creates a fresh one). Lets a tree-builder reuse one Namespace object as both a declaration and an element's active ns.
- `UnlinkNode(n)` — detach a `MutableNode` from parent and siblings (delegates to the internal `unlinkNode(Node)`)
- `unlinkNode(n)` — internal detach that works for ANY sealed node via `baseDocNode()`, including non-`MutableNode` nodes like `NamespaceNodeWrapper`. Attribute-aware: an `Attribute` under an `*Element` is detached via `spliceOutAttribute`, repairing `Element.properties`

All three insertion paths share `wouldCreateCycle(parent, cur)`: they reject inserting a node into itself or into one of its own descendants (which would put an ancestor below itself). The guard has TWO parts: an ANCESTOR walk (parent's parent chain, inclusive, looking for cur) that catches every cycle at O(depth) when parent/child links are consistent; and, ONLY when cur has children, a CHILD-pointer reachability search (`childReaches`, an ITERATIVE stack-based DFS with a per-call visited set) that catches a cycle formed through a FOREIGN child link. A foreign link is a child whose own parent pointer points elsewhere — an entity reference's child is the shared Entity node, whose parent stays the DTD — so `ent.AddChild(ref)` where ref's child is ent forms a child-pointer cycle the ancestor walk cannot see. `childReaches` enumerates each node's OWN children via `nextOwnedChild` (a foreign child's `next` belongs to another list, so it is not followed as a sibling) and descends fully (it does NOT skip foreign subtrees, since a live insertion parent CAN legitimately lie inside a shared entity expansion). It is iterative (no goroutine-stack overflow on a deep tree) and has NO depth cap — the popped-node visited set alone bounds it (each node visited once), so it is SOUND at any depth: a depth cap would fail OPEN and admit a cycle deeper than the cap. A separate per-list sibling-seen set bounds the INNER sibling enumeration (the popped-node visited set is consulted only on pop, not while walking siblings), so a cyclic sibling pointer — `child.next == child` or a longer sibling loop — terminates instead of spinning. The childless-cur fast path (the parser hot path appends fresh leaves) skips the search entirely, so plain parsing is unaffected; it only runs for a subtree/entity-ref insertion. addChild/addSibling auto-unlink an already-linked incoming node before relinking so it never lives in two places; rejection leaves the tree untouched. The shared guard + auto-unlink is factored into `addChildPreflight`/`addSiblingPreflight`. Leaf `AddChild`/`AddSibling` overrides that take a content-merge fast path (Text, Comment, and ProcessingInstruction — whose `AddChild` merges a Text/CDATA operand's content into the PI data string) run the matching preflight BEFORE merging, so `txt.AddChild(txt)`/`comment.AddChild(comment)` are rejected instead of doubling content, and an already-linked incoming leaf (including a Text/CDATA node merged into a PI) is unlinked from its old parent before its content is merged. These overrides also reject a nil or typed-nil operand with `ErrNilNode` before any method call on the operand, since a typed nil reaching the type switch / merge path would panic.

The read-only traversal APIs (`Walk`, `Children`, `ChildElements`, `Descendants`, and the aggregating `Content()`) share the same two safety properties, so a hand-built or foreign-linked graph cannot make them wander or loop. (1) OWNED BOUNDARY: they advance between siblings via `nextOwnedChild(owner, child)` (returns `child.NextSibling()` only when `child.Parent()` is `owner`), so a foreign child — an entity reference's shared Entity child, owned by the DTD — ends that child list instead of spilling into the DTD's unrelated declaration siblings. (2) CYCLE SAFETY: `Walk` carries the set of nodes currently ON the DFS stack (the active path, O(depth)) and returns the `ErrWalkCycle` sentinel when it would descend into a node already on that path (a child-pointer back-edge, e.g. an Entity whose child links back to its reference); it also carries a PER-FRAME `seenChildren` set that returns `ErrWalkCycle` when a child repeats within one sibling list — this covers BOTH a one-node self-loop (`child.next == child`) and a longer sibling cycle (`a -> b -> a`), which the active-path set misses because each child is popped before its next sibling is examined. `nextWalkSibling` does NOT special-case the self-loop: it lets the duplicate flow back so `seenChildren` reports `ErrWalkCycle`, rather than silently terminating and reporting a corrupt one-node cycle as fully traversed. `Descendants` mirrors this with an active-path set threaded through its recursion (visits a back-edge node once, does not descend through it) plus a per-list sibling-seen set; `Children`/`ChildElements` use a per-list sibling-seen set. None of them uses a GLOBAL visited set, so a shared DAG node reached on two different paths (e.g. `&e;&e;` — two references to one Entity) is still visited on EACH occurrence; only same-path back-edges are cut. On an acyclic, parent-consistent tree behavior is byte-identical to a naive descent — `Walk` returns nil, so its error return is a non-nil `ErrWalkCycle` ONLY on a corrupt (cyclic) tree the guarded insertion API refuses to build. Production `Walk` callers with an error/validity channel propagate a non-nil return (a cyclic document is not valid / a failed serialization check); the few callers with no channel (a best-effort id lookup, a result-tree mutation over a freshly-built acyclic tree) document why the error is safely ignored.

Every internal traversal that a corrupt (cyclic) tree could otherwise stall on routes through a cycle-safe scan rather than a raw `NextSibling()` walk, so a hang cannot precede the `Walk`-based cycle guards in validation. The document root-element scans (`Document.DocumentElement`/`SetDocumentElement`/`CreateInternalSubset`) and the deep-copy (`copyChildren`, `copyDTDChildren`) and serializer child descents iterate through `Children`; `setListDoc` (the `SetTreeDoc` sibling walker) and the serializer's attribute-chain walk each carry a per-list seen guard (the latter also terminates a non-`*Attribute` successor that would otherwise leave the cursor unadvanced). The element attribute-lookup hot paths (`addProperty`/`HasAttribute`/`Attributes`/`ForEachAttribute`) traverse the `properties` chain with a plain `NextAttribute` loop and NO guard: that chain is built exclusively through the guarded property-splice / `AddSibling` paths (which reject self/cycle insertion and install no foreign link), so a well-formed chain is a short, self-owned, acyclic list.

The auto-unlink and `replaceNode`'s splice operate through `unlinkNode`/`baseDocNode()` links rather than `MutableNode` setters. A non-`MutableNode` operand (e.g. a public `NamespaceNodeWrapper`, which embeds `docnode` directly) is therefore detached and spliced safely: the preflights perform the unlink through those links (leaving no stale old-parent links) and `replaceNode` splices without a `MutableNode` force-cast (which would panic on such an operand). `setListDoc(Node, doc)` (the `SetTreeDoc` sibling walker) likewise accepts any `Node`: a non-`MutableNode` sibling has its `doc` set directly via `baseDocNode()` instead of a `MutableNode` force-cast, so `SetTreeDoc` over a tree containing a `NamespaceNodeWrapper` does not panic.

`addChild`/`addSibling`/`replaceNode` reject a nil or typed-nil operand (every replacement operand is checked) with `ErrNilNode` BEFORE any `baseDocNode()` dereference, so the call returns an error and leaves the tree untouched instead of panicking.
