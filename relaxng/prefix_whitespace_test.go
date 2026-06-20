package relaxng_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// compileErrorsFor compiles the given RELAX NG schema and returns the fatal
// compile-error text collected during compilation (empty when the schema
// compiles cleanly).
func compileErrorsFor(t *testing.T, schema string) string {
	t.Helper()

	schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err, "schema should parse")

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err = relaxng.NewCompiler().ErrorHandler(collector).Compile(t.Context(), schemaDoc)
	require.NoError(t, err, "Compile should not return a hard error")
	_ = collector.Close()
	_, compileErrors := partitionCompileErrors(collector.Errors())
	return compileErrors
}

// TestUnboundPrefixInNameIsCompileError covers D-RNG-002: a QName whose prefix
// is not bound to any in-scope namespace declaration must be a fatal compile
// error rather than being silently treated as the empty namespace. Otherwise a
// schema such as <element name="p:admin"> (without xmlns:p) would wrongly match
// a no-namespace <admin/> instance.
func TestUnboundPrefixInNameIsCompileError(t *testing.T) {
	t.Parallel()

	t.Run("element name unbound prefix", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="p:admin" xmlns="http://relaxng.org/ns/structure/1.0">
  <empty/>
</element>`
		require.NotEmpty(t, compileErrorsFor(t, schema),
			"unbound prefix on <element name> must be a fatal compile error")
	})

	t.Run("attribute name unbound prefix", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <attribute name="p:id"/>
</element>`
		require.NotEmpty(t, compileErrorsFor(t, schema),
			"unbound prefix on <attribute name> must be a fatal compile error")
	})

	t.Run("name class unbound prefix", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <element>
    <name>p:admin</name>
    <empty/>
  </element>
</element>`
		require.NotEmpty(t, compileErrorsFor(t, schema),
			"unbound prefix in <name> name class must be a fatal compile error")
	})

	t.Run("default handler: unbound prefix does not spuriously validate", func(t *testing.T) {
		t.Parallel()
		// Regression for D-RNG-002: the DEFAULT compile path has no error
		// collector, so a fatal diagnostic is dropped and Compile still returns
		// a non-nil grammar with a nil error. An unbound prefix must therefore
		// install a never-matching name class so that validation of a
		// no-namespace <admin/> still fails (it must NOT match name="p:admin").
		schema := `<element name="p:admin" xmlns="http://relaxng.org/ns/structure/1.0">
  <empty/>
</element>`

		schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err, "schema should parse")

		grammar, err := relaxng.NewCompiler().Compile(t.Context(), schemaDoc)
		require.NoError(t, err, "default Compile returns a nil hard error")
		require.NotNil(t, grammar, "default Compile still returns a grammar")

		instanceDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<admin/>`))
		require.NoError(t, err, "instance should parse")

		require.Error(t, relaxng.NewValidator(grammar).Validate(t.Context(), instanceDoc),
			"an unbound-prefix schema name must not spuriously match a no-namespace element")
	})

	t.Run("bound prefix compiles cleanly", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="p:admin"
    xmlns="http://relaxng.org/ns/structure/1.0"
    xmlns:p="urn:example:p">
  <empty/>
</element>`
		require.Empty(t, compileErrorsFor(t, schema),
			"a bound prefix must compile without error")
	})

	t.Run("implicit xml prefix compiles cleanly", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <attribute name="xml:lang"/>
</element>`
		require.Empty(t, compileErrorsFor(t, schema),
			"the implicit xml prefix must always be bound")
	})
}

