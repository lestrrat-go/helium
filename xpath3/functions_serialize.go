package xpath3

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	htmlpkg "github.com/lestrrat-go/helium/html"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xmlchar"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
	"golang.org/x/text/unicode/norm"
)

const (
	serializeMethodXML      = "xml"
	serializeMethodXHTML    = "xhtml"
	serializeMethodHTML     = "html"
	serializeMethodText     = "text"
	serializeMethodJSON     = "json"
	serializeMethodAdaptive = "adaptive"
	serializeStandaloneOmit = "omit"
)

func init() {
	registerFn("serialize", 1, 2, fnSerialize)
}

const (
	errCodeSEPM0009 = "SEPM0009"
	errCodeSEPM0010 = "SEPM0010"
	errCodeSEPM0016 = "SEPM0016"
	errCodeSESU0011 = "SESU0011"
	errCodeSESU0013 = "SESU0013"
)

// effectiveSerializeVersion returns the effective output version, defaulting an
// unspecified version to "1.0".
func (o serializeOptions) effectiveSerializeVersion() string {
	if o.xmlVersion == "" {
		return "1.0"
	}
	return o.xmlVersion
}

// methodEmitsXMLDeclaration reports whether the output method produces an XML
// declaration (the xml and xhtml methods, and the unspecified default). The
// omit-xml-declaration / standalone / SEPM0009 rules apply only to these; for
// html/json/text/adaptive an XML declaration is never produced.
func (o serializeOptions) methodEmitsXMLDeclaration() bool {
	switch o.method {
	case "", serializeMethodXML, serializeMethodXHTML:
		return true
	}
	return false
}

func fnSerialize(ctx context.Context, args []Sequence) (Sequence, error) {
	opts, err := parseSerializeOptions(ctx, args)
	if err != nil {
		return nil, err
	}

	// An extension method (a prefixed QName) is a valid method-type value but
	// helium implements no extension output methods, so it is an unsupported
	// value (SEPM0016) rather than a silent fall-through to the xml method. The
	// built-in methods and the unspecified default ("") dispatch below.
	if opts.method != "" && !isBuiltinSerializeMethod(opts.method) {
		return nil, &XPathError{Code: errCodeSEPM0016, Message: fmt.Sprintf("output method %q is not supported", opts.method)}
	}

	// The version parameter, for the methods that emit an XML declaration, must
	// name an XML version the serializer supports (1.0 or 1.1); any other value
	// is the SESU0013 unsupported-version serialization error rather than a
	// bogus version pseudo-attribute. It also drives the XML 1.1 escaping /
	// undeclaration rules, so it must be validated even when the declaration is
	// omitted.
	if opts.methodEmitsXMLDeclaration() && opts.xmlVersion != "" && !isSupportedXMLOutputVersion(opts.xmlVersion) {
		return nil, &XPathError{Code: errCodeSESU0013, Message: fmt.Sprintf("XML output version %q is not supported", opts.xmlVersion)}
	}

	// SEPM0009 (Serialization 3.1): when the omit-xml-declaration parameter is
	// yes, it is an error if (a) the standalone parameter has a value other than
	// omit, OR (b) the version parameter has a value other than 1.0 AND the
	// doctype-system parameter is specified. Both sub-conditions are gated on
	// omit-xml-declaration=yes. It applies only to methods that emit an XML
	// declaration (xml/xhtml/default); this uses the EFFECTIVE
	// omit-xml-declaration (the map-form default of true) and the RESOLVED
	// standalone value. A DOCTYPE without an XML declaration is well-formed in
	// XML 1.0, so omit + doctype-system at version 1.0 is NOT an error.
	if opts.methodEmitsXMLDeclaration() && opts.omitXMLDeclaration {
		if opts.standalone == lexicon.ValueYes || opts.standalone == lexicon.ValueNo {
			return nil, &XPathError{Code: errCodeSEPM0009, Message: "omit-xml-declaration=yes conflicts with standalone=yes/no"}
		}
		if opts.effectiveSerializeVersion() != "1.0" && opts.doctypeSystem != "" {
			return nil, &XPathError{Code: errCodeSEPM0009, Message: "omit-xml-declaration=yes conflicts with a doctype-system value at version other than 1.0"}
		}
	}

	// Namespace undeclarations require XML/XHTML 1.1; requesting them at an
	// effective output version of 1.0 is a static error (Serialization 3.1
	// SEPM0010).
	if opts.undeclarePrefixes && opts.effectiveSerializeVersion() != "1.1" {
		return nil, &XPathError{Code: errCodeSEPM0010, Message: "undeclare-prefixes requires output version 1.1"}
	}

	var result string
	switch opts.method {
	case "", serializeMethodAdaptive:
		result, err = serializeAdaptiveSequence(args[0], opts)
	case serializeMethodJSON:
		result, err = serializeJSONSequence(args[0], opts)
	case serializeMethodHTML:
		result, err = serializeHTMLSequence(args[0], opts)
	case serializeMethodText:
		result, err = serializeTextSequence(args[0], opts)
	default:
		// The xml method (and xhtml, serialized as XML — a defensible
		// approximation, since helium implements no XHTML-specific rules).
		result, err = serializeXMLSequence(args[0], opts)
	}
	if err != nil {
		return nil, err
	}

	// Apply the requested Unicode normalization to the serialized output (the
	// last step, for the methods that support normalization-form). json/adaptive
	// ignore the parameter; "fully-normalized" is the SESU0011 unsupported form.
	result, err = applySerializeNormalization(result, opts)
	if err != nil {
		return nil, err
	}
	return SingleString(result), nil
}

// applySerializeNormalization applies the normalization-form parameter to the
// serialized string. none/"" is a no-op; NFC/NFD/NFKC/NFKD are applied via
// golang.org/x/text/unicode/norm; the W3C-specific "fully-normalized" form is
// not provided by that package and is the SESU0011 unsupported-normalization
// serialization error. The parameter is not applicable to the json/adaptive
// methods, which ignore it (Serialization 3.1 §9.1.9).
func applySerializeNormalization(s string, opts serializeOptions) (string, error) {
	form := opts.normalizationForm
	if form == "" || form == "none" || !opts.methodAppliesNormalization() {
		return s, nil
	}
	switch form {
	case "NFC":
		return norm.NFC.String(s), nil
	case "NFD":
		return norm.NFD.String(s), nil
	case "NFKC":
		return norm.NFKC.String(s), nil
	case "NFKD":
		return norm.NFKD.String(s), nil
	}
	return "", &XPathError{Code: errCodeSESU0011, Message: fmt.Sprintf("normalization-form %q is not supported", form)}
}

type serializeOptions struct {
	method              string
	itemSeparator       string
	indent              bool
	omitXMLDeclaration  bool
	allowDuplicateNames bool
	encoding            string
	// standalone is the resolved standalone pseudo-attribute request:
	// "yes"/"no" force it, "omit" forces omission. It defaults to "omit" (the
	// Serialization 3.1 default), so a source declaration's standalone is not
	// retained unless the parameter requests it.
	standalone string
	// undeclarePrefixes requests XML 1.1 namespace undeclarations (xmlns:pfx="").
	// It is honored only when the effective output version is 1.1; otherwise it
	// is a SEPM0010 static error.
	undeclarePrefixes bool
	// xmlVersion is the requested output version parameter ("" = unspecified,
	// treated as "1.0").
	xmlVersion string
	// cdataElements / suppressIndent hold the exact expanded {uri}local element
	// names for the cdata-section-elements and suppress-indentation parameters.
	cdataElements  map[string]struct{}
	suppressIndent map[string]struct{}
	// charMap is the resolved character map (use-character-maps).
	charMap map[rune]string
	// HTML output-method parameters (applied by serializeHTMLNode).
	// includeContentType / escapeURIAttributes default to true.
	includeContentType  bool
	escapeURIAttributes bool
	// htmlVersion is the requested html-version ("" = default 5.0). doctypePublic
	// / doctypeSystem are the requested DOCTYPE identifiers.
	htmlVersion   string
	doctypePublic string
	doctypeSystem string
	// mediaType is the media-type parameter ("" = default "text/html"), used in
	// the html Content-Type meta element.
	mediaType string
	// normalizationForm is the requested Unicode normalization form ("" / "none"
	// = no normalization). NFC/NFD/NFKC/NFKD are applied to the serialized output
	// for the methods that support normalization; "fully-normalized" is the
	// SESU0011 unsupported-normalization serialization error; the parameter is not
	// applicable to (ignored by) the json/adaptive methods.
	normalizationForm string
	// jsonNodeOutputMethod is the requested json-node-output-method value ("" =
	// the default xml). helium serializes a node embedded in JSON only with the
	// xml method, so any other value that would change node serialization is an
	// unsupported-feature error when a node is actually serialized under json.
	jsonNodeOutputMethod string
}

