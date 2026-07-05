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

	// The local-targetNamespace inspection helpers (localElementUnderNonAnyType-
	// Restriction / localAttributeUnderNonAnyTypeRestriction) route the restriction
	// @base through the QName chokepoint, so a PRESENT-but-empty base="" behaves like
	// the whitespace-only base="   ": both collapse to the invalidQName sentinel, whose
	// invalid-QName diagnostic already fired. A restriction with base="" plus a local
	// element/attribute targetNamespace thus emits EXACTLY the one invalid-QName error
	// and NO spurious "must appear in a restriction of a type other than xs:anyType"
	// secondary diagnostic. XSD 1.1 only (targetNamespace on a local declaration is a
	// 1.1 construct).
	baseEmptyLocalTargetNS := []struct {
		name   string
		schema string
	}{
		{"complexContent-element-empty", `<xs:complexType name="ct"><xs:complexContent><xs:restriction base=""><xs:sequence><xs:element name="c" type="xs:string" targetNamespace="urn:x"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>`},
		{"complexContent-element-ws", `<xs:complexType name="ct"><xs:complexContent><xs:restriction base="   "><xs:sequence><xs:element name="c" type="xs:string" targetNamespace="urn:x"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>`},
		{"simpleContent-attribute-empty", `<xs:complexType name="ct"><xs:simpleContent><xs:restriction base=""><xs:attribute name="a" type="xs:string" targetNamespace="urn:x"/></xs:restriction></xs:simpleContent></xs:complexType>`},
		{"simpleContent-attribute-ws", `<xs:complexType name="ct"><xs:simpleContent><xs:restriction base="   "><xs:attribute name="a" type="xs:string" targetNamespace="urn:x"/></xs:restriction></xs:simpleContent></xs:complexType>`},
	}
	for _, tc := range baseEmptyLocalTargetNS {
		t.Run("base-empty-local-targetns-no-secondary/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">%s</xs:schema>`, tc.schema)
			schema, errs, cerr := compileWith(t, xsd.Version11, schemaXML)
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "present-empty @base must reject")
			require.Nil(t, schema)
			require.Equal(t, 1, strings.Count(errs, "is not a valid QName"),
				"exactly one invalid-QName diagnostic; got: %s", errs)
			require.NotContains(t, errs, "must appear in a restriction of a type other than xs:anyType",
				"a present-empty/whitespace-only @base must not fire the secondary targetNamespace diagnostic; got: %s", errs)
		})
	}

	// The XSD 1.1 @substitutionGroup LIST branch filters invalidQName tokens exactly
	// like the 1.0 scalar branch, so a non-empty MALFORMED token (":bad") installs NO
	// spurious head: it yields exactly one invalid-QName diagnostic and no follow-on,
	// and a valid head listed alongside it is unaffected. 1.1 only (the list form is
	// 1.1; 1.0 takes a single scalar QName).
	substGroupInvalidToken := []struct {
		name   string
		schema string
	}{
		{"single-invalid", `<xs:element name="member" type="xs:string" substitutionGroup=":bad"/>`},
		{"valid-plus-invalid", `<xs:element name="head" type="xs:string"/><xs:element name="member" type="xs:string" substitutionGroup="head :bad"/>`},
	}
	for _, tc := range substGroupInvalidToken {
		t.Run("substitutiongroup-invalid-token-no-head/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">%s</xs:schema>`, tc.schema)
			schema, errs, cerr := compileWith(t, xsd.Version11, schemaXML)
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "malformed substitutionGroup token must reject")
			require.Nil(t, schema)
			require.Equal(t, 1, strings.Count(errs, "is not a valid QName"),
				"exactly one invalid-QName diagnostic for the malformed token; got: %s", errs)
			require.NotContains(t, errs, "does not resolve",
				"an invalid substitutionGroup token installs no head and produces no follow-on; got: %s", errs)
			require.NotContains(t, errs, "not validly substitutable",
				"a filtered sentinel head must not reach the affiliation check; got: %s", errs)
		})
	}
}

