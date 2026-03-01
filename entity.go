package helium

import (
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"
)

type EntityType int

const (
	InternalGeneralEntity EntityType = iota + 1
	ExternalGeneralParsedEntity
	ExternalGeneralUnparsedEntity
	InternalParameterEntity
	ExternalParameterEntity
	InternalPredefinedEntity
)

// Entity represents an XML entity declaration (libxml2: xmlEntity).
type Entity struct {
	node
	orig       string     // content without substitution
	content    string     // content or ndata if unparsed
	entityType EntityType // the entity type
	externalID string     // external identifier for PUBLIC
	systemID   string     // URI for a SYSTEM or PUBLIC entity
	uri        string     // the full URI as computed
	// owner      bool       // does the entity own children
	checked      int   // was the entity content checked
	expanding    bool  // guard against recursive expansion (mirrors XML_ENT_EXPANDING)
	expandedSize int64 // total expanded byte count after recursive resolution
	/* this is also used to count entities
	 * references done from that entity
	 * and if it contains '<' */
}

var (
	EntityLT         = newEntity("lt", InternalPredefinedEntity, "", "", "<", "&lt;")
	EntityGT         = newEntity("gt", InternalPredefinedEntity, "", "", ">", "&gt;")
	EntityAmpersand  = newEntity("amp", InternalPredefinedEntity, "", "", "&", "&amp;")
	EntityApostrophe = newEntity("apos", InternalPredefinedEntity, "", "", "'", "&apos;")
	EntityQuote      = newEntity("quot", InternalPredefinedEntity, "", "", `"`, "&quot;")
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
// &#xHHH;) in s, returning the resolved string. Used to normalize entity
// content before comparing against predefined entity values.
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
		var r rune
		var ok bool
		if len(s) > 0 && s[0] == 'x' {
			// hex: &#xHHH;
			s = s[1:]
			semi := strings.IndexByte(s, ';')
			if semi < 0 {
				b.WriteString("&#x")
				continue
			}
			v, err := strconv.ParseInt(s[:semi], 16, 32)
			if err == nil && v > 0 && utf8.ValidRune(rune(v)) {
				r = rune(v)
				ok = true
			}
			s = s[semi+1:]
		} else {
			// decimal: &#NNN;
			semi := strings.IndexByte(s, ';')
			if semi < 0 {
				b.WriteString("&#")
				continue
			}
			v, err := strconv.ParseInt(s[:semi], 10, 32)
			if err == nil && v > 0 && utf8.ValidRune(rune(v)) {
				r = rune(v)
				ok = true
			}
			s = s[semi+1:]
		}
		if ok {
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
		return EntityLT, nil
	case "gt":
		return EntityGT, nil
	case "amp":
		return EntityAmpersand, nil
	case "apos":
		return EntityApostrophe, nil
	case "quot":
		return EntityQuote, nil
	default:
		return nil, errors.New("entity not found")
	}
}

func newEntity(name string, typ EntityType, publicID, systemID, notation, orig string) *Entity {
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

func (e *Entity) Checked() bool {
	return e.checked & 1 == 1
}

func (e *Entity) MarkChecked() {
	e.checked |= 1
}

func (e *Entity) SetOrig(s string) {
	e.orig = s
}

func (e *Entity) EntityType() int {
	return int(e.entityType)
}

func (e *Entity) ExternalID() string {
	return e.externalID
}

func (e *Entity) SystemID() string {
	return e.systemID
}

func (e *Entity) Content() []byte {
	return []byte(e.content)
}

func (e *Entity) AddChild(cur Node) error {
	return addChild(e, cur)
}

func (e *Entity) AddContent(b []byte) error {
	return addContent(e, b)
}

func (e *Entity) AddSibling(cur Node) error {
	return addSibling(e, cur)
}

func (e *Entity) Replace(cur Node) error {
	return replaceNode(e, cur)
}

func (n *Entity) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}
