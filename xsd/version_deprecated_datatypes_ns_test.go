package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestVersion11DeprecatedDatatypesNamespaceRejected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		schema string
	}{
		{
			name: "attribute type QName",
			schema: `<schema xmlns="http://www.w3.org/2001/XMLSchema"
        xmlns:xsdt="http://www.w3.org/2001/XMLSchema-datatypes"
        targetNamespace="urn:test">
  <complexType name="TimerType">
    <attribute name="time" type="xsdt:gYear"/>
  </complexType>
</schema>`,
		},
		{
			name: "element type QName",
			schema: `<schema xmlns="http://www.w3.org/2001/XMLSchema"
        xmlns:xsdt="http://www.w3.org/2001/XMLSchema-datatypes">
  <element name="root" type="xsdt:string"/>
</schema>`,
		},
		{
			name: "complex type base QName",
			schema: `<schema xmlns="http://www.w3.org/2001/XMLSchema"
        xmlns:xsdt="http://www.w3.org/2001/XMLSchema-datatypes">
  <complexType name="T">
    <complexContent>
      <extension base="xsdt:anyType"/>
    </complexContent>
  </complexType>
</schema>`,
		},
		{
			name: "simple type base QName",
			schema: `<schema xmlns="http://www.w3.org/2001/XMLSchema"
        xmlns:xsdt="http://www.w3.org/2001/XMLSchema-datatypes">
  <simpleType name="S">
    <restriction base="xsdt:string"/>
  </simpleType>
</schema>`,
		},
		{
			name: "list itemType QName",
			schema: `<schema xmlns="http://www.w3.org/2001/XMLSchema"
        xmlns:xsdt="http://www.w3.org/2001/XMLSchema-datatypes">
  <simpleType name="S">
    <list itemType="xsdt:string"/>
  </simpleType>
</schema>`,
		},
		{
			name: "union memberTypes QName",
			schema: `<schema xmlns="http://www.w3.org/2001/XMLSchema"
        xmlns:xsdt="http://www.w3.org/2001/XMLSchema-datatypes">
  <simpleType name="S">
    <union memberTypes="xsdt:string string"/>
  </simpleType>
</schema>`,
		},
		{
			name: "alternative type QName",
			schema: `<schema xmlns="http://www.w3.org/2001/XMLSchema"
        xmlns:xsdt="http://www.w3.org/2001/XMLSchema-datatypes"
        xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning" vc:minVersion="1.1">
  <element name="root" type="string">
    <alternative test="true()" type="xsdt:string"/>
  </element>
</schema>`,
		},
		{
			name: "identity constraint ref QName",
			schema: `<schema xmlns="http://www.w3.org/2001/XMLSchema"
        targetNamespace="http://www.w3.org/2001/XMLSchema-datatypes"
        xmlns:xsdt="http://www.w3.org/2001/XMLSchema-datatypes">
  <element name="root">
    <complexType>
      <sequence>
        <element name="item" maxOccurs="unbounded">
          <complexType><attribute name="code" type="string"/></complexType>
        </element>
      </sequence>
    </complexType>
    <unique name="u"><selector xpath="item"/><field xpath="@code"/></unique>
    <unique ref="xsdt:u"/>
  </element>
</schema>`,
		},
		{
			name: "keyref refer QName",
			schema: `<schema xmlns="http://www.w3.org/2001/XMLSchema"
        targetNamespace="http://www.w3.org/2001/XMLSchema-datatypes"
        xmlns:xsdt="http://www.w3.org/2001/XMLSchema-datatypes">
  <element name="root">
    <complexType>
      <sequence>
        <element name="item" maxOccurs="unbounded">
          <complexType>
            <attribute name="code" type="string"/>
            <attribute name="ref" type="string"/>
          </complexType>
        </element>
      </sequence>
    </complexType>
    <key name="k"><selector xpath="item"/><field xpath="@code"/></key>
    <keyref name="kr" refer="xsdt:k"><selector xpath="item"/><field xpath="@ref"/></keyref>
  </element>
</schema>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.schema))
			require.NoError(t, err)

			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			_, err = xsd.NewCompiler().
				Version(xsd.Version11).
				Label("test.xsd").
				ErrorHandler(collector).
				Compile(t.Context(), doc)
			_ = collector.Close()

			require.ErrorIs(t, err, xsd.ErrCompilationFailed)
			errs := compileErrorsString(collector.Errors())
			require.Contains(t, errs, "http://www.w3.org/2001/XMLSchema-datatypes")
			require.Contains(t, errs, "deprecated")
			require.NotContains(t, strings.ToLower(errs), "does not resolve")
		})
	}
}
