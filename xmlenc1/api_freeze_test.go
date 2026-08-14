package xmlenc1_test

import (
	"reflect"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// TestExportedErrorStructsRejectUnkeyedLiterals pins the guard that lets this
// package's exported error structs GROW a field without breaking any caller.
//
// A struct whose fields are all exported can be built positionally, as in
// &UnsupportedAlgorithmError{"block algorithm", uri}. Adding a field then stops
// that literal compiling, so the field set is frozen by every caller who wrote
// one. An UNEXPORTED field removes the choice: an unkeyed literal written from
// another package fails with "implicit assignment to unexported field", so
// every caller writes keyed fields and a later field costs them nothing.
//
// The compile failure itself cannot be asserted from inside a test binary, so
// this asserts the mechanism: each type carries at least one unexported field.
// It is a structural check on purpose — it holds whatever the guard field is
// named, and it fails if someone removes it.
func TestExportedErrorStructsRejectUnkeyedLiterals(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  reflect.Type
	}{
		{name: "UnsupportedAlgorithmError", typ: reflect.TypeFor[xmlenc1.UnsupportedAlgorithmError]()},
		{name: "KeySizeError", typ: reflect.TypeFor[xmlenc1.KeySizeError]()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var unexported int
			for f := range tc.typ.Fields() {
				if !f.IsExported() {
					unexported++
				}
			}
			require.Positive(t, unexported,
				"%s must carry an unexported field so an unkeyed composite literal cannot compile from another package", tc.name)
		})
	}
}

// TestExportedErrorStructsStayUsable pins that the guard costs a caller
// nothing it is entitled to: a keyed literal still compiles, the fields still
// read back, and errors.As still recovers the type through a wrapped chain.
func TestExportedErrorStructsStayUsable(t *testing.T) {
	t.Run("UnsupportedAlgorithmError", func(t *testing.T) {
		err := &xmlenc1.UnsupportedAlgorithmError{Parameter: "block algorithm", Algorithm: "urn:nope"}
		require.Equal(t, "block algorithm", err.Parameter)
		require.Equal(t, "urn:nope", err.Algorithm)
		require.Contains(t, err.Error(), "urn:nope")
	})

	t.Run("KeySizeError", func(t *testing.T) {
		err := &xmlenc1.KeySizeError{Key: "session key", Algorithm: xmlenc1.AES256GCM11, Want: 32, Got: 16}
		require.Equal(t, 32, err.Want)
		require.Equal(t, 16, err.Got)
		require.Contains(t, err.Error(), "session key")
	})

	// The types are reached through errors.As in real use, so a live decrypt
	// must still hand one back.
	t.Run("recovered from a live failure", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		_, err := xmlenc1.NewEncryptor().
			BlockAlgorithm("urn:example:not-an-algorithm").
			SessionKey(randKey(t, 32)).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.Error(t, err)

		var unsupported *xmlenc1.UnsupportedAlgorithmError
		require.ErrorAs(t, err, &unsupported)
		require.Equal(t, "urn:example:not-an-algorithm", unsupported.Algorithm)
	})
}

// TestEncryptNilOperandSentinelsAgree pins that all three encrypt terminals
// report a nil operand alike. They took different shapes once: EncryptBytes
// returned a bare helium.ErrNilNode while EncryptElement and EncryptContent
// also wrapped ErrEncryptionFailed, so a caller matching the operation sentinel
// got a different answer depending on which entry point it called.
func TestEncryptNilOperandSentinelsAgree(t *testing.T) {
	sessionKey := randKey(t, 32)

	for _, tc := range []struct {
		name string
		call func() error
	}{
		{
			name: "EncryptBytes",
			call: func() error {
				_, err := xmlenc1.NewEncryptor().SessionKey(sessionKey).
					EncryptBytes(t.Context(), nil, []byte("payload"))
				return err
			},
		},
		{
			name: "EncryptElement",
			call: func() error {
				_, err := xmlenc1.NewEncryptor().SessionKey(sessionKey).
					EncryptElement(t.Context(), nil)
				return err
			},
		},
		{
			name: "EncryptContent",
			call: func() error {
				_, err := xmlenc1.NewEncryptor().SessionKey(sessionKey).
					EncryptContent(t.Context(), nil)
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			require.Error(t, err)
			require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
			require.ErrorIs(t, err, helium.ErrNilNode)
		})
	}
}
