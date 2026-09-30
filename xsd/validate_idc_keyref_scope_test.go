package xsd_test

import (
	"fmt"
	"strings"
	"sync"
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
// a fatal schema compilation error, and is never silently accepted (which the
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

// keyrefSubtreeSchema declares a unique ("ItemUnique") on "items" and keyrefs
// on three hosts that never declare ItemUnique themselves, so each keyref
// resolves against the tables of the host's descendant "items" occurrences:
// "node" nests itself, "root" holds sibling "node" chains, and "host" ends in a
// processContents="skip" wildcard. "box" declares its own unique ("BoxKey")
// and a keyref referring to it, and nests itself.
const keyrefSubtreeSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="items">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="xs:string" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="ItemUnique">
      <xs:selector xpath="item"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
  <xs:element name="node">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="items" minOccurs="0" maxOccurs="unbounded"/>
        <xs:element name="ref" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
        <xs:element ref="node" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:keyref name="ItemRef" refer="ItemUnique">
      <xs:selector xpath="ref"/>
      <xs:field xpath="."/>
    </xs:keyref>
  </xs:element>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="node" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="host">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="ref" type="xs:string"/>
        <xs:any processContents="skip" minOccurs="0"/>
      </xs:sequence>
    </xs:complexType>
    <xs:keyref name="HostRef" refer="ItemUnique">
      <xs:selector xpath="ref"/>
      <xs:field xpath="."/>
    </xs:keyref>
  </xs:element>
  <xs:element name="box">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
        <xs:element name="ref" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
        <xs:element ref="box" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="BoxKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="."/>
    </xs:unique>
    <xs:keyref name="BoxRef" refer="BoxKey">
      <xs:selector xpath="ref"/>
      <xs:field xpath="."/>
    </xs:keyref>
  </xs:element>
</xs:schema>`

// keyrefAtRefSchema is keyrefSubtreeSchema's "node" chain plus an "alt" host
// whose keyref is declared through XSD 1.1 @ref to the named "ItemRef".
const keyrefAtRefSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:element name="items">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="xs:string" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="ItemUnique">
      <xs:selector xpath="t:item"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
  <xs:element name="node">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="t:items" minOccurs="0" maxOccurs="unbounded"/>
        <xs:element name="ref" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
        <xs:element ref="t:alt" minOccurs="0" maxOccurs="unbounded"/>
        <xs:element ref="t:node" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:keyref name="ItemRef" refer="t:ItemUnique">
      <xs:selector xpath="t:ref"/>
      <xs:field xpath="."/>
    </xs:keyref>
  </xs:element>
  <xs:element name="alt">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="ref" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
        <xs:element ref="t:node" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:keyref ref="t:ItemRef"/>
  </xs:element>
</xs:schema>`

// keyrefChainDoc builds a "node" chain depth levels deep, one level per line.
// Level i holds the single key "v<i>" and a ref to refAt(i).
func keyrefChainDoc(depth int, refAt func(level int) string) string {
	var b strings.Builder
	for i := range depth {
		fmt.Fprintf(&b, "<node><items><item>v%d</item></items><ref>%s</ref>\n", i, refAt(i))
	}
	for range depth {
		b.WriteString("</node>")
	}
	return b.String()
}

// deepChainRefs refers every level to the innermost key of a 300-level chain,
// except a dangling ref at the outermost level and a ref at the innermost level
// to a key only the outermost level holds.
func deepChainRefs(level int) string {
	switch level {
	case 0:
		return "missing"
	case 299:
		return "v0"
	default:
		return "v299"
	}
}

// deepChainValidRefs refers every level of a 300-level chain to the innermost key.
func deepChainValidRefs(int) string {
	return "v299"
}

// keyrefScopeRecord validates instance with v and returns
// every diagnostic followed by the verdict.
func keyrefScopeRecord(t *testing.T, v xsd.Validator, instance string) string {
	t.Helper()
	doc, err := helium.NewParser().MaxDepth(-1).Parse(t.Context(), []byte(instance))
	require.NoError(t, err)
	var errs string
	if err := validateWithOutput(t, v, doc, &errs); err != nil {
		return errs + "fails to validate\n"
	}
	return errs + "validates\n"
}

// compileKeyrefScopeSchema compiles schemaXML at version and requires a clean compile.
func compileKeyrefScopeSchema(t *testing.T, schemaXML string, version xsd.Version) xsd.Validator {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Version(version).Label("test.xsd").Compile(t.Context(), doc)
	require.NoError(t, err)
	return xsd.NewValidator(schema).Label("test.xml")
}

// TestIDCKeyRefSubtreeScope pins the exact verdict and diagnostic text of
// keyref resolution against key/unique tables gathered from the host's
// descendants, in both XSD versions. A host that does not declare the
// referenced constraint resolves against the union of every descendant
// occurrence's table, excluding the host itself: a key-sequence present in two
// children still satisfies the keyref (no conflict dropping), and a key
// declared on an ancestor or a sibling does not. A host that declares the
// referenced constraint itself consults only its own table.
func TestIDCKeyRefSubtreeScope(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		schema   string
		instance string
		// v11Only marks a case whose schema uses XSD 1.1 syntax.
		v11Only bool
		// want is the record under XSD 1.1, and under XSD 1.0 unless want10 is set.
		want   string
		want10 string
	}{
		{
			name:   "duplicate key in two sibling children satisfies the keyref",
			schema: keyrefSubtreeSchema,
			instance: `<node>
<items><item>x</item></items>
<items><item>x</item></items>
<ref>x</ref>
</node>`,
			want: "validates\n",
		},
		{
			name:   "keyref at intermediate scopes",
			schema: keyrefSubtreeSchema,
			instance: `<node>
<items><item>o1</item></items>
<ref>i1</ref>
<node>
<items><item>m1</item></items>
<ref>m1</ref>
<node>
<items><item>i1</item></items>
<ref>o1</ref>
</node>
</node>
</node>`,
			want: "test.xml:9: Schemas validity error : Element 'ref': No match found for key-sequence ['o1'] of keyref 'ItemRef'.\n" +
				"fails to validate\n",
		},
		{
			name:   "host declaring the referenced constraint ignores descendant tables",
			schema: keyrefSubtreeSchema,
			instance: `<box>
<item>a</item>
<ref>a</ref>
<ref>b</ref>
<box>
<item>b</item>
<ref>b</ref>
</box>
</box>`,
			want: "test.xml:4: Schemas validity error : Element 'ref': No match found for key-sequence ['b'] of keyref 'BoxRef'.\n" +
				"fails to validate\n",
		},
		{
			name:    "keyref declared through @ref resolves like the named form",
			schema:  keyrefAtRefSchema,
			v11Only: true,
			instance: `<node xmlns="urn:t">
<ref>a</ref>
<alt>
<ref>a</ref>
<ref>zz</ref>
<node>
<items><item>a</item></items>
</node>
</alt>
</node>`,
			want: "test.xml:5: Schemas validity error : Element '{urn:t}ref': No match found for key-sequence ['zz'] of keyref '{urn:t}ItemRef'.\n" +
				"fails to validate\n",
		},
		{
			// XSD 1.1 drops skip content from the selector node-set, so the
			// gathered table is empty. XSD 1.0 selects it, reports the unassessed
			// field once from the "items" pass-2 visit, and gathers no key.
			name:   "keys inside a skip wildcard",
			schema: keyrefSubtreeSchema,
			instance: `<host>
<ref>v9</ref>
<items><item>v9</item></items>
</host>`,
			want: "test.xml:2: Schemas validity error : Element 'ref': No match found for key-sequence ['v9'] of keyref 'HostRef'.\n" +
				"fails to validate\n",
			want10: "test.xml:2: Schemas validity error : Element 'ref': No match found for key-sequence ['v9'] of keyref 'HostRef'.\n" +
				"test.xml:3: Schemas validity error : Element 'item': The XPath '.' of a field of unique identity-constraint 'ItemUnique' evaluates to a node whose type is not simple.\n" +
				"fails to validate\n",
		},
		{
			name:     "deep chain with dangling refs at both ends",
			schema:   keyrefSubtreeSchema,
			instance: keyrefChainDoc(300, deepChainRefs),
			want: "test.xml:1: Schemas validity error : Element 'ref': No match found for key-sequence ['missing'] of keyref 'ItemRef'.\n" +
				"test.xml:300: Schemas validity error : Element 'ref': No match found for key-sequence ['v0'] of keyref 'ItemRef'.\n" +
				"fails to validate\n",
		},
		{
			name:     "deep chain with every ref resolved",
			schema:   keyrefSubtreeSchema,
			instance: keyrefChainDoc(300, deepChainValidRefs),
			want:     "validates\n",
		},
		{
			name:   "sibling host after a nested chain",
			schema: keyrefSubtreeSchema,
			instance: `<root>
<node>
<ref>k2</ref>
<node>
<items><item>k2</item></items>
</node>
</node>
<node>
<ref>k2</ref>
</node>
</root>`,
			want: "test.xml:9: Schemas validity error : Element 'ref': No match found for key-sequence ['k2'] of keyref 'ItemRef'.\n" +
				"fails to validate\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !tc.v11Only {
				want10 := tc.want10
				if want10 == "" {
					want10 = tc.want
				}
				got := keyrefScopeRecord(t, compileKeyrefScopeSchema(t, tc.schema, xsd.Version10), tc.instance)
				require.Equal(t, want10, got, "XSD 1.0")
			}
			got := keyrefScopeRecord(t, compileKeyrefScopeSchema(t, tc.schema, xsd.Version11), tc.instance)
			require.Equal(t, tc.want, got, "XSD 1.1")
		})
	}
}

// TestIDCKeyRefSubtreeScopeConcurrent validates the deep keyref chain from 8
// goroutines sharing one Validator. The per-run keyref state must live on the
// validation run, never on the Validator or Schema, so under -race this fails
// on any shared write, and every goroutine must reproduce the single-goroutine
// record byte for byte.
func TestIDCKeyRefSubtreeScopeConcurrent(t *testing.T) {
	t.Parallel()

	const workers = 8
	for _, version := range []xsd.Version{xsd.Version10, xsd.Version11} {
		v := compileKeyrefScopeSchema(t, keyrefSubtreeSchema, version)
		instance := keyrefChainDoc(300, deepChainRefs)
		want := keyrefScopeRecord(t, v, instance)

		got := make([]string, workers)
		var wg sync.WaitGroup
		for i := range workers {
			wg.Go(func() {
				got[i] = keyrefScopeRecord(t, v, instance)
			})
		}
		wg.Wait()
		for i := range workers {
			require.Equal(t, want, got[i], "goroutine %d", i)
		}
	}
}