// methodAppliesNormalization reports whether the output method applies the
// normalization-form parameter. Per Serialization 3.1 it is a parameter of the
// xml/xhtml/html/text output methods (and the unspecified default); it is NOT
// applicable to the json (§9.1.9) or adaptive methods, which ignore it.
func (o serializeOptions) methodAppliesNormalization() bool {
	switch o.method {
	case "", serializeMethodXML, serializeMethodXHTML, serializeMethodHTML, serializeMethodText:
		return true
	}
	return false
}

func parseSerializeOptions(ctx context.Context, args []Sequence) (serializeOptions, error) {
	// The default output method is xml (Serialization 3.1 §2 / F&O 3.1
	// §18.9.1); adaptive is opt-in and must be requested explicitly. An empty
	// method means "unspecified default (xml family)", which the dispatch
	// treats like adaptive for atomic/map/array output but which the node-kind
	// guard distinguishes from an explicit "adaptive" so serializing an
	// attribute or namespace node under the default method raises SENR0001.
	opts := serializeOptions{
		method:        "",
		itemSeparator: " ",
		// The Serialization 3.1 default for the standalone parameter is "omit".
		standalone: serializeStandaloneOmit,
		// include-content-type and escape-uri-attributes default to yes.
		includeContentType:  true,
		escapeURIAttributes: true,
	}
	if len(args) <= 1 || seqLen(args[1]) == 0 {
		return opts, nil
	}

	m, ok := args[1].Get(0).(MapItem)
	if ok {
		return parseSerializeOptionsMap(ctx, opts, m)
	}
	if seqLen(args[1]) != 1 {
		return opts, &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize options must be a singleton"}
	}
	node, ok := args[1].Get(0).(NodeItem)
	if !ok {
		return opts, nil
	}
	return parseSerializeOptionsNode(opts, node.Node)
}

func parseSerializeOptionsMap(ctx context.Context, opts serializeOptions, m MapItem) (serializeOptions, error) {
	readBool := func(name string) (bool, bool, error) {
		v, found := m.Get(AtomicValue{TypeName: TypeString, Value: name})
		if !found {
			return false, false, nil
		}
		if seqLen(v) != 1 {
			return false, true, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option %q must be a single xs:boolean", name)}
		}
		av, ok := v.Get(0).(AtomicValue)
		if !ok {
			return false, true, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option %q must be xs:boolean", name)}
		}
		switch av.TypeName {
		case TypeBoolean:
			return av.BooleanVal(), true, nil
		case TypeUntypedAtomic:
			s, err := atomicToString(av)
			if err != nil {
				return false, true, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option %q must be xs:boolean", name)}
			}
			switch s {
			case lexicon.ValueTrue, "1":
				return true, true, nil
			case lexicon.ValueFalse, "0":
				return false, true, nil
			default:
				return false, true, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option %q must be xs:boolean", name)}
			}
		default:
			return false, true, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option %q must be xs:boolean", name)}
		}
	}

	if method, found, err := resolveSerializeMethodMap(ctx, m); err != nil {
		return opts, err
	} else if found {
		opts.method = method
	}
	if sep, found, err := readSerializeStringOption(ctx, m, "item-separator"); err != nil {
		return opts, err
	} else if found {
		opts.itemSeparator = sep
	}
	if indent, found, err := readBool("indent"); err != nil {
		return opts, err
	} else if found {
		opts.indent = indent
	}
	// The map form of fn:serialize defaults omit-xml-declaration to true:
	// supplying an empty map has the same effect as omitting the argument, and
	// the XML declaration is omitted unless the option requests otherwise
	// (F&O 3.1 §18.9.1; W3C serialize-xml-127a).
	opts.omitXMLDeclaration = true
	if omit, found, err := readBool("omit-xml-declaration"); err != nil {
		return opts, err
	} else if found {
		opts.omitXMLDeclaration = omit
	}
	if allow, found, err := readBool("allow-duplicate-names"); err != nil {
		return opts, err
	} else if found {
		opts.allowDuplicateNames = allow
	}
	if encoding, found, err := readSerializeStringOption(ctx, m, "encoding"); err != nil {
		return opts, err
	} else if found {
		opts.encoding = encoding
	}
	if undeclare, found, err := readBool("undeclare-prefixes"); err != nil {
		return opts, err
	} else if found {
		opts.undeclarePrefixes = undeclare
	}
	if include, found, err := readBool("include-content-type"); err != nil {
		return opts, err
	} else if found {
		opts.includeContentType = include
	}
	if escape, found, err := readBool("escape-uri-attributes"); err != nil {
		return opts, err
	} else if found {
		opts.escapeURIAttributes = escape
	}
	if version, found, err := readSerializeStringOption(ctx, m, "version"); err != nil {
		return opts, err
	} else if found {
		opts.xmlVersion = version
	}
	if hv, found, err := readSerializeNumberOption(m, "html-version"); err != nil {
		return opts, err
	} else if found {
		opts.htmlVersion = hv
	}
	if pub, found, err := readSerializeStringOption(ctx, m, "doctype-public"); err != nil {
		return opts, err
	} else if found {
		opts.doctypePublic = pub
	}
	if sys, found, err := readSerializeStringOption(ctx, m, "doctype-system"); err != nil {
		return opts, err
	} else if found {
		opts.doctypeSystem = sys
	}
	if mt, found, err := readSerializeStringOption(ctx, m, "media-type"); err != nil {
		return opts, err
	} else if found {
		opts.mediaType = mt
	}
	// Validation parity with the element form for the recognized-but-unapplied
	// (or limitation) parameters, so an invalid value is rejected consistently.
	if _, found, err := readBool("byte-order-mark"); err != nil {
		return opts, err
	} else {
		_ = found // validated (boolean lexical space) and ignored (UTF-8 emits no BOM)
	}
	if jn, found, err := readSerializeStringOption(ctx, m, "json-node-output-method"); err != nil {
		return opts, err
	} else if found {
		if !serializeJSONNodeOutputMethodValid(jn) {
			return opts, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option json-node-output-method has an invalid value %q", jn)}
		}
		opts.jsonNodeOutputMethod = jn
	}
	if nf, found, err := readSerializeStringOption(ctx, m, "normalization-form"); err != nil {
		return opts, err
	} else if found {
		if !serializeNormalizationFormValid(nf) {
			return opts, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option normalization-form has an invalid value %q", nf)}
		}
		opts.normalizationForm = nf
	}
	if v, found := m.Get(AtomicValue{TypeName: TypeString, Value: "standalone"}); found {
		standalone, err := resolveSerializeStandaloneMap(ctx, v)
		if err != nil {
			return opts, err
		}
		opts.standalone = standalone
	}
	if v, found := m.Get(AtomicValue{TypeName: TypeString, Value: "use-character-maps"}); found {
		charMap, err := resolveSerializeCharacterMaps(v)
		if err != nil {
			return opts, err
		}
		opts.charMap = charMap
	}
	if v, found := m.Get(AtomicValue{TypeName: TypeString, Value: "cdata-section-elements"}); found {
		names, err := resolveSerializeQNameNames(ctx, v, "cdata-section-elements")
		if err != nil {
			return opts, err
		}
		opts.cdataElements = names
	}
	if v, found := m.Get(AtomicValue{TypeName: TypeString, Value: "suppress-indentation"}); found {
		names, err := resolveSerializeQNameNames(ctx, v, "suppress-indentation")
		if err != nil {
			return opts, err
		}
		opts.suppressIndent = names
	}

	return opts, nil
}

