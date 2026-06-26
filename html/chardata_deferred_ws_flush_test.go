package html_test

import (
	"testing"

	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// When leading whitespace is DEFERRED into pendingWS and the next non-whitespace
// chunk flushes it, the flush must not short-circuit the rest of emitCharacters.
// A SAX handler with NO Characters callback makes the deferred-whitespace flush
// return ErrHandlerUnspecified; previously that aborted emitCharacters before the
// current chunk's encoding-error check ran, so the "Invalid bytes in character
// encoding" diagnostic for the U+FFFD that triggered the flush was lost.
//
// Input shape: a charset=utf-8 declaration plus an invalid 0xFF byte tucked inside
// a COMMENT (so it sets the document-wide encoding-error flag without ever flowing
// through emitCharacters); a deferred leading space after <html> (mode <
// insertInBody); and a NUL that breaks the whitespace run so the NUL-derived U+FFFD
// arrives as the SOLE char-data chunk. That single chunk both flushes the pending
// space AND is the only place the encoding-error check can fire — if the flush
// short-circuits, the diagnostic is lost for good.
func TestDeferredWhitespaceFlush_NoCharactersHandler_StillRunsEncodingCheck(t *testing.T) {
	input := []byte("<!-- charset=utf-8 \xFF --><html> \x00</html>")

	var sawEncodingError bool
	sax := &html.SAXCallbacks{}
	// Deliberately register OTHER callbacks but NOT OnCharacters, so the deferred
	// whitespace flush returns ErrHandlerUnspecified.
	sax.SetOnStartElement(html.StartElementFunc(func(string, []html.Attribute) error {
		return nil
	}))
	sax.SetOnError(html.ErrorFunc(func(err error) error {
		if err != nil && err.Error() == "Invalid bytes in character encoding" {
			sawEncodingError = true
		}
		return nil
	}))

	err := html.NewParser().ParseWithSAX(t.Context(), input, sax)
	require.NoError(t, err, "non-strict parse must keep going past an unset Characters handler")
	require.True(t, sawEncodingError,
		"deferred-whitespace flush must not short-circuit the current chunk's encoding-error check")
}
