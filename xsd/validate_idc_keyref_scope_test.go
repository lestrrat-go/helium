package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileXSD compiles a schema and returns the validator plus the fatal
// compile-error string (empty when the schema compiled clean).
func compileXSD(t *testing.T, schemaXML string) (xsd.Validator, string) {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	s, err := xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
	requireCompileResultErr(t, err)
	_, errors := partitionCompileErrors(collector.Errors())
	if errors != "" {
		return xsd.Validator{}, errors
	}
	return xsd.NewValidator(s), ""
}

// TestIDCFieldXPathFunctionRejected covers a field XPath that uses a function
// call. Such an expression is outside the XSD identity-constraint XPath subset
// (selectors/fields are restricted location paths), so it must be a fatal schema
// compilation error. Previously it compiled and the field's evaluation error was
// swallowed, silently disabling the constraint so a unique/keyref could miss a
// violation; now the out-of-subset XPath is rejected up front.
func TestIDCFieldXPathFunctionRejected(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="bogusfn(.)"/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	_, compileErrs := compileXSD(t, schemaXML)
	require.NotEmpty(t, compileErrs, "an out-of-subset field XPath must be a fatal schema error")
	require.Contains(t, compileErrs, "is not a valid field")
}

// TestIDCCrossElementKeyRefOutOfScope verifies the XSD identity-constraint scope
// rule for a keyref whose referenced key is declared on a DIFFERENT element. The
// keyref's referenced key/unique must be in the keyref host occurrence's scope;
// a key declared on a SIBLING element is NOT, so every key-sequence is a "no
// match" failure — even when an equal value exists under the sibling key. This
// matches xmllint, which rejects BOTH the "matching" and the dangling instance
// here. (A doc-wide merged key table would falsely accept the first.) The schema
// itself still COMPILES: @refer resolves in the schema-wide identity-constraint
// symbol space; only value resolution is occurrence-scoped.
func TestIDCCrossElementKeyRefOutOfScope(t *testing.T) {
	t.Parallel()

	// "productKey" is declared on the GLOBAL element <products>; "orderRef" is
	// declared on the GLOBAL element <orders>. They live on different elements, so
	// the key is out of the keyref's scope.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="catalog">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="products"/>
        <xs:element ref="orders"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="products">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="product" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="productKey">
      <xs:selector xpath="product"/>
      <xs:field xpath="@id"/>
    </xs:key>
  </xs:element>
  <xs:element name="orders">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="order" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="product" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:keyref name="orderRef" refer="productKey">
      <xs:selector xpath="order"/>
      <xs:field xpath="@product"/>
    </xs:keyref>
  </xs:element>
</xs:schema>`

	v, compileErrs := compileXSD(t, schemaXML)
	require.Empty(t, compileErrs, "cross-element keyref should compile clean (refer resolves in the symbol space)")

	t.Run("out-of-scope key does not satisfy keyref", func(t *testing.T) {
		t.Parallel()
		// An equal value DOES exist under the sibling productKey, but it is out of
		// the orderRef scope, so xmllint (and helium) reject it.
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<catalog><products><product id="a"/><product id="b"/></products><orders><order product="a"/></orders></catalog>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.Error(t, err, "a key on a sibling element is out of the keyref scope and must not satisfy it")
		require.Contains(t, errs, "No match found")
	})

	t.Run("dangling cross-element keyref fails", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<catalog><products><product id="a"/></products><orders><order product="missing"/></orders></catalog>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.Error(t, err, "a dangling cross-element keyref must be rejected, not silently skipped")
		require.Contains(t, errs, "No match found")
	})
}

// TestIDCKeyRefUnboundReferPrefix covers an xs:keyref/@refer that uses a
// namespace prefix not bound in scope. @refer is a QName, so an unbound prefix is
// a fatal schema compilation error rather than being silently accepted (which the
// previous local-name-only matching did).
func TestIDCKeyRefUnboundReferPrefix(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:string"/>
          </xs:complexType>
        </xs:element>
        <xs:element name="ref" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="to" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@id"/>
    </xs:key>
    <xs:keyref name="itemRef" refer="bogus:itemKey">
      <xs:selector xpath="ref"/>
      <xs:field xpath="@to"/>
    </xs:keyref>
  </xs:element>
</xs:schema>`

	_, compileErrs := compileXSD(t, schemaXML)
	require.NotEmpty(t, compileErrs, "an unbound @refer prefix must be a fatal schema error")
	require.Contains(t, compileErrs, "bogus")
}

