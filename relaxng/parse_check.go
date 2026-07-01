package relaxng

import (
	"context"
	"fmt"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/xsd/value"
	"github.com/lestrrat-go/helium/internal/xsdregex"
)

// ruleFlags tracks ancestor context for forbidden-pattern-nesting checks.
type ruleFlags int

const (
	inAttribute     ruleFlags = 1 << iota
	inOneOrMore               // inside oneOrMore or zeroOrMore
	inList                    // inside list
	inDataExcept              // inside data/except
	inStart                   // top-level start pattern
	inOOMGroup                // inside oneOrMore//group
	inOOMInterleave           // inside oneOrMore//interleave
)

// visitKey identifies a define visit by both the define pattern and the
// ancestor flag context it was reached under. Forbidden-nesting checks depend on
// ruleFlags, so the same define reached under a different context (e.g. once in
// a normal position and once under <list>) must be checked again rather than
// being suppressed by a pattern-only cache.
type visitKey struct {
	pat   *pattern
	flags ruleFlags
}

// checkRules walks the compiled pattern tree and reports forbidden nesting
// errors (e.g. list//element, attribute//attribute) and warnings.
func (c *compiler) checkRules(ctx context.Context) {
	if c.grammar.start == nil {
		return
	}
	visited := make(map[visitKey]int8) // 0=unseen, 1=in-progress, 2=done
	c.checkPattern(ctx, c.grammar.start, inStart, visited)
}

// checkPattern recursively checks a pattern node for forbidden nestings,
// then recurses into children with updated flags. Refs are followed via their
// compile-time-resolved scoped target; the visited set is keyed by {define
// pattern, flag context} so distinct same-named scopes are tracked
// independently and a define reached under a new flag context is re-checked.
func (c *compiler) checkPattern(ctx context.Context, pat *pattern, flags ruleFlags, visited map[visitKey]int8) {
	if pat == nil {
		return
	}

	switch pat.kind {
	case patternElement:
		if flags&inList != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern list//element")
		}
		if flags&inAttribute != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern attribute//element")
		}
		if flags&inDataExcept != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern data/except//element")
		}
		// Element resets context: recurse attrs and children with 0.
		// (Attribute patterns in pat.attrs set inAttribute on their own children.)
		for _, attr := range pat.attrs {
			c.checkPattern(ctx, attr, 0, visited)
		}
		for _, child := range pat.children {
			c.checkPattern(ctx, child, 0, visited)
		}
		return

	case patternAttribute:
		if flags&inAttribute != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern attribute//attribute")
		}
		if flags&inList != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern list//attribute")
		}
		if flags&inStart != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern start//attribute")
		}
		if flags&inDataExcept != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern data/except//attribute")
		}
		if flags&inOOMGroup != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern oneOrMore//group//attribute")
		}
		if flags&inOOMInterleave != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern oneOrMore//interleave//attribute")
		}
		// Warnings for anyName/nsName without oneOrMore ancestor.
		if pat.nameClass != nil && flags&inOneOrMore == 0 {
			if pat.nameClass.kind == ncAnyName {
				c.addPatternWarning(ctx, pat, "Found anyName attribute without oneOrMore ancestor")
			}
			if pat.nameClass.kind == ncNsName {
				c.addPatternWarning(ctx, pat, "Found nsName attribute without oneOrMore ancestor")
			}
		}
		childFlags := flags | inAttribute
		for _, child := range pat.children {
			c.checkPattern(ctx, child, childFlags, visited)
		}
		return

	case patternList:
		if flags&inList != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern list//list")
		}
		if flags&inDataExcept != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern data/except//list")
		}
		childFlags := flags | inList
		for _, child := range pat.children {
			c.checkPattern(ctx, child, childFlags, visited)
		}
		return

	case patternText:
		if flags&inList != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern list//text")
		}
		if flags&inDataExcept != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern data/except//text")
		}
		return

	case patternInterleave:
		if flags&inList != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern list//interleave")
		}
		if flags&inDataExcept != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern data/except//interleave")
		}
		childFlags := flags
		if flags&inOneOrMore != 0 {
			childFlags |= inOOMInterleave
		}
		for _, child := range pat.children {
			c.checkPattern(ctx, child, childFlags, visited)
		}
		return

	case patternOneOrMore, patternZeroOrMore:
		childFlags := flags | inOneOrMore
		for _, child := range pat.children {
			c.checkPattern(ctx, child, childFlags, visited)
		}
		return

	case patternGroup:
		childFlags := flags
		if flags&inOneOrMore != 0 {
			childFlags |= inOOMGroup
		}
		for _, child := range pat.children {
			c.checkPattern(ctx, child, childFlags, visited)
		}
		return

	case patternData:
		c.checkDataFacets(ctx, pat)
		// Children of data are except patterns.
		childFlags := flags | inDataExcept
		for _, child := range pat.children {
			c.checkPattern(ctx, child, childFlags, visited)
		}
		return

	case patternRef, patternParentRef:
		if flags&inDataExcept != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern data/except//ref")
		}
		def := pat.resolved
		if def == nil {
			return
		}
		key := visitKey{pat: def, flags: flags}
		state := visited[key]
		if state != 0 {
			return // in-progress or done in this flag context
		}
		visited[key] = 1 // in-progress
		c.checkPattern(ctx, def, flags, visited)
		visited[key] = 2 // done
		return
	}

	// Default: choice, optional, mixed — pass flags through.
	for _, child := range pat.children {
		c.checkPattern(ctx, child, flags, visited)
	}
}

