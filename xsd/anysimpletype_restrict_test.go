package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestAnySimpleTypeRestriction covers the version-INDEPENDENT rule that the
// simple ur-type xs:anySimpleType must NOT be DIRECTLY restricted — as a
// simpleType restriction base, or as the DIRECT base of a simpleContent
// restriction (cos-st-restricts / XML Schema Part 2 §2.4.1: a restriction's base
// must be atomic/list/union, and anySimpleType is none of those). libxml2 rejects
// this in XSD 1.0 too, so the check runs in BOTH versions (mirrors W3C msMeta
// groups stZ005/stZ006/stZ009/stZ011 and addB110, all expected invalid).
//
// These stay VALID in XSD 1.0 (rejected only in 1.1): a simpleContent EXTENSION
// of anySimpleType, USE of anySimpleType as an element/attribute type, and a
// simpleContent RESTRICTION of a COMPLEX base whose content type merely resolves
// to anySimpleType (W3C bug 14559 carve-out, stZ007/stZ047/stZ055 valid in 1.0).
func TestAnySimpleTypeRestriction(t *testing.T) {
	t.Parallel()

	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`
	const tail = `</xs:schema>`

	// Both XSD versions, so each case runs in 1.0 and 1.1 (versionName renders the
	// subtest label).
	versions := []xsd.Version{xsd.Version10, xsd.Version11}

	// simpleType restriction whose base is the ur-type — stZ005/stZ006/stZ011 shape.
	const simpleTypeRestrict = `<xs:simpleType name="t"><xs:restriction base="xs:anySimpleType"/></xs:simpleType>`
	// simpleContent complexType restriction directly of the ur-type — stZ009 shape.
	const simpleContentRestrict = `<xs:complexType name="t"><xs:simpleContent><xs:restriction base="xs:anySimpleType"/></xs:simpleContent></xs:complexType>`

	t.Run("rejected in both versions", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			schema string
		}{
			{"simpleType-restriction-base", simpleTypeRestrict},
			{"simpleContent-restriction-base", simpleContentRestrict},
		} {
			for _, ver := range versions {
				t.Run(tc.name+"/"+versionName(ver), func(t *testing.T) {
					t.Parallel()
					require.NotEmpty(t, compileSchemaErrorsVersion(t, head+tc.schema+tail, ver == xsd.Version11),
						"restricting xs:anySimpleType must be rejected in %s", versionName(ver))
				})
			}
		}
	})

	// A simpleContent restriction of a COMPLEX base whose content type resolves to
	// the ur-type is valid in XSD 1.0 but invalid in XSD 1.1 (stZ007 shape). The
	// 1.0 path must not over-reject it.
	const simpleContentRestrictComplexBase = `<xs:complexType name="t1"><xs:simpleContent><xs:extension base="xs:anySimpleType"/></xs:simpleContent></xs:complexType>` +
		`<xs:complexType name="t2"><xs:simpleContent><xs:restriction base="t1"/></xs:simpleContent></xs:complexType>`

	t.Run("simpleContent-restriction-of-complex-base valid in 1.0 only", func(t *testing.T) {
		t.Parallel()
		t.Run("xsd10-valid", func(t *testing.T) {
			t.Parallel()
			require.Empty(t, compileSchemaErrorsVersion(t, head+simpleContentRestrictComplexBase+tail, false),
				"restriction of a complex anySimpleType-content base must compile in XSD 1.0")
		})
		t.Run("xsd11-invalid", func(t *testing.T) {
			t.Parallel()
			require.NotEmpty(t, compileSchemaErrorsVersion(t, head+simpleContentRestrictComplexBase+tail, true),
				"restriction of a complex anySimpleType-content base must be rejected in XSD 1.1")
		})
	})

	// The extension/usage arms must stay valid in BOTH versions — the whole risk
	// of the fix is over-rejecting these.
	t.Run("valid in both versions", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			schema string
		}{
			// simpleContent EXTENSION of the ur-type (stZ007 shape).
			{"simpleContent-extension-base", `<xs:complexType name="t"><xs:simpleContent><xs:extension base="xs:anySimpleType"/></xs:simpleContent></xs:complexType>`},
			// Element typed xs:anySimpleType.
			{"element-type-anySimpleType", `<xs:element name="e" type="xs:anySimpleType"/>`},
			// Attribute typed xs:anySimpleType.
			{"attribute-type-anySimpleType", `<xs:attribute name="a" type="xs:anySimpleType"/>`},
			// simpleContent restriction of a base whose content is narrowed by a
			// nested <xs:simpleType> — the effective content type is xs:string, NOT
			// the ur-type, so it must NOT be rejected.
			{"simpleContent-restriction-narrowed", `<xs:complexType name="t"><xs:simpleContent><xs:restriction base="xs:anySimpleType"><xs:simpleType><xs:restriction base="xs:string"><xs:maxLength value="2"/></xs:restriction></xs:simpleType></xs:restriction></xs:simpleContent></xs:complexType>`},
		} {
			for _, ver := range versions {
				t.Run(tc.name+"/"+versionName(ver), func(t *testing.T) {
					t.Parallel()
					require.Empty(t, compileSchemaErrorsVersion(t, head+tc.schema+tail, ver == xsd.Version11),
						"anySimpleType extension/usage must compile in %s", versionName(ver))
				})
			}
		}
	})
}
