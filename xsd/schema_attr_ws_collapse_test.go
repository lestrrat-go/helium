package xsd_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// xs:NCName and xs:QName both have their whiteSpace facet fixed to "collapse",
// so every NCName-valued (@name) and QName-valued (@type/@ref/@base/@itemType/
// @memberTypes) schema attribute is whitespace-collapsed before it is stored,
// validated, and resolved. A padded-but-valid value must compile; an internal-
// whitespace value (still not a valid NCName/QName after collapsing) must be
// rejected. Version-independent: enforced under both XSD 1.0 and 1.1.
func TestSchemaAttrWhitespaceCollapse(t *testing.T) {
	t.Parallel()

	// Shared expected-fragment for the named-component (@name) NCName rejections.
	const wantNCName = "is not a valid 'xs:NCName'"

	// Finding 1: a collapsed @name is what is REGISTERED — a ref to the trimmed
	// name resolves against the registered {tns}child declaration. If the trailing
	// space were retained the ref="child" would dangle and compilation would fail.
	t.Run("collapsed-name-is-registered/global-ref-resolves", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="child"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="child " type="xs:string"/>
</xs:schema>`
		for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
			_, errs, cerr := compileWith(t, v, schemaXML)
			require.NoError(t, cerr, "version=%v must register the collapsed name so ref resolves: %s", v, errs)
		}
	})

	// The collapsed @name is what an instance is matched against, too: a global
	// element declared with a trailing-space @name validates an instance bearing
	// the trimmed name.
	t.Run("collapsed-name-is-registered/instance-validates", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root " type="xs:string"/>
</xs:schema>`
		errs, err := validateInstance(t, schemaXML, `<root>hello</root>`)
		require.NoError(t, err, "instance must validate against the collapsed declaration name: %s", errs)
	})

	// Findings 2 & 3: a QName-valued attribute with surrounding whitespace collapses
	// to a valid QName and resolves — at every QName-valued read site.
	validQName := []struct {
		name   string
		schema string
	}{
		{
			"element-type",
			`<xs:element name="e" type="  xs:string "/>`,
		},
		{
			"attribute-type",
			`<xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a" type=" xs:string "/>
    </xs:complexType>
  </xs:element>`,
		},
		{
			"attribute-ref",
			`<xs:attribute name="ga" type="xs:string"/>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute ref=" ga "/>
    </xs:complexType>
  </xs:element>`,
		},
		{
			"restriction-base",
			`<xs:simpleType name="st">
    <xs:restriction base="   xs:string ">
      <xs:maxLength value="3"/>
    </xs:restriction>
  </xs:simpleType>`,
		},
		{
			"list-itemType",
			`<xs:simpleType name="st">
    <xs:list itemType=" xs:int "/>
  </xs:simpleType>`,
		},
		{
			"union-memberTypes",
			`<xs:simpleType name="u">
    <xs:union memberTypes=" xs:int   xs:string "/>
  </xs:simpleType>`,
		},
		// A padded @name on each NAMED component is REGISTERED collapsed — proven by a
		// reference to the trimmed name resolving. If the trailing space survived, the
		// ref would dangle and compilation would fail.
		{
			"simpleType-name-registered",
			`<xs:simpleType name="st ">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
  <xs:element name="e" type="st"/>`,
		},
		{
			"complexType-name-registered",
			`<xs:complexType name="ct ">
    <xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:element name="e" type="ct"/>`,
		},
		{
			"group-name-registered",
			`<xs:group name="g ">
    <xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence>
  </xs:group>
  <xs:element name="e">
    <xs:complexType><xs:group ref="g"/></xs:complexType>
  </xs:element>`,
		},
		{
			"attributeGroup-name-registered",
			`<xs:attributeGroup name="ag ">
    <xs:attribute name="a" type="xs:string"/>
  </xs:attributeGroup>
  <xs:element name="e">
    <xs:complexType><xs:attributeGroup ref="ag"/></xs:complexType>
  </xs:element>`,
		},
		{
			"global-attribute-name-registered",
			`<xs:attribute name="ga " type="xs:string"/>
  <xs:element name="e">
    <xs:complexType><xs:attribute ref="ga"/></xs:complexType>
  </xs:element>`,
		},
		{
			"substitutionGroup-padded",
			`<xs:element name="head" type="xs:string"/>
  <xs:element name="member" type="xs:string" substitutionGroup=" head "/>`,
		},
	}
	for _, tc := range validQName {
		t.Run("padded-qname-resolves/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  %s
</xs:schema>`, tc.schema)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				_, errs, cerr := compileWith(t, v, schemaXML)
				require.NoError(t, cerr, "version=%v must accept padded QName value: %s", v, errs)
			}
		})
	}

	// An internal-whitespace value stays invalid after collapsing and must be
	// rejected — never routed into component lookup as a bogus local name.
	rejectInternalWS := []struct {
		name   string
		schema string
		want   string
	}{
		{
			"element-name",
			`<xs:element name="a b" type="xs:string"/>`,
			"is not a valid 'NCName'",
		},
		{
			"attribute-name",
			`<xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a b" type="xs:string"/>
    </xs:complexType>
  </xs:element>`,
			"is not a valid 'NCName'",
		},
		{
			"restriction-base",
			`<xs:simpleType name="st">
    <xs:restriction base="a b">
      <xs:maxLength value="3"/>
    </xs:restriction>
  </xs:simpleType>`,
			"'a b' is not a valid QName",
		},
		{
			"attribute-type",
			`<xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a" type="a b"/>
    </xs:complexType>
  </xs:element>`,
			"'a b' is not a valid QName",
		},
		{
			"element-type",
			`<xs:element name="e" type="a b"/>`,
			"'a b' is not a valid QName",
		},
		{
			"simpleType-name",
			`<xs:simpleType name="a b">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>`,
			wantNCName,
		},
		{
			"complexType-name",
			`<xs:complexType name="a b">
    <xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence>
  </xs:complexType>`,
			wantNCName,
		},
		{
			"group-name",
			`<xs:group name="a b">
    <xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence>
  </xs:group>`,
			wantNCName,
		},
		{
			"attributeGroup-name",
			`<xs:attributeGroup name="a b">
    <xs:attribute name="x" type="xs:string"/>
  </xs:attributeGroup>`,
			wantNCName,
		},
		{
			"union-memberTypes",
			`<xs:simpleType name="u">
    <xs:union memberTypes="a:b:c"/>
  </xs:simpleType>`,
			"is not a valid QName",
		},
	}
	for _, tc := range rejectInternalWS {
		t.Run("internal-whitespace-rejected/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  %s
</xs:schema>`, tc.schema)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v must reject internal-whitespace value", v)
				require.Nil(t, schema)
				require.Contains(t, errs, tc.want, "version=%v", v)
			}
		})
	}

	// xs:keyref/@refer is an xs:QName: a padded refer=" k " collapses at the read
	// point and resolves to the key "k" (a schema that compiles clean), while an
	// internal-whitespace refer="a b" stays an invalid QName and is rejected.
	const keyrefSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType><xs:attribute name="id" type="xs:string"/></xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="k">
      <xs:selector xpath="item"/>
      <xs:field xpath="@id"/>
    </xs:key>
    <xs:keyref name="kr" refer="%s">
      <xs:selector xpath="item"/>
      <xs:field xpath="@id"/>
    </xs:keyref>
  </xs:element>