// checkDataFacets validates, at compile time, the XSD <param> facets carried by a
// <data> pattern, failing closed on a facet that cannot apply to the datatype:
//
//   - an ordering facet (min/maxInclusive, min/maxExclusive) on a datatype whose
//     value space is not ordered (value.Orderable) — value.Compare returns a
//     deterministic order for boolean and the binary types so enumeration can use
//     it, but that order must never fire a range facet — or whose bound is not a
//     valid value of the datatype.
//   - a digit facet (totalDigits, fractionDigits) on a datatype outside the
//     xs:decimal family (value.IsDecimalFamily).
//
// These mirror the XSD facet-applicability rules (internal/xsd/value) so RELAX NG
// and XSD agree on which facets a datatype admits. An inapplicable facet is a
// fatal compile error, which makes the whole grammar unmatchable (compileSchema).
func (c *compiler) checkDataFacets(ctx context.Context, pat *pattern) {
	typeName, ok := effectiveXSDDatatype(pat.dataType)
	if !ok {
		return
	}
	for _, p := range pat.params {
		switch p.name {
		case "pattern":
			// The pattern facet is an XSD/XPath regular expression. Compile it once
			// with the shared XSD-regex engine (xsdregex) so XSD-only constructs (\i,
			// \c, \p{...}, character-class subtraction, …) are honoured and an invalid
			// pattern is a fatal schema error rather than a silent runtime no-op or a
			// false rejection. The compilation is cached on the param for validation.
			if p.patternChecked {
				continue
			}
			p.patternChecked = true
			re, err := xsdregex.Compile(p.value)
			if err != nil {
				c.addPatternError(ctx, pat, fmt.Sprintf("value '%s' for facet 'pattern' is not a valid regular expression", p.value))
				continue
			}
			p.compiledPattern = re
		case "length", "minLength", "maxLength":
			// Length facets apply only to string-derived, binary, anyURI, QName and
			// NOTATION datatypes (value.LengthApplicable). Applying one to a numeric,
			// boolean or date/time datatype is a schema error, mirroring XSD's
			// facet-applicability rules so RELAX NG and XSD agree.
			if !value.LengthApplicable(typeName) {
				c.addPatternError(ctx, pat, fmt.Sprintf("facet '%s' is not allowed on the datatype '%s'", p.name, typeName))
				continue
			}
			// The bound must itself be a valid xs:nonNegativeInteger. Validate here
			// with XSD whitespace/lexical rules (value.Normalize collapses only XSD
			// whitespace — NOT Go's TrimSpace, which would accept NBSP — and
			// value.ValidateBuiltin enforces the digit lexical and the >=0 range), so
			// an out-of-space bound (negative, fractional, non-digit, NBSP-padded)
			// is a fatal compile error rather than being parsed leniently at
			// validation time and turning the facet into a no-op or reject-all.
			if value.ValidateBuiltin(value.Normalize(p.value, "nonNegativeInteger"), "nonNegativeInteger", value.Version10) != nil {
				c.addPatternError(ctx, pat, fmt.Sprintf("value '%s' for facet '%s' is not a valid 'nonNegativeInteger'", p.value, p.name))
			}
		case "minInclusive", "maxInclusive", "minExclusive", "maxExclusive":
			if !value.Orderable(typeName) {
				c.addPatternError(ctx, pat, fmt.Sprintf("facet '%s' is not allowed on the non-ordered datatype '%s'", p.name, typeName))
				continue
			}
			if value.ValidateBuiltin(value.Normalize(p.value, typeName), typeName, value.Version10) != nil {
				c.addPatternError(ctx, pat, fmt.Sprintf("value '%s' for facet '%s' is not a valid '%s'", p.value, p.name, typeName))
			}
		case "totalDigits", "fractionDigits":
			if !value.IsDecimalFamily(typeName) {
				c.addPatternError(ctx, pat, fmt.Sprintf("facet '%s' is not allowed on the datatype '%s'", p.name, typeName))
				continue
			}
			// The digit-facet bound must itself be a valid integer in its XSD value
			// space: totalDigits is an xs:positiveInteger, fractionDigits an
			// xs:nonNegativeInteger. Validating here (with XSD whitespace/lexical
			// rules — so e.g. an NBSP-padded bound is rejected, not silently trimmed)
			// keeps an out-of-space bound from being parsed leniently at validation.
			boundType := "positiveInteger"
			if p.name == "fractionDigits" {
				boundType = "nonNegativeInteger"
			}
			if value.ValidateBuiltin(value.Normalize(p.value, boundType), boundType, value.Version10) != nil {
				c.addPatternError(ctx, pat, fmt.Sprintf("value '%s' for facet '%s' is not a valid '%s'", p.value, p.name, boundType))
			}
		}
	}
}

