package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
)

func FuzzCompile(f *testing.F) {
	f.Add([]byte(`<?xml version="1.0"?><xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="root" type="xs:string"/></xs:schema>`))
	f.Add([]byte(`<?xml version="1.0"?><xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:complexType name="T"><xs:sequence><xs:element name="a" type="xs:int"/></xs:sequence></xs:complexType></xs:schema>`))
	f.Add([]byte(`<?xml version="1.0"?><xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:simpleType name="S"><xs:restriction base="xs:string"><xs:pattern value="[a-z]+"/></xs:restriction></xs:simpleType></xs:schema>`))
	f.Add([]byte(``))
	f.Add([]byte(`not a schema`))

	// Compile may attempt os.ReadFile for xs:include/xs:import/xs:redefine schemaLocation
	// in fuzz-generated schemas. This is read-only and random paths will almost always
	// fail with an error. Injecting a stub loader would require API changes; accepted
	// risk for fuzz testing.
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			return
		}
		doc, err := helium.Parse(t.Context(), data)
		if err != nil {
			return
		}
		_, _ = xsd.Compile(t.Context(), doc)
	})
}

func FuzzValidate(f *testing.F) {
	f.Add(
		[]byte(`<?xml version="1.0"?><xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="root" type="xs:string"/></xs:schema>`),
		[]byte(`<?xml version="1.0"?><root>hello</root>`),
	)
	f.Add(
		[]byte(`<?xml version="1.0"?><xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="root" type="xs:int"/></xs:schema>`),
		[]byte(`<?xml version="1.0"?><root>42</root>`),
	)

	f.Fuzz(func(t *testing.T, schemaData, instanceData []byte) {
		if len(schemaData) > 1<<20 || len(instanceData) > 1<<20 {
			return
		}
		ctx := t.Context()
		schemaDom, err := helium.Parse(ctx, schemaData)
		if err != nil {
			return
		}

		schema, err := xsd.Compile(t.Context(), schemaDom)
		if err != nil {
			return
		}

		instanceDom, err := helium.Parse(ctx, instanceData)
		if err != nil {
			return
		}

		_ = xsd.Validate(t.Context(), instanceDom, schema)
	})
}
