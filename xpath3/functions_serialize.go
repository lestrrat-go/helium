package xpath3

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	htmlpkg "github.com/lestrrat-go/helium/html"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

const serializeMethodHTML = "html"

func init() {
	registerFn("serialize", 1, 2, fnSerialize)
}

func fnSerialize(ctx context.Context, args []Sequence) (Sequence, error) {
	opts, err := parseSerializeOptions(ctx, args)
	if err != nil {
		return nil, err
	}

	var result string
	switch opts.method {
	case "", "adaptive":
		result, err = serializeAdaptiveSequence(args[0], opts)
	case "json":
		result, err = serializeJSONSequence(args[0], opts)
	case serializeMethodHTML:
		result, err = serializeHTMLSequence(args[0], opts)
	default:
		result, err = serializeXMLSequence(args[0], opts)
	}
	if err != nil {
		return nil, err
	}
	return SingleString(result), nil
}

type serializeOptions struct {
	method              string
	itemSeparator       string
	indent              bool
	omitXMLDeclaration  bool
	allowDuplicateNames bool
	encoding            string
	// standalone is the resolved standalone pseudo-attribute request:
	// "yes"/"no" force it, "omit" or "" leave it out.
	standalone string
	// undeclarePrefixes emits XML 1.1 namespace undeclarations (xmlns:pfx="").
	undeclarePrefixes bool
	// cdataElements / suppressIndent hold element names (expanded {uri}local and
	// bare local form) for the cdata-section-elements and suppress-indentation
	// serialization parameters.
	cdataElements  map[string]struct{}
	suppressIndent map[string]struct{}
	// charMap is the resolved character map (use-character-maps).
	charMap map[rune]string
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

	if method, found, err := readSerializeStringOption(ctx, m, "method"); err != nil {
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
		names, err := resolveSerializeQNameNames(v, "cdata-section-elements")
		if err != nil {
			return opts, err
		}
		opts.cdataElements = names
	}
	if v, found := m.Get(AtomicValue{TypeName: TypeString, Value: "suppress-indentation"}); found {
		names, err := resolveSerializeQNameNames(v, "suppress-indentation")
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
// expanded {uri}local name (a no-namespace QName contributes its bare local name).
func resolveSerializeQNameNames(v Sequence, name string) (map[string]struct{}, error) {
	names := make(map[string]struct{})
	for item := range seqItems(v) {
		av, ok := item.(AtomicValue)
		if !ok || av.TypeName != TypeQName {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option %q must be a sequence of xs:QName", name)}
		}
		qn, ok := av.Value.(QNameValue)
		if !ok {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize option %q must be a sequence of xs:QName", name)}
		}
		names[nameKey(qn.URI, qn.Local)] = struct{}{}
	}
	return names, nil
}

// nameKey builds the writer's element-name key: the bare local name for the
// no-namespace case, otherwise the expanded {uri}local form.
func nameKey(uri, local string) string {
	if uri == "" {
		return local
	}
	return "{" + uri + "}" + local
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
			if strings.TrimSpace(string(child.Content())) == "" {
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
			opts.method = value
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
		case "byte-order-mark", "doctype-public", "doctype-system",
			"json-node-output-method", "media-type", "normalization-form",
			"version":
			if _, err := readSerializeParamValue(param); err != nil {
				return opts, err
			}
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

func readSerializeParamYesNo(elem *helium.Element) (bool, error) {
	value, err := readSerializeParamValue(elem)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case lexicon.ValueYes, "true", "1":
		return true, nil
	case lexicon.ValueNo, "false", "0":
		return false, nil
	default:
		return false, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize parameter %q must be yes/no", elem.LocalName())}
	}
}

// readSerializeParamStandalone reads and normalizes the element-form
// "standalone" parameter. The element form whitespace-collapses the value
// (Serialization 3.1), so " no " is the "no" enum value; it returns the
// collapsed "yes"/"no"/"omit".
func readSerializeParamStandalone(elem *helium.Element) (string, error) {
	value, err := readSerializeParamValue(elem)
	if err != nil {
		return "", err
	}
	switch collapsed := strings.ToLower(strings.TrimSpace(value)); collapsed {
	case lexicon.ValueYes, lexicon.ValueNo, "omit":
		return collapsed, nil
	default:
		return "", &XPathError{Code: lexicon.ErrXPTY0004, Message: "serialize parameter \"standalone\" must be yes/no/omit"}
	}
}

// readSerializeParamQNameNames reads a cdata-section-elements or
// suppress-indentation element-form parameter whose value attribute is a
// whitespace-separated list of lexical QNames, resolving each prefix against the
// parameter element's in-scope namespaces, and returns the writer's element-name
// key set.
func readSerializeParamQNameNames(elem *helium.Element) (map[string]struct{}, error) {
	value, err := readSerializeParamValue(elem)
	if err != nil {
		return nil, err
	}
	names := make(map[string]struct{})
	for token := range strings.FieldsSeq(value) {
		prefix, local, hasPrefix := strings.Cut(token, ":")
		if !hasPrefix {
			// No colon: strings.Cut puts the whole token in prefix; it is a
			// no-namespace name whose key is its bare local name.
			names[prefix] = struct{}{}
			continue
		}
		uri, ok := lookupInScopeNamespace(elem, prefix)
		if !ok {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("serialize parameter uses unbound namespace prefix %q", prefix)}
		}
		names[nameKey(uri, local)] = struct{}{}
	}
	return names, nil
}

// lookupInScopeNamespace resolves prefix against the in-scope namespace
// declarations of elem, walking up its ancestors.
func lookupInScopeNamespace(elem helium.Node, prefix string) (string, bool) {
	for node := elem; node != nil; node = node.Parent() {
		nser, ok := node.(helium.Namespacer)
		if !ok {
			continue
		}
		for _, ns := range nser.Namespaces() {
			if ns.Prefix() == prefix {
				return ns.URI(), true
			}
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
		if s == "omit" {
			return "omit", nil
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
			if strings.TrimSpace(string(child.Content())) == "" {
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

		var character, mapString string
		for _, attr := range charMap.Attributes() {
			if attr.URI() != "" {
				return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "character-map attributes must be unqualified"}
			}
			switch attr.LocalName() {
			case "character":
				character = attr.Value()
			case "map-string":
				mapString = attr.Value()
			default:
				return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "character-map has unsupported attribute"}
			}
		}
		if character == "" || mapString == "" {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "character-map requires character and map-string"}
		}
		if utf8.RuneCountInString(character) != 1 {
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
			if strings.TrimSpace(string(child.Content())) != "" {
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
		if opts.method != "adaptive" {
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
		valText, err := serializeAdaptiveSequence(value, serializeOptions{method: "adaptive", itemSeparator: ","})
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
		text, err := serializeAdaptiveSequence(member, serializeOptions{method: "adaptive", itemSeparator: ","})
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
		return `"` + encodeJSONStringForSerialization(s, opts.encoding) + `"`, nil
	}
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
// options for the xml output method: omit-xml-declaration, indent, standalone,
// undeclare-prefixes, cdata-section-elements, suppress-indentation, and character
// maps.
func newSerializeXMLWriter(opts serializeOptions) helium.Writer {
	writer := helium.NewWriter()
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
	}
	if opts.undeclarePrefixes {
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
// serialized as an HTML fragment (no DOCTYPE). Character maps are not applied on
// the HTML path.
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
	doc, ok := node.(*helium.Document)
	if !ok {
		// A non-document node is an HTML fragment: serialize without a DOCTYPE
		// and without mutating the input tree.
		var buf strings.Builder
		hw := htmlpkg.NewWriter().DefaultDTD(false).Format(false).PreserveCase(true)
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
	insertHTMLContentTypeMeta(clone, opts.encoding)

	var buf strings.Builder
	hw := htmlpkg.NewWriter().DefaultDTD(false).Format(false).PreserveCase(true)
	doctypeEmitted := false
	for child := range helium.Children(clone) {
		if child.Type() == helium.DTDNode {
			continue
		}
		if child.Type() == helium.ElementNode && !doctypeEmitted {
			buf.WriteString("<!DOCTYPE html>\n")
			doctypeEmitted = true
		}
		if err := hw.WriteTo(&buf, child); err != nil {
			return "", err
		}
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

// insertHTMLContentTypeMeta inserts a <meta http-equiv="Content-Type"> element
// as the first child of the document's <head>, unless one already exists. The
// encoding defaults to UTF-8. It mutates the given document, so callers pass a
// copy.
func insertHTMLContentTypeMeta(doc *helium.Document, encoding string) {
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
	if htmlHeadHasContentTypeMeta(head) {
		return
	}
	enc := encoding
	if enc == "" {
		enc = lexicon.EncodingUTF8U
	}
	meta := doc.CreateElement("meta")
	if headURI := head.URI(); headURI != "" {
		_ = meta.SetActiveNamespace(head.Prefix(), headURI)
	}
	_ = meta.SetLiteralAttribute("http-equiv", "Content-Type")
	_ = meta.SetLiteralAttribute("content", "text/html; charset="+enc)

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

func htmlHeadHasContentTypeMeta(head *helium.Element) bool {
	for child := range helium.Children(head) {
		e, ok := child.(*helium.Element)
		if !ok || !strings.EqualFold(e.LocalName(), "meta") {
			continue
		}
		for _, attr := range e.Attributes() {
			if strings.EqualFold(attr.Name(), "http-equiv") && strings.EqualFold(attr.Value(), "Content-Type") {
				return true
			}
		}
	}
	return false
}

func serializeNodeItem(item NodeItem, opts serializeOptions) (string, error) {
	if attr, ok := item.Node.(*helium.Attribute); ok {
		return fmt.Sprintf(`%s="%s"`, attr.Name(), attr.Value()), nil
	}
	var buf strings.Builder
	if err := newSerializeXMLWriter(opts).WriteTo(&buf, item.Node); err != nil {
		return "", err
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
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
