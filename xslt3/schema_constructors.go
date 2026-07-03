package xslt3

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

func (ec *execContext) registerSchemaConstructors(dst map[xpath3.QualifiedName]xpath3.Function) {
	for _, schema := range ec.stylesheet.schemas {
		for _, qn := range schema.NamedTypes() {
			td, ok := schema.LookupType(qn.Local, qn.NS)
			if !ok || td == nil || td.ContentType != xsd.ContentTypeSimple {
				continue
			}
			key := xpath3.QualifiedName{URI: qn.NS, Name: qn.Local}
			if _, exists := dst[key]; exists {
				continue
			}
			dst[key] = &xsltFunc{min: 1, max: 1, fn: ec.makeSchemaConstructor(td)}
		}
	}
}

func (ec *execContext) makeSchemaConstructor(td *xsd.TypeDef) func(context.Context, []xpath3.Sequence) (xpath3.Sequence, error) {
	typeName := ec.schemaTypeName(td.Name.NS, td.Name.Local)
	baseType := schemaBuiltinXPathType(td)
	variety := schemaTypeVariety(td)
	return func(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
		av, empty, err := schemaConstructorArg(args[0], typeName)
		if err != nil {
			return nil, err
		}
		if empty {
			return nil, nil
		}

		lexical, err := xpath3.AtomicToString(av)
		if err != nil {
			return nil, err
		}
		if baseType == xpath3.TypeQName || baseType == xpath3.TypeNOTATION {
			if err := td.Validate(ctx, lexical, ec.stylesheet.namespaces); err != nil {
				return nil, &xpath3.XPathError{
					Code:    "FORG0001",
					Message: fmt.Sprintf("cannot cast %q to %s", lexical, typeName),
				}
			}
			qv, err := schemaConstructorQNameValue(av, ec)
			if err != nil {
				return nil, err
			}
			return xpath3.SingleAtomic(xpath3.AtomicValue{
				TypeName: typeName,
				Value:    qv,
			}), nil
		}
		if err := td.Validate(ctx, lexical, nil); err != nil {
			return nil, &xpath3.XPathError{
				Code:    "FORG0001",
				Message: fmt.Sprintf("cannot cast %q to %s", lexical, typeName),
			}
		}

		if baseType == "" || variety != xsd.TypeVarietyAtomic {
			// A LIST-typed value is a SEQUENCE of item atoms (XDM list semantics),
			// not a single string atom — so its cardinality and per-item typing
			// match a real list value (e.g. `s:intListType1("1 2 3")` is three
			// xs:integer items, not one string, so it is not a singleton).
			if variety == xsd.TypeVarietyList {
				if seq, ok := ec.expandSchemaListValue(td, schemaNormalizeLexical(lexical, td)); ok {
					return seq, nil
				}
			}
			return xpath3.SingleAtomic(xpath3.AtomicValue{
				TypeName: typeName,
				Value:    schemaNormalizeLexical(lexical, td),
			}), nil
		}

		cast, err := xpath3.CastAtomic(av, baseType)
		if err != nil {
			return nil, err
		}
		// Preserve the actual typed value (e.g., *big.Int for integer types)
		// so that aggregate functions like sum() can operate on it. Stamp the
		// builtin BaseType (the AtomizeItem/schema-aware-cast convention) so the
		// user-typed atom normalizes to its base for cast dispatch and canonical
		// string formatting (e.g. a user xs:date atom stringifies as "2001-01-01").
		return xpath3.SingleAtomic(xpath3.AtomicValue{
			TypeName: typeName,
			BaseType: baseType,
			Value:    cast.Value,
		}), nil
	}
}

func schemaConstructorQNameValue(av xpath3.AtomicValue, ec *execContext) (xpath3.QNameValue, error) {
	promoted := xpath3.PromoteSchemaType(av)
	if promoted.TypeName == xpath3.TypeQName {
		return promoted.QNameVal(), nil
	}
	if qv, ok := av.Value.(xpath3.QNameValue); ok {
		return qv, nil
	}
	if av.TypeName == xpath3.TypeNOTATION {
		s, err := xpath3.AtomicToString(av)
		if err != nil {
			return xpath3.QNameValue{}, err
		}
		return resolveQNameFromMap(s, ec.stylesheet.namespaces)
	}

	if av.TypeName != xpath3.TypeString && av.TypeName != xpath3.TypeUntypedAtomic {
		return xpath3.QNameValue{}, &xpath3.XPathError{
			Code:    "XPTY0004",
			Message: fmt.Sprintf("cannot cast %s to schema-derived QName/NOTATION type", av.TypeName),
		}
	}
	return resolveQNameFromMap(av.StringVal(), ec.stylesheet.namespaces)
}

