package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestExtensionChainDeterministic guards against issue #437: the extension
// content-model merge in link_refs.go iterated c.typeRefs (a Go map, randomized
// per call) and captured the base type's ContentModel at merge time. When a
// derived type was merged before its base, the captured pointer was orphaned by
// the base's later reassignment, silently dropping transitively-inherited
// particles. A multi-level A->B->C extension chain therefore validated
// nondeterministically. Compiling many times exercises different map orders.
func TestExtensionChainDeterministic(t *testing.T) {
	const schemaXSD = `<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="A">
    <xs:sequence><xs:element name="Required" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="B">
    <xs:complexContent><xs:extension base="A">
      <xs:sequence><xs:element name="MidExtra" type="xs:string" minOccurs="0"/></xs:sequence>
    </xs:extension></xs:complexContent>
  </xs:complexType>
  <xs:complexType name="C">
    <xs:complexContent><xs:extension base="B">
      <xs:sequence><xs:element name="LeafExtra" type="xs:string" minOccurs="0"/></xs:sequence>
    </xs:extension></xs:complexContent>
  </xs:complexType>
  <xs:element name="Doc" type="C"/>
</xs:schema>`

	const docXML = `<?xml version="1.0"?><Doc><Required>x</Required></Doc>`

	for i := 0; i < 200; i++ {
		schemaDOM, err := helium.NewParser().Parse(t.Context(), []byte(schemaXSD))
		require.NoError(t, err, "iter %d: parse schema", i)
		schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDOM)
		require.NoError(t, err, "iter %d: compile", i)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(docXML))
		require.NoError(t, err, "iter %d: parse doc", i)

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		err = xsd.NewValidator(schema).ErrorHandler(collector).Validate(t.Context(), doc)
		if err != nil {
			var first string
			if errs := collector.Errors(); len(errs) > 0 {
				first = strings.SplitN(errs[0].Error(), "\n", 2)[0]
			}
			t.Fatalf("iter %d: expected valid; transitively-inherited particle dropped: %s", i, first)
		}
	}
}
