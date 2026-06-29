package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestVersion11DeprecatedDatatypesNamespaceRejected(t *testing.T) {
	t.Parallel()

	const schemaXML = `<schema xmlns="http://www.w3.org/2001/XMLSchema"
        xmlns:xsdt="http://www.w3.org/2001/XMLSchema-datatypes"
        targetNamespace="urn:test">
  <complexType name="TimerType">
    <attribute name="time" type="xsdt:gYear"/>
  </complexType>
</schema>`

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
	require.Contains(t, errs, "http://www.w3.org/2001/XMLSchema-datatypes")
	require.Contains(t, errs, "deprecated")
}
