package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestIDCFieldSkipContent covers the version-aware handling of an
// identity-constraint <xs:field> that selects an ELEMENT node admitted by a
// processContents="skip" wildcard (cvc-identity-constraint.3 / XSD 1.1 §3.11.4).
//
// In XSD 1.1 such a node is UNASSESSED (no type annotation), so the field
// contributes NO value rather than failing the simple-type requirement — the
// node simply drops out of the qualified node-set. In XSD 1.0 there is no such
// relaxation, so the unassessed complex-typed node is a validity error. An
// element field whose type is ACTUALLY assessed as complex is rejected in BOTH
// versions.
func TestIDCFieldSkipContent(t *testing.T) {
	t.Parallel()

	// A globally-declared complex <g> re-admitted under item via a
	// processContents="skip" wildcard, with a field selecting it.
	const skipSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="g">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="x" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:sequence>
              <xs:any processContents="skip" minOccurs="0"/>
            </xs:sequence>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="g"/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	// A field selecting a locally-declared, genuinely-assessed complex element.
	const assessedSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="g">
                <xs:complexType>
                  <xs:sequence>
                    <xs:element name="x" type="xs:string"/>
                  </xs:sequence>
                </xs:complexType>
              </xs:element>
            </xs:sequence>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="g"/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	const instance = `<root><item><g><x>foo</x></g></item></root>`

	compile := func(t *testing.T, src string, v11 bool) xsd.Validator {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		c := xsd.NewCompiler()
		if v11 {
			c = c.Version(xsd.Version11)
		}
		schema, err := c.Compile(t.Context(), doc)
		require.NoError(t, err)
		return xsd.NewValidator(schema)
	}

	validate := func(t *testing.T, v xsd.Validator) (string, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		return errs, err
	}

	t.Run("skip-content field is no value in 1.1", func(t *testing.T) {
		t.Parallel()
		errs, err := validate(t, compile(t, skipSchema, true))
		require.NoError(t, err, "1.1 skip-content field must contribute no value, got: %s", errs)
	})

	t.Run("skip-content field rejected in 1.0", func(t *testing.T) {
		t.Parallel()
		errs, err := validate(t, compile(t, skipSchema, false))
		require.Error(t, err, "1.0 unassessed complex field must be rejected")
		require.Contains(t, errs, "not simple")
	})

	t.Run("assessed complex field rejected in 1.1", func(t *testing.T) {
		t.Parallel()
		errs, err := validate(t, compile(t, assessedSchema, true))
		require.Error(t, err, "assessed complex-typed field must be rejected in 1.1")
		require.Contains(t, errs, "not simple")
	})

	t.Run("assessed complex field rejected in 1.0", func(t *testing.T) {
		t.Parallel()
		errs, err := validate(t, compile(t, assessedSchema, false))
		require.Error(t, err, "assessed complex-typed field must be rejected in 1.0")
		require.Contains(t, errs, "not simple")
	})
}

