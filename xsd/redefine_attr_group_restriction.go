package xsd

import (
	"context"
	"fmt"

	helium "github.com/lestrrat-go/helium"
)

func (c *compiler) checkRedefineAttrGroupRestriction(ctx context.Context, elem *helium.Element, qn QName, derivedAttrs []*AttrUse, derivedWildcard *Wildcard, baseAttrs []*AttrUse, baseWildcard *Wildcard) {
	if attrGroupValidRestriction(ctx, c, derivedAttrs, derivedWildcard, baseAttrs, baseWildcard) {
		return
	}
	msg := fmt.Sprintf("src-redefine.7.2: The redefinition of attributeGroup '%s' is not a valid restriction of the original attribute group.", qn.Local)
	c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemAttributeGroup, msg))
}

func attrGroupValidRestriction(ctx context.Context, c *compiler, derivedAttrs []*AttrUse, derivedWildcard *Wildcard, baseAttrs []*AttrUse, baseWildcard *Wildcard) bool {
	baseByName := make(map[QName]*AttrUse, len(baseAttrs))
	for _, au := range baseAttrs {
		if au == nil || au.Prohibited {
			continue
		}
		baseByName[au.Name] = au
	}

	derivedByName := make(map[QName]*AttrUse, len(derivedAttrs))
	for _, au := range derivedAttrs {
		if au == nil || au.Prohibited {
			continue
		}
		derivedByName[au.Name] = au
		baseAU := baseByName[au.Name]
		if baseAU == nil {
			if baseWildcard == nil || !wildcardAllowsExpandedName(baseWildcard, au.Name.Local, au.Name.NS, c.schema, true) {
				return false
			}
			continue
		}
		if !attrUseValidRestriction(ctx, c, au, baseAU) {
			return false
		}
	}

	for _, baseAU := range baseByName {
		if !baseAU.Required {
			continue
		}
		derivedAU := derivedByName[baseAU.Name]
		if derivedAU == nil || !derivedAU.Required {
			return false
		}
	}

	if derivedWildcard == nil {
		return true
	}
	if baseWildcard == nil {
		return false
	}
	if !wildcardConstraintSubset(derivedWildcard, baseWildcard, c.schema, true) {
		return false
	}
	return processContentsStrength(derivedWildcard.ProcessContents) >= processContentsStrength(baseWildcard.ProcessContents)
}

func attrUseValidRestriction(ctx context.Context, c *compiler, derived, base *AttrUse) bool {
	if base.Required && !derived.Required {
		return false
	}
	if c.version == Version11 && derived.Inheritable != base.Inheritable {
		return false
	}

	derivedTD := attrUseEffectiveTypeDef(derived, c.schema)
	baseTD := attrUseEffectiveTypeDef(base, c.schema)
	if derivedTD != nil && baseTD != nil && !simpleTypeValidlyRestricts(derivedTD, baseTD) {
		return false
	}

	if base.Fixed != nil {
		return derived.Fixed != nil &&
			fixedConstraintRestricts(ctx, *derived.Fixed, *base.Fixed, derivedTD, baseTD, derived.FixedNS, base.FixedNS, c.schema, c.version)
	}
	return true
}