// TestUnboundPrefixInExceptPoisonsNameClass covers the interaction with the
// unbound-prefix fix: an invalid/unbound name inside an <except> compiles to a
// never-matching name class. On the DEFAULT compile path (no error collector)
// the fatal diagnostic is dropped, so the enclosing anyName/nsName must poison
// itself rather than silently treat the exclusion as empty — otherwise it would
// match everything and spuriously validate the instance.
func TestUnboundPrefixInExceptPoisonsNameClass(t *testing.T) {
	t.Parallel()

	t.Run("default handler: anyName/except unbound prefix does not spuriously validate", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="root" xmlns="http://relaxng.org/ns/structure/1.0">
  <element>
    <anyName>
      <except><name>p:admin</name></except>
    </anyName>
    <empty/>
  </element>
</element>`

		schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err, "schema should parse")

		grammar, err := relaxng.NewCompiler().Compile(t.Context(), schemaDoc)
		require.NoError(t, err, "default Compile returns a nil hard error")
		require.NotNil(t, grammar, "default Compile still returns a grammar")

		instanceDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><child/></root>`))
		require.NoError(t, err, "instance should parse")

		require.Error(t, relaxng.NewValidator(grammar).Validate(t.Context(), instanceDoc),
			"an unbound-prefix name inside <except> must poison anyName, not be an empty exclusion")
	})

	t.Run("default handler: nsName/except unbound prefix does not spuriously validate", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="root"
    xmlns="http://relaxng.org/ns/structure/1.0"
    ns="urn:example:e">
  <element>
    <nsName>
      <except><name>p:admin</name></except>
    </nsName>
    <empty/>
  </element>
</element>`

		schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err, "schema should parse")

		grammar, err := relaxng.NewCompiler().Compile(t.Context(), schemaDoc)
		require.NoError(t, err, "default Compile returns a nil hard error")
		require.NotNil(t, grammar, "default Compile still returns a grammar")

		instanceDoc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root xmlns="urn:example:e"><child/></root>`))
		require.NoError(t, err, "instance should parse")

		require.Error(t, relaxng.NewValidator(grammar).Validate(t.Context(), instanceDoc),
			"an unbound-prefix name inside <except> must poison nsName, not be an empty exclusion")
	})

	t.Run("anyName/except unbound prefix is a fatal compile error", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="root" xmlns="http://relaxng.org/ns/structure/1.0">
  <element>
    <anyName>
      <except><name>p:admin</name></except>
    </anyName>
    <empty/>
  </element>
</element>`
		require.NotEmpty(t, compileErrorsFor(t, schema),
			"an unbound prefix inside <except> must be a fatal compile error")
	})

	t.Run("bound prefix in anyName/except compiles and excludes correctly", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="root"
    xmlns="http://relaxng.org/ns/structure/1.0"
    xmlns:p="urn:example:p">
  <element>
    <anyName>
      <except><name>p:admin</name></except>
    </anyName>
    <empty/>
  </element>
</element>`
		require.Empty(t, compileErrorsFor(t, schema),
			"a bound prefix inside <except> must compile cleanly")
		require.NoError(t, validateWith(t, schema, `<root><child/></root>`),
			"a no-namespace child is not the excluded p:admin and must match anyName")
	})
}

