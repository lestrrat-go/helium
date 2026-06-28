package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestConditionalInclusion exercises the XSD 1.1 version-control (vc:) namespace
// conditional-inclusion pre-pass: an element (with its subtree) is pruned from
// the schema unless every vc: condition on it holds for the active processor
// version. The cases mirror the W3C VC test suite (saxonData/VC).
func TestConditionalInclusion(t *testing.T) {
	compileVC := func(t *testing.T, c xsd.Compiler, schemaXML string) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		return c.Compile(t.Context(), doc)
	}
	validate := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), doc)
	}

	const ns = `xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"`

	t.Run("minVersion gates an xs:assert (vc001)", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema ` + ns + `>
  <xs:element name="temp">
    <xs:complexType>
       <xs:sequence/>
       <xs:attribute name="x" use="required"/>
       <xs:assert test="@x > 300" vc:minVersion="1.1"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		// Under 1.0 the assert is pruned, so x=204 is accepted.
		s10, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version10), schema)
		require.NoError(t, err)
		require.NoError(t, validate(t, s10, `<temp x="204"/>`))
		// Under 1.1 the assert is kept, so x=204 fails it.
		s11, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), schema)
		require.NoError(t, err)
		require.Error(t, validate(t, s11, `<temp x="204"/>`))
		require.NoError(t, validate(t, s11, `<temp x="304"/>`))
	})

	t.Run("typeAvailable known type keeps element (vc010)", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema ` + ns + `>
  <xs:element name="temp">
    <xs:complexType>
       <xs:sequence/>
       <xs:attribute name="x" use="required" vc:typeAvailable="xs:integer"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		s, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), schema)
		require.NoError(t, err)
		require.NoError(t, validate(t, s, `<temp x="204"/>`))
	})

	t.Run("typeUnavailable known type prunes element (vc011)", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema ` + ns + `>
  <xs:element name="temp">
    <xs:complexType>
       <xs:sequence/>
       <xs:attribute name="x" use="required" vc:typeUnavailable="xs:integer"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		s, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), schema)
		require.NoError(t, err)
		// x is pruned, so the attribute is not allowed.
		require.Error(t, validate(t, s, `<temp x="204"/>`))
		require.NoError(t, validate(t, s, `<temp/>`))
	})

	t.Run("typeAvailable mix prunes element (vc012)", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema ` + ns + `>
  <xs:element name="temp">
    <xs:complexType>
       <xs:sequence/>
       <xs:attribute name="x" use="required" vc:typeAvailable="xs:integer xs:bananaSkin"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		s, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), schema)
		require.NoError(t, err)
		require.Error(t, validate(t, s, `<temp x="204"/>`))
	})

	t.Run("typeUnavailable mix keeps element (vc013)", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema ` + ns + `>
  <xs:element name="temp">
    <xs:complexType>
       <xs:sequence/>
       <xs:attribute name="x" use="required" vc:typeUnavailable=" vc:list-of-QNames xs:integer "/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		s, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), schema)
		require.NoError(t, err)
		require.NoError(t, validate(t, s, `<temp x="204"/>`))
	})

	t.Run("xs:error availability switches branches (vc014)", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema ` + ns + ` elementFormDefault="qualified">
  <xs:element name="temp">
    <xs:complexType>
       <xs:sequence/>
       <xs:attribute name="x" use="required"/>
       <xs:attribute name="y" use="optional" type="xs:error" vc:typeAvailable="xs:error"/>
       <xs:attribute name="y" use="optional" vc:typeUnavailable="xs:error">
         <xs:simpleType>
           <xs:restriction base="xs:integer">
             <xs:pattern value="A"/>
           </xs:restriction>
         </xs:simpleType>
       </xs:attribute>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		// Both versions must compile (exactly one y survives) and reject y present.
		for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
			s, err := compileVC(t, xsd.NewCompiler().Version(v), schema)
			require.NoError(t, err)
			require.NoError(t, validate(t, s, `<temp x="204"/>`))
			require.Error(t, validate(t, s, `<temp x="204" y=""/>`))
		}
	})

	t.Run("facetAvailable mix prunes (vc022), facetUnavailable mix keeps (vc023)", func(t *testing.T) {
		t.Parallel()
		prune := `<xs:schema ` + ns + `>
  <xs:element name="temp">
    <xs:complexType>
       <xs:sequence/>
       <xs:attribute name="x" use="required" vc:facetAvailable="xs:pattern xs:bananaSkin"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		sp, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), prune)
		require.NoError(t, err)
		require.Error(t, validate(t, sp, `<temp x="204"/>`))

		keep := `<xs:schema ` + ns + `>
  <xs:element name="temp">
    <xs:complexType>
       <xs:sequence/>
       <xs:attribute name="x" use="required" vc:facetUnavailable=" vc:list-of-QNames xs:minInclusive "/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		sk, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), keep)
		require.NoError(t, err)
		require.NoError(t, validate(t, sk, `<temp x="204"/>`))
	})

	t.Run("empty typeAvailable is a no-op, empty typeUnavailable excludes (vc008)", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema ` + ns + `>
  <xs:element name="temp">
    <xs:complexType>
       <xs:sequence/>
       <xs:attribute name="keep" use="optional" vc:typeAvailable=""/>
       <xs:attribute name="drop" use="required" vc:typeUnavailable=""/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		s, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), schema)
		require.NoError(t, err)
		// keep survived (empty typeAvailable = no effect); drop pruned (so the
		// otherwise-required attribute is absent-OK and present-rejected).
		require.NoError(t, validate(t, s, `<temp keep="anything"/>`))
		require.Error(t, validate(t, s, `<temp drop="x"/>`))
	})

	t.Run("root vc condition makes the schema empty (vc006)", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema ` + ns + ` vc:maxVersion="0.9">
  <xs:element name="temp" type="xs:string"/>
</xs:schema>`
		s, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), schema)
		require.NoError(t, err)
		// temp was pruned along with the whole document, so it is undeclared.
		require.Error(t, validate(t, s, `<temp>hi</temp>`))
	})

	t.Run("misspelt vc attribute is ignored", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema ` + ns + `>
  <xs:element name="temp" vc:what-on-earth-is-this="surprise!" type="xs:string"/>
</xs:schema>`
		s, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), schema)
		require.NoError(t, err)
		require.NoError(t, validate(t, s, `<temp>hi</temp>`))
	})

	t.Run("invalid decimal: error under 1.1, tolerated under 1.0 (vc902/vc903)", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema ` + ns + `>
  <xs:element name="temp">
    <xs:complexType>
       <xs:sequence/>
       <xs:attribute name="x" use="required"/>
       <xs:attribute name="y" use="optional" vc:minVersion="10g"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		_, err11 := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), schema)
		require.Error(t, err11)
		_, err10 := compileVC(t, xsd.NewCompiler().Version(xsd.Version10), schema)
		require.NoError(t, err10)
	})

	t.Run("invalid minVersion lexical is an error under 1.1 (vc901)", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema ` + ns + `>
  <xs:element name="temp">
    <xs:complexType>
       <xs:sequence/>
       <xs:attribute name="x" use="required"/>
       <xs:assert test="@x > 300" vc:minVersion="1.1.3"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		_, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), schema)
		require.Error(t, err)
	})

	t.Run("unbound prefix in QName list is an error under 1.1 (vc904)", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema ` + ns + `>
  <xs:element name="temp">
    <xs:complexType>
       <xs:sequence/>
       <xs:attribute name="x" use="required" vc:typeUnavailable=" vx:list-of-QNames xs:integer"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		_, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), schema)
		require.Error(t, err)
	})

	t.Run("invalid QName lexical in list is an error under 1.1 (vc905)", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema ` + ns + `>
  <xs:element name="temp">
    <xs:complexType>
       <xs:sequence/>
       <xs:attribute name="x" use="required" vc:typeUnavailable=" xs:integer 23"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		_, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), schema)
		require.Error(t, err)
	})

	t.Run("high-precision decimal bounds compare exactly (no float rounding)", func(t *testing.T) {
		t.Parallel()
		// maxVersion is just above 1.1 — 1.1 < max, so the element is KEPT under
		// 1.1. A float64 parse would round this to 1.1 and wrongly exclude it.
		keep := `<xs:schema ` + ns + `>
  <xs:element name="temp">
    <xs:complexType>
       <xs:sequence/>
       <xs:attribute name="x" use="required" vc:maxVersion="1.1000000000000000000000000000000000001"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		sk, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), keep)
		require.NoError(t, err)
		require.NoError(t, validate(t, sk, `<temp x="204"/>`))

		// A many-digit (but lexically valid) decimal is NOT malformed: it would
		// overflow a float64, but exact comparison treats it as a huge minVersion,
		// so 1.1 < min and the element is pruned (attribute not allowed) — and the
		// schema still compiles without a "malformed decimal" error.
		bigMin := "9" + strings.Repeat("0", 400)
		prune := `<xs:schema ` + ns + `>
  <xs:element name="temp">
    <xs:complexType>
       <xs:sequence/>
       <xs:attribute name="x" use="required" vc:minVersion="` + bigMin + `"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		sp, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), prune)
		require.NoError(t, err)
		require.Error(t, validate(t, sp, `<temp x="204"/>`))
	})

	t.Run("excluded root with a bogus blockDefault compiles to empty (no error)", func(t *testing.T) {
		t.Parallel()
		// The root is vc-excluded under 1.1, so its (never-used) blockDefault must
		// not be validated; the schema compiles to an empty (valid) schema.
		schema := `<xs:schema ` + ns + ` vc:maxVersion="0.9" blockDefault="bogus">
  <xs:element name="temp" type="xs:string"/>
</xs:schema>`
		s, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), schema)
		require.NoError(t, err)
		require.Error(t, validate(t, s, `<temp>hi</temp>`)) // temp was pruned with the root
	})

	t.Run("conditional element declarations: only one root survives (ibm s4_2_2)", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema ` + ns + ` targetNamespace="a" elementFormDefault="qualified">
  <xs:element name="root" vc:minVersion="3.2">
    <xs:complexType><xs:sequence><xs:element name="ele32"/></xs:sequence></xs:complexType>
  </xs:element>
  <xs:element name="root" vc:minversion="1.0" vc:maxVersion="3.2">
    <xs:complexType><xs:sequence><xs:element name="ele11"/></xs:sequence></xs:complexType>
  </xs:element>
</xs:schema>`
		s, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), schema)
		require.NoError(t, err)
		require.NoError(t, validate(t, s, `<root xmlns="a"><ele11/></root>`))
		require.Error(t, validate(t, s, `<root xmlns="a"><ele32/></root>`))
	})
}
