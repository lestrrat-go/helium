package relaxng_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
)

func FuzzCompile(f *testing.F) {
	f.Add([]byte(`<?xml version="1.0"?><grammar xmlns="http://relaxng.org/ns/structure/1.0"><start><element name="root"><text/></element></start></grammar>`))
	f.Add([]byte(`<?xml version="1.0"?><element name="root" xmlns="http://relaxng.org/ns/structure/1.0"><zeroOrMore><element name="item"><text/></element></zeroOrMore></element>`))
	f.Add([]byte(`<?xml version="1.0"?><grammar xmlns="http://relaxng.org/ns/structure/1.0"><start><choice><element name="a"><empty/></element><element name="b"><text/></element></choice></start></grammar>`))
	f.Add([]byte(``))
	f.Add([]byte(`not a schema`))

	f.Fuzz(func(_ *testing.T, data []byte) {
		doc, err := helium.Parse(context.Background(), data)
		if err != nil {
			return
		}
		_, _ = relaxng.Compile(doc)
	})
}

func FuzzValidate(f *testing.F) {
	f.Add(
		[]byte(`<?xml version="1.0"?><grammar xmlns="http://relaxng.org/ns/structure/1.0"><start><element name="root"><text/></element></start></grammar>`),
		[]byte(`<?xml version="1.0"?><root>hello</root>`),
	)
	f.Add(
		[]byte(`<?xml version="1.0"?><element name="doc" xmlns="http://relaxng.org/ns/structure/1.0"><oneOrMore><element name="p"><text/></element></oneOrMore></element>`),
		[]byte(`<?xml version="1.0"?><doc><p>paragraph</p></doc>`),
	)

	f.Fuzz(func(_ *testing.T, schemaData, instanceData []byte) {
		schemaDom, err := helium.Parse(context.Background(), schemaData)
		if err != nil {
			return
		}

		grammar, err := relaxng.Compile(schemaDom)
		if err != nil {
			return
		}

		instanceDom, err := helium.Parse(context.Background(), instanceData)
		if err != nil {
			return
		}

		_ = relaxng.Validate(instanceDom, grammar)
	})
}