// addPatternError records a forbidden-nesting error.
func (c *compiler) addPatternError(ctx context.Context, p *pattern, msg string) {
	elemName := patternElemName(p.kind)
	var formatted string
	if c.filename != "" {
		formatted = rngParserErrorAt(c.filename, p.line, elemName, msg)
	} else {
		formatted = rngParserError(msg)
	}
	c.errorHandler.Handle(ctx, helium.NewLeveledError(formatted, helium.ErrorLevelFatal))
	c.errorCount++
}

// addPatternWarning records a forbidden-nesting warning.
func (c *compiler) addPatternWarning(ctx context.Context, p *pattern, msg string) {
	elemName := patternElemName(p.kind)
	var formatted string
	if c.filename != "" {
		formatted = fmt.Sprintf("%s:%d: element %s: Relax-NG parser warning : %s\n",
			c.filename, p.line, elemName, msg)
	} else {
		formatted = fmt.Sprintf("Relax-NG parser warning : %s\n", msg)
	}
	c.errorHandler.Handle(ctx, helium.NewLeveledError(formatted, helium.ErrorLevelWarning))
}

// patternElemName returns the XML element name for a pattern kind.
func patternElemName(k patternKind) string {
	switch k {
	case patternElement:
		return "element"
	case patternAttribute:
		return "attribute"
	case patternList:
		return "list"
	case patternText:
		return "text"
	case patternInterleave:
		return combineInterleave
	case patternRef:
		return "ref"
	case patternParentRef:
		return "parentRef"
	case patternData:
		return "data"
	case patternGroup:
		return "group"
	case patternChoice:
		return combineChoice
	case patternOneOrMore:
		return "oneOrMore"
	case patternZeroOrMore:
		return "zeroOrMore"
	case patternOptional:
		return "optional"
	default:
		return "unknown"
	}
}
