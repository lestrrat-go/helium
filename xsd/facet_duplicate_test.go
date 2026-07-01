package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestDuplicateSingletonFacetsRejected(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="zoned">
    <xs:restriction base="xs:dateTime">
      <xs:explicitTimezone value="optional"/>
      <xs:explicitTimezone value="prohibited"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
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
	require.Contains(t, errs, "facet 'explicitTimezone'")
	require.Contains(t, errs, "specified more than once")
}

func compileErrorsString(errs []error) string {
	var b strings.Builder
	for _, err := range errs {
		b.WriteString(err.Error())
		b.WriteByte('\n')
	}
	return b.String()
}