// TestUnboundPrefixInChoiceNameClassPoisonsChoice covers the direct <choice>
// name-class case: a <choice> whose branch is a <name> with an unbound prefix
// compiles that branch to a never-matching name class. On the DEFAULT compile
// path (no error collector) the fatal diagnostic is dropped, so the choice must
// inherit the taint rather than silently validate via its remaining branch —
// otherwise an unbound-prefix branch would be masked by a valid sibling branch.
func TestUnboundPrefixInChoiceNameClassPoisonsChoice(t *testing.T) {
	t.Parallel()

	t.Run("default handler: unbound-prefix choice branch does not spuriously validate", func(t *testing.T) {
		t.Parallel()
		// <choice> of <name>a</name> and <name>p:b</name>; p is unbound. The
		// unbound prefix is fatal and must taint the whole choice, so even a
		// no-namespace <a/> (which the first branch would match) is rejected.
		schema := `<element name="root" xmlns="http://relaxng.org/ns/structure/1.0">
  <element>
    <choice>
      <name>a</name>
      <name>p:b</name>
    </choice>
    <empty/>
  </element>
</element>`

		schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err, "schema should parse")

		grammar, err := relaxng.NewCompiler().Compile(t.Context(), schemaDoc)
		require.NoError(t, err, "default Compile returns a nil hard error")
		require.NotNil(t, grammar, "default Compile still returns a grammar")

		instanceDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a/></root>`))
		require.NoError(t, err, "instance should parse")

		require.Error(t, relaxng.NewValidator(grammar).Validate(t.Context(), instanceDoc),
			"an unbound-prefix branch must poison the whole choice, not be masked by a valid sibling")
	})

	t.Run("unbound-prefix choice branch is a fatal compile error", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="root" xmlns="http://relaxng.org/ns/structure/1.0">
  <element>
    <choice>
      <name>a</name>
      <name>p:b</name>
    </choice>
    <empty/>
  </element>
</element>`
		require.NotEmpty(t, compileErrorsFor(t, schema),
			"an unbound prefix in a <choice> name-class branch must be a fatal compile error")
	})

	t.Run("all-bound choice branches compile and validate", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="root"
    xmlns="http://relaxng.org/ns/structure/1.0"
    xmlns:p="urn:example:p">
  <element>
    <choice>
      <name>a</name>
      <name>p:b</name>
    </choice>
    <empty/>
  </element>
</element>`
		require.Empty(t, compileErrorsFor(t, schema),
			"a choice with only bound-prefix branches must compile cleanly")
		require.NoError(t, validateWith(t, schema, `<root><a/></root>`),
			"the no-namespace <a/> matches the first choice branch")
	})
}

// TestWhitespaceOnlyNameIsCompileError covers the presence-aware name lookup: a
// name attribute whose value is XML whitespace only trims to "" but is still
// PRESENT. It must be treated as an invalid (empty) QName — a fatal compile
// error installing a never-matching name class — rather than as an ABSENT name,
// which would leave no name class and make <attribute>/<element> match anything.
func TestWhitespaceOnlyNameIsCompileError(t *testing.T) {
	t.Parallel()

	t.Run("attribute name whitespace-only is a fatal compile error", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <attribute name=" "/>
</element>`
		require.NotEmpty(t, compileErrorsFor(t, schema),
			"a present-but-empty attribute name must be a fatal compile error")
	})

	t.Run("element name whitespace-only is a fatal compile error", func(t *testing.T) {
		t.Parallel()
		schema := `<element name=" " xmlns="http://relaxng.org/ns/structure/1.0">
  <empty/>
</element>`
		require.NotEmpty(t, compileErrorsFor(t, schema),
			"a present-but-empty element name must be a fatal compile error")
	})

	t.Run("default handler: whitespace-only attribute name does not match anything", func(t *testing.T) {
		t.Parallel()
		// On the DEFAULT compile path (no error collector) the fatal diagnostic
		// is dropped, so the whitespace-only name must still install a
		// never-matching name class. Otherwise <attribute name=" "> would have no
		// name class and accept ANY attribute, spuriously validating <a x=""/>.
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <attribute name=" "/>
</element>`

		schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err, "schema should parse")

		grammar, err := relaxng.NewCompiler().Compile(t.Context(), schemaDoc)
		require.NoError(t, err, "default Compile returns a nil hard error")
		require.NotNil(t, grammar, "default Compile still returns a grammar")

		instanceDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<a x=""/>`))
		require.NoError(t, err, "instance should parse")

		require.Error(t, relaxng.NewValidator(grammar).Validate(t.Context(), instanceDoc),
			"a whitespace-only attribute name must not accept an arbitrary attribute")
	})
}

