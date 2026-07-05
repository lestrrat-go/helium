package xsd

import (
	"context"
	"sort"

	"github.com/lestrrat-go/helium/internal/lexicon"
)

// checkIDAttributeUses enforces the XSD 1.0 Schema Component Constraint that a
// complex type must not have more than one attribute use whose {type definition}
// is or is derived from xs:ID (Part 1 §3.4.6 / cvc "Attribute Declarations
// Consistent"). This is the COMPILE-TIME (schema-component) manifestation of the
// one-ID-per-element rule: it rejects such a type even when the ID-typed uses are
// optional and never both present in any instance — the instance-level cap in
// validate_id.go only catches two ID-typed attributes that actually CO-OCCUR.
//
// XSD 1.1 removed the constraint (multiple ID attributes are legal — W3C
// ctM004), so the check is gated to Version10.
//
// It counts the effective {attribute uses} recorded on each complex type
// (td.Attributes), which in XSD 1.0 already includes attributes contributed by
// xs:attributeGroup references and inherited through extension/restriction (the
// resolveRefs merge appends base attribute uses), so an inherited/extended ID
// attribute combined with a locally declared one is caught too.
//
// A wildcard-admitted ID (the "wild IDs" static rule) is a separate deferred gap.
func (c *compiler) checkIDAttributeUses(ctx context.Context) {
	if c.filename == "" || c.version != Version10 {
		return
	}

	type issue struct {
		source string
		line   int
		local  string
		msg    string
	}
	var issues []issue

	for td, src := range c.typeDefSources {
		if !td.IsComplex {
			continue
		}
		count := 0
		for _, au := range td.Attributes {
			if au == nil || au.Prohibited {
				continue
			}
			if c.attrUseIsIDType(au) {
				count++
			}
		}
		if count <= 1 {
			continue
		}
		issues = append(issues, issue{
			source: c.diagSourceOrRecorded(src.source),
			line:   src.line,
			local:  src.elemKind,
			msg:    "A complex type definition must not have more than one attribute declaration whose type is or is derived from ID.",
		})
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].source != issues[j].source {
			return issues[i].source < issues[j].source
		}
		return issues[i].line < issues[j].line
	})
	for _, is := range issues {
		c.schemaError(ctx, schemaParserError(is.source, is.line, is.local, elemComplexType, is.msg))
	}
}

// attrUseIsIDType reports whether an attribute use's effective type is the
// atomic built-in xs:ID or a type derived from it. A list of xs:ID or a union
// with an xs:ID member is NOT "derived from ID" (its variety is list/union), so
// it does not count toward this SCC — matching the literal §3.4.6 wording.
func (c *compiler) attrUseIsIDType(au *AttrUse) bool {
	td := au.Type
	if td == nil && au.TypeName != (QName{}) {
		td = c.resolveNamedType(au.TypeName)
	}
	if td == nil {
		return false
	}
	if resolveVariety(td) != TypeVarietyAtomic {
		return false
	}
	return builtinBaseLocal(td) == lexicon.TypeID
}
