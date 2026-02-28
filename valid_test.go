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
