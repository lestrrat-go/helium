package xmlenc1_test

import (
	"context"
	"sync"
	"testing"
	"time"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

func TestEncryptorHonorsCancellation(t *testing.T) {
	newCancelledContext := func(t *testing.T) context.Context {
		t.Helper()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		return ctx
	}

	newEncryptor := func(t *testing.T) xmlenc1.Encryptor {
		t.Helper()
		return xmlenc1.NewEncryptor().SessionKey(randKey(t, 32))
	}

	t.Run("EncryptElement leaves the document unchanged", func(t *testing.T) {
		doc := mustParseXML(t, `<root><child/></root>`)
		before, err := helium.WriteString(doc)
		require.NoError(t, err)

		_, err = newEncryptor(t).EncryptElement(newCancelledContext(t), doc.DocumentElement())
		require.ErrorIs(t, err, context.Canceled)

		after, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Equal(t, before, after)
	})

	t.Run("EncryptContent leaves the document unchanged", func(t *testing.T) {
		doc := mustParseXML(t, `<root><child/></root>`)
		before, err := helium.WriteString(doc)
		require.NoError(t, err)

		_, err = newEncryptor(t).EncryptContent(newCancelledContext(t), doc.DocumentElement())
		require.ErrorIs(t, err, context.Canceled)

		after, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Equal(t, before, after)
	})

	t.Run("EncryptBytes returns no ciphertext", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		_, err := newEncryptor(t).EncryptBytes(newCancelledContext(t), doc, []byte("payload"))
		require.ErrorIs(t, err, context.Canceled)
	})
}

func TestDecryptorHonorsCancellationWithSessionKey(t *testing.T) {
	sessionKey := randKey(t, 32)

	t.Run("Decrypt returns no nodes", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		edElem, err := xmlenc1.NewEncryptor().SessionKey(sessionKey).EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err = xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(ctx, edElem)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("DecryptBytes returns no plaintext", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		edElem, err := xmlenc1.NewEncryptor().SessionKey(sessionKey).EncryptBytes(t.Context(), doc, []byte("payload"))
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err = xmlenc1.NewDecryptor().SessionKey(sessionKey).DecryptBytes(ctx, edElem)
		require.ErrorIs(t, err, context.Canceled)
	})
}

func TestDecryptReturnsPlaintextParserCancellation(t *testing.T) {
	sessionKey := randKey(t, 32)
	doc := mustParseXML(t, `<root/>`)
	edElem, err := xmlenc1.NewEncryptor().SessionKey(sessionKey).EncryptElement(t.Context(), doc.DocumentElement())
	require.NoError(t, err)

	ctx := newCancelAfterErrCalls(4)
	_, err = xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(ctx, edElem)
	require.ErrorIs(t, err, context.Canceled)
}

// TestDecryptorCancellationWinsOverCandidateError pins the rule that a
// cancellation landing while a candidate's key resolution runs is reported as
// the cancellation, not as the candidate's own failure. Key resolution and
// block decryption cannot be interrupted once entered, so each candidate stage
// is followed by a context poll; without it the stage's error is retained in
// lastErr and surfaces as ErrDecryptionFailed after the loop.
func TestDecryptorCancellationWinsOverCandidateError(t *testing.T) {
	newEncryptedData := func(t *testing.T, edType string, key *xmlenc1.EncryptedKey, cipher []byte) *helium.Element {
		t.Helper()
		doc := mustParseXML(t, `<root/>`)
		elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, &xmlenc1.EncryptedData{
			Type:             edType,
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
			EncryptedKeys:    []*xmlenc1.EncryptedKey{key},
			CipherValue:      cipher,
		})
		require.NoError(t, err)
		return elem
	}

	wrap := func(t *testing.T, kek, sessionKey []byte) *xmlenc1.EncryptedKey {
		t.Helper()
		wrapped, err := xmlenc1.AESKeyWrapForTest(kek, sessionKey)
		require.NoError(t, err)
		return &xmlenc1.EncryptedKey{
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256KeyWrap},
			CipherValue:      wrapped,
		}
	}

	// Three polls run before the candidate loop reaches key resolution
	// (terminal entry, post-parse, loop top), so cancelling after the third
	// puts the cancellation inside the resolution stage.
	const cancelDuringKeyResolution = 3

	t.Run("DecryptBytes after a successful key resolution", func(t *testing.T) {
		kek := randKey(t, 32)
		sessionKey := randKey(t, 32)
		cipher, err := xmlenc1.EncryptBytesForTest(xmlenc1.AES256GCM, sessionKey, []byte("payload"))
		require.NoError(t, err)
		// Corrupt the payload ciphertext so block decryption fails: the key
		// unwraps cleanly, so only the poll after resolution can report the
		// cancellation.
		cipher[len(cipher)-1] ^= 0xff

		elem := newEncryptedData(t, "", wrap(t, kek, sessionKey), cipher)

		ctx := newCancelAfterErrCalls(cancelDuringKeyResolution)
		_, err = xmlenc1.NewDecryptor().KeyEncryptionKey(kek).DecryptBytes(ctx, elem)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("Decrypt after a failed key resolution", func(t *testing.T) {
		kek := randKey(t, 32)
		wrongKEK := randKey(t, 32)
		sessionKey := randKey(t, 32)
		cipher, err := xmlenc1.EncryptBytesForTest(xmlenc1.AES256GCM, sessionKey, []byte(`<x>secret</x>`))
		require.NoError(t, err)

		// The KEK does not match, so the candidate's AES key unwrap fails its
		// integrity check and the loop would otherwise retain that error.
		elem := newEncryptedData(t, xmlenc1.TypeElement, wrap(t, kek, sessionKey), cipher)

		ctx := newCancelAfterErrCalls(cancelDuringKeyResolution)
		_, err = xmlenc1.NewDecryptor().KeyEncryptionKey(wrongKEK).Decrypt(ctx, elem)
		require.ErrorIs(t, err, context.Canceled)
	})
}

// cancelAfterErrCalls reports a cancelled error from Err() only after Err has
// been consulted cancelAfter times, simulating a context that is live when
// Decrypt is entered and becomes cancelled partway through decryption — while
// finishDecrypt, and the ParseInNodeContext call inside it, are running. It
// implements context.Context directly (no embedding) to satisfy the
// containedctx linter; it needs no parent context, because the test drives the
// cancellation itself and decryption reads no context values. Closing done on
// cancellation keeps Done() consistent with Err() for any derived context that
// waits on it, and the mutex keeps that transition safe for the concurrent use
// a context.Context must support.
type cancelAfterErrCalls struct {
	mu          sync.Mutex
	done        chan struct{}
	cancelAfter int
	calls       int
	err         error
}

func newCancelAfterErrCalls(cancelAfter int) *cancelAfterErrCalls {
	return &cancelAfterErrCalls{done: make(chan struct{}), cancelAfter: cancelAfter}
}

func (c *cancelAfterErrCalls) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *cancelAfterErrCalls) Done() <-chan struct{}       { return c.done }
func (c *cancelAfterErrCalls) Value(any) any               { return nil }

func (c *cancelAfterErrCalls) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	c.calls++
	if c.calls <= c.cancelAfter {
		return nil
	}
	c.err = context.Canceled
	close(c.done)
	return c.err
}