// resolveSerializeQNameNames converts a serialization parameter whose value is a
// sequence of xs:QName items (cdata-section-elements, suppress-indentation) into
// the element-name key set the writer matches against: each QName contributes its
// exact expanded {uri}local name (Clark notation; an explicit empty namespace is
// "{}local"). Matching is by exact expanded name, so QName("","b") does not match
// a namespaced <p:b>.
func resolveSerializeQNameNames(ctx context.Context, v Sequence, name string) (map[string]struct{}, error) {
	// Option (function) conversion (F&O 3.1 §2.5): atomize FIRST so an array
	// value (W3C serialize-xml-106a supplies [QName(...), ...]) flattens to its
	// members and each member atomizes to its xs:QName, THEN require every atom to
	// be an xs:QName.
	atoms, err := atomizeTypedValue(ctx, v)
	if err != nil {
		return nil, err
	}
	names := make(map[string]struct{})
	for _, av := range atoms {
		if av.TypeName != TypeQName {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option %q must be a sequence of xs:QName", name)}
		}
		qn, ok := av.Value.(QNameValue)
		if !ok {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option %q must be a sequence of xs:QName", name)}
		}
		names[helium.ClarkName(qn.URI, qn.Local)] = struct{}{}
	}
	return names, nil
}

// readSerializeNumberOption extracts a numeric-valued serialize option
// (html-version, an xs:decimal) from the option map as its canonical string
// form. A non-numeric value is err:XPTY0004. found is false only when the key is
// absent.
func readSerializeNumberOption(m MapItem, name string) (string, bool, error) {
	v, found := m.Get(AtomicValue{TypeName: TypeString, Value: name})
	if !found {
		return "", false, nil
	}
	if seqLen(v) != 1 {
		return "", true, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option %q must be a single number", name)}
	}
	av, ok := v.Get(0).(AtomicValue)
	if !ok || !av.IsNumeric() {
		return "", true, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option %q must be numeric", name)}
	}
	s, err := atomicToString(av)
	if err != nil {
		return "", true, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option %q must be numeric", name)}
	}
	return s, true, nil
}

// readSerializeStringOption extracts a string-valued serialize option from the
// option map, applying F&O 3.1 §2.5 option (function) conversion: it atomizes
// the value FIRST — through the ctx-aware typed-value atomization, so an
// element-only-typed node (which has no typed value) raises err:FOTY0012 and an
// array (e.g. an empty-array member) flattens to its members — THEN enforces the
// singleton cardinality. atomizeTypedValue keeps the atom's type (e.g. an
// xs:QName "method" value), so atomicToString handles it unchanged; a raw
// pre-atomization seqLen gate would wrongly reject map{"method": ([], "xml")}.
// found is false only when the option key is absent.
func readSerializeStringOption(ctx context.Context, m MapItem, name string) (string, bool, error) {
	v, found := m.Get(AtomicValue{TypeName: TypeString, Value: name})
	if !found {
		return "", false, nil
	}
	atoms, err := atomizeTypedValue(ctx, v)
	if err != nil {
		return "", true, err
	}
	if len(atoms) != 1 {
		return "", true, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option %q must be a singleton", name)}
	}
	s, err := atomicToString(atoms[0])
	if err != nil {
		return "", true, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option %q must be string-like", name)}
	}
	return s, true, nil
}

// resolveSerializeMethodMap resolves the map-form "method" option, whose value is
// an xs:string OR xs:QName (F&O 3.1 §18.9.1). It inspects the ATOM so a
// namespaced QName keeps its namespace instead of being stringified to its local
// part: a no-namespace value must be a built-in method token (else err:XPTY0004),
// and a value with a non-null namespace (a namespaced QName, or a prefixed-QName
// string) names an EXTENSION method — a valid value helium does not implement,
// which the SEPM0016 dispatch check reports as unsupported (the extension name is
// carried as an EQName so it is never mistaken for a built-in token). found is
// false only when the "method" key is absent.
func resolveSerializeMethodMap(ctx context.Context, m MapItem) (string, bool, error) {
	v, present := m.Get(AtomicValue{TypeName: TypeString, Value: "method"})
	if !present {
		return "", false, nil
	}
	atoms, err := atomizeTypedValue(ctx, v)
	if err != nil {
		return "", true, err
	}
	if len(atoms) != 1 {
		return "", true, &XPathError{Code: lexicon.ErrXPTY0004, Message: `serialize option "method" must be a singleton`}
	}
	av := atoms[0]
	if av.TypeName == TypeQName {
		qn, ok := av.Value.(QNameValue)
		if !ok {
			return "", true, &XPathError{Code: lexicon.ErrXPTY0004, Message: `serialize option "method" is not a valid xs:QName`}
		}
		if qn.URI == "" {
			// A no-namespace QName must name a built-in method token; any other
			// no-namespace name is an invalid value.
			if isBuiltinSerializeMethod(qn.Local) {
				return qn.Local, true, nil
			}
			return "", true, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option method has an invalid value %q", qn.Local)}
		}
		// A namespaced QName names an extension method: preserve its namespace as
		// an EQName so it is never a built-in token and the SEPM0016 dispatch check
		// reports it as unsupported.
		return "Q{" + qn.URI + "}" + qn.Local, true, nil
	}
	s, err := atomicToString(av)
	if err != nil {
		return "", true, &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize option method has an invalid value"}
	}
	if !serializeMethodValid(s) {
		return "", true, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option method has an invalid value %q", s)}
	}
	return s, true, nil
}

func parseSerializeOptionsNode(opts serializeOptions, n helium.Node) (serializeOptions, error) {
	elem, ok := n.(*helium.Element)
	if !ok {
		return opts, &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize options node must be an element"}
	}
	if elem.URI() != lexicon.NamespaceSerialization || elem.LocalName() != "serialization-parameters" {
		return opts, &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize options root must be output:serialization-parameters"}
	}
	if len(elem.Attributes()) != 0 {
		return opts, &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize options root must not have attributes"}
	}

	seen := make(map[string]struct{})
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			if isXSDWhitespaceOnly(string(child.Content())) {
				continue
			}
			return opts, &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize options root must not contain text"}
		case helium.CommentNode, helium.ProcessingInstructionNode:
			continue
		case helium.ElementNode:
		default:
			return opts, &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize options root has invalid child node"}
		}

		param, _ := helium.AsNode[*helium.Element](child)
		if param.URI() == "" {
			return opts, &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize options parameters must be namespace-qualified"}
		}

		key := param.URI() + "|" + param.LocalName()
		if _, exists := seen[key]; exists {
			return opts, &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize options parameter must not appear more than once"}
		}
		seen[key] = struct{}{}

		if param.URI() != lexicon.NamespaceSerialization {
			if _, err := readSerializeParamValue(param); err != nil {
				return opts, err
			}
			continue
		}

		switch param.LocalName() {
		case "method":
			value, err := readSerializeParamValue(param)
			if err != nil {
				return opts, err
			}
			// method-type is a union of the built-in method tokens and a QName /
			// EQName naming an extension method (XSD-whitespace-collapsed).
			method := strings.Trim(value, xsdWhitespaceCutset)
			if !serializeMethodValid(method) {
				return opts, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize parameter method has an invalid value %q", value)}
			}
			opts.method = method
		case "item-separator":
			value, err := readSerializeParamValue(param)
			if err != nil {
				return opts, err
			}
			opts.itemSeparator = value
		case "indent":
			value, err := readSerializeParamYesNo(param)
			if err != nil {
				return opts, err
			}
			opts.indent = value
		case "omit-xml-declaration":
			value, err := readSerializeParamYesNo(param)
			if err != nil {
				return opts, err
			}
			opts.omitXMLDeclaration = value
		case "allow-duplicate-names":
			value, err := readSerializeParamYesNo(param)
			if err != nil {
				return opts, err
			}
			opts.allowDuplicateNames = value
		case "undeclare-prefixes":
			value, err := readSerializeParamYesNo(param)
			if err != nil {
				return opts, err
			}
			opts.undeclarePrefixes = value
		case "encoding":
			value, err := readSerializeParamValue(param)
			if err != nil {
				return opts, err
			}
			opts.encoding = value
		case "cdata-section-elements":
			names, err := readSerializeParamQNameNames(param)
			if err != nil {
				return opts, err
			}
			opts.cdataElements = names
		case "suppress-indentation":
			names, err := readSerializeParamQNameNames(param)
			if err != nil {
				return opts, err
			}
			opts.suppressIndent = names
		case "version":
			value, err := readSerializeParamValue(param)
			if err != nil {
				return opts, err
			}
			opts.xmlVersion = strings.Trim(value, xsdWhitespaceCutset)
		case "byte-order-mark":
			// A yes-no boolean helium does not apply (UTF-8 output emits no BOM);
			// validate the boolean lexical space and ignore.
			if _, err := readSerializeParamYesNo(param); err != nil {
				return opts, err
			}
		case "escape-uri-attributes":
			value, err := readSerializeParamYesNo(param)
			if err != nil {
				return opts, err
			}
			opts.escapeURIAttributes = value
		case "include-content-type":
			value, err := readSerializeParamYesNo(param)
			if err != nil {
				return opts, err
			}
			opts.includeContentType = value
		case "html-version":
			value, err := readSerializeParamValue(param)
			if err != nil {
				return opts, err
			}
			hv := strings.Trim(value, xsdWhitespaceCutset)
			if !isValidXSDecimal(hv) {
				return opts, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize parameter html-version must be an xs:decimal, got %q", value)}
			}
			opts.htmlVersion = hv
		case "doctype-public":
			value, err := readSerializeParamValue(param)
			if err != nil {
				return opts, err
			}
			opts.doctypePublic = value
		case "doctype-system":
			value, err := readSerializeParamValue(param)
			if err != nil {
				return opts, err
			}
			opts.doctypeSystem = value
		case "json-node-output-method":
			value, err := readSerializeParamValue(param)
			if err != nil {
				return opts, err
			}
			// json-node-output-method-type: xml|html|xhtml|text or an extension QName
			// (NOT json/adaptive — a narrower domain than the method parameter).
			jn := strings.Trim(value, xsdWhitespaceCutset)
			if !serializeJSONNodeOutputMethodValid(jn) {
				return opts, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize parameter json-node-output-method has an invalid value %q", value)}
			}
			opts.jsonNodeOutputMethod = jn
		case "normalization-form":
			value, err := readSerializeParamValue(param)
			if err != nil {
				return opts, err
			}
			nf := strings.Trim(value, xsdWhitespaceCutset)
			if !serializeNormalizationFormValid(nf) {
				return opts, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize parameter normalization-form has an invalid value %q", value)}
			}
			opts.normalizationForm = nf
		case "media-type":
			value, err := readSerializeParamValue(param)
			if err != nil {
				return opts, err
			}
			opts.mediaType = value
		case "standalone":
			value, err := readSerializeParamStandalone(param)
			if err != nil {
				return opts, err
			}
			opts.standalone = value
		case "use-character-maps":
			charMap, err := resolveSerializeCharacterMapsElement(param)
			if err != nil {
				return opts, err
			}
			opts.charMap = charMap
		default:
			return opts, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("unsupported serialize parameter %q", param.LocalName())}
		}
	}

	return opts, nil
}

