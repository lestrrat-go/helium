package xsd_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// §3.3.6 "Element Default Valid (Immediate)" clause 2.1 (version-independent): a
// value constraint (default/fixed) may be present on an element declaration only
// when the {type definition} is a simple type or a complex type whose {content
// type} is a simple type definition or mixed. An element-only or empty complex
// content type carries no character-data value for the constraint, so a
// default/fixed on such an element is a schema-representation error. Mixed and
// simple(-content) types, and the untyped (xs:anyType, mixed) case, stay valid.
func TestElement_ValueConstraintRequiresSimpleOrMixedContent(t *testing.T) {
	t.Parallel()

	const shell = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">%s</xsd:schema>`

	invalid := map[string]string{
		// default on an element whose named type has element-only (sequence) content.
		"default/sequence-named-type": `
	<xsd:complexType name="author">
		<xsd:sequence>
			<xsd:element name="title" type="xsd:string"/>
		</xsd:sequence>
	</xsd:complexType>
	<xsd:element name="book" type="author" default="foo"/>`,
		// fixed on an element whose inline type has element-only (all) content.
		"fixed/all-inline-type": `
	<xsd:element name="myElem" fixed="foo">
		<xsd:complexType>
			<xsd:all>
				<xsd:element name="ele1" type="xsd:string"/>
			</xsd:all>
		</xsd:complexType>
	</xsd:element>`,
		// default on an element whose inline type has empty content.
		"default/empty-content": `
	<xsd:element name="e" default="foo">
		<xsd:complexType>
			<xsd:attribute name="a" type="xsd:string"/>
		</xsd:complexType>
	</xsd:element>`,
	}
	for name, body := range invalid {
		t.Run("invalid/"+name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(shell, body)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v must reject", v)
				require.Nil(t, schema)
				require.Contains(t, errs, "must be a simple type or a complex type with mixed or simple content", "version=%v", v)
			}
		})
	}

	valid := map[string]string{
		// simple-typed element with a default.
		"simple-type": `<xsd:element name="x" type="xsd:string" default="foo"/>`,
		// untyped element defaults to xs:anyType (mixed) — value constraint allowed.
		"untyped-anytype": `<xsd:element name="x" default="foo"/>`,
		// mixed complex content with an emptiable particle.
		"mixed-content": `
	<xsd:element name="x" default="foo">
		<xsd:complexType mixed="true">
			<xsd:sequence minOccurs="0">
				<xsd:element name="y" type="xsd:string"/>
			</xsd:sequence>
		</xsd:complexType>
	</xsd:element>`,
		// simpleContent complex type with a fixed value.
		"simple-content": `
	<xsd:element name="x" fixed="5">
		<xsd:complexType>
			<xsd:simpleContent>
				<xsd:extension base="xsd:int">
					<xsd:attribute name="a" type="xsd:string"/>
				</xsd:extension>
			</xsd:simpleContent>
		</xsd:complexType>
	</xsd:element>`,
	}
	for name, body := range valid {
		t.Run("valid/"+name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(shell, body)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, errs, cerr := compileWith(t, v, schemaXML)
				require.NoError(t, cerr, "version=%v must accept; errs=%s", v, errs)
				require.NotNil(t, schema)
			}
		})
	}
}
