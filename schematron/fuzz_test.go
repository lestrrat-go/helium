package schematron_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/schematron"
)

func FuzzCompile(f *testing.F) {
	f.Add([]byte(`<?xml version="1.0"?>
<schema xmlns="http://purl.oclc.org/dsdl/schematron">
  <pattern>
    <rule context="root">
      <assert test="child">missing child</assert>
    </rule>
  </pattern>
</schema>`))
	f.Add([]byte(`<?xml version="1.0"?>
<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern>
    <rule context="item">
      <report test="@id">has id</report>
    </rule>
  </pattern>
</schema>`))
	f.Add([]byte(``))
	f.Add([]byte(`not a schema`))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			return
		}

		doc, err := helium.NewParser().Parse(t.Context(), data)
		if err != nil {
			return
		}

		_, _ = schematron.NewCompiler().Compile(t.Context(), doc)
	})
}

func FuzzValidate(f *testing.F) {
	f.Add(
		[]byte(`<?xml version="1.0"?>
<schema xmlns="http://purl.oclc.org/dsdl/schematron">
  <pattern>
    <rule context="root">
      <assert test="child">missing child</assert>
    </rule>
  </pattern>
</schema>`),
		[]byte(`<?xml version="1.0"?><root><child/></root>`),
	)
	f.Add(
		[]byte(`<?xml version="1.0"?>
<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern>
    <rule context="item">
      <report test="@id">has id</report>
    </rule>
  </pattern>
</schema>`),
		[]byte(`<?xml version="1.0"?><item id="v"/>`),
	)

	f.Fuzz(func(t *testing.T, schemaData, instanceData []byte) {
		if len(schemaData) > 1<<20 || len(instanceData) > 1<<20 {
			return
		}

		ctx := t.Context()
		schemaDoc, err := helium.NewParser().Parse(ctx, schemaData)
		if err != nil {
			return
		}

		schema, err := schematron.NewCompiler().Compile(ctx, schemaDoc)
		if err != nil {
			return
		}

		instanceDoc, err := helium.NewParser().Parse(ctx, instanceData)
		if err != nil {
			return
		}

		_ = schematron.NewValidator(schema).Validate(ctx, instanceDoc)
	})
}
