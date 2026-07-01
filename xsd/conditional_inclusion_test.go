package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

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

	t.Run("excluded root with a malformed vc value still errors under 1.1", func(t *testing.T) {
		t.Parallel()
		// The root is vc-excluded (maxVersion 0.9 <= 1.1) AND carries a malformed
		// minVersion; the malformed-value schema error must NOT be swallowed by the
		// empty-schema short-circuit.
		schema := `<xs:schema ` + ns + ` vc:maxVersion="0.9" vc:minVersion="1.1.3">
  <xs:element name="temp" type="xs:string"/>
</xs:schema>`
		_, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), schema)
		require.Error(t, err)
		// Under 1.0 the malformed value is tolerated; the root (maxVersion 0.9) is
		// still excluded, so the schema compiles to an empty schema.
		s10, err10 := compileVC(t, xsd.NewCompiler().Version(xsd.Version10), schema)
		require.NoError(t, err10)
		require.Error(t, validate(t, s10, `<temp>hi</temp>`))
	})

	t.Run("NBSP in vc:minVersion is malformed (not ASCII-trimmed) under 1.1", func(t *testing.T) {
		t.Parallel()
		// A leading NBSP (U+00A0) is NOT XSD whitespace, so the value is not a valid
		// xs:decimal and is a fatal schema error under 1.1 — it must not be silently
		// trimmed to "1.1".
		schema := `<xs:schema ` + ns + `>
  <xs:element name="temp">
    <xs:complexType>
       <xs:sequence/>
       <xs:attribute name="x" use="required"/>
       <xs:assert test="@x > 300" vc:minVersion="` + " " + `1.1"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		_, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), schema)
		require.Error(t, err)
	})

	t.Run("resolveVersion: NBSP/below-1.1 minVersion hint does not auto-select 1.1", func(t *testing.T) {
		t.Parallel()
		// No explicit Compiler.Version(): the root vc:minVersion hint drives version
		// auto-selection. type xs:double reveals the selected version via the 1.1-only
		// "+INF" lexical (accepted only under 1.1). resolveVersion must use the same
		// ASCII-trim + exact-decimal rules as the pre-pass.
		schemaFor := func(minVer string) string {
			return `<xs:schema ` + ns + ` vc:minVersion="` + minVer + `">
  <xs:element name="v" type="xs:double"/>
</xs:schema>`
		}
		// "1.10" == 1.1 exactly → selects 1.1 → +INF accepted (and pv==minVersion so
		// the root is not self-excluded).
		sEq, err := compileVC(t, xsd.NewCompiler(), schemaFor("1.10"))
		require.NoError(t, err)
		require.NoError(t, validate(t, sEq, `<v>+INF</v>`))
		// NBSP-padded → not a valid xs:decimal → no hint → default 1.0 → +INF rejected
		// (a float-based strings.TrimSpace parse would have wrongly selected 1.1).
		sNBSP, err := compileVC(t, xsd.NewCompiler(), schemaFor("\u00a0"+"1.1"))
		require.NoError(t, err)
		require.Error(t, validate(t, sNBSP, `<v>+INF</v>`))
		// High-precision value just BELOW 1.1 → exact compare keeps 1.0 → +INF rejected
		// (a float64 parse would round it up to 1.1 and wrongly select 1.1).
		sBelow, err := compileVC(t, xsd.NewCompiler(), schemaFor("1.09999999999999999999999999"))
		require.NoError(t, err)
		require.Error(t, validate(t, sBelow, `<v>+INF</v>`))
	})

	t.Run("vc-excluded included root with mismatched TNS contributes empty (no error)", func(t *testing.T) {
		t.Parallel()
		// The included document's root is vc-excluded (maxVersion 0.9) AND declares a
		// targetNamespace incompatible with the including schema. Conditional
		// inclusion must run BEFORE the TNS compatibility check, so the excluded root
		// contributes an empty schema instead of failing the include for TNS mismatch.
		const mainXSD = "vc_inc_main.xsd"
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema ` + ns + ` targetNamespace="urn:main">
  <xs:include schemaLocation="vc_inc_other.xsd"/>
  <xs:element name="temp" type="xs:string"/>
</xs:schema>`)},
			"vc_inc_other.xsd": &fstest.MapFile{Data: []byte(`<xs:schema ` + ns + ` targetNamespace="urn:other" vc:maxVersion="0.9">
  <xs:element name="bogus" type="xs:string"/>
</xs:schema>`)},
		}
		data, err := fsys.ReadFile(mainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).Label(mainXSD).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, err)
		require.NotNil(t, schema)
	})

	t.Run("malformed vc in an imported schema is reported, not dropped", func(t *testing.T) {
		t.Parallel()
		// The imported schema records a malformed-vc fatal diagnostic in the import
		// sub-collector during the pre-pass, then fails parseSchemaChildren on a
		// nameless top-level complexType (an early `return err` path in loadImport).
		// The sub-collector's diagnostic and error count must still be propagated to
		// the parent (so the compile fails) rather than silently dropped behind the
		// import's I/O-warning demotion.
		const mainXSD = "vc_imp_main.xsd"
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema ` + ns + ` targetNamespace="urn:main" xmlns:imp="urn:imp">
  <xs:import namespace="urn:imp" schemaLocation="vc_imp_bad.xsd"/>
  <xs:element name="temp" type="xs:string"/>
</xs:schema>`)},
			"vc_imp_bad.xsd": &fstest.MapFile{Data: []byte(`<xs:schema ` + ns + ` targetNamespace="urn:imp">
  <xs:element name="e" vc:minVersion="1.1.3" type="xs:string"/>
  <xs:complexType/>
</xs:schema>`)},
		}
		data, err := fsys.ReadFile(mainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, compileErr := xsd.NewCompiler().Version(xsd.Version11).Label(mainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, collector.Close())
		require.Error(t, compileErr)
		var combined strings.Builder
		for _, e := range collector.Errors() {
			combined.WriteString(e.Error())
			combined.WriteByte('\n')
		}
		require.Contains(t, combined.String(), "1.1.3")
	})

	t.Run("xs:redefine of a vc-excluded schema rejects absent-target overrides", func(t *testing.T) {
		t.Parallel()
		// The redefined document's root is vc-excluded (minVersion 2.0 > 1.1), so it
		// contributes NO components. The <xs:redefine> override of "S" therefore
		// targets a now-absent component and must be REJECTED (not silently accepted
		// by skipping the override children).
		const mainXSD = "vc_redef_main.xsd"
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema ` + ns + ` targetNamespace="urn:t">
  <xs:redefine schemaLocation="vc_redef_base.xsd">
    <xs:simpleType name="S">
      <xs:restriction base="S">
        <xs:maxLength value="5"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:redefine>
</xs:schema>`)},
			"vc_redef_base.xsd": &fstest.MapFile{Data: []byte(`<xs:schema ` + ns + ` targetNamespace="urn:t" vc:minVersion="2.0">
  <xs:simpleType name="S">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
</xs:schema>`)},
		}
		data, err := fsys.ReadFile(mainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		_, compileErr := xsd.NewCompiler().Version(xsd.Version11).Label(mainXSD).FS(fsys).Compile(t.Context(), doc)
		require.Error(t, compileErr)
	})

	t.Run("Compile does not mutate the caller's document across versions", func(t *testing.T) {
		t.Parallel()
		// The xs:assert is gated by vc:minVersion="1.1": pruned under 1.0, kept under
		// 1.1. Compiling the SAME parsed *helium.Document first under 1.0 (which would
		// unlink the assert if it mutated the caller's DOM) then under 1.1 must STILL
		// see the assert — the conditional-inclusion pre-pass operates on a clone, so
		// the caller's document is never mutated and Compile is idempotent.
		const schemaXML = `<xs:schema ` + ns + `>
  <xs:element name="temp">
    <xs:complexType>
       <xs:sequence/>
       <xs:attribute name="x" use="required"/>
       <xs:assert test="@x > 300" vc:minVersion="1.1"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		s10, err := xsd.NewCompiler().Version(xsd.Version10).Compile(t.Context(), doc)
		require.NoError(t, err)
		require.NoError(t, validate(t, s10, `<temp x="204"/>`)) // assert pruned under 1.0
		// Re-compile the SAME doc under 1.1: the assert must still be present.
		s11, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		require.NoError(t, err)
		require.Error(t, validate(t, s11, `<temp x="204"/>`)) // assert kept under 1.1 → 204 fails
		require.NoError(t, validate(t, s11, `<temp x="304"/>`))
	})

	t.Run("typeAvailable uses built-in capability, not included-schema declarations", func(t *testing.T) {
		t.Parallel()
		// The INCLUDING schema (targetNamespace = the XSD namespace) declares a user
		// type literally named {XSD}error, which shares the compiler's type registry
		// with the chameleon include. The include gates element "gated" on
		// vc:typeAvailable="xs:error". Under Version10 xs:error is NOT a built-in, so
		// "gated" must be PRUNED — the leaked user {XSD}error must not make it
		// "available" (capability detection, not "is it declared somewhere").
		const xsdNS = "http://www.w3.org/2001/XMLSchema"
		const mainXSD = "leak_main.xsd"
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema ` + ns + ` targetNamespace="` + xsdNS + `">
  <xs:simpleType name="error"><xs:restriction base="xs:string"/></xs:simpleType>
  <xs:include schemaLocation="leak_inc.xsd"/>
</xs:schema>`)},
			"leak_inc.xsd": &fstest.MapFile{Data: []byte(`<xs:schema ` + ns + `>
  <xs:element name="gated" type="xs:string" vc:typeAvailable="xs:error"/>
</xs:schema>`)},
		}
		data, err := fsys.ReadFile(mainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		s10, err := xsd.NewCompiler().Version(xsd.Version10).Label(mainXSD).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, err)
		// "gated" is {XSD}gated (the chameleon include adopts the XSD-ns target). It
		// must be pruned under 1.0, so an instance of it is invalid (undeclared).
		require.Error(t, validate(t, s10, `<gated xmlns="`+xsdNS+`">x</gated>`))
	})

	t.Run("typeAvailable=xs:error is version-keyed (pruned 1.0, kept 1.1)", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema ` + ns + ` targetNamespace="urn:x" elementFormDefault="qualified">
  <xs:element name="gated" type="xs:string" vc:typeAvailable="xs:error"/>
</xs:schema>`
		s10, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version10), schema)
		require.NoError(t, err)
		require.Error(t, validate(t, s10, `<gated xmlns="urn:x">x</gated>`)) // pruned under 1.0
		s11, err := compileVC(t, xsd.NewCompiler().Version(xsd.Version11), schema)
		require.NoError(t, err)
		require.NoError(t, validate(t, s11, `<gated xmlns="urn:x">x</gated>`)) // kept under 1.1
	})

	t.Run("clone path preserves source line in diagnostics", func(t *testing.T) {
		t.Parallel()
		// A no-op vc:minVersion="1.0" forces the caller-document clone path
		// (documentHasVCDirective is true). A duplicate global element must still
		// report its REAL source line — the deep copier preserves Node.Line(), so the
		// diagnostic is "(string):3", NOT the "(string):0" line-loss regression.
		const schemaVC = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning" vc:minVersion="1.0">
  <xs:element name="dup" type="xs:string"/>
  <xs:element name="dup" type="xs:string"/>
</xs:schema>`
		const schemaNoVC = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="dup" type="xs:string"/>
  <xs:element name="dup" type="xs:string"/>
</xs:schema>`
		errText := func(s string) string {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
			require.NoError(t, err)
			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			_, _ = xsd.NewCompiler().ErrorHandler(collector).Compile(t.Context(), doc)
			require.NoError(t, collector.Close())
			var b strings.Builder
			for _, e := range collector.Errors() {
				b.WriteString(e.Error())
				b.WriteByte('\n')
			}
			return b.String()
		}
		vc := errText(schemaVC)
		require.Contains(t, vc, "(string):3")    // real line preserved on the clone
		require.NotContains(t, vc, "(string):0") // not the line-0 regression
		// The clone path's line matches the no-clone baseline exactly.
		require.Contains(t, errText(schemaNoVC), "(string):3")
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
