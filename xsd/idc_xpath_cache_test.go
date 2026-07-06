package xsd_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestIdentityConstraintXPathCacheLargeDocument(t *testing.T) {
	const rows = 5000
	schema := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="row" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="a" type="xs:NCName" use="required"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="k">
      <xs:selector xpath="t:row|t:row"/>
      <xs:field xpath="@a|@a"/>
    </xs:key>
  </xs:element>
</xs:schema>`

	var inst strings.Builder
	inst.WriteString(`<root xmlns="urn:t">`)
	for i := range rows {
		fmt.Fprintf(&inst, `<row a="v%d"/>`, i)
	}
	inst.WriteString(`</root>`)

	sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err)
	compiled, err := xsd.NewCompiler().Compile(t.Context(), sdoc)
	require.NoError(t, err)
	idoc, err := helium.NewParser().Parse(t.Context(), []byte(inst.String()))
	require.NoError(t, err)

	validateCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	require.NoError(t, xsd.NewValidator(compiled).Validate(validateCtx, idoc))
}
