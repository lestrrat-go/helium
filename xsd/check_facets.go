package xsd

import (
	"fmt"
	"math/big"
	"sort"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// compareDecimal compares two decimal string values using math/big.Rat.
// Returns -1 if a < b, 0 if a == b, 1 if a > b, or -2 on parse error.
func compareDecimal(a, b string) int {
	ra, ok1 := new(big.Rat).SetString(a)
	rb, ok2 := new(big.Rat).SetString(b)
	if !ok1 || !ok2 {
		return -2
	}
	return ra.Cmp(rb)
}

// baseFacets returns the FacetSet from the nearest base type in the chain.
func baseFacets(td *TypeDef) *FacetSet {
	if td.BaseType == nil {
		return nil
	}
	for cur := td.BaseType; cur != nil; cur = cur.BaseType {
		if cur.Facets != nil {
			return cur.Facets
		}
	}
	return nil
}

// checkFacetConsistency validates facet constraints for all named types.
// It checks same-type mutual exclusion, same-type consistency, and
// base-type restriction narrowing rules.
func (c *compiler) checkFacetConsistency() {
	if c.filename == "" {
		return
	}

	// Collect and sort types by name for deterministic error ordering.
	type facetEntry struct {
		qn QName
		td *TypeDef
	}
	var entries []facetEntry
	for qn, td := range c.schema.types {
		if td.Facets == nil {
			continue
		}
		if qn.NS == lexicon.NamespaceXSD {
			continue
		}
		entries = append(entries, facetEntry{qn: qn, td: td})
	}
	sort.Slice(entries, func(i, j int) bool {
		si, oki := c.typeDefSources[entries[i].td]
		sj, okj := c.typeDefSources[entries[j].td]
		if oki && okj {
			return si.line < sj.line
		}
		if oki != okj {
			return oki
		}
		return entries[i].qn.Local < entries[j].qn.Local
	})

	for _, entry := range entries {
		td := entry.td
		fs := td.Facets

		src, hasSrc := c.typeDefSources[td]
		component := td.Name.Local
		if component == "" {
			component = "local simple type"
		}
		line := 0
		if hasSrc {
			line = src.line
		}

		c.checkFacetMutualExclusion(fs, line, component)
		c.checkFacetSameTypeConsistency(fs, line, component)
		c.checkFacetBaseRestriction(td, fs, line, component)
	}
}

// checkFacetMutualExclusion checks that mutually exclusive facets are not
// both specified on the same type definition.
func (c *compiler) checkFacetMutualExclusion(fs *FacetSet, line int, component string) {
	if fs.Length != nil && (fs.MinLength != nil || fs.MaxLength != nil) {
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for both 'length' and either of 'minLength' or 'maxLength' to be specified on the same type definition."), helium.ErrorLevelFatal))
		c.errorCount++
	}
	if fs.MaxInclusive != nil && fs.MaxExclusive != nil {
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for both 'maxInclusive' and 'maxExclusive' to be specified."), helium.ErrorLevelFatal))
		c.errorCount++
	}
	if fs.MinInclusive != nil && fs.MinExclusive != nil {
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for both 'minInclusive' and 'minExclusive' to be specified."), helium.ErrorLevelFatal))
		c.errorCount++
	}
}

// checkFacetSameTypeConsistency checks consistency of facets within the same type.
func (c *compiler) checkFacetSameTypeConsistency(fs *FacetSet, line int, component string) {
	if fs.MinLength != nil && fs.MaxLength != nil && *fs.MinLength > *fs.MaxLength {
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for the value of 'minLength' to be greater than the value of 'maxLength'."), helium.ErrorLevelFatal))
		c.errorCount++
	}
	if fs.MinInclusive != nil && fs.MaxInclusive != nil {
		if compareDecimal(*fs.MinInclusive, *fs.MaxInclusive) > 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				"It is an error for the value of 'minInclusive' to be greater than the value of 'maxInclusive'."), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinExclusive != nil && fs.MaxExclusive != nil {
		if compareDecimal(*fs.MinExclusive, *fs.MaxExclusive) >= 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				"It is an error for the value of 'minExclusive' to be greater than or equal to the value of 'maxExclusive'."), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.FractionDigits != nil && fs.TotalDigits != nil && *fs.FractionDigits > *fs.TotalDigits {
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for the value of 'fractionDigits' to be greater than the value of 'totalDigits'."), helium.ErrorLevelFatal))
		c.errorCount++
	}
	if fs.MinExclusive != nil && fs.MaxInclusive != nil {
		if compareDecimal(*fs.MinExclusive, *fs.MaxInclusive) >= 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				"It is an error for the value of 'minExclusive' to be greater than or equal to the value of 'maxInclusive'."), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinInclusive != nil && fs.MaxExclusive != nil {
		if compareDecimal(*fs.MinInclusive, *fs.MaxExclusive) >= 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				"It is an error for the value of 'minInclusive' to be greater than or equal to the value of 'maxExclusive'."), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
}

// checkFacetBaseRestriction checks that facet values properly narrow (not widen)
// the base type's facets.
func (c *compiler) checkFacetBaseRestriction(td *TypeDef, fs *FacetSet, line int, component string) {
	base := baseFacets(td)
	if base == nil {
		return
	}

	// Length facets.
	if fs.MinLength != nil && base.MinLength != nil && *fs.MinLength < *base.MinLength {
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'minLength' value '%d' is less than the 'minLength' value of the base type '%d'.", *fs.MinLength, *base.MinLength)), helium.ErrorLevelFatal))
		c.errorCount++
	}
	if fs.MaxLength != nil && base.MaxLength != nil && *fs.MaxLength > *base.MaxLength {
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'maxLength' value '%d' is greater than the 'maxLength' value of the base type '%d'.", *fs.MaxLength, *base.MaxLength)), helium.ErrorLevelFatal))
		c.errorCount++
	}
	if fs.Length != nil && base.Length != nil && *fs.Length != *base.Length {
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'length' value '%d' does not match the 'length' value of the base type '%d'.", *fs.Length, *base.Length)), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// Digit facets.
	if fs.TotalDigits != nil && base.TotalDigits != nil && *fs.TotalDigits > *base.TotalDigits {
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'totalDigits' value '%d' is greater than the 'totalDigits' value of the base type '%d'.", *fs.TotalDigits, *base.TotalDigits)), helium.ErrorLevelFatal))
		c.errorCount++
	}
	if fs.FractionDigits != nil && base.FractionDigits != nil && *fs.FractionDigits > *base.FractionDigits {
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'fractionDigits' value '%d' is greater than the 'fractionDigits' value of the base type '%d'.", *fs.FractionDigits, *base.FractionDigits)), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// Inclusive/exclusive boundary facets vs base.
	if fs.MaxInclusive != nil && base.MaxInclusive != nil {
		if compareDecimal(*fs.MaxInclusive, *base.MaxInclusive) > 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxInclusive' value '%s' is greater than the 'maxInclusive' value of the base type '%s'.", *fs.MaxInclusive, *base.MaxInclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MaxInclusive != nil && base.MaxExclusive != nil {
		if compareDecimal(*fs.MaxInclusive, *base.MaxExclusive) >= 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxInclusive' value '%s' must be less than the 'maxExclusive' value of the base type '%s'.", *fs.MaxInclusive, *base.MaxExclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MaxInclusive != nil && base.MinInclusive != nil {
		if compareDecimal(*fs.MaxInclusive, *base.MinInclusive) < 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxInclusive' value '%s' is less than the 'minInclusive' value of the base type '%s'.", *fs.MaxInclusive, *base.MinInclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MaxInclusive != nil && base.MinExclusive != nil {
		if compareDecimal(*fs.MaxInclusive, *base.MinExclusive) <= 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxInclusive' value '%s' must be greater than the 'minExclusive' value of the base type '%s'.", *fs.MaxInclusive, *base.MinExclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MaxExclusive != nil && base.MaxExclusive != nil {
		if compareDecimal(*fs.MaxExclusive, *base.MaxExclusive) > 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxExclusive' value '%s' is greater than the 'maxExclusive' value of the base type '%s'.", *fs.MaxExclusive, *base.MaxExclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MaxExclusive != nil && base.MaxInclusive != nil {
		if compareDecimal(*fs.MaxExclusive, *base.MaxInclusive) > 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxExclusive' value '%s' is greater than the 'maxInclusive' value of the base type '%s'.", *fs.MaxExclusive, *base.MaxInclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MaxExclusive != nil && base.MinInclusive != nil {
		if compareDecimal(*fs.MaxExclusive, *base.MinInclusive) <= 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxExclusive' value '%s' must be greater than the 'minInclusive' value of the base type '%s'.", *fs.MaxExclusive, *base.MinInclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MaxExclusive != nil && base.MinExclusive != nil {
		if compareDecimal(*fs.MaxExclusive, *base.MinExclusive) <= 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxExclusive' value '%s' must be greater than the 'minExclusive' value of the base type '%s'.", *fs.MaxExclusive, *base.MinExclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinInclusive != nil && base.MinInclusive != nil {
		if compareDecimal(*fs.MinInclusive, *base.MinInclusive) < 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minInclusive' value '%s' is less than the 'minInclusive' value of the base type '%s'.", *fs.MinInclusive, *base.MinInclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinInclusive != nil && base.MinExclusive != nil {
		if compareDecimal(*fs.MinInclusive, *base.MinExclusive) <= 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minInclusive' value '%s' must be greater than the 'minExclusive' value of the base type '%s'.", *fs.MinInclusive, *base.MinExclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinInclusive != nil && base.MaxInclusive != nil {
		if compareDecimal(*fs.MinInclusive, *base.MaxInclusive) > 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minInclusive' value '%s' is greater than the 'maxInclusive' value of the base type '%s'.", *fs.MinInclusive, *base.MaxInclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinInclusive != nil && base.MaxExclusive != nil {
		if compareDecimal(*fs.MinInclusive, *base.MaxExclusive) >= 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minInclusive' value '%s' must be less than the 'maxExclusive' value of the base type '%s'.", *fs.MinInclusive, *base.MaxExclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinExclusive != nil && base.MinExclusive != nil {
		if compareDecimal(*fs.MinExclusive, *base.MinExclusive) < 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minExclusive' value '%s' is less than the 'minExclusive' value of the base type '%s'.", *fs.MinExclusive, *base.MinExclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinExclusive != nil && base.MinInclusive != nil {
		if compareDecimal(*fs.MinExclusive, *base.MinInclusive) < 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minExclusive' value '%s' is less than the 'minInclusive' value of the base type '%s'.", *fs.MinExclusive, *base.MinInclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinExclusive != nil && base.MaxInclusive != nil {
		if compareDecimal(*fs.MinExclusive, *base.MaxInclusive) > 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minExclusive' value '%s' is greater than the 'maxInclusive' value of the base type '%s'.", *fs.MinExclusive, *base.MaxInclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinExclusive != nil && base.MaxExclusive != nil {
		if compareDecimal(*fs.MinExclusive, *base.MaxExclusive) >= 0 {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minExclusive' value '%s' must be less than the 'maxExclusive' value of the base type '%s'.", *fs.MinExclusive, *base.MaxExclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
}