func resolveQNameFromMap(s string, ns map[string]string) (xpath3.QNameValue, error) {
	prefix, local, _, validNC := domutil.SplitLexicalQName(s)
	if !validNC {
		return xpath3.QNameValue{}, &xpath3.XPathError{
			Code:    "FORG0001",
			Message: fmt.Sprintf("invalid QName: %q", strings.TrimSpace(s)),
		}
	}

	uri := ""
	if prefix != "" {
		var ok bool
		uri, ok = ns[prefix]
		if !ok {
			return xpath3.QNameValue{}, &xpath3.XPathError{
				Code:    "FONS0004",
				Message: fmt.Sprintf("no namespace binding for prefix %q", prefix),
			}
		}
	} else if ns != nil {
		uri = ns[""]
	}

	return xpath3.QNameValue{Prefix: prefix, Local: local, URI: uri}, nil
}

func schemaConstructorArg(seq xpath3.Sequence, typeName string) (xpath3.AtomicValue, bool, error) {
	if seq == nil || sequence.Len(seq) == 0 {
		return xpath3.AtomicValue{}, true, nil
	}
	if sequence.Len(seq) > 1 {
		return xpath3.AtomicValue{}, false, &xpath3.XPathError{
			Code:    "XPTY0004",
			Message: fmt.Sprintf("%s constructor requires a singleton argument", typeName),
		}
	}
	av, err := xpath3.AtomizeItem(seq.Get(0))
	if err != nil {
		return xpath3.AtomicValue{}, false, err
	}
	return av, false, nil
}

func (ec *execContext) schemaTypeName(uri, local string) string {
	if uri == lexicon.NamespaceXSD {
		return "xs:" + local
	}
	// Use Q{ns}local annotation format for consistency with type annotations
	// from schema validation and XPath instance-of checks.
	if uri != "" {
		return xpath3.QAnnotation(uri, local)
	}
	return local
}

// listItemTypeDef returns the item type definition of a list type, following the
// base-type chain (a restriction of a list keeps the base's item type).
func listItemTypeDef(listTD *xsd.TypeDef) *xsd.TypeDef {
	for cur := listTD; cur != nil; cur = cur.BaseType {
		if cur.Variety == xsd.TypeVarietyList && cur.ItemType != nil {
			return cur.ItemType
		}
	}
	return nil
}

// expandSchemaListValue expands a (validated, whitespace-collapsed) list lexical
// into a sequence of per-item atoms, each typed as the item type — the XDM
// representation of a list value. It only handles an ATOMIC item type (the common
// case, e.g. xs:list itemType="xs:integer"); a union/list item type returns ok=false
// so the caller keeps the single-atom fallback rather than mis-typing the tokens.
func (ec *execContext) expandSchemaListValue(listTD *xsd.TypeDef, lexical string) (xpath3.Sequence, bool) {
	itemTD := listItemTypeDef(listTD)
	if itemTD == nil {
		return nil, false
	}
	itemBuiltin := schemaBuiltinXPathType(itemTD)
	if itemBuiltin == "" || schemaTypeVariety(itemTD) != xsd.TypeVarietyAtomic {
		return nil, false
	}
	itemName := ec.schemaTypeName(itemTD.Name.NS, itemTD.Name.Local)
	tokens := strings.Fields(lexical)
	items := make(xpath3.ItemSlice, 0, len(tokens))
	for _, tok := range tokens {
		cast, err := xpath3.CastFromString(tok, itemBuiltin)
		if err != nil {
			return nil, false
		}
		items = append(items, xpath3.AtomicValue{
			TypeName: itemName,
			BaseType: itemBuiltin,
			Value:    cast.Value,
		})
	}
	return items, true
}

func schemaTypeVariety(td *xsd.TypeDef) xsd.TypeVariety {
	for cur := td; cur != nil; cur = cur.BaseType {
		if cur.Variety != xsd.TypeVarietyAtomic {
			return cur.Variety
		}
	}
	return xsd.TypeVarietyAtomic
}

