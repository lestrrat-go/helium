package xsd_test

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
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

	// A PRESENT-but-empty QName attribute — the empty string OR a whitespace-only
	// value that collapses to empty — is an INVALID (empty) QName, NOT an absent
	// attribute: it dispatches on PRESENCE (hasAttr), routes through resolveQName,
	// yields exactly ONE "is not a valid QName" diagnostic, and produces no follow-on
	// "does not resolve" unresolved-reference error. This closes the empty/
	// whitespace-only cell for every QName-valued read site consistently.
	presentEmptyQName := []struct {
		name   string
		schema string
	}{
		{"element-type-empty", `<xs:element name="e" type=""/>`},
		{"element-type-ws", `<xs:element name="e" type="   "/>`},
		{"attribute-type-empty", `<xs:element name="e"><xs:complexType><xs:attribute name="a" type=""/></xs:complexType></xs:element>`},
		{"attribute-type-ws", `<xs:element name="e"><xs:complexType><xs:attribute name="a" type="  "/></xs:complexType></xs:element>`},
		{"restriction-base-empty", `<xs:simpleType name="st"><xs:restriction base=""/></xs:simpleType>`},
		{"restriction-base-ws", `<xs:simpleType name="st"><xs:restriction base="   "/></xs:simpleType>`},
		{"local-element-ref-empty", `<xs:element name="r"><xs:complexType><xs:sequence><xs:element ref=""/></xs:sequence></xs:complexType></xs:element>`},
		{"local-element-ref-ws", `<xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="  "/></xs:sequence></xs:complexType></xs:element>`},
		{"group-ref-empty", `<xs:element name="e"><xs:complexType><xs:group ref=""/></xs:complexType></xs:element>`},
		{"group-ref-ws", `<xs:element name="e"><xs:complexType><xs:group ref="  "/></xs:complexType></xs:element>`},
		{"attributeGroup-ref-empty", `<xs:element name="e"><xs:complexType><xs:attributeGroup ref=""/></xs:complexType></xs:element>`},
		{"attributeGroup-ref-ws", `<xs:element name="e"><xs:complexType><xs:attributeGroup ref="  "/></xs:complexType></xs:element>`},
		{"attribute-ref-empty", `<xs:element name="e"><xs:complexType><xs:attribute ref=""/></xs:complexType></xs:element>`},
		{"attribute-ref-ws", `<xs:element name="e"><xs:complexType><xs:attribute ref="   "/></xs:complexType></xs:element>`},
		// complexContent restriction/extension @base — routed through resolveQNameRef.
		// The invalidQName sentinel base is excluded from the extension/restriction
		// derivation loops, so no spurious cos-ct-extends/restriction follow-on fires.
		{"complexContent-restriction-base-empty", `<xs:complexType name="ct"><xs:complexContent><xs:restriction base=""><xs:sequence/></xs:restriction></xs:complexContent></xs:complexType>`},
		{"complexContent-extension-base-ws", `<xs:complexType name="ct"><xs:complexContent><xs:extension base="   "><xs:sequence/></xs:extension></xs:complexContent></xs:complexType>`},
		// simpleContent extension/restriction @base — routed through resolveQNameRef.
		{"simpleContent-extension-base-empty", `<xs:complexType name="ct"><xs:simpleContent><xs:extension base=""/></xs:simpleContent></xs:complexType>`},
		{"simpleContent-restriction-base-ws", `<xs:complexType name="ct"><xs:simpleContent><xs:restriction base="   "/></xs:simpleContent></xs:complexType>`},
		// @substitutionGroup — a single QName in 1.0, a QName-LIST in 1.1; both the
		// present-empty and the whitespace-only case (which splitSpace would tokenize to
		// nothing) yield exactly one invalid-QName and install no spurious head.
		{"substitutionGroup-empty", `<xs:element name="head" type="xs:string"/><xs:element name="member" type="xs:string" substitutionGroup=""/>`},
		{"substitutionGroup-ws", `<xs:element name="head" type="xs:string"/><xs:element name="member" type="xs:string" substitutionGroup="   "/>`},
		// @memberTypes — a QName-LIST whose present-empty/whitespace-only value is an
		// invalid list, reported once (and satisfies the union grammar's hasMemberTypes
		// presence check, so it is not also reported as "must have memberTypes/simpleType").
		{"union-memberTypes-empty", `<xs:simpleType name="u"><xs:union memberTypes=""/></xs:simpleType>`},
		{"union-memberTypes-ws", `<xs:simpleType name="u"><xs:union memberTypes="   "/></xs:simpleType>`},
	}
	for _, tc := range presentEmptyQName {
		t.Run("present-empty-qname-invalid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">%s</xs:schema>`, tc.schema)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v: present-empty QName must reject", v)
				require.Nil(t, schema)
				require.Equal(t, 1, strings.Count(errs, "is not a valid QName"),
					"version=%v: exactly one invalid-QName diagnostic; got: %s", v, errs)
				require.NotContains(t, errs, "does not resolve",
					"version=%v: a present-empty QName must not produce a follow-on unresolved error; got: %s", v, errs)
			}
		})
	}

	// A PRESENT-but-empty / whitespace-only list @itemType is reported ONCE with the
	// list-specific "must be a valid QName; it must not be empty" diagnostic (not the
	// generic invalid-QName message): the structural derivation-body check already
	// covers it, so resolveQName is not also invoked — no double diagnostic.
	itemTypeEmpty := []struct {
		name   string
		schema string
	}{
		{"empty", `<xs:simpleType name="st"><xs:list itemType=""/></xs:simpleType>`},
		{"ws", `<xs:simpleType name="st"><xs:list itemType="   "/></xs:simpleType>`},
	}
	for _, tc := range itemTypeEmpty {
		t.Run("present-empty-itemtype/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">%s</xs:schema>`, tc.schema)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v: present-empty itemType must reject", v)
				require.Nil(t, schema)
				require.Contains(t, errs, "must be a valid QName; it must not be empty",
					"version=%v: itemType uses the list-specific diagnostic; got: %s", v, errs)
				require.Equal(t, 0, strings.Count(errs, "is not a valid QName"),
					"version=%v: itemType must NOT also emit the generic invalid-QName diagnostic; got: %s", v, errs)
			}
		})
	}

	// An ABSENT QName attribute keeps its established default — presence-gating must
	// distinguish "attribute exists but empty" (invalid) from "attribute absent"
	// (default). An absent @type falls through to an inline type or the ur-type; an
	// absent restriction @base falls through to an inline <xs:simpleType> base.
	absentKeepsDefault := []struct {
		name   string
		schema string
	}{
		{"absent-element-type-inline", `<xs:element name="e"><xs:complexType><xs:sequence/></xs:complexType></xs:element>`},
		{"absent-element-type-urtype", `<xs:element name="e"/>`},
		{"absent-attribute-type-inline", `<xs:element name="e"><xs:complexType><xs:attribute name="a"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:attribute></xs:complexType></xs:element>`},
		{"absent-restriction-base-inline", `<xs:simpleType name="st"><xs:restriction><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType><xs:maxLength value="3"/></xs:restriction></xs:simpleType>`},
	}
	for _, tc := range absentKeepsDefault {
		t.Run("absent-keeps-default/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">%s</xs:schema>`, tc.schema)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				_, errs, cerr := compileWith(t, v, schemaXML)
				require.NoError(t, cerr, "version=%v: an absent QName attribute must keep its default; got: %s", v, errs)
			}
		})
	}

	// The nested <xs:attributeGroup ref="..."> inside an xs:redefine attributeGroup
	// OVERRIDE is a QName store site too: a PRESENT-but-empty ref="" routes through
	// resolveQNameRef and is reported ONCE as an invalid QName (the invalidQName
	// sentinel it yields never equals the redefined group's name, so it routes to the
	// non-self branch and checkAttrGroupRefsResolve's sentinel guard suppresses any
	// follow-on "does not resolve"), rather than being silently dropped.
	t.Run("redefine-override-attributeGroup-ref-empty", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			"redef_ag_main.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="redef_ag_base.xsd">
    <xs:attributeGroup name="g">
      <xs:attribute name="a" type="xs:string"/>
      <xs:attributeGroup ref=""/>
    </xs:attributeGroup>
  </xs:redefine>
  <xs:complexType name="t"><xs:attributeGroup ref="g"/></xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`)},
			"redef_ag_base.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g"><xs:attribute name="z" type="xs:string"/></xs:attributeGroup>
</xs:schema>`)},
		}
		for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
			data, err := fsys.ReadFile("redef_ag_main.xsd")
			require.NoError(t, err)
			doc, err := helium.NewParser().Parse(t.Context(), data)
			require.NoError(t, err)
			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			schema, cerr := xsd.NewCompiler().Version(v).Label("redef_ag_main.xsd").
				ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
			require.NoError(t, collector.Close())
			errs := compileErrorsString(collector.Errors())
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v: present-empty override ref must reject", v)
			require.Nil(t, schema)
			require.Equal(t, 1, strings.Count(errs, "is not a valid QName"),
				"version=%v: exactly one invalid-QName diagnostic; got: %s", v, errs)
			require.NotContains(t, errs, "does not resolve",
				"version=%v: a present-empty override ref must not produce a follow-on unresolved error; got: %s", v, errs)
		}
	})
}
