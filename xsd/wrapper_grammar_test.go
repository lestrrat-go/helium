package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestComplexContentGrammar exercises the full XSD 1.1 (annotation?, (restriction
// | extension)) content model of <xs:complexContent>: exactly one derivation,
// annotation only before it, no stray/trailing children. Each violation class is a
// schema error in 1.1; the valid canonical case compiles; 1.0 tolerates a stray.
func TestComplexContentGrammar(t *testing.T) {
	t.Parallel()

	const base = `  <xs:complexType name="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`
	wrap := func(inner string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + base + `
  <xs:complexType name="R"><xs:complexContent>
` + inner + `
  </xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
	}
	const derivation = `    <xs:restriction base="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:restriction>`

	t.Run("valid annotation then one derivation compiles", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(`    <xs:annotation><xs:documentation>ok</xs:documentation></xs:annotation>
`+derivation))
		require.NoError(t, cerr)
	})

	t.Run("zero derivations is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(`    <xs:annotation><xs:documentation>only annotation</xs:documentation></xs:annotation>`))
		require.Error(t, cerr, "complexContent requires exactly one restriction/extension")
	})

	t.Run("two derivations is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(derivation+"\n"+derivation))
		require.Error(t, cerr)
	})

	t.Run("annotation after the derivation is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(derivation+`
    <xs:annotation><xs:documentation>too late</xs:documentation></xs:annotation>`))
		require.Error(t, cerr, "annotation must precede the derivation")
	})

	t.Run("stray child before the derivation is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(`    <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
`+derivation))
		require.Error(t, cerr)
	})

	t.Run("stray child after the derivation is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(derivation+`
    <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="skip"/></xs:openContent>`))
		require.Error(t, cerr)
	})

	t.Run("XSD 1.0 also rejects a stray child after the derivation", func(t *testing.T) {
		t.Parallel()
		// The (annotation?, (restriction|extension)) wrapper grammar is version-
		// independent, so 1.0 rejects the trailing stray sequence just like 1.1.
		schema := wrap(derivation + `
    <xs:sequence><xs:element name="z" type="xs:string"/></xs:sequence>`)
		_, v10err := compileV10(t, schema)
		require.Error(t, v10err, "1.0 enforces the wrapper grammar")
		_, _, v11err := compileV11(t, schema)
		require.Error(t, v11err, "1.1 rejects the stray")
	})
}

// TestSimpleContentGrammar exercises the same (annotation?, (restriction |
// extension)) grammar enforced in the OUTER parseSimpleContent wrapper.
func TestSimpleContentGrammar(t *testing.T) {
	t.Parallel()

	wrap := func(inner string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="R"><xs:simpleContent>
` + inner + `
  </xs:simpleContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
	}
	const derivation = `    <xs:extension base="xs:string"><xs:attribute name="a" type="xs:string"/></xs:extension>`

	t.Run("valid annotation then one derivation compiles", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(`    <xs:annotation><xs:documentation>ok</xs:documentation></xs:annotation>
`+derivation))
		require.NoError(t, cerr)
	})

	t.Run("zero derivations is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(`    <xs:annotation><xs:documentation>only annotation</xs:documentation></xs:annotation>`))
		require.Error(t, cerr, "simpleContent requires exactly one restriction/extension")
	})

	t.Run("two derivations is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(derivation+"\n"+derivation))
		require.Error(t, cerr)
	})

	t.Run("annotation after the derivation is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(derivation+`
    <xs:annotation><xs:documentation>too late</xs:documentation></xs:annotation>`))
		require.Error(t, cerr)
	})

	t.Run("direct trailing openContent under simpleContent is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(derivation+`
    <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="skip"/></xs:openContent>`))
		require.Error(t, cerr, "a trailing openContent directly under simpleContent must be rejected")
	})

	t.Run("stray element under simpleContent is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(`    <xs:sequence><xs:element name="z" type="xs:string"/></xs:sequence>
`+derivation))
		require.Error(t, cerr)
	})

	t.Run("XSD 1.0 also rejects a stray child under simpleContent", func(t *testing.T) {
		t.Parallel()
		// The simpleContent wrapper grammar is version-independent: a trailing
		// openContent (not a restriction/extension) is a stray child rejected in 1.0
		// too (openContent is not a 1.0 construct, so it is genuinely invalid there).
		schema := wrap(derivation + `
    <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="skip"/></xs:openContent>`)
		_, v10err := compileV10(t, schema)
		require.Error(t, v10err, "1.0 enforces the wrapper grammar")
		_, _, v11err := compileV11(t, schema)
		require.Error(t, v11err)
	})
}

// TestComplexTypeDirectAssertOrdering covers the trailing assert* region of the
// direct complexType grammar (annotation?, openContent?, (group|all|sequence|
// choice)?, ((attribute|attributeGroup)*, anyAttribute?), assert*): nothing but
// further assertions may follow an xs:assert. (assert is 1.1-only, so these guards
// never affect 1.0.)
func TestComplexTypeDirectAssertOrdering(t *testing.T) {
	t.Parallel()

	wrap := func(inner string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="R">
` + inner + `
  </xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
	}

	t.Run("valid model group, attributes, anyAttribute, assert compiles", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(`    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:attribute name="x" type="xs:string"/>
    <xs:anyAttribute processContents="skip"/>
    <xs:assert test="true()"/>`))
		require.NoError(t, cerr)
	})

	t.Run("attribute after assert is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(`    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:assert test="true()"/>
    <xs:attribute name="x" type="xs:string"/>`))
		require.Error(t, cerr)
	})

	t.Run("anyAttribute after assert is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(`    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:assert test="true()"/>
    <xs:anyAttribute processContents="skip"/>`))
		require.Error(t, cerr)
	})

	t.Run("content model particle after assert is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(`    <xs:assert test="true()"/>
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>`))
		require.Error(t, cerr)
	})
}

// TestComplexTypeDirectStrayChild covers the direct <xs:complexType> child switch:
// only annotation, simpleContent, complexContent, openContent, a model-group
// particle, attribute, attributeGroup, anyAttribute, and assert are allowed; any
// other child is a schema error in 1.1 and tolerated in 1.0.
func TestComplexTypeDirectStrayChild(t *testing.T) {
	t.Parallel()

	wrap := func(inner string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="R">
` + inner + `
  </xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
	}

	t.Run("valid complexType compiles", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(`    <xs:annotation><xs:documentation>ok</xs:documentation></xs:annotation>
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:attribute name="x" type="xs:string"/>`))
		require.NoError(t, cerr)
	})

	t.Run("direct xs:element under complexType is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(`    <xs:element name="x" type="xs:string"/>`))
		require.Error(t, cerr, "a direct xs:element is not a valid child of xs:complexType")
	})

	t.Run("unknown XSD element under complexType is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(`    <xs:notAThing/>`))
		require.Error(t, cerr)
	})

	t.Run("XSD 1.0 tolerates a stray child under complexType", func(t *testing.T) {
		t.Parallel()
		schema := wrap(`    <xs:element name="x" type="xs:string"/>`)
		_, v10err := compileV10(t, schema)
		require.NoError(t, v10err, "1.0 keeps its lenient behavior (byte-identity)")
		_, _, v11err := compileV11(t, schema)
		require.Error(t, v11err)
	})
}

// TestComplexTypeDirectAnnotationCardinality covers F2: the direct complexType
// grammar is (annotation?, ...), so at most ONE xs:annotation is allowed, and it
// must be the first child. This is version-independent (enforced in 1.0 and 1.1).
func TestComplexTypeDirectAnnotationCardinality(t *testing.T) {
	t.Parallel()

	wrap := func(inner string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="R">
` + inner + `
  </xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
	}

	t.Run("one annotation compiles", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(`    <xs:annotation><xs:documentation>ok</xs:documentation></xs:annotation>
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>`))
		require.NoError(t, cerr)
	})

	t.Run("two annotations is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(`    <xs:annotation><xs:documentation>one</xs:documentation></xs:annotation>
    <xs:annotation><xs:documentation>two</xs:documentation></xs:annotation>
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>`))
		require.Error(t, cerr)
	})

	t.Run("XSD 1.0 also rejects two annotations", func(t *testing.T) {
		t.Parallel()
		schema := wrap(`    <xs:annotation><xs:documentation>one</xs:documentation></xs:annotation>
    <xs:annotation><xs:documentation>two</xs:documentation></xs:annotation>
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>`)
		_, v10err := compileV10(t, schema)
		require.Error(t, v10err, "the annotation cardinality rule is version-independent")
	})

	t.Run("XSD 1.0 also rejects an annotation after content", func(t *testing.T) {
		t.Parallel()
		schema := wrap(`    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:annotation><xs:documentation>too late</xs:documentation></xs:annotation>`)
		_, v10err := compileV10(t, schema)
		require.Error(t, v10err, "the annotation must be the first child (version-independent)")
	})
}

// TestComplexTypeWrapperAssertExclusivity covers the direct-complexType CHOICE between
// the wrapper branch (simpleContent|complexContent) and the non-wrapper branch
// (which includes the trailing assert*): an xs:assert may not be a sibling of a
// wrapper, in EITHER order. Version11-gated; 1.0 tolerant.
func TestComplexTypeWrapperAssertExclusivity(t *testing.T) {
	t.Parallel()

	const base = `  <xs:complexType name="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`
	wrap := func(inner string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + base + `
  <xs:complexType name="R">
` + inner + `
  </xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
	}
	const cc = `    <xs:complexContent><xs:restriction base="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:restriction></xs:complexContent>`
	const sc = `    <xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent>`
	const assertChild = `    <xs:assert test="true()"/>`

	t.Run("assert before complexContent is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(assertChild+"\n"+cc))
		require.Error(t, cerr)
	})

	t.Run("complexContent followed by sibling assert is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(cc+"\n"+assertChild))
		require.Error(t, cerr)
	})

	t.Run("assert before simpleContent is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(assertChild+"\n"+sc))
		require.Error(t, cerr)
	})

	t.Run("simpleContent followed by sibling assert is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(sc+"\n"+assertChild))
		require.Error(t, cerr)
	})

	t.Run("valid wrapper-only complexType compiles", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(cc))
		require.NoError(t, cerr)
	})

	t.Run("valid non-wrapper complexType with trailing assert compiles", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, wrap(`    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:assert test="true()"/>`))
		require.NoError(t, cerr)
	})

	t.Run("XSD 1.0 tolerates assert beside a wrapper (assert ignored)", func(t *testing.T) {
		t.Parallel()
		// In 1.0 xs:assert is not a recognized complexType child, so it is ignored and
		// the wrapper alone governs — byte-identical to origin.
		_, v10err := compileV10(t, wrap(cc+"\n"+assertChild))
		require.NoError(t, v10err)
	})
}
