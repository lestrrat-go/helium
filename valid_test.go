package helium

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// buildTestDoc creates a Document with intSubset and extSubset for testing.
// The intSubset declares element "root" with ANY content.
// The extSubset declares element "child" with EMPTY content, an entity "extEnt",
// and a REQUIRED attribute "role" on element "child".
func buildTestDoc(standalone DocumentStandaloneType) *Document {
	doc := NewDocument("1.0", "utf-8", standalone)

	// Internal subset: declares "root" element with ANY content.
	intDTD := newDTD()
	intDTD.doc = doc
	intDTD.etype = DTDNode
	doc.intSubset = intDTD

	rootDecl := newElementDecl()
	rootDecl.name = "root"
	rootDecl.decltype = AnyElementType
	rootDecl.doc = doc
	intDTD.elements = map[string]*ElementDecl{
		"root:": rootDecl,
	}
	intDTD.entities = map[string]*Entity{}
	intDTD.pentities = map[string]*Entity{}
	intDTD.attributes = map[string]*AttributeDecl{}

	// External subset: declares "child" element, entity, and attribute.
	extDTD := newDTD()
	extDTD.doc = doc
	extDTD.etype = DTDNode
	doc.extSubset = extDTD

	childDecl := newElementDecl()
	childDecl.name = "child"
	childDecl.decltype = EmptyElementType
	childDecl.doc = doc
	extDTD.elements = map[string]*ElementDecl{
		"child:": childDecl,
	}

	extEnt := newEntity("extEnt", InternalGeneralEntity, "", "", "hello", "")
	extEnt.doc = doc
	extDTD.entities = map[string]*Entity{
		"extEnt": extEnt,
	}
	extDTD.pentities = map[string]*Entity{}

	attrDecl := newAttributeDecl()
	attrDecl.name = "role"
	attrDecl.elem = "child"
	attrDecl.atype = AttrCDATA
	attrDecl.def = AttrDefaultRequired
	attrDecl.doc = doc
	extDTD.attributes = map[string]*AttributeDecl{
		"role::child": attrDecl,
	}

	return doc
}

func TestExtSubsetLookup_ElementInExtSubset(t *testing.T) {
	doc := buildTestDoc(StandaloneImplicitNo)

	root := newElement("root")
	root.doc = doc
	child := newElement("child")
	child.doc = doc
	_ = child.SetAttribute("role", "main")
	_ = root.AddChild(child)
	_ = doc.AddChild(root)

	ve := validateDocument(doc)
	require.Nil(t, ve, "validation should pass when element is declared in extSubset")
}

func TestExtSubsetLookup_EntityInExtSubset(t *testing.T) {
	doc := buildTestDoc(StandaloneImplicitNo)

	ent, found := doc.GetEntity("extEnt")
	require.True(t, found, "entity in extSubset should be found")
	require.Equal(t, "hello", string(ent.Content()))
}

func TestExtSubsetLookup_AttributeInExtSubset(t *testing.T) {
	doc := buildTestDoc(StandaloneImplicitNo)

	root := newElement("root")
	root.doc = doc
	child := newElement("child")
	child.doc = doc
	// Missing required "role" attribute
	_ = root.AddChild(child)
	_ = doc.AddChild(root)

	ve := validateDocument(doc)
	require.NotNil(t, ve, "validation should report missing REQUIRED attribute from extSubset")
	require.Contains(t, ve.Error(), "attribute role is required")
}

func TestExtSubsetLookup_StandaloneYesPreventsExtSubset(t *testing.T) {
	doc := buildTestDoc(StandaloneExplicitYes)

	// Entity lookup should NOT fall through to extSubset
	_, found := doc.GetEntity("extEnt")
	require.False(t, found, "standalone=yes should prevent extSubset entity lookup")

	// Element declared only in extSubset should not be found
	root := newElement("root")
	root.doc = doc
	child := newElement("child")
	child.doc = doc
	_ = child.SetAttribute("role", "main")
	_ = root.AddChild(child)
	_ = doc.AddChild(root)

	ve := validateDocument(doc)
	require.NotNil(t, ve, "standalone=yes should prevent finding element in extSubset")
	require.Contains(t, ve.Error(), "no declaration found")
}

