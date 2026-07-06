package xsd_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

// src-redefine.6.1.1/6.1.2 (XSD Part 1 §4.2.3): a <group> child of <xs:redefine>
// that references itself must do so exactly once and with the reference's
// minOccurs = maxOccurs = 1. A group that does NOT reference itself is governed
// by clause 6.2 (valid restriction of the original) and is not constrained by
// this rule. The rule is version-independent, so it is enforced in the default
// XSD 1.0 compiler. Mirrors W3C msMeta/Schema_w3c.xml schR3/schR4.
func TestRedefine_GroupSelfReferenceCardinality(t *testing.T) {
	t.Parallel()

	const baseXSD = "base.xsd"
	base := &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g">
    <xs:choice>
      <xs:element name="c1" type="xs:int"/>
      <xs:element name="c2" type="xs:int"/>
    </xs:choice>
  </xs:group>
  <xs:group name="other">
    <xs:sequence><xs:element name="o1" type="xs:int"/></xs:sequence>
  </xs:group>
</xs:schema>`)}

	for _, tc := range []struct {
		name    string
		redef   string
		wantErr string
	}{
		{
			name: "self-reference with occurrence 1/1 is valid",
			redef: `<xs:group name="g">
    <xs:choice>
      <xs:element name="c1" type="xs:int"/>
      <xs:group ref="g"/>
      <xs:element name="c2" type="xs:int"/>
    </xs:choice>
  </xs:group>`,
		},
		{
			name: "self-reference with minOccurs=0 is invalid",
			redef: `<xs:group name="g">
    <xs:choice>
      <xs:element name="c1" type="xs:int"/>
      <xs:group ref="g" minOccurs="0"/>
      <xs:element name="c2" type="xs:int"/>
    </xs:choice>
  </xs:group>`,
			wantErr: "src-redefine.6.1.2",
		},
		{
			name: "self-reference with maxOccurs=2 is invalid",
			redef: `<xs:group name="g">
    <xs:choice>
      <xs:element name="c1" type="xs:int"/>
      <xs:group ref="g" maxOccurs="2"/>
      <xs:element name="c2" type="xs:int"/>
    </xs:choice>
  </xs:group>`,
			wantErr: "src-redefine.6.1.2",
		},
		{
			name: "two self-references are invalid",
			redef: `<xs:group name="g">
    <xs:choice>
      <xs:group ref="g"/>
      <xs:group ref="g"/>
    </xs:choice>
  </xs:group>`,
			wantErr: "src-redefine.6.1.1",
		},
		{
			name: "different group reference is invalid",
			redef: `<xs:group name="g">
    <xs:choice>
      <xs:element name="c1" type="xs:int"/>
      <xs:group ref="other"/>
    </xs:choice>
  </xs:group>`,
			wantErr: "src-redefine.6.1.1",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			const mainXSD = "main.xsd"
			fsys := fstest.MapFS{
				baseXSD: base,
				mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base.xsd">
    ` + tc.redef + `
  </xs:redefine>
</xs:schema>`)},
			}
			errStr, err := compileRedefineFS(t, fsys, mainXSD)
			if tc.wantErr != "" {
				require.Error(t, err, "expected src-redefine.6.1 rejection")
				require.Contains(t, errStr, tc.wantErr)
				return
			}
			require.NoError(t, err, "valid redefine must compile; got: %q", errStr)
		})
	}
}
