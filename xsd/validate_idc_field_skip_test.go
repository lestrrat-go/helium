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
