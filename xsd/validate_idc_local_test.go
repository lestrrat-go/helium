package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestIDCLocalKeyRefEvaluated is the regression test for the conformance gap
// where identity constraints declared on a LOCAL element declaration (buried
// inside a content model) were compile-checked but never EVALUATED at
// instance-validation time. Pass-2 resolved the host declaration via
// lookupElemDecl, which only finds GLOBAL declarations, so a key/unique/keyref
// on a local element was silently skipped: a dangling local keyref VALIDATED.
// xmllint rejects it. The fix records the actual *ElementDecl for each element
// instance during pass-1 and uses it in pass-2 before falling back to the
// global lookup.
func TestIDCLocalKeyRefEvaluated(t *testing.T) {
	t.Parallel()

	// Both the key (localItemKey) and the keyref (localRef) are declared on the
	// LOCAL element <items> nested inside <root>'s content model.
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
              <xs:element name="ref" maxOccurs="unbounded">
                <xs:complexType>
                  <xs:attribute name="r" type="xs:string"/>
                </xs:complexType>
              </xs:element>
            </xs:sequence>
          </xs:complexType>
          <xs:key name="localItemKey">
            <xs:selector xpath="item"/>
            <xs:field xpath="@id"/>
          </xs:key>
          <xs:keyref name="localRef" refer="localItemKey">
            <xs:selector xpath="ref"/>
            <xs:field xpath="@r"/>
          </xs:keyref>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	v, compileErrs := compileXSD(t, schemaXML)
	require.Empty(t, compileErrs, "local IDC schema should compile clean")

	t.Run("dangling local keyref is rejected", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root><items><item id="a"/><ref r="missing"/></items></root>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.Error(t, err, "a dangling keyref on a local element must be rejected, not silently skipped")
		require.Contains(t, errs, "No match found for key-sequence ['missing'] of keyref 'localRef'.")
	})

	t.Run("matching local keyref validates", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root><items><item id="a"/><ref r="a"/></items></root>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.NoError(t, err, "expected valid, got: %s", errs)
	})
}

// TestIDCLocalKeyRefDescendantKey confirms XSD identity-constraint table
// propagation: a keyref on a LOCAL element referring to a key/unique declared on
// a DESCENDANT element is IN scope (descendant tables propagate up to the host),
// so a matching value validates and a dangling one is rejected. This mirrors the
// bug322411 golden case but with both constraints local. xmllint validates the
// matching instance and rejects the dangling one.
func TestIDCLocalKeyRefDescendantKey(t *testing.T) {
	t.Parallel()

	// keyref "ItemRef" is on the LOCAL <ELEMENT>; the unique "ItemUnique" it
	// refers to is on the GLOBAL <items>, a CHILD of <ELEMENT>.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="items">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="ItemUnique">
      <xs:selector xpath="item"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
  <xs:element name="ELEMENTS">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="ELEMENT">
          <xs:complexType>
            <xs:sequence>
              <xs:element ref="items"/>
              <xs:element name="ref" type="xs:string" maxOccurs="unbounded"/>
            </xs:sequence>
          </xs:complexType>
          <xs:keyref name="ItemRef" refer="ItemUnique">
            <xs:selector xpath="ref"/>
            <xs:field xpath="."/>
          </xs:keyref>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	v, compileErrs := compileXSD(t, schemaXML)
	require.Empty(t, compileErrs, "descendant-key keyref schema should compile clean")

	t.Run("descendant key satisfies keyref", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<ELEMENTS><ELEMENT><items><item>a</item><item>b</item></items><ref>a</ref></ELEMENT></ELEMENTS>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.NoError(t, err, "a key on a descendant element propagates up and satisfies the keyref; got: %s", errs)
	})

	t.Run("dangling keyref still rejected", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<ELEMENTS><ELEMENT><items><item>a</item></items><ref>missing</ref></ELEMENT></ELEMENTS>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.Error(t, err, "a dangling keyref must still be rejected even with descendant propagation")
		require.Contains(t, errs, "No match found for key-sequence ['missing'] of keyref 'ItemRef'.")
	})
}