// TestNBSPNotXMLWhitespace covers D-RNG-003: XML whitespace is only #x20, #x9,
// #xA, #xD. A U+00A0 NBSP must NOT be treated as ignorable whitespace, so an
// NBSP between element children, or an NBSP value for an <empty/> pattern, is
// significant content and must make the instance invalid.
func TestNBSPNotXMLWhitespace(t *testing.T) {
	t.Parallel()

	const nbsp = " "

	t.Run("empty pattern rejects NBSP content", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <empty/>
</element>`

		err := validateWith(t, schema, `<a></a>`)
		require.NoError(t, err, "truly empty element matches <empty/>")

		err = validateWith(t, schema, `<a> </a>`)
		require.NoError(t, err, "XML-whitespace-only content matches <empty/>")

		err = validateWith(t, schema, "<a>"+nbsp+"</a>")
		require.Error(t, err, "NBSP is significant content and must not match <empty/>")
	})

	t.Run("NBSP between element children is significant", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="root" xmlns="http://relaxng.org/ns/structure/1.0">
  <element name="a"><empty/></element>
  <element name="b"><empty/></element>
</element>`

		err := validateWith(t, schema, "<root><a/> <b/></root>")
		require.NoError(t, err, "XML whitespace between children is ignorable")

		err = validateWith(t, schema, "<root><a/>"+nbsp+"<b/></root>")
		require.Error(t, err, "NBSP between children is significant text, not ignorable whitespace")
	})

	t.Run("empty attribute pattern rejects NBSP value", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <attribute name="x"><empty/></attribute>
</element>`

		err := validateWith(t, schema, `<a x=""/>`)
		require.NoError(t, err, "empty attribute value matches <empty/>")

		err = validateWith(t, schema, "<a x=\""+nbsp+"\"/>")
		require.Error(t, err, "NBSP attribute value is significant and must not match <empty/>")
	})
}

// TestNameContentXMLWhitespaceTrim covers D-RNG-002 follow-up: <name> content
// is QName-parsed only AFTER leading/trailing XML whitespace is removed (spec
// §4.2). A bound prefix surrounded by ordinary spaces must therefore compile,
// while an NBSP is significant and must keep the prefix unbound (rejected).
func TestNameContentXMLWhitespaceTrim(t *testing.T) {
	t.Parallel()

	const nbsp = " "

	t.Run("bound prefix with surrounding spaces compiles", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a"
    xmlns="http://relaxng.org/ns/structure/1.0"
    xmlns:p="urn:example:p">
  <element>
    <name> p:admin </name>
    <empty/>
  </element>
</element>`
		require.Empty(t, compileErrorsFor(t, schema),
			"surrounding XML whitespace must be trimmed before QName parsing")
	})

	t.Run("NBSP before prefix keeps prefix unbound", func(t *testing.T) {
		t.Parallel()
		// A leading NBSP is not XML whitespace, so it stays part of the QName.
		// The token "<NBSP>p" is then an unbound prefix and must be rejected.
		schema := `<element name="a"
    xmlns="http://relaxng.org/ns/structure/1.0"
    xmlns:p="urn:example:p">
  <element>
    <name>` + nbsp + `p:admin</name>
    <empty/>
  </element>
</element>`
		require.NotEmpty(t, compileErrorsFor(t, schema),
			"NBSP is significant, so the prefix '<NBSP>p' is unbound and must be a fatal compile error")
	})
}

// TestSchemaAttrNBSPNotTrimmed covers the schema-attribute side of the XML
// whitespace rule: name/type/combine attribute values are trimmed of XML
// whitespace only, so a leading/trailing NBSP is significant and must NOT be
// silently stripped (which would otherwise turn an invalid value into a valid
// one).
func TestSchemaAttrNBSPNotTrimmed(t *testing.T) {
	t.Parallel()

	const nbsp = " "

	t.Run("leading XML space on name is trimmed", func(t *testing.T) {
		t.Parallel()
		// A leading ordinary space on element name is XML whitespace and must
		// be trimmed, leaving a valid NCName so the schema compiles and matches.
		schema := `<element name="root" xmlns="http://relaxng.org/ns/structure/1.0">
  <element name=" a"><empty/></element>
</element>`
		require.Empty(t, compileErrorsFor(t, schema),
			"leading XML whitespace on name must be trimmed")
		require.NoError(t, validateWith(t, schema, `<root><a/></root>`),
			"trimmed name 'a' must match a no-namespace <a/>")
	})

	t.Run("leading NBSP on name is an invalid NCName", func(t *testing.T) {
		t.Parallel()
		// A leading NBSP is not XML whitespace, so after XML-space trimming the
		// element name is still "<NBSP>a", which is not a valid NCName. The
		// schema must therefore fail to compile.
		schema := `<element name="root" xmlns="http://relaxng.org/ns/structure/1.0">
  <element name="` + nbsp + `a"><empty/></element>
</element>`
		require.NotEmpty(t, compileErrorsFor(t, schema),
			"a leading NBSP makes the name an invalid NCName, which must be a fatal compile error")
	})

	t.Run("trailing NBSP on datatype name is significant", func(t *testing.T) {
		t.Parallel()
		// "integer " (trailing NBSP) is not a known XSD datatype after
		// XML-whitespace trimming, so the data type must not be recognized as
		// xs:integer and must reject a valid integer value.
		schema := `<element name="root"
    xmlns="http://relaxng.org/ns/structure/1.0"
    datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <element name="a"><data type="integer` + nbsp + `"/></element>
</element>`
		require.Error(t, validateWith(t, schema, `<root><a>42</a></root>`),
			"a trailing NBSP on the type name must not be trimmed, so 'integer ' is not xs:integer")
	})
}