</xs:schema>`
	t.Run("keyref-refer-padded-resolves", func(t *testing.T) {
		t.Parallel()
		schemaXML := fmt.Sprintf(keyrefSchema, " k ")
		for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
			_, errs, cerr := compileWith(t, v, schemaXML)
			require.NoError(t, cerr, "version=%v must accept padded @refer resolving to key: %s", v, errs)
		}
	})
	t.Run("keyref-refer-internal-whitespace-rejected", func(t *testing.T) {
		t.Parallel()
		schemaXML := fmt.Sprintf(keyrefSchema, "a b")
		for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
			schema, errs, cerr := compileWith(t, v, schemaXML)
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v must reject internal-whitespace @refer", v)
			require.Nil(t, schema)
			require.Contains(t, errs, "not a valid QName", "version=%v", v)
		}
	})

	// The invalid-QName dedup keys on the ATTRIBUTE name, so two DIFFERENT
	// QName-valued attributes on the SAME one-line element carrying the SAME invalid
	// value each report — neither is suppressed by the other. Both @type and
	// @substitutionGroup here resolve through resolveQName; before the fix the shared
	// (element, value) key collapsed the two diagnostics into one.
	t.Run("dedup-per-attribute/two-invalid-qnames-one-line", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="e" type=":bad" substitutionGroup=":bad"/></xs:schema>`
		for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
			_, errs, cerr := compileWith(t, v, schemaXML)
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v", v)
			require.GreaterOrEqual(t, strings.Count(errs, "is not a valid QName"), 2,
				"version=%v: each invalid-QName attribute must report; got: %s", v, errs)
		}
	})

	// A lexically-malformed QName value is reported ONCE at its read point and must
	// NOT also produce a spurious follow-on "does not resolve to a(n) …" diagnostic:
	// resolveQName returns a distinguished sentinel so downstream ref-resolution
	// skips the malformed value instead of routing it into a bogus component lookup.
	noFollowOn := []struct {
		name   string
		schema string
	}{
		{"element-type", `<xs:element name="e" type="a b"/>`},
		{"element-ref", `<xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="a b"/></xs:sequence></xs:complexType></xs:element>`},
		{"attribute-ref", `<xs:element name="e"><xs:complexType><xs:attribute ref="a b"/></xs:complexType></xs:element>`},
		{"restriction-base", `<xs:simpleType name="st"><xs:restriction base="a b"><xs:maxLength value="3"/></xs:restriction></xs:simpleType>`},
		{"list-itemType", `<xs:simpleType name="st"><xs:list itemType="a b"/></xs:simpleType>`},
		{"union-memberTypes", `<xs:simpleType name="u"><xs:union memberTypes="a:b:c"/></xs:simpleType>`},
		{"group-ref", `<xs:element name="e"><xs:complexType><xs:group ref="a b"/></xs:complexType></xs:element>`},
		{"attributeGroup-ref", `<xs:element name="e"><xs:complexType><xs:attributeGroup ref="a b"/></xs:complexType></xs:element>`},
	}
	for _, tc := range noFollowOn {
		t.Run("no-follow-on-unresolved/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">%s</xs:schema>`, tc.schema)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				_, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v", v)
				require.Equal(t, 1, strings.Count(errs, "is not a valid QName"),
					"version=%v: exactly one invalid-QName diagnostic; got: %s", v, errs)
				require.NotContains(t, errs, "does not resolve",
					"version=%v: a malformed value must not produce a follow-on unresolved error; got: %s", v, errs)
			}
		})
	}

	// Two SIBLING declarations minified onto ONE physical line carrying the SAME
	// malformed value in the SAME attribute each report: the dedup keys on the
	// element's IDENTITY, not on (source, line, local name) which siblings share.
	t.Run("dedup-per-element/two-siblings-one-line", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="a" type=":bad"/><xs:element name="b" type=":bad"/></xs:schema>`
		for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
			_, errs, cerr := compileWith(t, v, schemaXML)
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v", v)
			require.Equal(t, 2, strings.Count(errs, "is not a valid QName"),
				"version=%v: each sibling's invalid @type must report; got: %s", v, errs)
		}
	})

	// A well-formed but genuinely UNDECLARED ref still gets the normal unresolved
	// diagnostic — the sentinel skip is only for LEXICALLY malformed values, not for
	// a lexically-valid name that happens to resolve to nothing.
	t.Run("undeclared-ref-still-unresolved", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="e" type="missing"/></xs:schema>`
		for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
			_, errs, cerr := compileWith(t, v, schemaXML)
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v", v)
			require.NotContains(t, errs, "is not a valid QName",
				"version=%v: a well-formed name is not a lexical error; got: %s", v, errs)
			require.Contains(t, errs, "does not resolve",
				"version=%v: an undeclared type must still report unresolved; got: %s", v, errs)
		}
	})
}
