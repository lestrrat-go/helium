package xsd_test

import (
	"fmt"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestWhiteSpaceValidRestriction verifies the "whiteSpace valid restriction"
// schema component constraint (XSD Part 2 §4.3.6): a simple type restricting a
// base with a more restrictive whiteSpace facet may only tighten, never loosen,
// that facet. The permitted ordering is preserve → replace → collapse (a derived
// value may equal or move rightward). A base 'collapse' admits only 'collapse';
// a base 'replace' forbids 'preserve'. This is version-independent, enforced in
// both XSD 1.0 (the default) and 1.1. (W3C msMeta/SimpleType_w3c stZ013, stZ018,
// stZ019, stZ022, stZ026-stZ028.)
func TestWhiteSpaceValidRestriction(t *testing.T) {
	t.Parallel()

	const (
		wantMsg    = "is not a valid restriction of the 'whiteSpace' value"
		wsPreserve = "preserve"
		wsReplace  = "replace"
		wsCollapse = "collapse"
	)

	// Two-step derivation: type_a restricts xs:string with baseWS, then type_b
	// restricts type_a with derivedWS.
	chainSchema := func(baseWS, derivedWS string) string {
		return fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="type_a">
    <xs:restriction base="xs:string">
      <xs:whiteSpace value="%s"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="type_b">
    <xs:restriction base="type_a">
      <xs:whiteSpace value="%s"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`, baseWS, derivedWS)
	}

	// Nested-simpleType base (no @base attribute): the inner anonymous simpleType
	// supplies the base whiteSpace, the outer restriction the derived value.
	nestedSchema := func(baseWS, derivedWS string) string {
		return fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="type_c">
    <xs:restriction>
      <xs:simpleType>
        <xs:restriction base="xs:string">
          <xs:whiteSpace value="%s"/>
        </xs:restriction>
      </xs:simpleType>
      <xs:whiteSpace value="%s"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`, baseWS, derivedWS)
	}

	compile := func(t *testing.T, schemaXML string, v ...xsd.Version) (string, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		compiler := xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector)
		for _, ver := range v {
			compiler = compiler.Version(ver)
		}
		_, cerr := compiler.Compile(t.Context(), doc)
		_, errStr := partitionCompileErrors(collector.Errors())
		return errStr, cerr
	}

	// Loosening a more-restrictive base whiteSpace is a schema error.
	invalid := []struct{ baseWS, derivedWS string }{
		{wsCollapse, wsReplace},  // stZ018
		{wsCollapse, wsPreserve}, // stZ019
		{wsReplace, wsPreserve},  // stZ013 type_b, stZ022
	}
	for _, tc := range invalid {
		t.Run(fmt.Sprintf("reject chain %s->%s", tc.baseWS, tc.derivedWS), func(t *testing.T) {
			t.Parallel()
			errStr, cerr := compile(t, chainSchema(tc.baseWS, tc.derivedWS))
			requireCompileResultErr(t, cerr)
			require.Contains(t, errStr, wantMsg)
		})
		t.Run(fmt.Sprintf("reject nested %s->%s", tc.baseWS, tc.derivedWS), func(t *testing.T) {
			t.Parallel()
			errStr, cerr := compile(t, nestedSchema(tc.baseWS, tc.derivedWS))
			requireCompileResultErr(t, cerr)
			require.Contains(t, errStr, wantMsg)
		})
	}

	// Tightening or keeping a base whiteSpace is valid (stZ017/020/021/023-025/029).
	valid := []struct{ baseWS, derivedWS string }{
		{wsCollapse, wsCollapse},
		{wsReplace, wsCollapse},
		{wsReplace, wsReplace},
		{wsPreserve, wsCollapse},
		{wsPreserve, wsReplace}, // stZ013 type_d, stZ029
		{wsPreserve, wsPreserve},
	}
	for _, tc := range valid {
		t.Run(fmt.Sprintf("accept chain %s->%s", tc.baseWS, tc.derivedWS), func(t *testing.T) {
			t.Parallel()
			_, cerr := compile(t, chainSchema(tc.baseWS, tc.derivedWS))
			require.NoError(t, cerr)
		})
		t.Run(fmt.Sprintf("accept nested %s->%s", tc.baseWS, tc.derivedWS), func(t *testing.T) {
			t.Parallel()
			_, cerr := compile(t, nestedSchema(tc.baseWS, tc.derivedWS))
			require.NoError(t, cerr)
		})
	}

	// Version-independence: an invalid loosening is rejected under 1.1 too.
	t.Run("reject under XSD 1.1", func(t *testing.T) {
		t.Parallel()
		errStr, cerr := compile(t, chainSchema(wsCollapse, wsPreserve), xsd.Version11)
		requireCompileResultErr(t, cerr)
		require.Contains(t, errStr, wantMsg)
	})
}