// isXSDWhitespaceOnly reports whether s is empty or consists only of XSD
// whitespace (#x20/#x9/#xA/#xD). NBSP and other Unicode whitespace are NOT
// whitespace, so content containing them is significant.
func isXSDWhitespaceOnly(s string) bool {
	return strings.Trim(s, xsdWhitespaceCutset) == ""
}

// isValidXSDecimal reports whether s is a valid xs:decimal lexical form: an
// optional sign followed by digits with an optional fractional part (no
// exponent). Used to validate the html-version parameter.
func isValidXSDecimal(s string) bool {
	if s == "" {
		return false
	}
	if s[0] == '+' || s[0] == '-' {
		s = s[1:]
	}
	intPart, fracPart, _ := strings.Cut(s, ".")
	// A second "." (e.g. "5.0.0") leaves a "." in fracPart — reject it. A lone
	// "." with no digits either side is invalid; "5." and ".5" are valid.
	if strings.IndexByte(fracPart, '.') >= 0 {
		return false
	}
	if intPart == "" && fracPart == "" {
		return false
	}
	return allASCIIDigits(intPart) && allASCIIDigits(fracPart)
}

func allASCIIDigits(s string) bool {
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// isSupportedXMLOutputVersion reports whether v is an XML output version the
// serializer supports. helium serializes XML 1.0 and 1.1; any other value
// (including a malformed one or an unsupported version such as "1.2") is the
// SESU0013 unsupported-version error.
func isSupportedXMLOutputVersion(v string) bool {
	return v == "1.0" || v == "1.1"
}

// serializeNormalizationFormValid reports whether v is a valid normalization-form
// value: one of the Unicode normalization forms, "none", or empty.
func serializeNormalizationFormValid(v string) bool {
	switch v {
	case "", "none", "NFC", "NFD", "NFKC", "NFKD", "fully-normalized":
		return true
	}
	return false
}

// isBuiltinSerializeMethod reports whether v names a built-in output method that
// helium implements.
func isBuiltinSerializeMethod(v string) bool {
	switch v {
	case serializeMethodXML, serializeMethodHTML, serializeMethodXHTML, serializeMethodText, serializeMethodJSON, serializeMethodAdaptive:
		return true
	}
	return false
}

// isExtensionMethodName reports whether v is a syntactically valid extension
// method name: a lexical QName WITH a prefix (its expanded name has a non-null
// namespace), or an `Q{uri}local` EQName with a NON-EMPTY namespace URI. A bare
// NCName (null namespace) is NOT an extension method — per Serialization 3.1 the
// method-type restricts extension methods to prefixed QNames, so an unprefixed
// value must be one of the built-in tokens.
func isExtensionMethodName(v string) bool {
	if uri, local, ok := parseEQNameToken(v); ok {
		return uri != "" && xmlchar.IsValidNCName(local)
	}
	prefix, local, hasPrefix := strings.Cut(v, ":")
	if !hasPrefix {
		return false
	}
	return xmlchar.IsValidNCName(prefix) && xmlchar.IsValidNCName(local)
}

// serializeMethodValid reports whether v is a valid method-type value: a
// built-in output method, or a prefixed QName / non-null EQName naming an
// extension method. A bare non-built-in NCName (e.g. "bogus") is NOT valid.
// Note: passing validation does not imply the method is SUPPORTED — an extension
// method is valid but unimplemented, and fnSerialize rejects it at dispatch
// rather than silently falling through to the xml method.
func serializeMethodValid(v string) bool {
	return isBuiltinSerializeMethod(v) || isExtensionMethodName(v)
}

// serializeJSONNodeOutputMethodValid reports whether v is a valid
// json-node-output-method value. Per Serialization 3.1 §9.1 its domain is the
// tokens xml|html|xhtml|text (NOT json or adaptive) or a prefixed extension
// QName — a SEPARATE, narrower domain than the method parameter.
func serializeJSONNodeOutputMethodValid(v string) bool {
	switch v {
	case serializeMethodXML, serializeMethodHTML, serializeMethodXHTML, serializeMethodText:
		return true
	}
	return isExtensionMethodName(v)
}

func readSerializeParamValue(elem *helium.Element) (string, error) {
	if hasNonWhitespaceContent(elem) {
		return "", &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize parameter must not have child content"}
	}
	attrs := elem.Attributes()
	if len(attrs) != 1 {
		return "", &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize parameter must have exactly one value attribute"}
	}
	attr := attrs[0]
	if attr.URI() != "" || attr.LocalName() != "value" {
		return "", &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize parameter must use an unqualified value attribute"}
	}
	return attr.Value(), nil
}

// xsdWhitespaceCutset is the XSD list / whiteSpace-collapse whitespace set
// (#x20, #x9, #xA, #xD). Element-form enumeration/boolean/QNames values are
// token-derived, so leading/trailing runs of these characters are stripped —
// but NBSP and other Unicode whitespace are NOT whitespace and must remain in
// the value (where they then fail lexical validation).
const xsdWhitespaceCutset = " \t\r\n"

// serializeBooleanValue normalizes a Serialization 3.1 boolean lexical to
// "yes"/"no". The value space (after collapsing leading/trailing XSD whitespace)
// is the full xs:boolean lexical space plus the yes/no synonyms:
// {yes, no, true, false, 1, 0}. ok is false for any other value — including
// uppercase forms (xs:boolean is lowercase-only) and NBSP-padded values.
func serializeBooleanValue(value string) (string, bool) {
	switch strings.Trim(value, xsdWhitespaceCutset) {
	case lexicon.ValueYes, lexicon.ValueTrue, "1":
		return lexicon.ValueYes, true
	case lexicon.ValueNo, lexicon.ValueFalse, "0":
		return lexicon.ValueNo, true
	}
	return "", false
}

func readSerializeParamYesNo(elem *helium.Element) (bool, error) {
	value, err := readSerializeParamValue(elem)
	if err != nil {
		return false, err
	}
	norm, ok := serializeBooleanValue(value)
	if !ok {
		return false, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize parameter %q must be a boolean (yes/no/true/false/1/0)", elem.LocalName())}
	}
	return norm == lexicon.ValueYes, nil
}

// readSerializeParamStandalone reads and normalizes the element-form
// "standalone" parameter, whose schema type is yes-no-omit-type: the boolean
// lexical space {yes, no, true, false, 1, 0} plus "omit" (Serialization 3.1).
// The value is XSD-whitespace-collapsed, so " no " is the "no" value; it returns
// the normalized "yes"/"no"/"omit".
func readSerializeParamStandalone(elem *helium.Element) (string, error) {
	value, err := readSerializeParamValue(elem)
	if err != nil {
		return "", err
	}
	if strings.Trim(value, xsdWhitespaceCutset) == serializeStandaloneOmit {
		return serializeStandaloneOmit, nil
	}
	if norm, ok := serializeBooleanValue(value); ok {
		return norm, nil
	}
	return "", &XPathError{Code: lexicon.ErrXPTY0004, Message: `serialize parameter "standalone" must be yes/no/omit or a boolean (true/false/1/0)`}
}

// readSerializeParamQNameNames reads a cdata-section-elements or
// suppress-indentation element-form parameter whose value attribute is a
// whitespace-separated list of names, and returns the writer's exact expanded-
// name key set. Per the Serialization 3.1 schema each token is a QName OR an
// EQName: an `Q{uri}local` EQName supplies its namespace directly; a prefixed
// lexical QName resolves its prefix against the parameter element's in-scope
// namespaces; an UNPREFIXED lexical QName resolves through the in-scope DEFAULT
// namespace (these parameters use the default namespace for unprefixed QNames).
func readSerializeParamQNameNames(elem *helium.Element) (map[string]struct{}, error) {
	value, err := readSerializeParamValue(elem)
	if err != nil {
		return nil, err
	}
	names := make(map[string]struct{})
	// QNames-type is an xs:list, so tokens split ONLY on XSD list whitespace
	// (#x20/#x9/#xA/#xD) — NBSP and other Unicode whitespace stay inside the token
	// and then fail NCName validation, rather than being silently split.
	for _, token := range xsdListFields(value) {
		key, err := resolveSerializeNameToken(elem, token)
		if err != nil {
			return nil, err
		}
		names[key] = struct{}{}
	}
	return names, nil
}

// resolveSerializeNameToken resolves one cdata-section-elements /
// suppress-indentation name token (a QName or an `Q{uri}local` EQName) to its
// exact expanded {uri}local key. NCName parts are validated before resolution.
func resolveSerializeNameToken(elem *helium.Element, token string) (string, error) {
	badName := func() error {
		return &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize parameter has an invalid name %q", token)}
	}
	if uri, local, ok := parseEQNameToken(token); ok {
		if !xmlchar.IsValidNCName(local) {
			return "", badName()
		}
		return helium.ClarkName(uri, local), nil
	}
	prefix, local, hasPrefix := strings.Cut(token, ":")
	if !hasPrefix {
		// No colon: strings.Cut puts the whole token in prefix. Resolve through
		// the in-scope default namespace (absent → no namespace).
		if !xmlchar.IsValidNCName(prefix) {
			return "", badName()
		}
		uri, _ := lookupInScopeNamespace(elem, "")
		return helium.ClarkName(uri, prefix), nil
	}
	if !xmlchar.IsValidNCName(prefix) || !xmlchar.IsValidNCName(local) {
		return "", badName()
	}
	uri, ok := lookupInScopeNamespace(elem, prefix)
	if !ok {
		return "", &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize parameter uses unbound namespace prefix %q", prefix)}
	}
	return helium.ClarkName(uri, local), nil
}

// parseEQNameToken recognizes an EQName of the form `Q{uri}local`, returning the
// namespace URI and local name. Per the grammar `Q\{[^{}]*\}...` the URI part
// admits neither "{" nor "}"; the first "}" after "Q{" ends it (so the URI never
// contains "}"), and a "{" inside the URI part means the token is not a
// well-formed EQName (the caller then rejects it as an invalid name).
func parseEQNameToken(token string) (string, string, bool) {
	if !strings.HasPrefix(token, "Q{") {
		return "", "", false
	}
	uri, local, ok := strings.Cut(token[2:], "}")
	if !ok || strings.ContainsRune(uri, '{') {
		return "", "", false
	}
	return uri, local, true
}

// lookupInScopeNamespace resolves prefix against the in-scope namespace
// declarations of elem, walking up its ancestors. The reserved "xml" prefix is
// always bound (no declaration required). For a non-default prefix, an empty URI
// is an UNDECLARATION (XML 1.1 namespace masking): the prefix becomes unbound and
// the masking hides any ancestor binding. The default prefix ("") may be bound to
// the empty URI (xmlns="" = no default namespace), which is a valid state.
func lookupInScopeNamespace(elem helium.Node, prefix string) (string, bool) {
	if prefix == lexicon.PrefixXML {
		return lexicon.NamespaceXML, true
	}
	for node := elem; node != nil; node = node.Parent() {
		nser, ok := node.(helium.Namespacer)
		if !ok {
			continue
		}
		for _, ns := range nser.Namespaces() {
			if ns.Prefix() != prefix {
				continue
			}
			uri := ns.URI()
			if prefix != "" && uri == "" {
				// Undeclaration: the prefix is unbound from here up (masking).
				return "", false
			}
			return uri, true
		}
	}
	return "", false
}

// resolveSerializeStandaloneMap validates and resolves the "standalone" option
// value in the map form of fn:serialize. Its value space is
// union(xs:boolean, enum("omit")) (F&O 3.1 §18.9.1). The map form does NOT
// whitespace-collapse the value, so a string such as " omit " (with surrounding
// spaces) is not the "omit" enum value and is a type error [err:XPTY0004]. It
// returns "yes"/"no"/"omit", or "" to select the parameter default (omit).
func resolveSerializeStandaloneMap(ctx context.Context, v Sequence) (string, error) {
	xerr := &XPathError{Code: lexicon.ErrXPTY0004, Message: `serialize option "standalone" must be xs:boolean or "omit"`}
	// Option (function) conversion (F&O 3.1 §2.5): atomize FIRST so an array
	// (e.g. an empty-array member) flattens to its members, THEN apply the
	// singleton cardinality. An empty atomized value selects the parameter
	// default (omit); it is not a bad value (QT3 serialize-xml-131 supplies
	// map{"standalone":()}). Atomization is ctx-aware: an element-only-typed node
	// has no typed value, so it surfaces err:FOTY0012 rather than being masked as
	// the XPTY0004 bad-value error.
	atoms, err := atomizeTypedValue(ctx, v)
	if err != nil {
		if isNoTypedValueError(err) {
			return "", err
		}
		return "", xerr
	}
	if len(atoms) == 0 {
		return "", nil
	}
	if len(atoms) > 1 {
		return "", xerr
	}
	av := atoms[0]
	switch av.TypeName {
	case TypeBoolean:
		if av.BooleanVal() {
			return lexicon.ValueYes, nil
		}
		return lexicon.ValueNo, nil
	case TypeString, TypeUntypedAtomic:
		s, err := atomicToString(av)
		if err != nil {
			return "", xerr
		}
		if s == serializeStandaloneOmit {
			return serializeStandaloneOmit, nil
		}
		// An untypedAtomic may carry an xs:boolean lexical; accept the same
		// value space the other boolean map options do (readBool): true/false/1/0.
		// A TYPED xs:string is not an xs:boolean member of the union, and yes/no
		// are not xs:boolean lexicals — both stay rejected.
		if av.TypeName == TypeUntypedAtomic {
			switch s {
			case lexicon.ValueTrue, "1":
				return lexicon.ValueYes, nil
			case lexicon.ValueFalse, "0":
				return lexicon.ValueNo, nil
			}
		}
		return "", xerr
	default:
		return "", xerr
	}
}

// serializeNodeKindError reports err:SENR0001 when a bare attribute or namespace
// node is serialized under a markup output method (xml/xhtml/html/text or the
// unspecified default). Under the adaptive and json methods these node kinds are
// serialized specially, so callers must not apply this guard for those methods.
func serializeNodeKindError(item NodeItem) error {
	if _, ok := item.Node.(*helium.Attribute); ok {
		return &XPathError{Code: errCodeSENR0001, Message: "cannot serialize an attribute node using the xml output method"}
	}
	if item.Node != nil && item.Node.Type() == helium.NamespaceNode {
		return &XPathError{Code: errCodeSENR0001, Message: "cannot serialize a namespace node using the xml output method"}
	}
	return nil
}

// resolveSerializeCharacterMapsElement validates the element form of
// use-character-maps and builds the rune→replacement map from its
// output:character-map children.
func resolveSerializeCharacterMapsElement(elem *helium.Element) (map[rune]string, error) {
	if len(elem.Attributes()) != 0 {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize parameter use-character-maps must not have attributes"}
	}
	result := make(map[rune]string)
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			if isXSDWhitespaceOnly(string(child.Content())) {
				continue
			}
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "use-character-maps must not contain text"}
		case helium.CommentNode, helium.ProcessingInstructionNode:
			continue
		case helium.ElementNode:
		default:
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "use-character-maps has invalid child node"}
		}

		charMap, _ := helium.AsNode[*helium.Element](child)
		if charMap.URI() != lexicon.NamespaceSerialization || charMap.LocalName() != "character-map" {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "use-character-maps children must be output:character-map"}
		}
		if hasNonWhitespaceContent(charMap) {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "character-map must not have child content"}
		}

		// map-string is an xs:string: an absent OR empty value maps the character
		// to the EMPTY replacement (deletion), so presence is tracked separately
		// from value and an empty value is not rejected. Only the character
		// attribute is required (and must be a single character).
		var character, mapString string
		var hasCharacter bool
		for _, attr := range charMap.Attributes() {
			if attr.URI() != "" {
				return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "character-map attributes must be unqualified"}
			}
			switch attr.LocalName() {
			case "character":
				character = attr.Value()
				hasCharacter = true
			case "map-string":
				mapString = attr.Value()
			default:
				return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "character-map has unsupported attribute"}
			}
		}
		if !hasCharacter || utf8.RuneCountInString(character) != 1 {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "character-map character must be a single character"}
		}
		key := []rune(character)[0]
		if _, exists := result[key]; exists {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "character-map entries must be unique"}
		}
		result[key] = mapString
	}
	return result, nil
}