// TestIDCSameNamespaceKeyRef confirms that a valid same-target-namespace keyref
// (prefixed @refer resolving through the schema's namespace context) still
// compiles and enforces referential integrity after the namespace-aware refer
// resolution change.
func TestIDCSameNamespaceKeyRef(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:test" targetNamespace="urn:test" elementFormDefault="qualified">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:string"/>
          </xs:complexType>
        </xs:element>
        <xs:element name="ref" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="to" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="itemKey">
      <xs:selector xpath="t:item"/>
      <xs:field xpath="@id"/>
    </xs:key>
    <xs:keyref name="itemRef" refer="t:itemKey">
      <xs:selector xpath="t:ref"/>
      <xs:field xpath="@to"/>
    </xs:keyref>
  </xs:element>
</xs:schema>`

	v, compileErrs := compileXSD(t, schemaXML)
	require.Empty(t, compileErrs, "valid prefixed same-namespace keyref should compile clean")

	t.Run("matching validates", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root xmlns="urn:test"><item id="a"/><item id="b"/><ref to="a"/></root>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.NoError(t, err, "expected valid, got: %s", errs)
	})

	t.Run("dangling fails", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root xmlns="urn:test"><item id="a"/><ref to="missing"/></root>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.Error(t, err)
		require.Contains(t, errs, "No match found")
	})
}

// TestIDCKeyRefOccurrenceScope is the regression test for the leak that the
// document-level key table introduced: a key/unique declared on a REPEATING host
// element must be scoped to each occurrence, so a keyref on a later occurrence
// cannot satisfy itself using an earlier occurrence's keys. xmllint rejects the
// cross-occurrence case; a doc-wide merged table would falsely accept it. The
// host <group> is a GLOBAL element referenced with maxOccurs="unbounded", which
// is the path pass-2 IDC evaluation actually walks.
func TestIDCKeyRefOccurrenceScope(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="group" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="group">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:string"/>
          </xs:complexType>
        </xs:element>
        <xs:element name="ref" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="r" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@id"/>
    </xs:key>
    <xs:keyref name="itemRef" refer="itemKey">
      <xs:selector xpath="ref"/>
      <xs:field xpath="@r"/>
    </xs:keyref>
  </xs:element>
</xs:schema>`

	v, compileErrs := compileXSD(t, schemaXML)
	require.Empty(t, compileErrs, "repeated-host keyref schema should compile clean")

	t.Run("each occurrence satisfies its own keyref", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root>
  <group><item id="a"/><ref r="a"/></group>
  <group><item id="b"/><ref r="b"/></group>
</root>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.NoError(t, err, "expected valid, got: %s", errs)
	})

	t.Run("cross-occurrence key does not leak", func(t *testing.T) {
		t.Parallel()
		// group2's ref points to group1's item key — out of group2's scope.
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root>
  <group><item id="a"/><ref r="a"/></group>
  <group><item id="b"/><ref r="a"/></group>
</root>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.Error(t, err, "a later occurrence must not reuse an earlier occurrence's keys")
		require.Contains(t, errs, "No match found for key-sequence ['a'] of keyref 'itemRef'.")
	})
}

// TestIDCLocalKeyRefMissingRefer is the regression test for finding #2: a keyref
// declared on a LOCAL element declaration (buried inside a content model) with a
// @refer that names no existing key/unique must be a fatal schema compile error.
// The prior registry scanned only GLOBAL element declarations, so a local keyref's
// dangling refer was never checked and the constraint was silently disabled.
func TestIDCLocalKeyRefMissingRefer(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="child">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="ref" maxOccurs="unbounded">
                <xs:complexType>
                  <xs:attribute name="r" type="xs:string"/>
                </xs:complexType>
              </xs:element>
            </xs:sequence>
          </xs:complexType>
          <xs:keyref name="danglingRef" refer="nonexistentKey">
            <xs:selector xpath="ref"/>
            <xs:field xpath="@r"/>
          </xs:keyref>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	_, compileErrs := compileXSD(t, schemaXML)
	require.NotEmpty(t, compileErrs, "a local keyref with a missing refer must be a fatal schema error")
	require.Contains(t, compileErrs, "nonexistentKey")
}

// TestIDCLocalKeyRefCrossLocalKey confirms the registry also reaches a key/unique
// declared on a LOCAL element when validating a local keyref's @refer: a local
// keyref referring to a key on another local element compiles clean (the refer
// resolves), exercising the recursive content-model walk in collectAllIDCs.
func TestIDCLocalKeyRefCrossLocalKey(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="items">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="item" maxOccurs="unbounded">
                <xs:complexType>
                  <xs:attribute name="id" type="xs:string"/>
                </xs:complexType>
              </xs:element>
            </xs:sequence>
          </xs:complexType>
          <xs:key name="localItemKey">
            <xs:selector xpath="item"/>
            <xs:field xpath="@id"/>
          </xs:key>
        </xs:element>
        <xs:element name="refs">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="ref" maxOccurs="unbounded">
                <xs:complexType>
                  <xs:attribute name="r" type="xs:string"/>
                </xs:complexType>
              </xs:element>
            </xs:sequence>
          </xs:complexType>
          <xs:keyref name="localRef" refer="localItemKey">
            <xs:selector xpath="ref"/>
            <xs:field xpath="@r"/>
          </xs:keyref>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	_, compileErrs := compileXSD(t, schemaXML)
	require.Empty(t, compileErrs, "a local keyref referring to a local key must compile clean")
}
