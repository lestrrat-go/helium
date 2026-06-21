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
| Attribute | `Attribute` | docnode | ✓ (text/entityref for value) | via children | ✓ (linked list) | ns, atype, defaultAttr |
| Text | `Text` | node | ✗ (merges) | ✓ content | ✓ | Adjacent text nodes auto-merge |
| CDATASection | `CDATASection` | node | ✗ | ✓ content | ✓ | — |
| Comment | `Comment` | node | ✗ | ✓ content | ✓ | — |
| PI | `ProcessingInstruction` | docnode | ✗ | data field | ✓ | target, data (Name() returns target). AddChild/AppendText route text into `data`; non-text children rejected |
| EntityRef | `EntityRef` | node | ✓ (if expanded) | ✓ (if resolved) | ✓ | References Entity by name |
| Entity | `Entity` | node | ✓ (parsed) | ✓ content | ✓ | entityType, externalID, systemID, uri, checked, expanding, expandedSize |
| DTD | `DTD` | docnode | ✓ (decls) | — | ✓ | attributes/elements/entities/pentities/notations maps, externalID, systemID |
| ElementDecl | `ElementDecl` | docnode | — | — | ✓ | decltype, content (grammar tree), attributes, prefix |
| AttributeDecl | `AttributeDecl` | docnode | — | — | ✓ | atype, def, defvalue, tree (enumeration), prefix, elem |
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
`Document.GetElementByID(id)` — `idsSkip` is authoritative and checked FIRST: when `SkipIDs()` is true it returns nil immediately, resolving NO ids regardless of the `ids` table or DTD subsets. Otherwise it is O(1) via `ids map[string]*Element` (populated during parse), falling back to an O(n) tree walk if the map is empty. The fallback walk consults ID-typed attribute declarations in BOTH the internal and external DTD subsets (`intSubset`, `extSubset`). `Document.SkipIDs()`/`SetSkipIDs(bool)` read/write the `idsSkip` flag (set from parser `SkipIDs(true)`); because it is checked before the table, `SetSkipIDs(true)` suppresses resolution even on a doc with a populated ID table, and `SetSkipIDs(false)` restores it. Carry the flag onto derived docs so id() semantics match the source. A derived doc that drops the ID table (e.g. an xsl:strip-space copy) relies on the fallback walk, so it must also carry over `extSubset` — `helium.CopyExtSubset(src, dst *Document)` DEEP-COPIES the source's external subset into `dst` (independent `*DTD`; mutating one never affects the other), unlike `CopyDTDInfo`/`CopyDoc` which copy only the internal subset and link it into the document tree.

### Content() Default
`docnode.Content()` walks children and concatenates (returns a fresh buffer). Overridden by Text, CDATA, Comment, PI, EntityRef.

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

All three insertion paths share `wouldCreateCycle(parent, cur)`: they reject inserting a node into itself or into one of its own descendants (which would put an ancestor below itself). addChild/addSibling auto-unlink an already-linked incoming node before relinking so it never lives in two places; rejection leaves the tree untouched. The shared guard + auto-unlink is factored into `addChildPreflight`/`addSiblingPreflight`. Leaf `AddChild`/`AddSibling` overrides that take a content-merge fast path (Text, Comment, and ProcessingInstruction — whose `AddChild` merges a Text/CDATA operand's content into the PI data string) run the matching preflight BEFORE merging, so `txt.AddChild(txt)`/`comment.AddChild(comment)` are rejected instead of doubling content, and an already-linked incoming leaf (including a Text/CDATA node merged into a PI) is unlinked from its old parent before its content is merged. These overrides also reject a nil or typed-nil operand with `ErrNilNode` before any method call on the operand, since a typed nil reaching the type switch / merge path would panic.

The auto-unlink and `replaceNode`'s splice operate through `unlinkNode`/`baseDocNode()` links rather than `MutableNode` setters. A non-`MutableNode` operand (e.g. a public `NamespaceNodeWrapper`, which embeds `docnode` directly) is therefore detached and spliced safely: the preflights no longer silently skip the unlink (which left stale old-parent links) and `replaceNode` no longer force-casts to `MutableNode` (which could panic). `setListDoc(Node, doc)` (the `SetTreeDoc` sibling walker) likewise accepts any `Node`: a non-`MutableNode` sibling has its `doc` set directly via `baseDocNode()` instead of a `MutableNode` force-cast, so `SetTreeDoc` over a tree containing a `NamespaceNodeWrapper` does not panic.

`addChild`/`addSibling`/`replaceNode` reject a nil or typed-nil operand (every replacement operand is checked) with `ErrNilNode` BEFORE any `baseDocNode()` dereference, so the call returns an error and leaves the tree untouched instead of panicking.