func hasNonWhitespaceContent(elem *helium.Element) bool {
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			if !isXSDWhitespaceOnly(string(child.Content())) {
				return true
			}
		case helium.CommentNode, helium.ProcessingInstructionNode:
			continue
		default:
			return true
		}
	}
	return false
}

// resolveSerializeCharacterMaps validates the map form of the use-character-maps
// option and builds the rune→replacement map. Keys must be single-character
// strings and values strings (option (function) conventions apply — a QName key
// is err:XPTY0004; W3C serialize-xml-139/139b).
func resolveSerializeCharacterMaps(v Sequence) (map[rune]string, error) {
	if seqLen(v) != 1 {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize option 'use-character-maps' must be a singleton map"}
	}
	m, ok := v.Get(0).(MapItem)
	if !ok {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize option 'use-character-maps' must be a map"}
	}
	charMap := make(map[rune]string)
	err := m.forEach0(func(key AtomicValue, value Sequence) error {
		if key.TypeName != TypeString && key.TypeName != TypeUntypedAtomic {
			return &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize use-character-maps keys must be strings"}
		}
		keyString, err := atomicToString(key)
		if err != nil || utf8.RuneCountInString(keyString) != 1 {
			return &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize use-character-maps keys must be single characters"}
		}
		if seqLen(value) != 1 {
			return &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize use-character-maps values must be singleton strings"}
		}
		av, ok := value.Get(0).(AtomicValue)
		if !ok || (av.TypeName != TypeString && av.TypeName != TypeUntypedAtomic) {
			return &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize use-character-maps values must be strings"}
		}
		mapString, err := atomicToString(av)
		if err != nil {
			return err
		}
		charMap[[]rune(keyString)[0]] = mapString
		return nil
	})
	if err != nil {
		return nil, err
	}
	return charMap, nil
}

