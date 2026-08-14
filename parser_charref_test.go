package helium_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
)

// foreignEntity is a sax.Entity implementation that is NOT *helium.Entity.
// A custom SAX handler returning such a value must produce an error rather
// than panic from a forced type assertion.
type foreignEntity struct {
	name    string
	content []byte
}

func (e *foreignEntity) Name() string                { return e.name }
func (e *foreignEntity) SetOrig(string)              {}
func (e *foreignEntity) EntityType() enum.EntityType { return enum.InternalGeneralEntity }
func (e *foreignEntity) Content() []byte             { return e.content }
func (e *foreignEntity) Checked() bool               { return true }
func (e *foreignEntity) MarkChecked()                {}

func TestCharRef(t *testing.T) {
	t.Parallel()

	// parses documents with valid hex/decimal char refs to
	// drive the success branches of parseCharRef.
	t.Run("valid forms", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root>&#65;&#x42;&#x4A;</root>`))
		require.NoError(t, err)
		root := doc.DocumentElement()
		require.NotNil(t, root)
		require.Equal(t, "ABJ", string(root.Content()))
	})

	t.Run("a valid reference is accepted", func(t *testing.T) {
		for _, tc := range []struct {
			input string
			want  string
		}{
			{`<root>&#65;</root>`, "A"},
			{`<root>&#x41;</root>`, "A"},
		} {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.input))
			require.NoError(t, err, "valid char ref must parse: %s", tc.input)
			require.NotNil(t, doc)

			root := doc.DocumentElement()
			require.NotNil(t, root)
			require.Equal(t, tc.want, string(root.Content()))
		}
	})

	t.Run("missing digits are rejected", func(t *testing.T) {
		for _, input := range []string{
			`<root>&#;</root>`,
			`<root>&#x;</root>`,
		} {
			_, err := helium.NewParser().Parse(t.Context(), []byte(input))
			require.Error(t, err, "char ref without digits must be rejected: %s", input)
		}
	})

	t.Run("a missing semicolon is rejected", func(t *testing.T) {
		for _, input := range []string{
			`<root>&#65</root>`,
			`<root>&#x41</root>`,
		} {
			_, err := helium.NewParser().Parse(t.Context(), []byte(input))
			require.Error(t, err, "char ref without terminating ';' must be rejected: %s", input)
		}
	})

	// drives parseCharRef / parseEntityRef error
	// branches with a table of malformed-reference documents.
	t.Run("malformed character and entity references", func(t *testing.T) {
		bad := []struct {
			name string
			src  string
		}{
			{"hex-missing-digits", `<root>&#x;</root>`},
			{"dec-missing-digits", `<root>&#;</root>`},
			{"hex-invalid-digit", `<root>&#xZZ;</root>`},
			{"dec-invalid-digit", `<root>&#12A3;</root>`},
			{"charref-out-of-range", `<root>&#x110000;</root>`},
			{"charref-control", `<root>&#x0;</root>`},
			{"undeclared-entity-standalone", `<?xml version="1.0" standalone="yes"?><root>&undeclared;</root>`},
			{"entity-empty-name", `<root>&;</root>`},
		}

		for _, tc := range bad {
			t.Run(tc.name, func(t *testing.T) {
				_, err := helium.NewParser().Parse(t.Context(), []byte(tc.src))
				require.Error(t, err, "expected parse error for %q", tc.src)
			})
		}
	})

	// indirectly exercises resolveCharRefs by
	// round-tripping a document whose internal entity content contains char refs.
	t.Run("resolved through Parse", func(t *testing.T) {
		const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY e "A&#66;C&#x44;E">
]>
<doc>&e;</doc>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)

		out, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, out, "doc")
	})

	// against the index-out-of-range panic in
	// parseStringCharRef when malformed character references appear inside entity
	// declarations. Each malformed input must yield a structured
	// helium.ErrParseError, never a panic.
	t.Run("a malformed reference does not panic", func(t *testing.T) {
		malformed := []string{
			`<!DOCTYPE root [<!ENTITY e "&#">]><root>&e;</root>`,
			`<!DOCTYPE root [<!ENTITY e "&#x">]><root>&e;</root>`,
			`<!DOCTYPE root [<!ENTITY e "&#1">]><root>&e;</root>`,
			`<!DOCTYPE root [<!ENTITY e "&#xZ">]><root>&e;</root>`,
			`<!DOCTYPE root [<!ENTITY e "&#;">]><root>&e;</root>`,
		}

		for _, input := range malformed {
			t.Run(input, func(t *testing.T) {
				require.NotPanics(t, func() {
					_, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(input))
					require.Error(t, err, "malformed char ref must return an error")
					var pe helium.ErrParseError
					require.ErrorAs(t, err, &pe, "malformed char ref must return a helium.ErrParseError")
				})
			})
		}
	})

	// against an out-of-range character
	// reference wrapping int32 into a valid-looking rune. These exercise the
	// content-path parser (parseCharRef) and the parameter-entity declaration
	// branch, both of which must report an error and produce no
	// bogus character.
	t.Run("an out-of-range reference is rejected", func(t *testing.T) {
		cases := []string{
			// Content path: 4294967337 = 0x100000029 wraps to 0x29 (')').
			`<root>&#4294967337;</root>`,
			`<root>&#x100000029;</root>`,
			// Parameter-entity value must propagate the same error.
			`<!DOCTYPE root [<!ENTITY % p "&#4294967337;"> %p;]><root/>`,
		}

		for _, input := range cases {
			t.Run(input, func(t *testing.T) {
				require.NotPanics(t, func() {
					_, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(input))
					require.Error(t, err, "out-of-range char ref must return an error")
					var pe helium.ErrParseError
					require.ErrorAs(t, err, &pe, "out-of-range char ref must return a helium.ErrParseError")
				})
			})
		}
	})

	// against over-rejection: a valid decimal
	// character reference entity must still parse and expand correctly.
	t.Run("a valid reference still works", func(t *testing.T) {
		input := `<!DOCTYPE root [<!ENTITY e "&#65;">]><root>&e;</root>`
		doc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(input))
		require.NoError(t, err, "valid char ref must parse")
		require.NotNil(t, doc)

		root := doc.DocumentElement()
		require.NotNil(t, root)
		child := root.FirstChild()
		require.NotNil(t, child, "expanded entity must produce a text child")
		require.Equal(t, "A", string(child.Content()), "&#65; must expand to 'A'")
	})

	// numeric and hex character references.
	t.Run("character references", func(t *testing.T) {
		const src = `<root>dec=&#65; hex=&#x42; high=&#x1F600;</root>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		require.Equal(t, "dec=A hex=B high=\U0001F600", string(doc.DocumentElement().Content()))
	})

	t.Run("the replacement character", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte("<root>&#xFFFD;</root>"))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)
		require.Equal(t, "�", string(root.Content()))
	})
}

func TestCreateCharRef(t *testing.T) {
	t.Parallel()

	// the CreateCharRef name-stripping branches.
	t.Run("forms", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)

		plain, err := doc.CreateCharRef("foo")
		require.NoError(t, err)
		require.Equal(t, "foo", plain.Name())

		// "&foo;" -> "foo"
		full, err := doc.CreateCharRef("&foo;")
		require.NoError(t, err)
		require.Equal(t, "foo", full.Name())

		// "&foo" (no trailing semicolon) -> "foo"
		noSemi, err := doc.CreateCharRef("&foo")
		require.NoError(t, err)
		require.Equal(t, "foo", noSemi.Name())

		// Empty name and a name that decodes to empty are rejected.
		_, err = doc.CreateCharRef("")
		require.Error(t, err)
		_, err = doc.CreateCharRef("&;")
		require.Error(t, err)
	})

	// locks the create->serialize round-trip: an
	// EntityRefNode built by CreateCharRef with a "#NNN"/"#xHH" name serializes as
	// a character reference, and a plain name serializes as a named entity
	// reference, mirroring libxml2's xmlNewCharRef (both are XML_ENTITY_REF_NODE).
	t.Run("serializes", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			want string
		}{
			{name: "#123", want: "&#123;"},
			{name: "#xAB", want: "&#xAB;"},
			{name: "amp", want: "&amp;"},
		} {
			doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))

			ref, err := doc.CreateCharRef(tc.name)
			require.NoError(t, err)
			require.Equal(t, helium.EntityRefNode, ref.Type(), "CreateCharRef yields an EntityRefNode (libxml2 xmlNewCharRef)")
			require.Equal(t, tc.name, ref.Name())
			require.NoError(t, root.AddChild(ref))

			str, err := helium.WriteString(root)
			require.NoError(t, err)
			require.Equal(t, "<root>"+tc.want+"</root>", str)
		}
	})
}

func TestGetEntityErrors(t *testing.T) {
	t.Parallel()

	t.Run("a foreign type in an attribute value does not panic", func(t *testing.T) {
		// Exercise the nested string-decoding path (parseStringEntityRef ->
		// entityCheck), in place of the direct attribute parseEntityRef path.
		//
		// `foo` is declared in the internal subset as "&bar;" and resolves to a
		// real *helium.Entity (the handler returns nil for it). Decoding "&foo;"
		// recurses into its content, calling parseStringEntityRef("&bar;"), which
		// returns a FOREIGN (non-*helium.Entity) sax.Entity for `bar`. That foreign
		// value then reaches entityCheck, which must handle it gracefully via its
		// comma-ok assertion, triggering no forced-cast panic.
		h := sax.New()
		h.SetOnGetEntity(sax.GetEntityFunc(func(_ context.Context, name string) (sax.Entity, error) {
			if name == "bar" {
				return &foreignEntity{name: name, content: []byte("hello")}, nil
			}
			return nil, nil //nolint:nilnil
		}))

		const input = `<!DOCTYPE root [<!ENTITY foo "&bar;">]><root attr="&foo;"/>`

		require.NotPanics(t, func() {
			_, _ = helium.NewParser().SAXHandler(h).Parse(t.Context(), []byte(input))
		})
	})

	t.Run("a foreign type returns an error, not a panic", func(t *testing.T) {
		h := sax.New()
		h.SetOnGetEntity(sax.GetEntityFunc(func(_ context.Context, name string) (sax.Entity, error) {
			return &foreignEntity{name: name, content: []byte("hello")}, nil
		}))

		const input = `<!DOCTYPE root [<!ENTITY foo "ignored">]><root>&foo;</root>`

		require.NotPanics(t, func() {
			_, err := helium.NewParser().SAXHandler(h).Parse(t.Context(), []byte(input))
			require.Error(t, err, "foreign sax.Entity type should yield an error")
		})
	})

	t.Run("a nil handler error does not panic", func(t *testing.T) {
		// A handler returning (nil, err) (e.g. "entity not found") must not panic.
		// Per libxml2 semantics this falls through to undeclared-entity handling.
		h := sax.New()
		h.SetOnGetEntity(sax.GetEntityFunc(func(_ context.Context, name string) (sax.Entity, error) {
			return nil, context.Canceled
		}))

		const input = `<!DOCTYPE root [<!ENTITY foo "ignored">]><root>&foo;</root>`

		require.NotPanics(t, func() {
			_, _ = helium.NewParser().SAXHandler(h).Parse(t.Context(), []byte(input))
		})
	})
}
