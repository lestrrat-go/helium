package xslt3

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/lestrrat-go/helium/internal/catalog"
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
	return func(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
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
		if err := xsd.ValidateSimpleValue(lexical, td); err != nil {
			return nil, &xpath3.XPathError{
				Code:    "FORG0001",
				Message: fmt.Sprintf("cannot cast %q to %s", lexical, typeName),
			}
		}

		if baseType == "" || variety != xsd.TypeVarietyAtomic {
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
		// so that aggregate functions like sum() can operate on it.
		return xpath3.SingleAtomic(xpath3.AtomicValue{
			TypeName: typeName,
			Value:    cast.Value,
		}), nil
	}
}

func schemaConstructorArg(seq xpath3.Sequence, typeName string) (xpath3.AtomicValue, bool, error) {
	if len(seq) == 0 {
		return xpath3.AtomicValue{}, true, nil
	}
	if len(seq) > 1 {
		return xpath3.AtomicValue{}, false, &xpath3.XPathError{
			Code:    "XPTY0004",
			Message: fmt.Sprintf("%s constructor requires a singleton argument", typeName),
		}
	}
	av, err := xpath3.AtomizeItem(seq[0])
	if err != nil {
		return xpath3.AtomicValue{}, false, err
	}
	return av, false, nil
}

func (ec *execContext) schemaTypeName(uri, local string) string {
	if uri == catalog.XSD {
		return "xs:" + local
	}
	// Use Q{ns}local annotation format for consistency with type annotations
	// from schema validation and XPath instance-of checks.
	if uri != "" {
		return xpath3.QAnnotation(uri, local)
	}
	return local
}

func (ec *execContext) schemaPrefixes(uri string) []string {
	if uri == "" {
		return nil
	}
	ns := make(map[string]string, len(ec.stylesheet.namespaces)+1)
	collectPackageNamespaces(ec.stylesheet, ns)
	for k, v := range ec.stylesheet.namespaces {
		ns[k] = v
	}
	if ec.currentPackage != nil {
		for k, v := range ec.currentPackage.namespaces {
			ns[k] = v
		}
	}
	var prefixes []string
	for prefix, nsURI := range ns {
		if prefix == "" || nsURI != uri {
			continue
		}
		prefixes = append(prefixes, prefix)
	}
	slices.Sort(prefixes)
	return prefixes
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
	case "string":
		return xpath3.TypeString
	case "boolean":
		return xpath3.TypeBoolean
	case "decimal":
		return xpath3.TypeDecimal
	case "double":
		return xpath3.TypeDouble
	case "float":
		return xpath3.TypeFloat
	case "integer":
		return xpath3.TypeInteger
	case "date":
		return xpath3.TypeDate
	case "dateTime":
		return xpath3.TypeDateTime
	case "dateTimeStamp":
		return xpath3.TypeDateTimeStamp
	case "time":
		return xpath3.TypeTime
	case "duration":
		return xpath3.TypeDuration
	case "dayTimeDuration":
		return xpath3.TypeDayTimeDuration
	case "yearMonthDuration":
		return xpath3.TypeYearMonthDuration
	case "anyURI":
		return xpath3.TypeAnyURI
	case "base64Binary":
		return xpath3.TypeBase64Binary
	case "hexBinary":
		return xpath3.TypeHexBinary
	case "untypedAtomic":
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
		if cur.Name.NS == catalog.XSD && cur.Name.Local != "" {
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