func serializeAdaptiveSequence(seq Sequence, opts serializeOptions) (string, error) {
	parts := make([]string, 0, seqLen(seq))
	for item := range seqItems(seq) {
		s, err := serializeAdaptiveItem(item, opts)
		if err != nil {
			return "", err
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, opts.itemSeparator), nil
}

func serializeAdaptiveItem(item Item, opts serializeOptions) (string, error) {
	switch v := item.(type) {
	case AtomicValue:
		if v.TypeName == TypeBoolean {
			if v.BooleanVal() {
				return "true()", nil
			}
			return "false()", nil
		}
		s, err := atomicToString(v)
		if err != nil {
			return "", err
		}
		// Per XPath 3.1 §12.2 (Adaptive): string and untypedAtomic values
		// are serialized enclosed in double quotes, with internal quotes
		// escaped as "".
		if v.TypeName == TypeString || v.TypeName == TypeUntypedAtomic {
			escaped := strings.ReplaceAll(s, `"`, `""`)
			return `"` + escaped + `"`, nil
		}
		return s, nil
	case NodeItem:
		// The default (empty) method is the xml family, under which an
		// attribute or namespace node is a serialization error. An explicit
		// method="adaptive" serializes them specially, so only guard when the
		// method is not adaptive.
		if opts.method != serializeMethodAdaptive {
			if err := serializeNodeKindError(v); err != nil {
				return "", err
			}
		}
		return serializeNodeItem(v, opts)
	case MapItem:
		return serializeAdaptiveMap(v, opts)
	case ArrayItem:
		return serializeAdaptiveArray(v, opts)
	case FunctionItem:
		return "", &XPathError{Code: errCodeFOER0000, Message: "cannot serialize function item"}
	default:
		return "", &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("cannot serialize %T", item)}
	}
}

func serializeAdaptiveMap(m MapItem, opts serializeOptions) (string, error) {
	parts := make([]string, 0, m.Size())
	err := m.forEach0(func(key AtomicValue, value Sequence) error {
		keyText, err := serializeAdaptiveItem(key, opts)
		if err != nil {
			return err
		}
		valText, err := serializeAdaptiveSequence(value, serializeOptions{method: serializeMethodAdaptive, itemSeparator: ","})
		if err != nil {
			return err
		}
		parts = append(parts, keyText+":"+valText)
		return nil
	})
	if err != nil {
		return "", err
	}
	return "map{" + strings.Join(parts, ",") + "}", nil
}

func serializeAdaptiveArray(a ArrayItem, _ serializeOptions) (string, error) {
	parts := make([]string, 0, a.Size())
	for _, member := range a.members0() {
		text, err := serializeAdaptiveSequence(member, serializeOptions{method: serializeMethodAdaptive, itemSeparator: ","})
		if err != nil {
			return "", err
		}
		parts = append(parts, text)
	}
	return "[" + strings.Join(parts, ",") + "]", nil
}

func serializeJSONSequence(seq Sequence, opts serializeOptions) (string, error) {
	if seqLen(seq) > 1 {
		return "", &XPathError{Code: errCodeFOER0000, Message: "json serialization requires at most one top-level item"}
	}
	if seqLen(seq) == 0 {
		return jsonKindNull, nil
	}
	return serializeJSONItem(seq.Get(0), opts)
}

func serializeJSONItem(item Item, opts serializeOptions) (string, error) {
	switch v := item.(type) {
	case AtomicValue:
		return serializeJSONAtomic(v, opts)
	case ArrayItem:
		parts := make([]string, 0, v.Size())
		for _, member := range v.members0() {
			if seqLen(member) == 0 {
				parts = append(parts, jsonKindNull)
				continue
			}
			for memberItem := range seqItems(member) {
				text, err := serializeJSONItem(memberItem, opts)
				if err != nil {
					return "", err
				}
				parts = append(parts, text)
			}
		}
		return formatJSONComposite("[", "]", parts, 0, opts.indent), nil
	case MapItem:
		seen := make(map[string]struct{}, v.Size())
		parts := make([]string, 0, v.Size())
		err := v.forEach0(func(key AtomicValue, value Sequence) error {
			keyText, err := atomicToString(key)
			if err != nil {
				return &XPathError{Code: errCodeFOER0000, Message: "json serialization map keys must be stringifiable"}
			}
			if _, exists := seen[keyText]; exists && !opts.allowDuplicateNames {
				return &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("json serialization duplicate key %q", keyText)}
			}
			seen[keyText] = struct{}{}
			if seqLen(value) > 1 {
				return &XPathError{Code: errCodeFOER0000, Message: "json serialization map values must be singleton or empty"}
			}
			valText := jsonKindNull
			if seqLen(value) == 1 {
				valText, err = serializeJSONItem(value.Get(0), opts)
				if err != nil {
					return err
				}
			}
			parts = append(parts, `"`+encodeJSONStringContent(keyText)+`":`+valueSeparator(opts.indent)+valText)
			return nil
		})
		if err != nil {
			return "", err
		}
		return formatJSONComposite("{", "}", parts, 0, opts.indent), nil
	case NodeItem:
		// helium serializes a node embedded in JSON only with the xml method (the
		// json-node-output-method default). A non-default value (html/xhtml/text or
		// an extension) would change the node's serialization, which helium does
		// not implement — so rather than silently emitting xml, it is an explicit
		// unsupported-feature error (SEPM0016). The empty/"xml" default is honored.
		if m := opts.jsonNodeOutputMethod; m != "" && m != lexicon.PrefixXML {
			return "", &XPathError{Code: errCodeSEPM0016, Message: fmt.Sprintf("json-node-output-method %q is not supported for a node embedded in JSON", m)}
		}
		text, err := serializeNodeItem(v, serializeOptions{method: lexicon.PrefixXML, omitXMLDeclaration: true})
		if err != nil {
			return "", err
		}
		return `"` + encodeJSONStringContent(text) + `"`, nil
	case FunctionItem:
		return "", &XPathError{Code: errCodeFOER0000, Message: "cannot serialize function item as JSON"}
	default:
		return "", &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("cannot serialize %T as JSON", item)}
	}
}