// TestIDCLocalKeyRefSiblingKeyOutOfScope confirms the scope boundary: a keyref
// on a LOCAL element referring to a key/unique declared on a SIBLING local
// element is out of scope (the key is not in the keyref host occurrence's
// subtree), so even a value that exists under the sibling key does NOT satisfy
// it. xmllint rejects this case; subtree-scoped key-table gathering must too.
func TestIDCLocalKeyRefSiblingKeyOutOfScope(t *testing.T) {
	t.Parallel()

	// localItemKey is on <items>; localRef is on the SIBLING <refs>.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="items">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="item" maxOccurs="unbounded">
                <xs:complexType><xs:attribute name="id" type="xs:string"/></xs:complexType>
              </xs:element>
            </xs:sequence>
          </xs:complexType>
          <xs:key name="localItemKey"><xs:selector xpath="item"/><xs:field xpath="@id"/></xs:key>
        </xs:element>
        <xs:element name="refs">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="ref" maxOccurs="unbounded">
                <xs:complexType><xs:attribute name="r" type="xs:string"/></xs:complexType>
              </xs:element>
            </xs:sequence>
          </xs:complexType>
          <xs:keyref name="localRef" refer="localItemKey"><xs:selector xpath="ref"/><xs:field xpath="@r"/></xs:keyref>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	v, compileErrs := compileXSD(t, schemaXML)
	require.Empty(t, compileErrs, "sibling-key keyref schema should compile clean (refer resolves in the symbol space)")

	// "a" exists under the sibling localItemKey, but it is out of localRef's scope.
	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><items><item id="a"/></items><refs><ref r="a"/></refs></root>`))
	require.NoError(t, err)
	var errs string
	err = validateWithOutput(t, v, doc, &errs)
	require.Error(t, err, "a key on a sibling local element is out of the keyref scope")
	require.Contains(t, errs, "No match found for key-sequence ['a'] of keyref 'localRef'.")
}

// TestIDCLocalShadowsGlobalNoInherit confirms a LOCAL element declaration that
// merely shadows a same-named GLOBAL declaration does NOT inherit the global's
// identity constraints. The local <item> here carries no IDC of its own; the
// global <item> carries a key on its <sub> children. An instance whose LOCAL
// <item> has duplicate @k values across <sub> must VALIDATE — the global key
// is out of scope. xmllint validates this instance and only enforces
// globalSubKey on the global item. Regression for idcHostDecl falling back to
// the global lookup whenever the recorded local decl carried zero IDCs.
func TestIDCLocalShadowsGlobalNoInherit(t *testing.T) {
	t.Parallel()

	// Global <item> carries a key (globalSubKey); the LOCAL <item> nested in
	// <root> shadows the NAME but declares no identity constraint.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="item">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="sub" maxOccurs="unbounded">
          <xs:complexType><xs:attribute name="k" type="xs:string"/></xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="globalSubKey">
      <xs:selector xpath="sub"/>
      <xs:field xpath="@k"/>
    </xs:key>
  </xs:element>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="sub" maxOccurs="unbounded">
                <xs:complexType><xs:attribute name="k" type="xs:string"/></xs:complexType>
              </xs:element>
            </xs:sequence>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	v, compileErrs := compileXSD(t, schemaXML)
	require.Empty(t, compileErrs, "shadowing schema should compile clean")

	t.Run("local item does not inherit global key", func(t *testing.T) {
		t.Parallel()
		// Duplicate @k under the LOCAL item: only valid because globalSubKey
		// does not apply to the shadowing local declaration.
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root><item><sub k="dup"/><sub k="dup"/></item></root>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.NoError(t, err, "a local element shadowing a global name must not inherit the global's key; got: %s", errs)
	})

	t.Run("global item still enforces its key", func(t *testing.T) {
		t.Parallel()
		// Same duplicate, but under the GLOBAL item where globalSubKey applies.
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<item><sub k="dup"/><sub k="dup"/></item>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.Error(t, err, "the global item's key must still be enforced")
		require.Contains(t, errs, "Duplicate key-sequence ['dup']")
		require.Contains(t, errs, "globalSubKey")
	})
}

// TestIDCLocalShadowsGlobalInvalidXsiType is the regression test for the
// recordElemDecl ordering bug: pass-1 recorded the LOCAL host declaration only
// AFTER xsi:type/abstract resolution succeeded. So when a LOCAL element shadowing
// a same-named GLOBAL declaration carried an INVALID xsi:type, the type-resolution
// branch did `continue` BEFORE recording the local decl, leaving pass-2's
// idcHostDecl to fall back to the GLOBAL declaration and apply the global's key
// — producing a spurious duplicate-key diagnostic on top of the real xsi:type
// error. xmllint reports ONLY the xsi:type error here. The fix records the local
// host decl immediately after it is resolved, before any validation branch can
// continue.
func TestIDCLocalShadowsGlobalInvalidXsiType(t *testing.T) {
	t.Parallel()

	// Global <item> carries globalSubKey on its <sub> children. The LOCAL <item>
	// nested in <root> shadows the NAME (type localItemType) and declares no
	// identity constraint of its own.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="item">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="sub" maxOccurs="unbounded">
          <xs:complexType><xs:attribute name="k" type="xs:string"/></xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="globalSubKey">
      <xs:selector xpath="sub"/>
      <xs:field xpath="@k"/>
    </xs:key>
  </xs:element>
  <xs:complexType name="localItemType">
    <xs:sequence>
      <xs:element name="sub" maxOccurs="unbounded">
        <xs:complexType><xs:attribute name="k" type="xs:string"/></xs:complexType>
      </xs:element>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="localItemType"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	v, compileErrs := compileXSD(t, schemaXML)
	require.Empty(t, compileErrs, "shadowing-with-xsi:type schema should compile clean")

	// The LOCAL <item> carries an unresolvable xsi:type AND duplicate @k under its
	// <sub> children. The duplicate would only matter if globalSubKey (the GLOBAL
	// item's key) were wrongly applied; it must NOT be, so the only error is the
	// xsi:type one.
	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><item xsi:type="nonExistentType"><sub k="dup"/><sub k="dup"/></item></root>`))
	require.NoError(t, err)
	var errs string
	err = validateWithOutput(t, v, doc, &errs)
	require.Error(t, err, "the invalid xsi:type must be reported")
	require.Contains(t, errs, "xsi:type", "expected the xsi:type error; got: %s", errs)
	require.NotContains(t, errs, "globalSubKey",
		"a local element shadowing a global must not inherit the global's key even when its xsi:type is invalid; got: %s", errs)
	require.NotContains(t, errs, "Duplicate key-sequence",
		"no spurious IDC diagnostic should fire for the shadowing local element; got: %s", errs)
}

