package helium

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/enum"
)

// attrEntityWFC classifies whether a general entity's TRANSITIVE replacement
// text violates one of the XML 1.0 attribute-value well-formedness constraints:
// "No External Entity References" or "No < in Attribute Values".
type attrEntityWFC int

const (
	attrWFCNone     attrEntityWFC = iota // no violation
	attrWFCExternal                      // reaches an external parsed general entity
	attrWFCUnparsed                      // reaches an unparsed general entity
	attrWFCLessThan                      // replacement text contains a literal '<'
)

// Attribute-value WFC memoization flags, mirroring libxml2's
// XML_ENT_CHECKED / XML_ENT_VALIDATED (xmlCheckEntityInAttValue). They record
// that an entity's transitive replacement text has been walked for the
// attribute-value WFCs so a repeated reference — or a shared nested entity —
// skips the re-walk (and the getEntity callbacks it would emit). entWFCChecked
// is set only when the walk ran in a reliable (non-DTD-subset) context;
// entWFCValidated is set even inside the DTD subset, where a nested entity may
// still be forward-declared, so a later body reference re-checks with the
// entWFCChecked target.
const (
	entWFCValidated = 1 << iota // checked, possibly against an incomplete DTD subset
	entWFCChecked               // fully checked in a reliable (body) context
)

// Entity represents an XML entity declaration (libxml2: xmlEntity).
type Entity struct {
	node
	orig       string          // content without substitution
	content    string          // content or ndata if unparsed
	entityType enum.EntityType // the entity type
	externalID string          // external identifier for PUBLIC
	systemID   string          // URI for a SYSTEM or PUBLIC entity
	uri        string          // the full URI as computed
	// resolvedURI is the URI an external parameter entity's content was actually
	// loaded from (a catalog/custom-resolver URI or the entity's resolved system
	// URI), cached alongside `content` so a later reference parses the cached
	// bytes against the SAME base the first load used — not the declared URI().
	resolvedURI string
	// textDeclVersion is the effective XML version of cached external parameter
	// entity replacement text. It scopes version-dependent validation while the cached
	// input is parsed without changing the owning document's recorded version.
	textDeclVersion string
	// owner      bool       // does the entity own children
	checked      int   // was the entity content checked
	attrWFCFlags int   // attribute-value WFC memoization (entWFCValidated/entWFCChecked)
	expanding    bool  // guard against recursive expansion (mirrors XML_ENT_EXPANDING)
	expandedSize int64 // total expanded byte count after recursive resolution
	/* this is also used to count entities
	 * references done from that entity
	 * and if it contains '<' */
}

// Predefined XML entity singletons (XML §4.6). Internal: callers resolve
// these through the parser, never by name from outside the package.
var (
	entityLT         = newEntity("lt", enum.InternalPredefinedEntity, "", "", "<", "&lt;")
	entityGT         = newEntity("gt", enum.InternalPredefinedEntity, "", "", ">", "&gt;")
	entityAmpersand  = newEntity("amp", enum.InternalPredefinedEntity, "", "", "&", "&amp;")
	entityApostrophe = newEntity("apos", enum.InternalPredefinedEntity, "", "", "'", "&apos;")
	entityQuote      = newEntity("quot", enum.InternalPredefinedEntity, "", "", `"`, "&quot;")
)

// predefinedEntityContent maps predefined entity names to their required
// content per XML §4.6. Used by DTD.AddEntity to reject invalid redeclarations.
var predefinedEntityContent = map[string]string{
	"lt":   "<",
	"gt":   ">",
	"amp":  "&",
	"apos": "'",
	"quot": `"`,
}

// resolveCharRefs resolves all numeric character references (&#NNN; and
// &#xHHH;) in s, returning the resolved string. It shares parseCharRefBody with
// writer output so lowercase-x lexical validation cannot diverge. Used to
// normalize entity content before comparing against predefined entity values.
func resolveCharRefs(s string) string {
	if !strings.Contains(s, "&#") {
		return s
	}
	var b strings.Builder
	for len(s) > 0 {
		idx := strings.Index(s, "&#")
		if idx < 0 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:idx])
		s = s[idx+2:] // skip "&#"
		semi := strings.IndexByte(s, ';')
		if semi < 0 {
			b.WriteString("&#")
			continue
		}
		r, ok := parseCharRefBody(s[:semi])
		s = s[semi+1:]
		if ok && r > 0 && utf8.ValidRune(r) {
			b.WriteRune(r)
		} else {
			b.WriteString("&#") // malformed ref, keep literal
		}
	}
	return b.String()
}

func resolvePredefinedEntity(name string) (*Entity, error) {
	switch name {
	case "lt":
		return entityLT, nil
	case "gt":
		return entityGT, nil
	case "amp":
		return entityAmpersand, nil
	case "apos":
		return entityApostrophe, nil
	case "quot":
		return entityQuote, nil
	default:
		return nil, errors.New("entity not found")
	}
}

func newEntity(name string, typ enum.EntityType, publicID, systemID, notation, orig string) *Entity {
	e := &Entity{
		content:    notation,
		entityType: typ,
		externalID: publicID,
		systemID:   systemID,
		orig:       orig,
	}
	e.etype = EntityNode
	e.name = name
	return e
}

// Checked reports whether this entity's content has been parsed and validated,
// used to prevent infinite recursion during entity expansion (libxml2: ent->checked).
func (e *Entity) Checked() bool {
	return e.checked&1 == 1
}

// MarkChecked marks this entity as having been parsed and validated (libxml2: ent->checked).
func (e *Entity) MarkChecked() {
	e.checked |= 1
}

func (e *Entity) SetOrig(s string) {
	e.orig = s
}

func (e *Entity) EntityType() enum.EntityType {
	return e.entityType
}

func (e *Entity) ExternalID() string {
	return e.externalID
}

func (e *Entity) SystemID() string {
	return e.systemID
}

// URI returns the fully resolved URI for external entities.
// Falls back to SystemID if no resolved URI is available.
func (e *Entity) URI() string {
	if e.uri != "" {
		return e.uri
	}
	return e.systemID
}

func (e *Entity) Content() []byte {
	return []byte(e.content)
}

func (e *Entity) AddChild(cur Node) error {
	return addChild(e, cur)
}

func (e *Entity) AppendText(b []byte) error {
	return appendText(e, b)
}

func (e *Entity) AddSibling(cur Node) error {
	return addSibling(e, cur)
}

func (e *Entity) Replace(nodes ...Node) error {
	return replaceNode(e, nodes...)
}

func (n *Entity) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}