func serializeJSONAtomic(v AtomicValue, opts serializeOptions) (string, error) {
	switch {
	case v.TypeName == TypeBoolean:
		if v.BooleanVal() {
			return "true", nil
		}
		return "false", nil
	case v.IsNumeric():
		s, err := atomicToString(v)
		if err != nil {
			return "", err
		}
		if s == "NaN" || s == "INF" || s == "-INF" {
			return "", &XPathError{Code: errCodeFOER0000, Message: "cannot serialize NaN or infinity as JSON"}
		}
		return s, nil
	default:
		s, err := atomicToString(v)
		if err != nil {
			return "", err
		}
		// Character maps (use-character-maps) are NOT applicable to the json
		// output method (Serialization 3.1 §9.1.11), so opts.charMap is
		// intentionally not consulted here — only JSON string escaping applies.
		return `"` + encodeJSONStringForSerialization(s, opts.encoding) + `"`, nil
	}
}

// serializeTextSequence implements the text output method (Serialization 3.1
// §8): the concatenation of the string values of the items, with the
// item-separator between adjacent items and no markup. A node contributes its
// string value (an attribute or namespace node its value — the text method has
// no SENR0001 node-kind restriction); an atomic its lexical form. Character maps
// apply to the resulting text.
func serializeTextSequence(seq Sequence, opts serializeOptions) (string, error) {
	parts := make([]string, 0, seqLen(seq))
	for item := range seqItems(seq) {
		s, err := serializeTextItem(item)
		if err != nil {
			return "", err
		}
		parts = append(parts, s)
	}
	return applyCharMapToString(strings.Join(parts, opts.itemSeparator), opts.charMap), nil
}

func serializeTextItem(item Item) (string, error) {
	switch v := item.(type) {
	case AtomicValue:
		return atomicToString(v)
	case NodeItem:
		return ixpath.StringValue(v.Node), nil
	case FunctionItem:
		return "", &XPathError{Code: errCodeFOER0000, Message: "cannot serialize function item"}
	default:
		// A map or array has no string value; it can only be serialized with the
		// json or adaptive method.
		return "", &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("cannot serialize %T using the text output method", item)}
	}
}

