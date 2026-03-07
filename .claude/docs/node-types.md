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
| PI | `ProcessingInstruction` | docnode | ✓ | data field | ✓ | target, data (Name() returns target) |
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

### DTD Map Keys
- Elements: `name:prefix`
- Attributes: `name:prefix:elem` (scoped to element)
- Entities: `name` (flat)

### Document ID Lookup
`Document.GetElementByID(id)` — O(1) via `ids map[string]*Element` (populated during parse). Falls back to O(n) tree walk if map empty.

### Content() Default
`docnode.Content()` walks children and concatenates. Overridden by Text, CDATA, Comment, PI, EntityRef (which return their own content directly).

### Predefined Entities
5 singletons: `EntityLT`, `EntityGT`, `EntityAmpersand`, `EntityApostrophe`, `EntityQuote`. Type `InternalPredefinedEntity`. Cannot be redeclared.

### NamespaceDeclNode Special Case
Skipped in `setTreeDoc()` — sentinel type rarely instantiated.

### Tree Operations
- `addChild(parent, child)` — append to end of children
- `addSibling(node, sibling)` — append to end of siblings
- `replaceNode(old, new)` — swap in same position
- `UnlinkNode(n)` — detach from parent and siblings
