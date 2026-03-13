package relaxng_test

import (
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

	// Compile may attempt os.ReadFile for include/externalRef hrefs in fuzz-generated
	// schemas. This is read-only and random paths will almost always fail with an error.
	// Injecting a stub loader would require API changes; accepted risk for fuzz testing.
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			return
		}
		doc, err := helium.Parse(t.Context(), data)
		if err != nil {
			return
		}
		_, _ = relaxng.Compile(t.Context(), doc)
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

	f.Fuzz(func(t *testing.T, schemaData, instanceData []byte) {
		if len(schemaData) > 1<<20 || len(instanceData) > 1<<20 {
			return
		}
		ctx := t.Context()
		schemaDom, err := helium.Parse(ctx, schemaData)
		if err != nil {
			return
		}

		grammar, err := relaxng.Compile(t.Context(), schemaDom)
		if err != nil {
			return
		}

		instanceDom, err := helium.Parse(ctx, instanceData)
		if err != nil {
			return
		}

		_ = relaxng.Validate(instanceDom, grammar)
	})
}
