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