func TestEnumerationAttributeValidation(t *testing.T) {
	t.Run("valid value accepted", func(t *testing.T) {
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root color (red|green|blue) #REQUIRED>
]>
<root color="green"/>`
		p := NewParser()
		p.SetOption(ParseDTDValid)
		p.SetOption(ParseDTDAttr)
		_, err := p.Parse([]byte(xml))
		require.NoError(t, err)
	})

	t.Run("invalid value rejected", func(t *testing.T) {
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root color (red|green|blue) #REQUIRED>
]>
<root color="yellow"/>`
		p := NewParser()
		p.SetOption(ParseDTDValid)
		p.SetOption(ParseDTDAttr)
		_, err := p.Parse([]byte(xml))
		require.Error(t, err)
		require.Contains(t, err.Error(), "not among the enumerated set")
	})

	t.Run("default value used when absent", func(t *testing.T) {
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root color (red|green|blue) "red">
]>
<root/>`
		p := NewParser()
		p.SetOption(ParseDTDValid)
		p.SetOption(ParseDTDAttr)
		_, err := p.Parse([]byte(xml))
		require.NoError(t, err)
	})
}

// Helper functions for building ElementContent trees in tests.

func ecElem(name string) *ElementContent {
	return &ElementContent{ctype: ElementContentElement, coccur: ElementContentOnce, name: name}
}

func ecElemOpt(name string) *ElementContent {
	return &ElementContent{ctype: ElementContentElement, coccur: ElementContentOpt, name: name}
}

func ecElemStar(name string) *ElementContent {
	return &ElementContent{ctype: ElementContentElement, coccur: ElementContentMult, name: name}
}

func ecElemPlus(name string) *ElementContent {
	return &ElementContent{ctype: ElementContentElement, coccur: ElementContentPlus, name: name}
}

// ecSeq builds a sequence node with the given occur from a list of parts.
// Parts are linked as a right-nested c1/c2 chain.
func ecSeq(occur ElementContentOccur, parts ...*ElementContent) *ElementContent {
	if len(parts) == 0 {
		return &ElementContent{ctype: ElementContentSeq, coccur: occur}
	}
	if len(parts) == 1 {
		return &ElementContent{ctype: ElementContentSeq, coccur: occur, c1: parts[0]}
	}
	root := &ElementContent{ctype: ElementContentSeq, coccur: occur, c1: parts[0]}
	cur := root
	for i := 1; i < len(parts)-1; i++ {
		next := &ElementContent{ctype: ElementContentSeq, coccur: ElementContentOnce, c1: parts[i]}
		cur.c2 = next
		cur = next
	}
	cur.c2 = parts[len(parts)-1]
	return root
}

// ecOr builds a choice node with the given occur from a list of alternatives.
// Alternatives are linked as a right-nested c1/c2 chain.
func ecOr(occur ElementContentOccur, alts ...*ElementContent) *ElementContent {
	if len(alts) == 0 {
		return &ElementContent{ctype: ElementContentOr, coccur: occur}
	}
	if len(alts) == 1 {
		return &ElementContent{ctype: ElementContentOr, coccur: occur, c1: alts[0]}
	}
	root := &ElementContent{ctype: ElementContentOr, coccur: occur, c1: alts[0]}
	cur := root
	for i := 1; i < len(alts)-1; i++ {
		next := &ElementContent{ctype: ElementContentOr, coccur: ElementContentOnce, c1: alts[i]}
		cur.c2 = next
		cur = next
	}
	cur.c2 = alts[len(alts)-1]
	return root
}

func TestMatchContentModel(t *testing.T) {
	tests := []struct {
		name     string
		content  *ElementContent
		children []string
		want     bool
	}{
		// Simple sequence: (a, b, c)
		{
			name:     "seq/exact match",
			content:  ecSeq(ElementContentOnce, ecElem("a"), ecElem("b"), ecElem("c")),
			children: []string{"a", "b", "c"},
			want:     true,
		},
		{
			name:     "seq/wrong order",
			content:  ecSeq(ElementContentOnce, ecElem("a"), ecElem("b"), ecElem("c")),
			children: []string{"a", "c", "b"},
			want:     false,
		},
		{
			name:     "seq/missing element",
			content:  ecSeq(ElementContentOnce, ecElem("a"), ecElem("b"), ecElem("c")),
			children: []string{"a", "b"},
			want:     false,
		},
		{
			name:     "seq/extra element rejected",
			content:  ecSeq(ElementContentOnce, ecElem("a"), ecElem("b"), ecElem("c")),
			children: []string{"a", "b", "c", "d"},
			want:     false,
		},

		// Choice: (a | b | c)
		{
			name:     "choice/first alt",
			content:  ecOr(ElementContentOnce, ecElem("a"), ecElem("b"), ecElem("c")),
			children: []string{"a"},
			want:     true,
		},
		{
			name:     "choice/second alt",
			content:  ecOr(ElementContentOnce, ecElem("a"), ecElem("b"), ecElem("c")),
			children: []string{"b"},
			want:     true,
		},
		{
			name:     "choice/third alt",
			content:  ecOr(ElementContentOnce, ecElem("a"), ecElem("b"), ecElem("c")),
			children: []string{"c"},
			want:     true,
		},
		{
			name:     "choice/no match",
			content:  ecOr(ElementContentOnce, ecElem("a"), ecElem("b"), ecElem("c")),
			children: []string{"d"},
			want:     false,
		},
		{
			name:     "choice/extra unconsumed rejected",
			content:  ecOr(ElementContentOnce, ecElem("a"), ecElem("b"), ecElem("c")),
			children: []string{"a", "b"},
			want:     false,
		},

		// Optional element: (a, b?, c)
		{
			name:     "optional/with optional present",
			content:  ecSeq(ElementContentOnce, ecElem("a"), ecElemOpt("b"), ecElem("c")),
			children: []string{"a", "b", "c"},
			want:     true,
		},
		{
			name:     "optional/with optional absent",
			content:  ecSeq(ElementContentOnce, ecElem("a"), ecElemOpt("b"), ecElem("c")),
			children: []string{"a", "c"},
			want:     true,
		},

		// Star repetition: (a, b*, c)
		{
			name:     "star/zero occurrences",
			content:  ecSeq(ElementContentOnce, ecElem("a"), ecElemStar("b"), ecElem("c")),
			children: []string{"a", "c"},
			want:     true,
		},
		{
			name:     "star/one occurrence",
			content:  ecSeq(ElementContentOnce, ecElem("a"), ecElemStar("b"), ecElem("c")),
			children: []string{"a", "b", "c"},
			want:     true,
		},
		{
			name:     "star/multiple occurrences",
			content:  ecSeq(ElementContentOnce, ecElem("a"), ecElemStar("b"), ecElem("c")),
			children: []string{"a", "b", "b", "b", "c"},
			want:     true,
		},

		// Plus repetition: (a, b+, c)
		{
			name:     "plus/zero occurrences fails",
			content:  ecSeq(ElementContentOnce, ecElem("a"), ecElemPlus("b"), ecElem("c")),
			children: []string{"a", "c"},
			want:     false,
		},
		{
			name:     "plus/one occurrence",
			content:  ecSeq(ElementContentOnce, ecElem("a"), ecElemPlus("b"), ecElem("c")),
			children: []string{"a", "b", "c"},
			want:     true,
		},
		{
			name:     "plus/multiple occurrences",
			content:  ecSeq(ElementContentOnce, ecElem("a"), ecElemPlus("b"), ecElem("c")),
			children: []string{"a", "b", "b", "c"},
			want:     true,
		},

		// Nested sequence: ((a, b), c)
		{
			name: "nested seq/match",
			content: ecSeq(ElementContentOnce,
				ecSeq(ElementContentOnce, ecElem("a"), ecElem("b")),
				ecElem("c"),
			),
			children: []string{"a", "b", "c"},
			want:     true,
		},
		{
			name: "nested seq/missing inner",
			content: ecSeq(ElementContentOnce,
				ecSeq(ElementContentOnce, ecElem("a"), ecElem("b")),
				ecElem("c"),
			),
			children: []string{"a", "c"},
			want:     false,
		},

		// Nested choice: (a, (b | c), d)
		{
			name: "nested choice/first alt",
			content: ecSeq(ElementContentOnce,
				ecElem("a"),
				ecOr(ElementContentOnce, ecElem("b"), ecElem("c")),
				ecElem("d"),
			),
			children: []string{"a", "b", "d"},
			want:     true,
		},
		{
			name: "nested choice/second alt",
			content: ecSeq(ElementContentOnce,
				ecElem("a"),
				ecOr(ElementContentOnce, ecElem("b"), ecElem("c")),
				ecElem("d"),
			),
			children: []string{"a", "c", "d"},
			want:     true,
		},
		{
			name: "nested choice/no match",
			content: ecSeq(ElementContentOnce,
				ecElem("a"),
				ecOr(ElementContentOnce, ecElem("b"), ecElem("c")),
				ecElem("d"),
			),
			children: []string{"a", "x", "d"},
			want:     false,
		},

		// Repeated sequence: (a, b)+
		{
			name:     "seq plus/one rep",
			content:  ecSeq(ElementContentPlus, ecElem("a"), ecElem("b")),
			children: []string{"a", "b"},
			want:     true,
		},
		{
			name:     "seq plus/two reps",
			content:  ecSeq(ElementContentPlus, ecElem("a"), ecElem("b")),
			children: []string{"a", "b", "a", "b"},
			want:     true,
		},
		{
			name:     "seq plus/zero reps fails",
			content:  ecSeq(ElementContentPlus, ecElem("a"), ecElem("b")),
			children: nil,
			want:     false,
		},

		// Repeated choice: (a | b)+
		{
			name:     "choice plus/single",
			content:  ecOr(ElementContentPlus, ecElem("a"), ecElem("b")),
			children: []string{"a"},
			want:     true,
		},
		{
			name:     "choice plus/multiple mixed",
			content:  ecOr(ElementContentPlus, ecElem("a"), ecElem("b")),
			children: []string{"a", "b", "a"},
			want:     true,
		},
		{
			name:     "choice plus/zero fails",
			content:  ecOr(ElementContentPlus, ecElem("a"), ecElem("b")),
			children: nil,
			want:     false,
		},

		// Repeated choice star: (a | b)*
		{
			name:     "choice star/zero ok",
			content:  ecOr(ElementContentMult, ecElem("a"), ecElem("b")),
			children: nil,
			want:     true,
		},
		{
			name:     "choice star/multiple",
			content:  ecOr(ElementContentMult, ecElem("a"), ecElem("b")),
			children: []string{"b", "a", "b"},
			want:     true,
		},

		// Empty children against required content
		{
			name:     "empty children/required seq fails",
			content:  ecSeq(ElementContentOnce, ecElem("a")),
			children: nil,
			want:     false,
		},
		{
			name:     "empty children/optional seq ok",
			content:  ecSeq(ElementContentOpt, ecElem("a")),
			children: nil,
			want:     true,
		},

		// Wrong element name
		{
			name:     "wrong element",
			content:  ecElem("a"),
			children: []string{"b"},
			want:     false,
		},

		// Single element exact match
		{
			name:     "single element/match",
			content:  ecElem("a"),
			children: []string{"a"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchContentModel(tt.content, tt.children)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestIsValidNameStartChar(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want bool
	}{
		// ASCII letters and underscore
		{"A", 'A', true},
		{"Z", 'Z', true},
		{"a", 'a', true},
		{"z", 'z', true},
		{"underscore", '_', true},

		// Rejected ASCII
		{"colon", ':', false},    // colon excluded in helium (namespace-aware)
		{"digit 0", '0', false},  // digit not a start char
		{"hyphen", '-', false},   // not a start char
		{"dot", '.', false},      // not a start char
		{"space", ' ', false},    // not a name char at all
		{"at", '@', false},       // between Z and a
		{"bracket", '[', false},  // between Z and a

		// Latin supplement boundaries
		{"U+00BF (before range)", 0xBF, false},   // just before 0xC0
		{"U+00C0 (start)", 0xC0, true},           // À
		{"U+00D6 (end)", 0xD6, true},             // Ö
		{"U+00D7 (multiply sign)", 0xD7, false},  // × excluded
		{"U+00D8 (start)", 0xD8, true},           // Ø
		{"U+00F6 (end)", 0xF6, true},             // ö
		{"U+00F7 (division sign)", 0xF7, false},  // ÷ excluded
		{"U+00F8 (start)", 0xF8, true},           // ø

		// Range gaps
		{"U+0300 (combining)", 0x0300, false},     // combining diacritical — not a start char
		{"U+036F (combining end)", 0x036F, false},  // not a start char
		{"U+0370 (Greek)", 0x0370, true},
		{"U+037D (end)", 0x037D, true},
		{"U+037E (Greek question mark)", 0x037E, false}, // excluded
		{"U+037F (start)", 0x037F, true},

		// Zero-width range
		{"U+200C (ZWNJ)", 0x200C, true},
		{"U+200D (ZWJ)", 0x200D, true},
		{"U+200B (before range)", 0x200B, false},
		{"U+200E (after range)", 0x200E, false},

		// CJK gap: 0x2070-0x218F
		{"U+2070", 0x2070, true},
		{"U+218F (end)", 0x218F, true},
		{"U+2190 (arrows)", 0x2190, false},

		// Gap between ranges: 0x2FEF end, 0x3001 start
		{"U+2FEF (end)", 0x2FEF, true},
		{"U+2FF0 (ideographic desc)", 0x2FF0, false}, // gap
		{"U+3000 (CJK space)", 0x3000, false},        // gap
		{"U+3001 (start)", 0x3001, true},

		// Upper BMP boundaries
		{"U+D7FF (end)", 0xD7FF, true},
		{"U+F900 (CJK compat)", 0xF900, true},
		{"U+FDCF (end)", 0xFDCF, true},
		{"U+FDD0 (nonchar)", 0xFDD0, false},
		{"U+FDF0 (start)", 0xFDF0, true},
		{"U+FFFD (end)", 0xFFFD, true},
		{"U+FFFE (nonchar)", 0xFFFE, false},

		// Supplementary planes
		{"U+10000 (start)", 0x10000, true},
		{"U+EFFFF (end)", 0xEFFFF, true},
		{"U+F0000 (private use)", 0xF0000, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidNameStartChar(tt.r)
			require.Equal(t, tt.want, got, "isValidNameStartChar(U+%04X)", tt.r)
		})
	}
}

func TestIsValidNameChar(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want bool
	}{
		// NameChar additions beyond NameStartChar
		{"digit 0", '0', true},
		{"digit 9", '9', true},
		{"hyphen", '-', true},
		{"dot", '.', true},
		{"middle dot U+00B7", 0xB7, true},
		{"combining U+0300", 0x0300, true},
		{"combining U+036F", 0x036F, true},
		{"U+02FF (before combining)", 0x02FF, true}, // in NameStartChar range 0xF8-0x2FF
		{"extender U+203F", 0x203F, true},
		{"extender U+2040", 0x2040, true},

		// Still not valid
		{"space", ' ', false},
		{"U+00B6 (pilcrow)", 0xB6, false},
		{"U+00B8 (cedilla)", 0xB8, false},
		{"U+2041 (caret insertion)", 0x2041, false},
		{"colon", ':', false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidNameChar(tt.r)
			require.Equal(t, tt.want, got, "isValidNameChar(U+%04X)", tt.r)
		})
	}
}

func TestIsValidName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"simple", "foo", true},
		{"with digits", "foo123", true},
		{"with hyphen", "foo-bar", true},
		{"with dot", "foo.bar", true},
		{"starts with underscore", "_foo", true},
		{"starts with digit", "1foo", false},
		{"starts with hyphen", "-foo", false},
		{"empty", "", false},
		{"unicode letter start", "\u00C0foo", true},      // À
		{"middle dot in name", "foo\u00B7bar", true},      // ·
		{"combining in name", "foo\u0300bar", true},       // combining grave
		{"multiply sign start", "\u00D7foo", false},       // ×
		{"division sign start", "\u00F7foo", false},       // ÷
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidName(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestStandaloneWhitespaceCheck(t *testing.T) {
	t.Run("whitespace in element-only content from ext subset", func(t *testing.T) {
		doc := NewDocument("1.0", "utf-8", StandaloneExplicitYes)

		// Internal subset: declares "root" with ANY content
		intDTD := newDTD()
		intDTD.doc = doc
		intDTD.etype = DTDNode
		doc.intSubset = intDTD
		rootDecl := newElementDecl()
		rootDecl.name = "root"
		rootDecl.decltype = AnyElementType
		rootDecl.doc = doc
		intDTD.elements = map[string]*ElementDecl{"root:": rootDecl}
		intDTD.entities = map[string]*Entity{}
		intDTD.pentities = map[string]*Entity{}
		intDTD.attributes = map[string]*AttributeDecl{}

		// External subset: declares "container" with element-only content (child)+
		extDTD := newDTD()
		extDTD.doc = doc
		extDTD.etype = DTDNode
		doc.extSubset = extDTD
		containerDecl := newElementDecl()
		containerDecl.name = "container"
		containerDecl.decltype = ElementElementType
		containerDecl.content = &ElementContent{
			ctype:  ElementContentElement,
			coccur: ElementContentPlus,
			name:   "child",
		}
		containerDecl.doc = doc
		extDTD.elements = map[string]*ElementDecl{"container:": containerDecl}
		extDTD.entities = map[string]*Entity{}
		extDTD.pentities = map[string]*Entity{}
		extDTD.attributes = map[string]*AttributeDecl{}

		// Build DOM: <root><container> <child/> </container></root>
		root := newElement("root")
		root.doc = doc
		container := newElement("container")
		container.doc = doc
		_ = container.AddContent([]byte(" "))
		child := newElement("child")
		child.doc = doc
		_ = container.AddChild(child)
		_ = container.AddContent([]byte(" "))
		_ = root.AddChild(container)
		_ = doc.AddChild(root)

		ve := validateDocument(doc)
		require.NotNil(t, ve)
		require.Contains(t, ve.Error(), "standalone")
		require.Contains(t, ve.Error(), "white spaces")
	})

	t.Run("no whitespace no error", func(t *testing.T) {
		doc := NewDocument("1.0", "utf-8", StandaloneExplicitYes)

		intDTD := newDTD()
		intDTD.doc = doc
		intDTD.etype = DTDNode
		doc.intSubset = intDTD
		rootDecl := newElementDecl()
		rootDecl.name = "root"
		rootDecl.decltype = AnyElementType
		rootDecl.doc = doc
		intDTD.elements = map[string]*ElementDecl{"root:": rootDecl}
		intDTD.entities = map[string]*Entity{}
		intDTD.pentities = map[string]*Entity{}
		intDTD.attributes = map[string]*AttributeDecl{}

		extDTD := newDTD()
		extDTD.doc = doc
		extDTD.etype = DTDNode
		doc.extSubset = extDTD
		containerDecl := newElementDecl()
		containerDecl.name = "container"
		containerDecl.decltype = ElementElementType
		containerDecl.content = &ElementContent{
			ctype:  ElementContentElement,
			coccur: ElementContentPlus,
			name:   "child",
		}
		containerDecl.doc = doc
		extDTD.elements = map[string]*ElementDecl{"container:": containerDecl}
		extDTD.entities = map[string]*Entity{}
		extDTD.pentities = map[string]*Entity{}
		extDTD.attributes = map[string]*AttributeDecl{}

		// Build DOM: <root><container><child/></container></root> (no whitespace)
		root := newElement("root")
		root.doc = doc
		container := newElement("container")
		container.doc = doc
		child := newElement("child")
		child.doc = doc
		_ = container.AddChild(child)
		_ = root.AddChild(container)
		_ = doc.AddChild(root)

		ve := validateDocument(doc)
		require.NotNil(t, ve) // still fails with "no declaration found"
		// But should NOT contain the standalone whitespace error
		for _, e := range ve.Errors {
			require.NotContains(t, e, "white spaces")
		}
	})

	t.Run("not standalone no whitespace error", func(t *testing.T) {
		doc := NewDocument("1.0", "utf-8", StandaloneImplicitNo)

		intDTD := newDTD()
		intDTD.doc = doc
		intDTD.etype = DTDNode
		doc.intSubset = intDTD
		rootDecl := newElementDecl()
		rootDecl.name = "root"
		rootDecl.decltype = AnyElementType
		rootDecl.doc = doc
		intDTD.elements = map[string]*ElementDecl{"root:": rootDecl}
		intDTD.entities = map[string]*Entity{}
		intDTD.pentities = map[string]*Entity{}
		intDTD.attributes = map[string]*AttributeDecl{}

		extDTD := newDTD()
		extDTD.doc = doc
		extDTD.etype = DTDNode
		doc.extSubset = extDTD
		containerDecl := newElementDecl()
		containerDecl.name = "container"
		containerDecl.decltype = ElementElementType
		containerDecl.content = &ElementContent{
			ctype:  ElementContentElement,
			coccur: ElementContentPlus,
			name:   "child",
		}
		containerDecl.doc = doc
		extDTD.elements = map[string]*ElementDecl{"container:": containerDecl}
		extDTD.entities = map[string]*Entity{}
		extDTD.pentities = map[string]*Entity{}
		extDTD.attributes = map[string]*AttributeDecl{}

		// Build DOM: <root><container> <child/> </container></root>
		root := newElement("root")
		root.doc = doc
		container := newElement("container")
		container.doc = doc
		_ = container.AddContent([]byte(" "))
		child := newElement("child")
		child.doc = doc
		_ = container.AddChild(child)
		_ = container.AddContent([]byte(" "))
		_ = root.AddChild(container)
		_ = doc.AddChild(root)

		ve := validateDocument(doc)
		// Non-standalone: extSubset is searched, so container is found and
		// validation should pass (or at least not have standalone error)
		if ve != nil {
			for _, e := range ve.Errors {
				require.NotContains(t, e, "standalone")
			}
		}
	})

	t.Run("element in internal subset not flagged", func(t *testing.T) {
		doc := NewDocument("1.0", "utf-8", StandaloneExplicitYes)

		intDTD := newDTD()
		intDTD.doc = doc
		intDTD.etype = DTDNode
		doc.intSubset = intDTD
		// "root" declared in internal subset with element-only content
		rootDecl := newElementDecl()
		rootDecl.name = "root"
		rootDecl.decltype = ElementElementType
		rootDecl.content = &ElementContent{
			ctype:  ElementContentElement,
			coccur: ElementContentPlus,
			name:   "child",
		}
		rootDecl.doc = doc
		childDecl := newElementDecl()
		childDecl.name = "child"
		childDecl.decltype = EmptyElementType
		childDecl.doc = doc
		intDTD.elements = map[string]*ElementDecl{
			"root:":  rootDecl,
			"child:": childDecl,
		}
		intDTD.entities = map[string]*Entity{}
		intDTD.pentities = map[string]*Entity{}
		intDTD.attributes = map[string]*AttributeDecl{}

		// Build DOM: <root> <child/> </root> (whitespace around child)
		root := newElement("root")
		root.doc = doc
		_ = root.AddContent([]byte(" "))
		child := newElement("child")
		child.doc = doc
		_ = root.AddChild(child)
		_ = root.AddContent([]byte(" "))
		_ = doc.AddChild(root)

		ve := validateDocument(doc)
		// Element is declared in the internal subset, so standalone whitespace
		// check should NOT apply. Whitespace is ignorable whitespace.
		if ve != nil {
			for _, e := range ve.Errors {
				require.NotContains(t, e, "standalone")
			}
		}
	})
}

func TestExtSubsetLookup_ParameterEntityInExtSubset(t *testing.T) {
	doc := buildTestDoc(StandaloneImplicitNo)

	// Add a parameter entity to extSubset
	pent := newEntity("pEnt", InternalParameterEntity, "", "", "param-value", "")
	pent.doc = doc
	doc.extSubset.pentities["pEnt"] = pent

	ent, found := doc.GetParameterEntity("pEnt")
	require.True(t, found, "parameter entity in extSubset should be found")
	require.Equal(t, "param-value", string(ent.Content()))

	// standalone=yes should block it
	doc.standalone = StandaloneExplicitYes
	_, found = doc.GetParameterEntity("pEnt")
	require.False(t, found, "standalone=yes should prevent extSubset parameter entity lookup")
}

func TestEntityAttributeValidation(t *testing.T) {
	t.Run("valid unparsed entity", func(t *testing.T) {
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!NOTATION gif SYSTEM "image/gif">
  <!ENTITY logo SYSTEM "logo.gif" NDATA gif>
  <!ATTLIST root img ENTITY #REQUIRED>
]>
<root img="logo"/>`
		p := NewParser()
		p.SetOption(ParseDTDValid)
		p.SetOption(ParseDTDAttr)
		_, err := p.Parse([]byte(xml))
		require.NoError(t, err)
	})

	t.Run("undeclared entity", func(t *testing.T) {
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root img ENTITY #REQUIRED>
]>
<root img="noSuchEntity"/>`
		p := NewParser()
		p.SetOption(ParseDTDValid)
		p.SetOption(ParseDTDAttr)
		_, err := p.Parse([]byte(xml))
		require.Error(t, err)
		require.Contains(t, err.Error(), "undeclared entity")
	})

	t.Run("wrong entity type (internal)", func(t *testing.T) {
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ENTITY internalEnt "hello">
  <!ATTLIST root img ENTITY #REQUIRED>
]>
<root img="internalEnt"/>`
		p := NewParser()
		p.SetOption(ParseDTDValid)
		p.SetOption(ParseDTDAttr)
		_, err := p.Parse([]byte(xml))
		require.Error(t, err)
		require.Contains(t, err.Error(), "not unparsed")
	})
}