// TestIDCFieldSkipAttribute covers the version-aware handling of an
// identity-constraint <xs:field> that selects an ATTRIBUTE admitted only by a
// processContents="skip" xs:anyAttribute wildcard (cvc-identity-constraint.3 /
// XSD 1.1 §3.11.4).
//
// Such an attribute is UNASSESSED — it was never validated against any
// declaration, so it carries no type annotation — even when a same-named GLOBAL
// attribute declaration exists in the schema. A matching global declaration must
// therefore NOT make it count as simple-typed: in XSD 1.1 it contributes NO
// value (the node drops out of the qualified node-set), and in XSD 1.0 it is a
// validity error, exactly as when NO matching global exists. An attribute that
// IS genuinely assessed against a declared attribute use stays simple-typed and
// is accepted.
func TestIDCFieldSkipAttribute(t *testing.T) {
	t.Parallel()

	// @a on <item> is admitted only by the processContents="skip" wildcard; a
	// same-named GLOBAL <xs:attribute name="a"> also exists. The field selects @a.
	const skipWithGlobalSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="a" type="xs:string"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:anyAttribute processContents="skip"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@a"/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	// Identical, but with NO same-named global attribute — the baseline that
	// already rejects the unassessed attribute in 1.0.
	const skipNoGlobalSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:anyAttribute processContents="skip"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@a"/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	// @a is a genuinely-assessed declared attribute use on <item>.
	const assessedAttrSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="a" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@a"/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	const instance = `<root><item a="foo"/></root>`

	compile := func(t *testing.T, src string, v11 bool) xsd.Validator {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		c := xsd.NewCompiler()
		if v11 {
			c = c.Version(xsd.Version11)
		}
		schema, err := c.Compile(t.Context(), doc)
		require.NoError(t, err)
		return xsd.NewValidator(schema)
	}

	validate := func(t *testing.T, v xsd.Validator) (string, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		return errs, err
	}

	t.Run("skip-attr with global is no value in 1.1", func(t *testing.T) {
		t.Parallel()
		errs, err := validate(t, compile(t, skipWithGlobalSchema, true))
		require.NoError(t, err, "1.1 skip-admitted attribute field must contribute no value even with a matching global, got: %s", errs)
	})

	t.Run("skip-attr with global rejected in 1.0", func(t *testing.T) {
		t.Parallel()
		errs, err := validate(t, compile(t, skipWithGlobalSchema, false))
		require.Error(t, err, "1.0 unassessed skip-admitted attribute field must be rejected even with a matching global")
		require.Contains(t, errs, "not simple")
	})

	t.Run("skip-attr without global is no value in 1.1", func(t *testing.T) {
		t.Parallel()
		errs, err := validate(t, compile(t, skipNoGlobalSchema, true))
		require.NoError(t, err, "1.1 skip-admitted attribute field must contribute no value, got: %s", errs)
	})

	t.Run("skip-attr without global rejected in 1.0", func(t *testing.T) {
		t.Parallel()
		errs, err := validate(t, compile(t, skipNoGlobalSchema, false))
		require.Error(t, err, "1.0 unassessed skip-admitted attribute field must be rejected")
		require.Contains(t, errs, "not simple")
	})

	t.Run("assessed attribute field accepted in 1.1", func(t *testing.T) {
		t.Parallel()
		errs, err := validate(t, compile(t, assessedAttrSchema, true))
		require.NoError(t, err, "assessed attribute field must be accepted, got: %s", errs)
	})

	t.Run("assessed attribute field accepted in 1.0", func(t *testing.T) {
		t.Parallel()
		errs, err := validate(t, compile(t, assessedAttrSchema, false))
		require.NoError(t, err, "assessed attribute field must be accepted, got: %s", errs)
	})
}

