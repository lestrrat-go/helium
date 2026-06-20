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

func TestGetEntityForeignTypeReturnsErrorNotPanic(t *testing.T) {
	h := sax.New()
	h.SetOnGetEntity(sax.GetEntityFunc(func(_ context.Context, name string) (sax.Entity, error) {
		return &foreignEntity{name: name, content: []byte("hello")}, nil
	}))

	const input = `<!DOCTYPE root [<!ENTITY foo "ignored">]><root>&foo;</root>`

	require.NotPanics(t, func() {
		_, err := helium.NewParser().SAXHandler(h).Parse(t.Context(), []byte(input))
		require.Error(t, err, "foreign sax.Entity type should yield an error")
	})
}

func TestGetEntityHandlerNilErrorDoesNotPanic(t *testing.T) {
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
}

func TestCharRefMissingSemicolonRejected(t *testing.T) {
	for _, input := range []string{
		`<root>&#65</root>`,
		`<root>&#x41</root>`,
	} {
		_, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.Error(t, err, "char ref without terminating ';' must be rejected: %s", input)
	}
}

func TestCharRefMissingDigitsRejected(t *testing.T) {
	for _, input := range []string{
		`<root>&#;</root>`,
		`<root>&#x;</root>`,
	} {
		_, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.Error(t, err, "char ref without digits must be rejected: %s", input)
	}
}

func TestCharRefValidAccepted(t *testing.T) {
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
}
