package xsd_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

// A chain of xs:redefine targeting the SAME type name — a schema that redefines
// a document that itself redefines the type — must compile: each level restricts
// (or, for a complexType, extends) the previous level's redefinition, and the
// self-reference base at every level resolves to the immediately-preceding
// definition, not to the type itself. Mirrors W3C msMeta/SimpleType_w3c.xml
// stZ033/stZ034 and msMeta/Additional_w3c.xml addB007.
func TestRedefine_Chain(t *testing.T) {
	t.Parallel()

	t.Run("simpleType chain (stZ033/stZ034)", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			"m.xsd": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="utf-8" ?>
<xs:schema targetNamespace="foo" elementFormDefault="qualified" xmlns="foo" xmlns:xs="http://www.w3.org/2001/XMLSchema">
	<xs:redefine schemaLocation="m2.xsd">
	<xs:simpleType name="B">
		<xs:restriction base="B">
			<xs:enumeration value="3"/>
		</xs:restriction>
	</xs:simpleType>
	</xs:redefine>
	<xs:element name="b1" type="B"/>
</xs:schema>`)},
			"m2.xsd": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="utf-8" ?>
<xs:schema targetNamespace="foo" elementFormDefault="qualified" xmlns="foo" xmlns:xs="http://www.w3.org/2001/XMLSchema">
	<xs:redefine schemaLocation="m3.xsd">
	<xs:simpleType name="B">
		<xs:restriction base="B">
			<xs:enumeration value="2"/>
			<xs:enumeration value="3"/>
		</xs:restriction>
	</xs:simpleType>
	</xs:redefine>
	<xs:element name="b2" type="B"/>
</xs:schema>`)},
			"m3.xsd": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="utf-8" ?>
<xs:schema targetNamespace="foo" elementFormDefault="qualified" xmlns="foo" xmlns:xs="http://www.w3.org/2001/XMLSchema">
	<xs:simpleType name="B">
		<xs:restriction base="xs:int">
			<xs:enumeration value="1"/>
			<xs:enumeration value="2"/>
			<xs:enumeration value="3"/>
		</xs:restriction>
	</xs:simpleType>
</xs:schema>`)},
		}
		errStr, err := compileRedefineFS(t, fsys, "m.xsd")
		require.NoError(t, err, "valid redefine chain must compile; got: %q", errStr)
	})

	t.Run("complexType chain (addB007)", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			"t.xsd": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="utf-8" ?>
<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
	<xsd:redefine schemaLocation="t1.xsd">
		<xsd:complexType name="ctFeed">
			<xsd:complexContent>
				<xsd:extension base="ctFeed">
					<xsd:attribute ref="a3" />
				</xsd:extension>
			</xsd:complexContent>
		</xsd:complexType>
	</xsd:redefine>
 	<xsd:attribute name="a3"/>
</xsd:schema>`)},
			"t1.xsd": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="utf-8" ?>
<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
	<xsd:redefine schemaLocation="t2.xsd">
		<xsd:complexType name="ctFeed">
			<xsd:complexContent>
				<xsd:extension base="ctFeed">
					<xsd:attribute ref="a2" />
				</xsd:extension>
			</xsd:complexContent>
		</xsd:complexType>
	</xsd:redefine>
 	<xsd:attribute name="a2"/>
</xsd:schema>`)},
			"t2.xsd": &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="utf-8" ?>
<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
	<xsd:complexType name="ctFeed">
		<xsd:attribute name="msnns" type="xsd:string" use="required" />
	</xsd:complexType>
	<xsd:element name="feed" type="ctFeed" />
</xsd:schema>`)},
		}
		errStr, err := compileRedefineFS(t, fsys, "t.xsd")
		require.NoError(t, err, "valid redefine chain must compile; got: %q", errStr)
	})
}

// xs:redefine has a CLOSED schema-for-schemas attribute set {id, schemaLocation}.
// A `namespace` attribute (which belongs to xs:import, not xs:redefine) is a
// schema-representation error. Mirrors W3C msMeta/Schema_w3c.xml schH4.
func TestRedefine_NamespaceAttrRejected(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"a.xsd": &fstest.MapFile{Data: []byte(`<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema"
targetNamespace="ns-a" xmlns="ns-a">
	<xsd:redefine namespace="foo" schemaLocation="b.xsd" />
	<xsd:complexType name="ct-A">
		<xsd:sequence minOccurs="1">
			<xsd:group ref="g-b"/>
		</xsd:sequence>
	</xsd:complexType>
	<xsd:element name="e1" type="ct-A" />
</xsd:schema>`)},
		"b.xsd": &fstest.MapFile{Data: []byte(`<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema"
	targetNamespace="ns-a" xmlns="ns-a">
	<xsd:group name="g-b">
		<xsd:sequence>
			<xsd:element name="b1" type="xsd:boolean"/>
			<xsd:element name="b2" type="xsd:int"/>
		</xsd:sequence>
	</xsd:group>
</xsd:schema>`)},
	}
	errStr, err := compileRedefineFS(t, fsys, "a.xsd")
	require.Error(t, err, "namespace attribute on xs:redefine must be rejected")
	require.Contains(t, errStr, "The attribute 'namespace' is not allowed")
}