// applyCharMapToString substitutes each mapped rune in s with its replacement
// string (used by the text output method, where the whole output is text).
func applyCharMapToString(s string, charMap map[rune]string) string {
	if len(charMap) == 0 {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if repl, ok := charMap[r]; ok {
			b.WriteString(repl)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func serializeXMLSequence(seq Sequence, opts serializeOptions) (string, error) {
	parts := make([]string, 0, seqLen(seq))
	for item := range seqItems(seq) {
		switch v := item.(type) {
		case FunctionItem:
			return "", &XPathError{Code: errCodeFOER0000, Message: "cannot serialize function item"}
		case NodeItem:
			// xml/xhtml/html/text output methods cannot serialize a bare
			// attribute or namespace node [err:SENR0001].
			if err := serializeNodeKindError(v); err != nil {
				return "", err
			}
			text, err := serializeNodeItem(v, opts)
			if err != nil {
				return "", err
			}
			parts = append(parts, text)
		default:
			s, err := serializeAdaptiveItem(item, opts)
			if err != nil {
				return "", err
			}
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, opts.itemSeparator), nil
}

// newSerializeXMLWriter builds a helium.Writer configured from the serialization
// options for the xml output method: version, omit-xml-declaration, indent,
// standalone, undeclare-prefixes, cdata-section-elements, suppress-indentation,
// and character maps.
func newSerializeXMLWriter(opts serializeOptions) helium.Writer {
	writer := helium.NewWriter()
	// The effective output version (version param, default "1.0") drives the XML
	// declaration text AND the XML 1.1 escaping/undeclaration behavior, so they
	// stay consistent regardless of the source document's own version.
	writer = writer.OutputVersion(opts.effectiveSerializeVersion())
	if opts.omitXMLDeclaration {
		writer = writer.XMLDeclaration(false)
	}
	if opts.indent {
		writer = writer.Format(true)
	}
	switch opts.standalone {
	case lexicon.ValueYes:
		writer = writer.Standalone(true)
	case lexicon.ValueNo:
		writer = writer.Standalone(false)
	default:
		// "omit" (the default) or an empty-sequence value: force omission of the
		// standalone pseudo-attribute, overriding the source declaration.
		writer = writer.OmitStandalone()
	}
	// undeclare-prefixes is honored only at output version 1.1 (fnSerialize has
	// already rejected the 1.0 case with SEPM0010).
	if opts.undeclarePrefixes && opts.effectiveSerializeVersion() == "1.1" {
		writer = writer.AllowPrefixUndeclarations(true)
	}
	if len(opts.cdataElements) > 0 {
		writer = writer.CDATASectionElements(opts.cdataElements)
	}
	if len(opts.suppressIndent) > 0 {
		writer = writer.SuppressIndentElements(opts.suppressIndent)
	}
	if len(opts.charMap) > 0 {
		writer = writer.CharacterMap(opts.charMap)
	}
	return writer
}

// serializeHTMLSequence serializes a sequence under the html output method.
// A document node is serialized with an HTML5 <!DOCTYPE html> and a
// <meta http-equiv="Content-Type"> injected into <head>; any other node is
// serialized as an HTML fragment (no DOCTYPE). Character maps (use-character-maps)
// are applied to text and attribute content.
func serializeHTMLSequence(seq Sequence, opts serializeOptions) (string, error) {
	parts := make([]string, 0, seqLen(seq))
	for item := range seqItems(seq) {
		node, ok := item.(NodeItem)
		if !ok {
			s, err := serializeAdaptiveItem(item, opts)
			if err != nil {
				return "", err
			}
			parts = append(parts, s)
			continue
		}
		if err := serializeNodeKindError(node); err != nil {
			return "", err
		}
		text, err := serializeHTMLNode(node.Node, opts)
		if err != nil {
			return "", err
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, opts.itemSeparator), nil
}

func serializeHTMLNode(node helium.Node, opts serializeOptions) (string, error) {
	hw := htmlpkg.NewWriter().DefaultDTD(false).Format(false).PreserveCase(true).
		EscapeURIAttributes(opts.escapeURIAttributes).CharacterMap(opts.charMap)

	doc, ok := node.(*helium.Document)
	if !ok || !htmlDocumentElementIsHTML(doc) {
		// A non-document node, or a document whose document element is not <html>,
		// is serialized under the html output method WITHOUT the DOCTYPE and
		// WITHOUT meta injection (Serialization 3.1: those apply to an html-rooted
		// result). The input tree is not mutated. escape-uri-attributes still
		// applies.
		var buf strings.Builder
		if err := hw.WriteTo(&buf, node); err != nil {
			return "", err
		}
		return strings.TrimSuffix(buf.String(), "\n"), nil
	}

	// Work on a copy so the <meta> injection never mutates the caller's tree.
	clone, err := helium.CopyDoc(doc)
	if err != nil {
		return "", err
	}
	// include-content-type (default yes) controls the Content-Type meta injection.
	if opts.includeContentType {
		insertHTMLContentTypeMeta(clone, opts.encoding, opts.mediaType)
	}

	doctype := htmlDoctype(opts, clone)
	var buf strings.Builder
	doctypeEmitted := false
	for child := range helium.Children(clone) {
		if child.Type() == helium.DTDNode {
			continue
		}
		if child.Type() == helium.ElementNode && !doctypeEmitted {
			buf.WriteString(doctype)
			doctypeEmitted = true
		}
		if err := hw.WriteTo(&buf, child); err != nil {
			return "", err
		}
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

// htmlDoctype computes the DOCTYPE declaration (with trailing newline) for the
// html output method: an explicit doctype-public/doctype-system produces a
// PUBLIC/SYSTEM declaration; otherwise HTML5 (html-version ≥ 5, the default)
// produces `<!DOCTYPE html>` and HTML 4 produces the HTML 4.01 declaration.
func htmlDoctype(opts serializeOptions, doc *helium.Document) string {
	if opts.doctypePublic != "" || opts.doctypeSystem != "" {
		rootName := serializeMethodHTML
		if root := doc.DocumentElement(); root != nil {
			rootName = root.Name()
		}
		var b strings.Builder
		b.WriteString("<!DOCTYPE ")
		b.WriteString(rootName)
		if opts.doctypePublic != "" {
			b.WriteString(` PUBLIC "`)
			b.WriteString(opts.doctypePublic)
			b.WriteString(`"`)
			if opts.doctypeSystem != "" {
				b.WriteString(` "`)
				b.WriteString(opts.doctypeSystem)
				b.WriteString(`"`)
			}
		} else {
			b.WriteString(` SYSTEM "`)
			b.WriteString(opts.doctypeSystem)
			b.WriteString(`"`)
		}
		b.WriteString(">\n")
		return b.String()
	}
	if serializeHTMLIsVersion5(opts) {
		return "<!DOCTYPE html>\n"
	}
	return `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd">` + "\n"
}

// serializeHTMLIsVersion5 reports whether the effective html-version selects
// HTML5 serialization (version ≥ 5). The default (unspecified) html-version is
// 5.0; an unparseable value is treated permissively as HTML5.
func serializeHTMLIsVersion5(opts serializeOptions) bool {
	if opts.htmlVersion == "" {
		return true
	}
	v, err := strconv.ParseFloat(opts.htmlVersion, 64)
	if err != nil {
		return true
	}
	return v >= 5
}

// htmlDocumentElementIsHTML reports whether the document element's local name is
// "html" (case-insensitive) — the condition under which the html output method
// emits the HTML5 DOCTYPE and injects the Content-Type meta element.
func htmlDocumentElementIsHTML(doc *helium.Document) bool {
	root := doc.DocumentElement()
	return root != nil && strings.EqualFold(root.LocalName(), "html")
}

// insertHTMLContentTypeMeta discards EVERY existing Content-Type meta element in
// the document's <head> and inserts a freshly-computed
// <meta http-equiv="Content-Type" content="{media-type}; charset={encoding}"> as
// the head's first child (Serialization 3.1 §4/§7 include-content-type: an
// existing meta is discarded and the computed one added). The encoding defaults
// to UTF-8 and the media type to text/html. It mutates the given document, so
// callers pass a copy.
func insertHTMLContentTypeMeta(doc *helium.Document, encoding, mediaType string) {
	root := doc.DocumentElement()
	if root == nil {
		return
	}
	var head *helium.Element
	for child := range helium.Children(root) {
		e, ok := child.(*helium.Element)
		if ok && strings.EqualFold(e.LocalName(), "head") {
			head = e
			break
		}
	}
	if head == nil {
		return
	}
	enc := encoding
	if enc == "" {
		enc = lexicon.EncodingUTF8U
	}
	mt := mediaType
	if mt == "" {
		mt = "text/html"
	}
	contentValue := mt + "; charset=" + enc

	// Discard every stale Content-Type meta (matched case-insensitively and
	// whitespace-trimmed) so no conflicting declaration survives, regardless of
	// its position or how many exist.
	removeHTMLContentTypeMetas(head)

	meta := doc.CreateElement("meta")
	if headURI := head.URI(); headURI != "" {
		_ = meta.SetActiveNamespace(head.Prefix(), headURI)
	}
	_ = meta.SetLiteralAttribute("http-equiv", "Content-Type")
	_ = meta.SetLiteralAttribute("content", contentValue)

	// Detach the current children, add meta first, then re-add them, so meta
	// becomes the head's first child.
	var children []helium.Node
	for child := head.FirstChild(); child != nil; {
		next := child.NextSibling()
		mut, ok := child.(helium.MutableNode)
		if ok {
			helium.UnlinkNode(mut)
			children = append(children, child)
		}
		child = next
	}
	_ = head.AddChild(meta)
	for _, child := range children {
		_ = head.AddChild(child)
	}
}

// removeHTMLContentTypeMetas unlinks every <meta http-equiv="Content-Type"> child
// of head so a stale Content-Type declaration cannot survive the freshly-inserted
// one.
func removeHTMLContentTypeMetas(head *helium.Element) {
	for child := head.FirstChild(); child != nil; {
		next := child.NextSibling()
		if isHTMLContentTypeMeta(child) {
			if mut, ok := child.(helium.MutableNode); ok {
				helium.UnlinkNode(mut)
			}
		}
		child = next
	}
}

// isHTMLContentTypeMeta reports whether n is a <meta> element whose http-equiv
// attribute is "Content-Type", compared case-insensitively after trimming
// surrounding whitespace so " Content-Type " and "CONTENT-TYPE" both match.
func isHTMLContentTypeMeta(n helium.Node) bool {
	e, ok := n.(*helium.Element)
	if !ok || !strings.EqualFold(e.LocalName(), "meta") {
		return false
	}
	for _, attr := range e.Attributes() {
		if attr.URI() != "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(attr.LocalName()), "http-equiv") &&
			strings.EqualFold(strings.TrimSpace(attr.Value()), "Content-Type") {
			return true
		}
	}
	return false
}

func serializeNodeItem(item NodeItem, opts serializeOptions) (string, error) {
	if attr, ok := item.Node.(*helium.Attribute); ok {
		return fmt.Sprintf(`%s="%s"`, attr.Name(), attr.Value()), nil
	}
	node := item.Node
	// The xml method emits a document type declaration when serializing a
	// document node ONLY if doctype-system is specified — per Serialization 3.1
	// §5.1 the doctype-public parameter MUST be ignored unless doctype-system is
	// also present (public is an optional additive). It is injected as an
	// internal subset on a COPY so the caller's tree is never mutated.
	if doc, ok := node.(*helium.Document); ok && opts.doctypeSystem != "" {
		withDT, err := documentWithDoctype(doc, opts)
		if err != nil {
			return "", err
		}
		node = withDT
	}
	var buf strings.Builder
	if err := newSerializeXMLWriter(opts).WriteTo(&buf, node); err != nil {
		return "", err
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

// documentWithDoctype returns a copy of doc carrying an internal-subset DTD with
// the requested doctype-public / doctype-system identifiers (named after the
// document element), so the XML writer emits `<!DOCTYPE name PUBLIC/SYSTEM ...>`.
// A document that already has a DTD is returned unchanged (a second declaration
// cannot be emitted).
func documentWithDoctype(doc *helium.Document, opts serializeOptions) (*helium.Document, error) {
	clone, err := helium.CopyDoc(doc)
	if err != nil {
		return nil, err
	}
	if clone.IntSubset() != nil {
		return clone, nil
	}
	root := clone.DocumentElement()
	if root == nil {
		return clone, nil
	}
	if _, err := clone.CreateInternalSubset(root.Name(), opts.doctypePublic, opts.doctypeSystem); err != nil {
		return nil, err
	}
	return clone, nil
}

func encodeJSONStringForSerialization(s, encoding string) string {
	if encoding == "" || strings.EqualFold(encoding, "utf-8") || strings.EqualFold(encoding, "utf8") {
		return encodeJSONStringContent(s)
	}

	var b strings.Builder
	for _, r := range s {
		switch {
		case r <= 0x7F:
			appendSerializedJSONStringRune(&b, r)
		case r <= 0xFFFF:
			fmt.Fprintf(&b, `\u%04X`, r)
		default:
			hi, lo := utf16SurrogatePair(r)
			fmt.Fprintf(&b, `\u%04X\u%04X`, hi, lo)
		}
	}
	return b.String()
}

func utf16SurrogatePair(r rune) (uint16, uint16) {
	cp := uint32(r - 0x10000)
	hi := uint16(0xD800 + (cp >> 10))
	lo := uint16(0xDC00 + (cp & 0x3FF))
	return hi, lo
}
