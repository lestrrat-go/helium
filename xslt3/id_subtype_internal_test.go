package xslt3

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestAnnotateAttrRegistersIDSubtype(t *testing.T) {
	schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:test">
  <xs:simpleType name="myID">
    <xs:restriction base="xs:ID"/>
  </xs:simpleType>
</xs:schema>`))
	require.NoError(t, err)

	schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDoc)
	require.NoError(t, err)

	doc := helium.NewDefaultDocument()
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))
	require.NoError(t, root.SetAttribute("id", "alpha"))

	ec := &execContext{
		schemaRegistry:  &schemaRegistry{schemas: []*xsd.Schema{schema}},
		typeAnnotations: make(map[helium.Node]string),
		outputStack:     []*outputFrame{{doc: doc, current: root}},
		resultDoc:       doc,
	}

	ec.annotateAttr(root, xpath3.QAnnotation("urn:test", "myID"), "id", "", "alpha")

	require.Equal(t, root, doc.GetElementByID("alpha"))
	attr := root.Attributes()[0]
	require.Equal(t, xpath3.QAnnotation("urn:test", "myID"), ec.typeAnnotations[attr])
}