// TestCheckElementsRepresentationGateSymmetry closes the check_elements.go
// representation-gate axis: every structural gate that keys on a QName/NCName-valued
// companion attribute (a with-ref prohibition, an inline-type mutual-exclusion, or
// the ref/name dispatch) is symmetric between a PRESENT-but-collapse-empty value —
// the literal "" AND the whitespace-only "   " (xs:QName/xs:NCName both fix
// whiteSpace "collapse") — each of which emits EXACTLY the one invalid-QName /
// invalid-NCName value diagnostic with NO spurious structural/companion/
// mutual-exclusion/prohibition follow-on. A genuinely-present two-valid-attribute
// mutual-exclusion/prohibition still fires. Version-INDEPENDENT (the gates live in
// checkGlobalElement/checkLocalElement/checkAttributeUse, ungated on version).
func TestCheckElementsRepresentationGateSymmetry(t *testing.T) {
	t.Parallel()

	const wantQName = "is not a valid QName"
	const wantNCName = "is not a valid 'NCName'"
	const wantMutEx = "mutually exclusive"

	// A collapse-empty companion (""/whitespace-only) must emit EXACTLY the one
	// value diagnostic (wantValue) and NONE of the structural secondaries (notWant).
	// The %s is the companion value. wantOther is the OTHER value message (asserted
	// absent) so a QName case can't leak an NCName diagnostic and vice versa.
	symmetric := []struct {
		name      string
		schema    string
		wantValue string
		wantOther string
		notWant   []string
	}{
		// gate 119 — global element @ref prohibition.
		{"global-elem-ref", `<xs:element name="e" type="xs:string" ref="%s"/>`,
			wantQName, wantNCName, []string{"attribute 'ref' is not allowed"}},
		// gate 165 — global element @type + inline complexType mutual-exclusion.
		{"global-elem-type-inline", `<xs:element name="e" type="%s"><xs:complexType><xs:sequence/></xs:complexType></xs:element>`,
			wantQName, wantNCName, []string{wantMutEx}},
		// gate 441 — local element @ref dispatch with name+type companions present.
		{"local-elem-ref-dispatch", `<xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="%s" name="x" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`,
			wantQName, wantNCName, []string{wantMutEx, "Only the attributes"}},
		// gate 484 — local element @ref + @name mutual-exclusion (NCName companion).
		{"local-elem-ref-name", `<xs:element name="c" type="xs:string"/><xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="c" name="%s"/></xs:sequence></xs:complexType></xs:element>`,
			wantNCName, wantQName, []string{wantMutEx}},
		// gate 601 — local element @type + inline complexType mutual-exclusion.
		{"local-elem-type-inline", `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="x" type="%s"><xs:complexType><xs:sequence/></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`,
			wantQName, wantNCName, []string{wantMutEx}},
		// gate 752 — attribute @name + @ref prohibition (NCName companion).
		{"attr-ref-name", `<xs:attribute name="ga" type="xs:string"/><xs:element name="e"><xs:complexType><xs:attribute ref="ga" name="%s"/></xs:complexType></xs:element>`,
			wantNCName, wantQName, []string{"attribute 'name' is not allowed"}},
		// gate 758 — attribute @type + @ref prohibition (no store site validates the
		// ref-branch @type, so the gate is the SOLE reporter of the invalid QName).
		{"attr-ref-type", `<xs:attribute name="ga" type="xs:string"/><xs:element name="e"><xs:complexType><xs:attribute ref="ga" type="%s"/></xs:complexType></xs:element>`,
			wantQName, wantNCName, []string{"attribute 'type' is not allowed"}},
		// gate 890 — attribute @type + inline simpleType mutual-exclusion.
		{"attr-type-inline", `<xs:element name="e"><xs:complexType><xs:attribute name="a" type="%s"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:attribute></xs:complexType></xs:element>`,
			wantQName, wantNCName, []string{wantMutEx}},
	}
	for _, tc := range symmetric {
		for _, val := range []struct{ label, v string }{{"literal-empty", ""}, {"whitespace-only", "   "}} {
			t.Run("collapse-empty/"+tc.name+"/"+val.label, func(t *testing.T) {
				t.Parallel()
				schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">%s</xs:schema>`,
					fmt.Sprintf(tc.schema, val.v))
				for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
					schema, errs, cerr := compileWith(t, v, schemaXML)
					require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v: %s", v, errs)
					require.Nil(t, schema)
					require.Equal(t, 1, strings.Count(errs, tc.wantValue),
						"version=%v: exactly one %q diagnostic; got: %s", v, tc.wantValue, errs)
					require.Equal(t, 0, strings.Count(errs, tc.wantOther),
						"version=%v: must not emit the other value diagnostic %q; got: %s", v, tc.wantOther, errs)
					for _, nw := range tc.notWant {
						require.NotContains(t, errs, nw,
							"version=%v: no spurious structural secondary %q; got: %s", v, nw, errs)
					}
				}
			})
		}
	}

	// A genuinely-present two-valid-attribute companion still fires its structural
	// gate — the symmetry fix suppresses only the collapse-empty cell, never a real
	// mutual-exclusion/prohibition between two present valid attributes.
	twoValid := []struct {
		name   string
		schema string
		want   string
	}{
		{"global-elem-ref", `<xs:element name="e" type="xs:string" ref="foo"/>`,
			"attribute 'ref' is not allowed"},
		{"global-elem-type-inline", `<xs:element name="e" type="xs:string"><xs:complexType><xs:sequence/></xs:complexType></xs:element>`,
			wantMutEx},
		{"local-elem-ref-name", `<xs:element name="c" type="xs:string"/><xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="c" name="x"/></xs:sequence></xs:complexType></xs:element>`,
			"The attributes 'ref' and 'name' are mutually exclusive"},
		{"local-elem-type-inline", `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="x" type="xs:string"><xs:complexType><xs:sequence/></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`,
			wantMutEx},
		{"attr-ref-name", `<xs:attribute name="ga" type="xs:string"/><xs:element name="e"><xs:complexType><xs:attribute ref="ga" name="x"/></xs:complexType></xs:element>`,
			"The attribute 'name' is not allowed"},
		{"attr-ref-type", `<xs:attribute name="ga" type="xs:string"/><xs:element name="e"><xs:complexType><xs:attribute ref="ga" type="xs:string"/></xs:complexType></xs:element>`,
			"The attribute 'type' is not allowed"},
		{"attr-type-inline", `<xs:element name="e"><xs:complexType><xs:attribute name="a" type="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:attribute></xs:complexType></xs:element>`,
			wantMutEx},
	}
	for _, tc := range twoValid {
		t.Run("two-valid-still-fires/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">%s</xs:schema>`, tc.schema)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				_, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v", v)
				require.Contains(t, errs, tc.want,
					"version=%v: a genuine two-valid-attribute rule must still fire; got: %s", v, errs)
				require.NotContains(t, errs, wantQName,
					"version=%v: a valid companion is not an invalid QName; got: %s", v, errs)
				require.NotContains(t, errs, wantNCName,
					"version=%v: a valid companion is not an invalid NCName; got: %s", v, errs)
			}
		})
	}
}

// TestReadParticlesTypesNameProhibitionGateSymmetry closes the @name-prohibition
// axis in read_particles.go and read_types.go: every gate that prohibits @name on a
// reference form (attributeGroup ref / model group ref), an inline model group, or a
// LOCAL type definition (complexType / simpleType) is symmetric between a
// PRESENT-but-collapse-empty @name — the literal "" AND the whitespace-only "   "
// (xs:NCName fixes whiteSpace "collapse") — each of which emits EXACTLY the one
// invalid-NCName value diagnostic and NO spurious "name not allowed" / "must not have
// a name" structural secondary. A genuinely-present VALID @name still fires the
// prohibition. Version-INDEPENDENT (all five gates are ungated on version).
func TestReadParticlesTypesNameProhibitionGateSymmetry(t *testing.T) {
	t.Parallel()

	const wantNCName = "is not a valid 'NCName'"

	// Each schema has one %s for the @name value under test; notWant is the
	// structural prohibition wording that must NOT appear for a collapse-empty @name.
	gates := []struct {
		name    string
		schema  string
		notWant string
	}{
		// read_particles.go:359 — @name on an attributeGroup REFERENCE.
		{"attrgroup-ref", `<xs:attributeGroup name="ag"><xs:attribute name="a" type="xs:string"/></xs:attributeGroup>` +
			`<xs:complexType name="t"><xs:attributeGroup ref="ag" name="%s"/></xs:complexType>`,
			"not allowed on an attributeGroup reference"},
		// read_particles.go:400 — @name on a model group REFERENCE.
		{"group-ref", `<xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>` +
			`<xs:complexType name="t"><xs:sequence><xs:group ref="g" name="%s"/></xs:sequence></xs:complexType>`,
			"not allowed on a model group reference"},
		// read_particles.go:483 — @name on an inline model group.
		{"inline-model-group", `<xs:complexType name="t"><xs:sequence name="%s"><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`,
			"not allowed on an inline model group"},
		// read_types.go:136 — @name on a LOCAL complexType.
		{"local-complextype", `<xs:element name="e"><xs:complexType name="%s"><xs:sequence/></xs:complexType></xs:element>`,
			"must not have a 'name' attribute"},
		// read_types.go:1166 — @name on a LOCAL simpleType.
		{"local-simpletype", `<xs:element name="e"><xs:simpleType name="%s"><xs:restriction base="xs:string"/></xs:simpleType></xs:element>`,
			"must not have a 'name' attribute"},
	}

	for _, tc := range gates {
		// present-empty ≡ whitespace-only: each yields exactly one invalid-NCName
		// value diagnostic and no structural prohibition secondary.
		for _, val := range []struct{ label, v string }{{"literal-empty", ""}, {"whitespace-only", "   "}} {
			t.Run("collapse-empty/"+tc.name+"/"+val.label, func(t *testing.T) {
				t.Parallel()
				schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">%s</xs:schema>`,
					fmt.Sprintf(tc.schema, val.v))
				for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
					schema, errs, cerr := compileWith(t, v, schemaXML)
					require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v: %s", v, errs)
					require.Nil(t, schema)
					require.Equal(t, 1, strings.Count(errs, wantNCName),
						"version=%v: exactly one invalid-NCName diagnostic; got: %s", v, errs)
					require.NotContains(t, errs, tc.notWant,
						"version=%v: no spurious %q prohibition secondary; got: %s", v, tc.notWant, errs)
				}
			})
		}

		// A genuinely-present VALID @name still fires the structural prohibition and
		// leaks no invalid-NCName diagnostic.
		t.Run("valid-name-still-fires/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">%s</xs:schema>`,
				fmt.Sprintf(tc.schema, "bogusName"))
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				_, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v", v)
				require.Contains(t, errs, tc.notWant,
					"version=%v: a valid @name must still fire the prohibition; got: %s", v, errs)
				require.NotContains(t, errs, wantNCName,
					"version=%v: a valid @name is not an invalid NCName; got: %s", v, errs)
			}
		})
	}
}

// TestDerivationBodyAndAlternativeQNameGateSymmetry closes the remaining
// QName mutual-exclusion gates outside check_elements.go: the simpleType
// derivation-body base/itemType-vs-inline-simpleType mutual exclusion
// (read_types.go checkSimpleTypeDerivationBody) and the xs:alternative
// type-vs-inline-type mutual exclusion (alternative.go parseTypeAlternative). A
// PRESENT-but-collapse-empty @base/@itemType/@type — the literal "" AND the
// whitespace-only "   " (all xs:QName, whiteSpace fixed "collapse") — emits EXACTLY
// the one value diagnostic and NO "must not have both …" structural secondary, so
// present-empty ≡ whitespace-only. A genuinely-present VALID value alongside the
// inline type still fires the mutual exclusion.
func TestDerivationBodyAndAlternativeQNameGateSymmetry(t *testing.T) {
	t.Parallel()

	const wantQName = "is not a valid QName"

	// Each gate: the collapse-empty %s value, its ONE value diagnostic (wantValue),
	// the mutual-exclusion secondary that must be suppressed (mutEx), a genuinely-
	// present valid value (validVal) that must still fire mutEx, and the versions to
	// exercise (xs:alternative is 1.1-only).
	gates := []struct {
		name      string
		schema    string
		wantValue string
		mutEx     string
		validVal  string
		versions  []xsd.Version
	}{
		// read_types.go — restriction @base vs inline <xs:simpleType>. The empty @base
		// value diagnostic comes from resolveQName at the store site.
		{
			"simpletype-restriction-base",
			`<xs:simpleType name="st"><xs:restriction base="%s"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:restriction></xs:simpleType>`,
			wantQName,
			"A restriction must not have both a 'base' attribute and a simpleType child",
			"xs:string",
			[]xsd.Version{xsd.Version10, xsd.Version11},
		},
		// read_types.go — list @itemType vs inline <xs:simpleType>. The empty @itemType
		// value diagnostic is the list-specific "must be a valid QName; it must not be empty".
		{
			"simpletype-list-itemtype",
			`<xs:simpleType name="lt"><xs:list itemType="%s"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:list></xs:simpleType>`,
			"it must not be empty",
			"A list must not have both an 'itemType' attribute and a simpleType child",
			"xs:string",
			[]xsd.Version{xsd.Version10, xsd.Version11},
		},
		// alternative.go — xs:alternative @type vs inline type definition (1.1-only).
		{
			"alternative-type-inline",
			`<xs:element name="e" type="xs:string"><xs:alternative test="true()" type="%s"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:alternative></xs:element>`,
			wantQName,
			"must not have both a 'type' attribute and an inline type definition",
			"xs:string",
			[]xsd.Version{xsd.Version11},
		},
	}

	for _, tc := range gates {
		for _, val := range []struct{ label, v string }{{"literal-empty", ""}, {"whitespace-only", "   "}} {
			t.Run("collapse-empty/"+tc.name+"/"+val.label, func(t *testing.T) {
				t.Parallel()
				schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">%s</xs:schema>`,
					fmt.Sprintf(tc.schema, val.v))
				for _, v := range tc.versions {
					schema, errs, cerr := compileWith(t, v, schemaXML)
					require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v: %s", v, errs)
					require.Nil(t, schema)
					require.Equal(t, 1, strings.Count(errs, tc.wantValue),
						"version=%v: exactly one %q value diagnostic; got: %s", v, tc.wantValue, errs)
					require.NotContains(t, errs, tc.mutEx,
						"version=%v: no spurious mutual-exclusion secondary %q; got: %s", v, tc.mutEx, errs)
				}
			})
		}

		// A genuinely-present VALID value alongside the inline type still fires the
		// mutual exclusion — the fix suppresses only the collapse-empty cell.
		t.Run("two-valid-still-fires/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">%s</xs:schema>`,
				fmt.Sprintf(tc.schema, tc.validVal))
			for _, v := range tc.versions {
				_, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v", v)
				require.Contains(t, errs, tc.mutEx,
					"version=%v: a valid value alongside an inline type must still fire the mutual exclusion; got: %s", v, errs)
				require.NotContains(t, errs, wantQName,
					"version=%v: a valid value is not an invalid QName; got: %s", v, errs)
			}
		})
	}
}

// TestPresentEmptyQNameNCNameSiteSymmetry closes the remaining reference /
// required-@name / targetNamespace-secondary / keyref-@refer gates that key on a
// QName- or NCName-valued schema attribute. For every gate a PRESENT-but-collapse-
// empty value — the literal "" AND the whitespace-only "   " (xs:QName / xs:NCName
// both fix whiteSpace "collapse") — must produce the IDENTICAL diagnostics: exactly
// the one invalid-value diagnostic, no spurious structural / prohibition / "missing"
// secondary. A genuinely-ABSENT attribute keeps its own missing/required diagnostic
// (the absent-regression table below).
func TestPresentEmptyQNameNCNameSiteSymmetry(t *testing.T) {
	t.Parallel()

	const (
		wantQName  = "is not a valid QName"
		wantNCName = "is not a valid 'NCName'"
	)

	// Each case: a %s-templated schema whose empty and whitespace-only fills must
	// yield byte-identical diagnostics; wantPresent is the one value diagnostic that
	// MUST appear; wantAbsent is the spurious secondary that must NOT.
	cases := []struct {
		name        string
		schema      string
		wantPresent string
		wantAbsent  string
		versions    []xsd.Version
	}{
		// Finding 1 — model-group reference: a present-empty @ref alongside a valid
		// @name reports the one invalid-QName, NOT the "name not allowed" secondary
		// (uniform with the attributeGroup reference below).
		{
			"model-group-ref-empty-with-name",
			`<xs:complexType name="ct"><xs:sequence><xs:group ref="%s" name="g"/></xs:sequence></xs:complexType>`,
			wantQName,
			"not allowed on a model group reference",
			[]xsd.Version{xsd.Version10, xsd.Version11},
		},
		// Finding 1 — attributeGroup reference: same, the mirror of the model-group
		// case (previously reported BOTH; now the single invalid-QName).
		{
			"attributegroup-ref-empty-with-name",
			`<xs:complexType name="ct"><xs:attributeGroup ref="%s" name="g"/></xs:complexType>`,
			wantQName,
			"name' is not allowed on an attributeGroup reference",
			[]xsd.Version{xsd.Version10, xsd.Version11},
		},
		// Finding 2 — global element @name: present-empty is an invalid (empty)
		// NCName, NOT "required but missing" (which is reserved for a genuinely-absent
		// @name, exercised in the absent table).
		{
			"global-element-name-empty",
			`<xs:element name="%s" type="xs:string"/>`,
			wantNCName,
			"is required but missing",
			[]xsd.Version{xsd.Version10, xsd.Version11},
		},
		// Finding 2 — local element @name: present-empty enters the named branch and
		// is an invalid (empty) NCName for both forms (no divergence with the raw
		// whitespace-only case).
		{
			"local-element-name-empty",
			`<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="%s" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`,
			wantNCName,
			"",
			[]xsd.Version{xsd.Version10, xsd.Version11},
		},
		// Finding 3 — local attribute @name + @targetNamespace: present-empty @name is
		// the one invalid-NCName; the "requires a ... name" targetNamespace secondary
		// is suppressed (previously fired only for the literal "").
		{
			"local-attribute-name-empty-targetns",
			`<xs:complexType name="b"><xs:simpleContent><xs:extension base="xs:string"><xs:attribute name="keep" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType>` +
				`<xs:complexType name="ct"><xs:simpleContent><xs:restriction base="b"><xs:attribute name="%s" type="xs:string" targetNamespace="urn:x"/></xs:restriction></xs:simpleContent></xs:complexType>`,
			wantNCName,
			"requires a local attribute declaration with a 'name'",
			[]xsd.Version{xsd.Version11},
		},
		// Finding 3 (sibling) — local element @name + @targetNamespace: the same
		// suppression for the local ELEMENT targetNamespace secondary.
		{
			"local-element-name-empty-targetns",
			`<xs:complexType name="b"><xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence></xs:complexType>` +
				`<xs:complexType name="ct"><xs:complexContent><xs:restriction base="b"><xs:sequence><xs:element name="%s" type="xs:string" targetNamespace="urn:x"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>`,
			wantNCName,
			"requires a local element declaration with a 'name'",
			[]xsd.Version{xsd.Version11},
		},
		// Finding 4 — keyref @refer: present-empty is the one invalid-QName, NOT the
		// "no refer attribute" (absent) diagnostic.
		{
			"keyref-refer-empty",
			`<xs:element name="root"><xs:complexType><xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence></xs:complexType>` +
				`<xs:key name="k"><xs:selector xpath="c"/><xs:field xpath="."/></xs:key>` +
				`<xs:keyref name="kr" refer="%s"><xs:selector xpath="c"/><xs:field xpath="."/></xs:keyref></xs:element>`,
			wantQName,
			"has no 'refer' attribute naming a key or unique",
			[]xsd.Version{xsd.Version10, xsd.Version11},
		},
	}

	for _, tc := range cases {
		t.Run("symmetry/"+tc.name, func(t *testing.T) {
			t.Parallel()
			for _, v := range tc.versions {
				emptyXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">%s</xs:schema>`, fmt.Sprintf(tc.schema, ""))
				wsXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">%s</xs:schema>`, fmt.Sprintf(tc.schema, "   "))

				schemaEmpty, errsEmpty, cerrEmpty := compileWith(t, v, emptyXML)
				_, errsWs, cerrWs := compileWith(t, v, wsXML)

				require.ErrorIs(t, cerrEmpty, xsd.ErrCompilationFailed, "version=%v: present-empty must reject; got: %s", v, errsEmpty)
				require.ErrorIs(t, cerrWs, xsd.ErrCompilationFailed, "version=%v: whitespace-only must reject; got: %s", v, errsWs)
				require.Nil(t, schemaEmpty)

				// The core symmetry claim: "" and "   " collapse to the same value, so
				// the emitted diagnostics are byte-identical.
				require.Equal(t, errsEmpty, errsWs,
					"version=%v: present-empty and whitespace-only must emit identical diagnostics", v)

				require.Contains(t, errsEmpty, tc.wantPresent,
					"version=%v: the one value diagnostic %q must appear; got: %s", v, tc.wantPresent, errsEmpty)
				if tc.wantAbsent != "" {
					require.NotContains(t, errsEmpty, tc.wantAbsent,
						"version=%v: no spurious secondary %q; got: %s", v, tc.wantAbsent, errsEmpty)
				}
			}
		})
	}

	// A genuinely-ABSENT attribute keeps its own missing/required diagnostic and does
	// NOT emit the present-empty invalid-value diagnostic.
	absent := []struct {
		name        string
		schema      string
		wantMissing string
		wantAbsent  string
		versions    []xsd.Version
	}{
		{
			"global-element-name-absent",
			`<xs:element type="xs:string"/>`,
			"is required but missing",
			wantNCName,
			[]xsd.Version{xsd.Version10, xsd.Version11},
		},
		{
			"keyref-refer-absent",
			`<xs:element name="root"><xs:complexType><xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence></xs:complexType>` +
				`<xs:key name="k"><xs:selector xpath="c"/><xs:field xpath="."/></xs:key>` +
				`<xs:keyref name="kr"><xs:selector xpath="c"/><xs:field xpath="."/></xs:keyref></xs:element>`,
			"has no 'refer' attribute naming a key or unique",
			wantQName,
			[]xsd.Version{xsd.Version10, xsd.Version11},
		},
		{
			"local-attribute-targetns-name-absent",
			`<xs:complexType name="b"><xs:simpleContent><xs:extension base="xs:string"><xs:attribute name="keep" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType>` +
				`<xs:complexType name="ct"><xs:simpleContent><xs:restriction base="b"><xs:attribute type="xs:string" targetNamespace="urn:x"/></xs:restriction></xs:simpleContent></xs:complexType>`,
			"requires a local attribute declaration with a 'name'",
			wantNCName,
			[]xsd.Version{xsd.Version11},
		},
	}
	for _, tc := range absent {
		t.Run("absent/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">%s</xs:schema>`, tc.schema)
			for _, v := range tc.versions {
				_, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v: absent-attribute case must reject; got: %s", v, errs)
				require.Contains(t, errs, tc.wantMissing,
					"version=%v: a genuinely-absent attribute keeps its missing/required diagnostic; got: %s", v, errs)
				require.NotContains(t, errs, tc.wantAbsent,
					"version=%v: an absent attribute is not a present-empty invalid value; got: %s", v, errs)
			}
		})
	}
}

// compileRedefineVersioned compiles a two-document redefine schema (base + main) from an
// in-memory FS, returning the compiled schema, the joined error string, and the
// compile error. The main document is the entry point.
func compileRedefineVersioned(t *testing.T, v xsd.Version, baseXSD, mainXSD string) (*xsd.Schema, string, error) {
	t.Helper()
	fsys := fstest.MapFS{
		"redef_base.xsd": &fstest.MapFile{Data: []byte(baseXSD)},
		"redef_main.xsd": &fstest.MapFile{Data: []byte(mainXSD)},
	}
	doc, err := helium.NewParser().Parse(t.Context(), []byte(mainXSD))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	schema, cerr := xsd.NewCompiler().Version(v).Label("redef_main.xsd").
		ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
	require.NoError(t, collector.Close())
	return schema, compileErrorsString(collector.Errors()), cerr
}

// TestComponentChildNameKeyedDispatchValidity covers EVERY name-keyed
// component-dispatch loop — xs:redefine (processRedefineOverrides) AND xs:override
// (collectOverrideChildren) — for all four named-component kinds
// (complexType/simpleType/group/attributeGroup). A direct component child whose
// @name is PRESENT but not a valid xs:NCName — the literal "" AND the whitespace-only
// "   " (both collapse to empty) AND a malformed NON-empty name like "a b" — must be
// REJECTED as an invalid NCName, not silently dropped (matching no target) and left to
// compile. Present-empty and whitespace-only produce byte-identical diagnostics; a
// well-formed child still compiles. xs:redefine runs in both XSD 1.0 and 1.1;
// xs:override is 1.1-only.
func TestComponentChildNameKeyedDispatchValidity(t *testing.T) {
	t.Parallel()

	const (
		wantNCName   = "is not a valid 'xs:NCName'"
		wrapRedefine = "redefine"
		wrapOverride = "override"
		wsName       = "   " // whitespace-only @name (collapses to empty)
	)

	cases := []struct {
		name       string
		wrapper    string // wrapRedefine or wrapOverride
		baseBody   string
		childTmpl  string // %s = the dispatch child's @name value
		extraBody  string // consumer/root in the main document
		wellFormed string // a valid @name that matches a base target
		versions   []xsd.Version
	}{
		// xs:redefine — the child must SELF-derive from the redefined component.
		{
			"redefine-complexType",
			wrapRedefine,
			`<xs:complexType name="ct"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`,
			`<xs:complexType name="%s"><xs:complexContent><xs:restriction base="ct"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>`,
			`<xs:element name="root" type="ct"/>`,
			"ct",
			[]xsd.Version{xsd.Version10, xsd.Version11},
		},
		{
			"redefine-simpleType",
			wrapRedefine,
			`<xs:simpleType name="st"><xs:restriction base="xs:string"/></xs:simpleType>`,
			`<xs:simpleType name="%s"><xs:restriction base="st"><xs:maxLength value="5"/></xs:restriction></xs:simpleType>`,
			`<xs:element name="root" type="st"/>`,
			"st",
			[]xsd.Version{xsd.Version10, xsd.Version11},
		},
		{
			"redefine-group",
			wrapRedefine,
			`<xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`,
			`<xs:group name="%s"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`,
			`<xs:complexType name="rt"><xs:group ref="g"/></xs:complexType><xs:element name="root" type="rt"/>`,
			"g",
			[]xsd.Version{xsd.Version10, xsd.Version11},
		},
		{
			"redefine-attributeGroup",
			wrapRedefine,
			`<xs:attributeGroup name="ag"><xs:attribute name="x" type="xs:string"/></xs:attributeGroup>`,
			`<xs:attributeGroup name="%s"><xs:attribute name="x" type="xs:string"/></xs:attributeGroup>`,
			`<xs:complexType name="rt"><xs:attributeGroup ref="ag"/></xs:complexType><xs:element name="root" type="rt"/>`,
			"ag",
			[]xsd.Version{xsd.Version10, xsd.Version11},
		},
		// xs:override — the child WHOLESALE-replaces the same-named base component.
		{
			"override-complexType",
			wrapOverride,
			`<xs:complexType name="ct"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`,
			`<xs:complexType name="%s"><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:complexType>`,
			`<xs:element name="root" type="ct"/>`,
			"ct",
			[]xsd.Version{xsd.Version11},
		},
		{
			"override-simpleType",
			wrapOverride,
			`<xs:simpleType name="st"><xs:restriction base="xs:string"/></xs:simpleType>`,
			`<xs:simpleType name="%s"><xs:restriction base="xs:string"><xs:maxLength value="3"/></xs:restriction></xs:simpleType>`,
			`<xs:element name="root" type="st"/>`,
			"st",
			[]xsd.Version{xsd.Version11},
		},
		{
			"override-group",
			wrapOverride,
			`<xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`,
			`<xs:group name="%s"><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:group>`,
			`<xs:complexType name="rt"><xs:group ref="g"/></xs:complexType><xs:element name="root" type="rt"/>`,
			"g",
			[]xsd.Version{xsd.Version11},
		},
		{
			"override-attributeGroup",
			wrapOverride,
			`<xs:attributeGroup name="ag"><xs:attribute name="x" type="xs:string"/></xs:attributeGroup>`,
			`<xs:attributeGroup name="%s"><xs:attribute name="y" type="xs:string"/></xs:attributeGroup>`,
			`<xs:complexType name="rt"><xs:attributeGroup ref="ag"/></xs:complexType><xs:element name="root" type="rt"/>`,
			"ag",
			[]xsd.Version{xsd.Version11},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			baseXSD := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">%s</xs:schema>`, tc.baseBody)
			buildMain := func(nameVal string) string {
				child := fmt.Sprintf(tc.childTmpl, nameVal)
				return fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:%s schemaLocation="redef_base.xsd">%s</xs:%s>%s</xs:schema>`,
					tc.wrapper, child, tc.wrapper, tc.extraBody)
			}
			for _, v := range tc.versions {
				// Every present-but-invalid @name is rejected as an invalid NCName.
				for _, bad := range []string{"", wsName, "a b"} {
					schema, errs, cerr := compileRedefineVersioned(t, v, baseXSD, buildMain(bad))
					require.ErrorIs(t, cerr, xsd.ErrCompilationFailed,
						"version=%v: a %s child with name=%q must be REJECTED, not silently compiled; got: %s", v, tc.wrapper, bad, errs)
					require.Nil(t, schema)
					require.Contains(t, errs, wantNCName,
						"version=%v: name=%q must emit an invalid-NCName diagnostic; got: %s", v, bad, errs)
				}

				// Present-empty ("") and whitespace-only ("   ") emit byte-identical
				// diagnostics (both collapse to "").
				_, errsEmpty, _ := compileRedefineVersioned(t, v, baseXSD, buildMain(""))
				_, errsWs, _ := compileRedefineVersioned(t, v, baseXSD, buildMain(wsName))
				require.Equal(t, errsEmpty, errsWs,
					"version=%v: present-empty and whitespace-only %s-child @name must emit identical diagnostics", v, tc.wrapper)

				// A well-formed child still compiles.
				schemaValid, errsValid, cerrValid := compileRedefineVersioned(t, v, baseXSD, buildMain(tc.wellFormed))
				require.NoError(t, cerrValid,
					"version=%v: a well-formed %s child must still compile; got: %s", v, tc.wrapper, errsValid)
				require.NotNil(t, schemaValid)
			}
		})
	}
}

// TestOverrideNotationNameSingleDiagnostic pins that an xs:override xs:notation
// child with a malformed @name is reported EXACTLY ONCE — by checkNotations (which
// owns notation @name NCName validity) — and does NOT also go through the
// component-child name gate (validateComponentChildName), which would emit a spurious
// SECOND invalid-name diagnostic. Present-empty ("") and whitespace-only ("   ") are
// byte-identical; a well-formed override notation child still compiles. xs:override
// is 1.1-only.
func TestOverrideNotationNameSingleDiagnostic(t *testing.T) {
	t.Parallel()

	const (
		wantNotation = "A notation declaration must have a 'name' attribute that is a valid NCName."
		spurious     = "is not a valid 'xs:NCName'"
		wsName       = "   " // whitespace-only @name (collapses to empty)
	)

	baseXSD := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:notation name="n" public="pub"/><xs:element name="root" type="xs:string"/></xs:schema>`
	buildMain := func(nameVal string) string {
		return fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:override schemaLocation="redef_base.xsd"><xs:notation name="%s" public="pub2"/></xs:override></xs:schema>`, nameVal)
	}

	for _, bad := range []string{"", wsName, "a b"} {
		schema, errs, cerr := compileRedefineVersioned(t, xsd.Version11, baseXSD, buildMain(bad))
		require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "name=%q: malformed override notation name must reject; got: %s", bad, errs)
		require.Nil(t, schema)
		require.Equal(t, 1, strings.Count(errs, wantNotation),
			"name=%q: exactly ONE checkNotations diagnostic; got: %s", bad, errs)
		require.NotContains(t, errs, spurious,
			"name=%q: no spurious component-child-gate NCName diagnostic; got: %s", bad, errs)
	}

	// Present-empty and whitespace-only emit byte-identical diagnostics.
	_, errsEmpty, _ := compileRedefineVersioned(t, xsd.Version11, baseXSD, buildMain(""))
	_, errsWs, _ := compileRedefineVersioned(t, xsd.Version11, baseXSD, buildMain(wsName))
	require.Equal(t, errsEmpty, errsWs,
		"present-empty and whitespace-only override notation @name must emit identical diagnostics")

	// A well-formed override notation child (matching the base notation) still compiles.
	schemaValid, errsValid, cerrValid := compileRedefineVersioned(t, xsd.Version11, baseXSD, buildMain("n"))
	require.NoError(t, cerrValid, "a well-formed override notation child must still compile; got: %s", errsValid)
	require.NotNil(t, schemaValid)
}