// TestPrefixedNameOverridesNSAttr covers RELAX NG §4.10: resolving a prefixed
// <name> QName replaces any existing ns attribute (inherited or explicit) with
// the prefix's namespace. A <name ns="urn:wrong">p:admin</name> with a bound
// prefix p must therefore match an element in p's namespace, not in urn:wrong.
func TestPrefixedNameOverridesNSAttr(t *testing.T) {
	t.Parallel()

	schema := `<element name="root"
    xmlns="http://relaxng.org/ns/structure/1.0"
    xmlns:p="urn:p">
  <element>
    <name ns="urn:wrong">p:admin</name>
    <empty/>
  </element>
</element>`

	require.Empty(t, compileErrorsFor(t, schema),
		"a prefixed <name> with a bound prefix must compile cleanly")

	require.NoError(t,
		validateWith(t, schema, `<root><admin xmlns="urn:p"/></root>`),
		"the resolved prefix namespace urn:p must override the ns attribute")

	require.Error(t,
		validateWith(t, schema, `<root><admin xmlns="urn:wrong"/></root>`),
		"the ns attribute urn:wrong must be overridden by the prefix namespace")
}

// TestNSAttrXMLWhitespaceTrim covers the ns-attribute side of the XML
// whitespace rule for unprefixed names: the value of an ns attribute is trimmed
// of leading/trailing XML whitespace before it becomes the element namespace,
// both when declared directly on the <element name> pattern and when inherited
// from an ancestor. Otherwise ns=" urn:x " would compile with spaces in the
// namespace and fail to match a properly-namespaced instance.
func TestNSAttrXMLWhitespaceTrim(t *testing.T) {
	t.Parallel()

	t.Run("direct ns with surrounding spaces is trimmed", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="root" ns=" urn:x "
    xmlns="http://relaxng.org/ns/structure/1.0">
  <empty/>
</element>`
		require.Empty(t, compileErrorsFor(t, schema),
			"surrounding XML whitespace on a direct ns attribute must be trimmed")
		require.NoError(t,
			validateWith(t, schema, `<root xmlns="urn:x"/>`),
			"the trimmed namespace urn:x must match a urn:x instance")
	})

	t.Run("inherited ns with surrounding spaces is trimmed", func(t *testing.T) {
		t.Parallel()
		// The ns attribute lives on the ancestor <grammar>/<element> and is
		// inherited via getInheritedNS for the unprefixed inner <element name>.
		schema := `<element name="outer" ns=" urn:x "
    xmlns="http://relaxng.org/ns/structure/1.0">
  <element name="inner">
    <empty/>
  </element>
</element>`
		require.Empty(t, compileErrorsFor(t, schema),
			"surrounding XML whitespace on an inherited ns attribute must be trimmed")
		require.NoError(t,
			validateWith(t, schema, `<outer xmlns="urn:x"><inner/></outer>`),
			"the trimmed inherited namespace urn:x must match a urn:x instance")
	})
}