func schemaBuiltinXPathType(td *xsd.TypeDef) string {
	switch schemaBuiltinBaseLocal(td) {
	case lexicon.TypeString:
		return xpath3.TypeString
	case lexicon.TypeBoolean:
		return xpath3.TypeBoolean
	case lexicon.TypeDecimal:
		return xpath3.TypeDecimal
	case lexicon.TypeDouble:
		return xpath3.TypeDouble
	case lexicon.TypeFloat:
		return xpath3.TypeFloat
	case lexicon.TypeInteger:
		return xpath3.TypeInteger
	case "date":
		return xpath3.TypeDate
	case lexicon.TypeDateTime:
		return xpath3.TypeDateTime
	case "dateTimeStamp":
		return xpath3.TypeDateTimeStamp
	case lexicon.TypeTime:
		return xpath3.TypeTime
	case lexicon.TypeDuration:
		return xpath3.TypeDuration
	case lexicon.TypeDayTimeDuration:
		return xpath3.TypeDayTimeDuration
	case lexicon.TypeYearMonthDuration:
		return xpath3.TypeYearMonthDuration
	case "anyURI":
		return xpath3.TypeAnyURI
	case "base64Binary":
		return xpath3.TypeBase64Binary
	case "hexBinary":
		return xpath3.TypeHexBinary
	case lexicon.TypeUntypedAtomic:
		return xpath3.TypeUntypedAtomic
	case "normalizedString":
		return xpath3.TypeNormalizedString
	case "token":
		return xpath3.TypeToken
	case "language":
		return xpath3.TypeLanguage
	case "Name":
		return xpath3.TypeName
	case "NCName":
		return xpath3.TypeNCName
	case "NMTOKEN":
		return xpath3.TypeNMTOKEN
	case "ENTITY":
		return xpath3.TypeENTITY
	case "ID":
		return xpath3.TypeID
	case "IDREF":
		return xpath3.TypeIDREF
	case "long":
		return xpath3.TypeLong
	case "int":
		return xpath3.TypeInt
	case "short":
		return xpath3.TypeShort
	case "byte":
		return xpath3.TypeByte
	case "unsignedLong":
		return xpath3.TypeUnsignedLong
	case "unsignedInt":
		return xpath3.TypeUnsignedInt
	case "unsignedShort":
		return xpath3.TypeUnsignedShort
	case "unsignedByte":
		return xpath3.TypeUnsignedByte
	case "nonNegativeInteger":
		return xpath3.TypeNonNegativeInteger
	case "nonPositiveInteger":
		return xpath3.TypeNonPositiveInteger
	case "positiveInteger":
		return xpath3.TypePositiveInteger
	case "negativeInteger":
		return xpath3.TypeNegativeInteger
	case "gDay":
		return xpath3.TypeGDay
	case "gMonth":
		return xpath3.TypeGMonth
	case "gMonthDay":
		return xpath3.TypeGMonthDay
	case "gYear":
		return xpath3.TypeGYear
	case "gYearMonth":
		return xpath3.TypeGYearMonth
	case "QName":
		return xpath3.TypeQName
	case "NOTATION":
		return xpath3.TypeNOTATION
	default:
		return ""
	}
}

func schemaBuiltinBaseLocal(td *xsd.TypeDef) string {
	for cur := td; cur != nil; cur = cur.BaseType {
		if cur.Name.NS == lexicon.NamespaceXSD && cur.Name.Local != "" {
			return cur.Name.Local
		}
	}
	return ""
}

func schemaNormalizeLexical(value string, td *xsd.TypeDef) string {
	switch schemaWhitespaceMode(td) {
	case "preserve":
		return value
	case "replace":
		return strings.Map(func(r rune) rune {
			switch r {
			case '\t', '\n', '\r':
				return ' '
			default:
				return r
			}
		}, value)
	default:
		value = strings.Map(func(r rune) rune {
			switch r {
			case '\t', '\n', '\r':
				return ' '
			default:
				return r
			}
		}, value)
		return strings.Join(strings.Fields(value), " ")
	}
}

func schemaWhitespaceMode(td *xsd.TypeDef) string {
	for cur := td; cur != nil; cur = cur.BaseType {
		if cur.Facets != nil && cur.Facets.WhiteSpace != nil {
			return *cur.Facets.WhiteSpace
		}
	}
	return "collapse"
}
