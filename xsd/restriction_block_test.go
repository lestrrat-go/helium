package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestElementRestrictionDisallowedSubstitutions covers Particle Valid
// (Restriction) NameAndTypeOK clause 3.2.4 (§3.9.6): a derived element
// declaration's {disallowed substitutions} (the @block value) must be a SUPERSET
// of the base element declaration's. A restriction may tighten the disallowed set
// but never loosen it. The rule is version-INDEPENDENT and exercised here in the
// default (XSD 1.0) compiler, mirroring W3C msData/particles particlesIg007-016
// and particlesL014-L027.
func TestElementRestrictionDisallowedSubstitutions(t *testing.T) {
	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">`

	// base type carries a single element "e" with the given @block; the derived
	// type restricts it with the given @block on "e".
	mk := func(baseBlock, derBlock string) string {
		be := `<xs:element name="e"`
		if baseBlock != "" {
			be += ` block="` + baseBlock + `"`
		}
		be += `/>`
		de := `<xs:element name="e"`
		if derBlock != "" {
			de += ` block="` + derBlock + `"`
		}
		de += `/>`
		return head + `
  <xs:complexType name="base"><xs:choice>` + be + `</xs:choice></xs:complexType>
  <xs:complexType name="der"><xs:complexContent>
    <xs:restriction base="t:base"><xs:choice>` + de + `</xs:choice></xs:restriction>
  </xs:complexContent></xs:complexType>
  <xs:element name="doc" type="t:der"/>
</xs:schema>`
	}

	compile := func(t *testing.T, src string) error {
		t.Helper()
		doc, perr := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, perr)
		_, err := xsd.NewCompiler().Compile(t.Context(), doc)
		return err
	}

	const (
		all = "#all"
		ext = "extension"
		res = "restriction"
		sub = "substitution"
	)
	tests := []struct {
		name      string
		baseBlock string
		derBlock  string
		wantErr   bool
	}{
		{"all restricted to extension is invalid", all, ext, true},
		{"all restricted to restriction is invalid", all, res, true},
		{"extension restricted to substitution is invalid", ext, sub, true},
		{"substitution restricted to restriction is invalid", sub, res, true},
		{"substitution restricted to absent is invalid", sub, "", true},
		{"extension restricted to all is valid", ext, all, false},
		{"extension restricted to extension is valid", ext, ext, false},
		{"substitution restricted to all is valid", sub, all, false},
		{"absent base with substitution derived is valid", "", sub, false},
		{"both absent is valid", "", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := compile(t, mk(tc.baseBlock, tc.derBlock))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
