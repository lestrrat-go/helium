package xsd_test

import (
	"context"
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

	f.Fuzz(func(_ *testing.T, data []byte) {
		doc, err := helium.Parse(context.Background(), data)
		if err != nil {
			return
		}
		_, _ = xsd.Compile(doc)
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

	f.Fuzz(func(_ *testing.T, schemaData, instanceData []byte) {
		schemaDom, err := helium.Parse(context.Background(), schemaData)
		if err != nil {
			return
		}

		schema, err := xsd.Compile(schemaDom)
		if err != nil {
			return
		}

		instanceDom, err := helium.Parse(context.Background(), instanceData)
		if err != nil {
			return
		}

		_ = xsd.Validate(instanceDom, schema)
	})
}
