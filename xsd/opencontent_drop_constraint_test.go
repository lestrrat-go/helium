package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpenContent_DropsBaseLocalConstraint covers the soundness finding that a
// restriction which DROPS a base LOCAL element declaration and re-admits the name
// through an (enforced) open-content wildcard governed by the GLOBAL declaration must
// reject when the global is NOT at least as restrictive as the dropped local on the
// constraints the dynamic wildcard-EDC does not enforce: fixed, nillable, and identity
// constraints. The base's declared local wins attribution (element-over-wildcard
// precedence), so the base rejects content the dropped+re-admitted derived accepts.
func TestOpenContent_DropsBaseLocalConstraint(t *testing.T) {
	t.Parallel()

	// build a schema where base B declares interleave-strict open content over the
	// target namespace AND a local element e (type etype, carrying localExtra); the
	// global e is declared by globalDecl. R restricts B keeping the same open content
	// but DROPPING e, so e spills to open content governed by the global.
	build := func(globalDecl, etype, localAttr, localChild string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  ` + globalDecl + `
  <xs:complexType name="ET"><xs:attribute name="id" type="xs:int"/></xs:complexType>
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##targetNamespace" processContents="strict"/></xs:openContent>
    <xs:sequence>
      <xs:element name="e" type="` + etype + `" minOccurs="0"` + localAttr + `>` + localChild + `</xs:element>
      <xs:element name="keep" type="xs:string" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="R">
    <xs:complexContent><xs:restriction base="t:B">
      <xs:openContent mode="interleave"><xs:any namespace="##targetNamespace" processContents="strict"/></xs:openContent>
      <xs:sequence>
        <xs:element name="keep" type="xs:string" minOccurs="0"/>
      </xs:sequence>
    </xs:restriction></xs:complexContent>
  </xs:complexType>
  <xs:element name="doc" type="t:R"/>
</xs:schema>`
	}

	t.Run("dropped local fixed re-admitted via no-fixed global is rejected", func(t *testing.T) {
		t.Parallel()
		schema := build(`<xs:element name="e" type="xs:int"/>`, "xs:int", ` fixed="5"`, ``)
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "the global has no fixed value, so it admits content the base local forbade")
	})

	t.Run("dropped local re-admitted via a more-nillable global is rejected", func(t *testing.T) {
		t.Parallel()
		schema := build(`<xs:element name="e" type="xs:int" nillable="true"/>`, "xs:int", ``, ``)
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "the global is nillable while the base local is not")
	})

	t.Run("dropped local with an identity constraint the global lacks is rejected", func(t *testing.T) {
		t.Parallel()
		schema := build(`<xs:element name="e" type="t:ET"/>`, "t:ET", ``,
			`<xs:key name="ek"><xs:selector xpath="."/><xs:field xpath="@id"/></xs:key>`)
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "the global does not impose the base local's xs:key")
	})

	t.Run("dropped local re-admitted via a fixed-compatible global is accepted", func(t *testing.T) {
		t.Parallel()
		// global carries the SAME fixed value, is not more nillable, and the local has
		// no identity constraints: nothing is lost, so the drop is a sound restriction.
		schema := build(`<xs:element name="e" type="xs:int" fixed="5"/>`, "xs:int", ` fixed="5"`, ``)
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "the global is at least as restrictive on fixed/nillable/IDC")
	})
}
