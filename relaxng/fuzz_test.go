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

	// NewCompiler defaults to a deny-all FS, so Compile never touches the host
	// filesystem for include/externalRef hrefs in fuzz-generated schemas; those
	// loads fail closed without an os.ReadFile.
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			return
		}
		doc, err := helium.NewParser().Parse(t.Context(), data)
		if err != nil {
			return
		}
		_, _ = relaxng.NewCompiler().Compile(t.Context(), doc)
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
		schemaDom, err := helium.NewParser().Parse(ctx, schemaData)
		if err != nil {
			return
		}

		grammar, err := relaxng.NewCompiler().Compile(t.Context(), schemaDom)
		if err != nil {
			return
		}

		instanceDom, err := helium.NewParser().Parse(ctx, instanceData)
		if err != nil {
			return
		}

		_ = relaxng.NewValidator(grammar).Validate(ctx, instanceDom)
	})
}