// TestIDCFieldClassificationMatrix exercises the shared §3.11.4 / cvc-identity-
// constraint.3 field-node classification (GOVERNED | SKIPPED | VIOLATION) across
// the full matrix of node kind (element/attribute) × assessment state × XSD
// version. classifyFieldNode bases the verdict ONLY on genuine-assessment records
// (assessedElemType / assessedAttrs), so a governed node contributes its value, a
// skipped node (unassessed in 1.1) contributes none, and a violation (an assessed
// complex element, or — in XSD 1.0, which has no skipped-node relaxation — any
// unassessed node) invalidates the constraint.
//
// Each instance is authored with DISTINCT field values so a GOVERNED unique/key
// stays valid; a KEY host is used where "no value" would surface as a keyMissing
// violation, making the SKIPPED vs GOVERNED distinction observable.
func TestIDCFieldClassificationMatrix(t *testing.T) {
	t.Parallel()

	// Two <item>s bearing a distinct @a — the instance shared by every attribute
	// cell whose selector is "item" and field is "@a".
	const distinctAttrInstance = `<root><item a="x"/><item a="y"/></root>`

	cases := []struct {
		name     string
		schema   string
		instance string
		v11valid bool // expected validity under XSD 1.1
		v10valid bool // expected validity under XSD 1.0
	}{
		{
			// ELEMENT, genuinely-assessed SIMPLE type → GOVERNED in both versions.
			name: "element assessed simple governed",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType><xs:sequence>
      <xs:element name="item" maxOccurs="unbounded">
        <xs:complexType><xs:sequence>
          <xs:element name="k" type="xs:integer"/>
        </xs:sequence></xs:complexType>
      </xs:element>
    </xs:sequence></xs:complexType>
    <xs:unique name="itemKey"><xs:selector xpath="item"/><xs:field xpath="k"/></xs:unique>
  </xs:element>
</xs:schema>`,
			instance: `<root><item><k>1</k></item><item><k>2</k></item></root>`,
			v11valid: true, v10valid: true,
		},
		{
			// ELEMENT, genuinely-assessed COMPLEX (element-only) type → VIOLATION in
			// both versions (idG006/idK012 shape).
			name: "element assessed complex violation",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType><xs:sequence>
      <xs:element name="item" maxOccurs="unbounded">
        <xs:complexType><xs:sequence>
          <xs:element name="k">
            <xs:complexType><xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence></xs:complexType>
          </xs:element>
        </xs:sequence></xs:complexType>
      </xs:element>
    </xs:sequence></xs:complexType>
    <xs:unique name="itemKey"><xs:selector xpath="item"/><xs:field xpath="k"/></xs:unique>
  </xs:element>
</xs:schema>`,
			instance: `<root><item><k><x>a</x></k></item><item><k><x>b</x></k></item></root>`,
			v11valid: false, v10valid: false,
		},
		{
			// ATTRIBUTE, genuinely-assessed TYPED declared use → GOVERNED both.
			name: "attribute assessed typed governed",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType><xs:sequence>
      <xs:element name="item" maxOccurs="unbounded">
        <xs:complexType><xs:attribute name="a" type="xs:string"/></xs:complexType>
      </xs:element>
    </xs:sequence></xs:complexType>
    <xs:key name="itemKey"><xs:selector xpath="item"/><xs:field xpath="@a"/></xs:key>
  </xs:element>
</xs:schema>`,
			instance: distinctAttrInstance,
			v11valid: true, v10valid: true,
		},
		{
			// ATTRIBUTE, genuinely-assessed UNTYPED declared use (= xs:anySimpleType,
			// itself simple) → GOVERNED both. A KEY host makes it observable: a
			// wrongly-SKIPPED field would be a keyMissing violation.
			name: "attribute assessed untyped governed",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType><xs:sequence>
      <xs:element name="item" maxOccurs="unbounded">
        <xs:complexType><xs:attribute name="a"/></xs:complexType>
      </xs:element>
    </xs:sequence></xs:complexType>
    <xs:key name="itemKey"><xs:selector xpath="item"/><xs:field xpath="@a"/></xs:key>
  </xs:element>
</xs:schema>`,
			instance: distinctAttrInstance,
			v11valid: true, v10valid: true,
		},
		{
			// ATTRIBUTE admitted by a STRICT anyAttribute wildcard WITH a matching
			// UNTYPED global (= xs:anySimpleType) → GOVERNED both (the wildcard-admit
			// records the assessment even though the global has no @type). A KEY host
			// makes it observable.
			name: "attribute wildcard untyped global governed",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="a"/>
  <xs:element name="root">
    <xs:complexType><xs:sequence>
      <xs:element name="item" maxOccurs="unbounded">
        <xs:complexType><xs:anyAttribute processContents="strict"/></xs:complexType>
      </xs:element>
    </xs:sequence></xs:complexType>
    <xs:key name="itemKey"><xs:selector xpath="item"/><xs:field xpath="@a"/></xs:key>
  </xs:element>
</xs:schema>`,
			instance: distinctAttrInstance,
			v11valid: true, v10valid: true,
		},
		{
			// ATTRIBUTE admitted by a STRICT anyAttribute wildcard WITH a matching
			// TYPED global → GOVERNED both.
			name: "attribute wildcard typed global governed",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="a" type="xs:integer"/>
  <xs:element name="root">
    <xs:complexType><xs:sequence>
      <xs:element name="item" maxOccurs="unbounded">
        <xs:complexType><xs:anyAttribute processContents="strict"/></xs:complexType>
      </xs:element>
    </xs:sequence></xs:complexType>
    <xs:key name="itemKey"><xs:selector xpath="item"/><xs:field xpath="@a"/></xs:key>
  </xs:element>
</xs:schema>`,
			instance: `<root><item a="1"/><item a="2"/></root>`,
			v11valid: true, v10valid: true,
		},
		{
			// ATTRIBUTE admitted only by a SKIP anyAttribute wildcard, NO global.
			// Unassessed → SKIPPED (no value) in 1.1, VIOLATION in 1.0. A UNIQUE host
			// so 1.1 stays valid (the node simply drops out of the qualified set).
			name: "attribute skip no global",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType><xs:sequence>
      <xs:element name="item" maxOccurs="unbounded">
        <xs:complexType><xs:anyAttribute processContents="skip"/></xs:complexType>
      </xs:element>
    </xs:sequence></xs:complexType>
    <xs:unique name="itemKey"><xs:selector xpath="item"/><xs:field xpath="@a"/></xs:unique>
  </xs:element>
</xs:schema>`,
			instance: distinctAttrInstance,
			v11valid: true, v10valid: false,
		},
		{
			// ATTRIBUTE admitted only by a SKIP anyAttribute wildcard WITH a matching
			// global (finding b): the global must NOT make it counted — it is still
			// UNASSESSED. SKIPPED in 1.1 (valid), VIOLATION in 1.0.
			name: "attribute skip with global unassessed",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="a" type="xs:string"/>
  <xs:element name="root">
    <xs:complexType><xs:sequence>
      <xs:element name="item" maxOccurs="unbounded">
        <xs:complexType><xs:anyAttribute processContents="skip"/></xs:complexType>
      </xs:element>
    </xs:sequence></xs:complexType>
    <xs:unique name="itemKey"><xs:selector xpath="item"/><xs:field xpath="@a"/></xs:unique>
  </xs:element>
</xs:schema>`,
			instance: distinctAttrInstance,
			v11valid: true, v10valid: false,
		},
		{
			// ATTRIBUTE whose ANCESTOR element is skip content: <inner> is matched by
			// a skip wildcard, so @a on it is unassessed. SKIPPED in 1.1 (valid),
			// VIOLATION in 1.0. The selector host <wrap> is itself assessed.
			name: "attribute ancestor skipped",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType><xs:sequence>
      <xs:element name="wrap" maxOccurs="unbounded">
        <xs:complexType><xs:sequence>
          <xs:any processContents="skip" minOccurs="0"/>
        </xs:sequence></xs:complexType>
      </xs:element>
    </xs:sequence></xs:complexType>
    <xs:unique name="wrapKey"><xs:selector xpath="wrap"/><xs:field xpath="inner/@a"/></xs:unique>
  </xs:element>
</xs:schema>`,
			instance: `<root><wrap><inner a="x"/></wrap><wrap><inner a="y"/></wrap></root>`,
			v11valid: true, v10valid: false,
		},
		{
			// MULTI-NODE, all SKIPPED: field "*" selects TWO skip-content children.
			// §3.11.4 drops both (no value), so the UNIQUE field is absent and the
			// entry drops out → valid in 1.1. In 1.0 an unassessed node is a
			// VIOLATION, so the multi-node field is rejected (not the cardinality
			// error — the first node already fails the simple-type requirement).
			name: "field selects two skipped nodes",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType><xs:sequence>
      <xs:element name="item" maxOccurs="unbounded">
        <xs:complexType><xs:sequence>
          <xs:any processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
        </xs:sequence></xs:complexType>
      </xs:element>
    </xs:sequence></xs:complexType>
    <xs:unique name="itemKey"><xs:selector xpath="item"/><xs:field xpath="*"/></xs:unique>
  </xs:element>
</xs:schema>`,
			instance: `<root><item><g>1</g><h>2</h></item><item><g>3</g><h>4</h></item></root>`,
			v11valid: true, v10valid: false,
		},
		{
			// MULTI-NODE, ONE GOVERNED + ONE SKIPPED: field "*" selects a declared
			// simple <k> (GOVERNED) and a skip-content sibling (SKIPPED). §3.11.4
			// drops the skipped node, leaving exactly one governed node whose value
			// is the field value — a KEY host with distinct <k> makes it observable
			// (a wrongly-absent field would be keyMissing). Valid in 1.1. In 1.0 the
			// skip-content node is a VIOLATION → rejected.
			name: "field selects one governed and one skipped",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType><xs:sequence>
      <xs:element name="item" maxOccurs="unbounded">
        <xs:complexType><xs:sequence>
          <xs:element name="k" type="xs:integer"/>
          <xs:any processContents="skip" minOccurs="0"/>
        </xs:sequence></xs:complexType>
      </xs:element>
    </xs:sequence></xs:complexType>
    <xs:key name="itemKey"><xs:selector xpath="item"/><xs:field xpath="*"/></xs:key>
  </xs:element>
</xs:schema>`,
			instance: `<root><item><k>1</k><extra>a</extra></item><item><k>2</k><extra>b</extra></item></root>`,
			v11valid: true, v10valid: false,
		},
		{
			// MULTI-NODE, TWO GOVERNED: field "*" selects two declared simple
			// elements, both GOVERNED. After dropping skipped nodes (none) more than
			// one governed node remains — the genuine cardinality error (a node-set
			// with more than one member), invalid in BOTH versions.
			name: "field selects two governed nodes",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType><xs:sequence>
      <xs:element name="item" maxOccurs="unbounded">
        <xs:complexType><xs:sequence>
          <xs:element name="k" type="xs:integer"/>
          <xs:element name="j" type="xs:integer"/>
        </xs:sequence></xs:complexType>
      </xs:element>
    </xs:sequence></xs:complexType>
    <xs:unique name="itemKey"><xs:selector xpath="item"/><xs:field xpath="*"/></xs:unique>
  </xs:element>
</xs:schema>`,
			instance: `<root><item><k>1</k><j>2</j></item></root>`,
			v11valid: false, v10valid: false,
		},
	}

	for _, tc := range cases {
		for _, ver := range []struct {
			label string
			v11   bool
			valid bool
		}{
			{"1.1", true, tc.v11valid},
			{"1.0", false, tc.v10valid},
		} {
			t.Run(tc.name+"/"+ver.label, func(t *testing.T) {
				t.Parallel()
				v := compileValidatorVersion(t, tc.schema, ver.v11)
				doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
				require.NoError(t, err)
				var errs string
				err = validateWithOutput(t, v, doc, &errs)
				if ver.valid {
					require.NoError(t, err, "expected valid, got errors: %s", errs)
					return
				}
				require.Error(t, err, "expected a validation error; got none")
			})
		}
	}
}