// TestIDCAllMissingRequiredShadowsGlobal is the regression test for the matchAll
// recordElemDecl ordering bug: in an xs:all group, the matched local child
// declaration was recorded only AFTER the post-scan missing-required/duplicate
// early returns. So when a matched LOCAL child shadowing a same-named GLOBAL sat
// alongside a MISSING required sibling, matchAll returned at the missing-required
// check BEFORE recording the local decl, leaving pass-2's idcHostDecl to fall
// back to the GLOBAL declaration and apply its key — producing a spurious
// globalSubKey duplicate-key diagnostic on top of the real missing-child error.
// xmllint reports ONLY the missing-child error here. The fix records each matched
// child decl during the initial xs:all scan, before any early return.
func TestIDCAllMissingRequiredShadowsGlobal(t *testing.T) {
	t.Parallel()

	// Global <item> carries globalSubKey on its <sub> children. Inside <root>'s
	// xs:all group the LOCAL <item> shadows the NAME and declares no identity
	// constraint; a sibling required <other> is also part of the all group.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="item">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="sub" maxOccurs="unbounded">
          <xs:complexType><xs:attribute name="k" type="xs:string"/></xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="globalSubKey">
      <xs:selector xpath="sub"/>
      <xs:field xpath="@k"/>
    </xs:key>
  </xs:element>
  <xs:element name="root">
    <xs:complexType>
      <xs:all>
        <xs:element name="item">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="sub" maxOccurs="unbounded">
                <xs:complexType><xs:attribute name="k" type="xs:string"/></xs:complexType>
              </xs:element>
            </xs:sequence>
          </xs:complexType>
        </xs:element>
        <xs:element name="other" type="xs:string"/>
      </xs:all>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	v, compileErrs := compileXSD(t, schemaXML)
	require.Empty(t, compileErrs, "xs:all shadowing schema should compile clean")

	// The required sibling <other> is OMITTED, and the LOCAL <item> carries
	// duplicate @k under its <sub> children. The only error must be the missing
	// required child; the duplicate must NOT trigger globalSubKey.
	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item><sub k="dup"/><sub k="dup"/></item></root>`))
	require.NoError(t, err)
	var errs string
	err = validateWithOutput(t, v, doc, &errs)
	require.Error(t, err, "the missing required child must be reported")
	require.Contains(t, errs, "Missing child element(s)",
		"expected the missing-child error; got: %s", errs)
	require.NotContains(t, errs, "globalSubKey",
		"a local element shadowing a global in xs:all must not inherit the global's key when a sibling is missing; got: %s", errs)
	require.NotContains(t, errs, "Duplicate key-sequence",
		"no spurious IDC diagnostic should fire for the shadowing local element; got: %s", errs)
}

// TestIDCAllDuplicateShadowsGlobal is the regression test for the matchAll
// DUPLICATE-child recordElemDecl ordering bug: in an xs:all group, the matched
// local child declaration was recorded only AFTER the seen[idx] duplicate check,
// so when a DUPLICATE local child shadowing a same-named GLOBAL declaration
// appeared, matchAll returned at the duplicate check BEFORE recording the local
// decl for that occurrence, leaving pass-2's idcHostDecl to fall back to the
// GLOBAL declaration and apply its key — producing a spurious globalSubKey
// duplicate-key diagnostic on top of the real duplicate/cardinality error.
// xmllint reports ONLY the "This element is not expected" duplicate error here.
// The fix records each matched child decl at the absolute earliest point in the
// xs:all scan, before the duplicate check and any other early return.
func TestIDCAllDuplicateShadowsGlobal(t *testing.T) {
	t.Parallel()

	// Global <item> carries globalSubKey on its <sub> children. Inside <root>'s
	// xs:all group the LOCAL <item> shadows the NAME and declares no identity
	// constraint; a sibling required <other> is also part of the all group.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="item">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="sub" maxOccurs="unbounded">
          <xs:complexType><xs:attribute name="k" type="xs:string"/></xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="globalSubKey">
      <xs:selector xpath="sub"/>
      <xs:field xpath="@k"/>
    </xs:key>
  </xs:element>
  <xs:element name="root">
    <xs:complexType>
      <xs:all>
        <xs:element name="item">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="sub" maxOccurs="unbounded">
                <xs:complexType><xs:attribute name="k" type="xs:string"/></xs:complexType>
              </xs:element>
            </xs:sequence>
          </xs:complexType>
        </xs:element>
        <xs:element name="other" type="xs:string"/>
      </xs:all>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	v, compileErrs := compileXSD(t, schemaXML)
	require.Empty(t, compileErrs, "xs:all duplicate-shadowing schema should compile clean")

	// The LOCAL <item> appears TWICE in the xs:all group (a duplicate), and the
	// FIRST one carries duplicate @k under its <sub> children. The FIRST item
	// matches the local declaration (no identity constraints), so globalSubKey must
	// NOT run on it — no "Duplicate key-sequence" from the two k="dup" siblings.
	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item><sub k="dup"/><sub k="dup"/></item><item><sub k="x"/></item><other>o</other></root>`))
	require.NoError(t, err)
	var errs string
	err = validateWithOutput(t, v, doc, &errs)
	require.Error(t, err, "the duplicate xs:all child must be reported")
	require.Contains(t, errs, "This element is not expected",
		"expected the duplicate/unexpected-child error; got: %s", errs)
	// The matched (first) local <item> shadows the global name and carries no key,
	// so it never produces a duplicate-key error from its two k="dup" siblings.
	// (The unexpected SECOND <item> is unmatched — hence unassessed — so its global
	// fallback declaration's globalSubKey field @k is a cvc-identity-constraint.3
	// non-simple violation, which is legitimate noise on an already-invalid tree
	// and does not represent the shadowing local inheriting the key.)
	require.NotContains(t, errs, "Duplicate key-sequence",
		"a duplicate local element shadowing a global in xs:all must not inherit the global's key; got: %s", errs)
}

// TestIDCLocalUniqueEvaluated confirms an xs:unique declared on a LOCAL element
// is evaluated at validation time: a duplicate key-sequence is rejected.
func TestIDCLocalUniqueEvaluated(t *testing.T) {
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
          <xs:unique name="localItemUnique">
            <xs:selector xpath="item"/>
            <xs:field xpath="@id"/>
          </xs:unique>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	v, compileErrs := compileXSD(t, schemaXML)
	require.Empty(t, compileErrs, "local unique schema should compile clean")

	t.Run("duplicate rejected", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root><items><item id="a"/><item id="a"/></items></root>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.Error(t, err, "a duplicate key-sequence under a local unique must be rejected")
		require.Contains(t, errs, "Duplicate key-sequence")
	})

	t.Run("distinct validates", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root><items><item id="a"/><item id="b"/></items></root>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.NoError(t, err, "expected valid, got: %s", errs)
	})
}