func TestEntitiesAttributeValidation(t *testing.T) {
	t.Run("valid multiple unparsed entities", func(t *testing.T) {
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!NOTATION gif SYSTEM "image/gif">
  <!ENTITY logo1 SYSTEM "logo1.gif" NDATA gif>
  <!ENTITY logo2 SYSTEM "logo2.gif" NDATA gif>
  <!ATTLIST root imgs ENTITIES #REQUIRED>
]>
<root imgs="logo1 logo2"/>`
		p := NewParser()
		p.SetOption(ParseDTDValid)
		p.SetOption(ParseDTDAttr)
		_, err := p.Parse([]byte(xml))
		require.NoError(t, err)
	})

	t.Run("one undeclared entity", func(t *testing.T) {
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!NOTATION gif SYSTEM "image/gif">
  <!ENTITY logo1 SYSTEM "logo1.gif" NDATA gif>
  <!ATTLIST root imgs ENTITIES #REQUIRED>
]>
<root imgs="logo1 noSuchEntity"/>`
		p := NewParser()
		p.SetOption(ParseDTDValid)
		p.SetOption(ParseDTDAttr)
		_, err := p.Parse([]byte(xml))
		require.Error(t, err)
		require.Contains(t, err.Error(), "undeclared entity")
	})
}

func TestNotationAttributeValidation(t *testing.T) {
	t.Run("valid notation", func(t *testing.T) {
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!NOTATION gif SYSTEM "image/gif">
  <!NOTATION png SYSTEM "image/png">
  <!ATTLIST root fmt NOTATION (gif|png) #REQUIRED>
]>
<root fmt="gif"/>`
		p := NewParser()
		p.SetOption(ParseDTDValid)
		p.SetOption(ParseDTDAttr)
		_, err := p.Parse([]byte(xml))
		require.NoError(t, err)
	})

	t.Run("undeclared notation", func(t *testing.T) {
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!NOTATION gif SYSTEM "image/gif">
  <!ATTLIST root fmt NOTATION (gif|png) #REQUIRED>
]>
<root fmt="png"/>`
		p := NewParser()
		p.SetOption(ParseDTDValid)
		p.SetOption(ParseDTDAttr)
		_, err := p.Parse([]byte(xml))
		require.Error(t, err)
		require.Contains(t, err.Error(), "undeclared notation")
	})
}
