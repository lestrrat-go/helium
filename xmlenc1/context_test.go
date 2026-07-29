package xmlenc1_test

import (
	"context"
	"strings"
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

// TestCancellationVerdictIsTheSameFromEveryTerminal drives one cancellation
// scenario through every terminal it applies to and asserts a single verdict
// from all of them. The rule the package implements is uniform — a live
// cancellation wins over a stage's own error — so the encrypt terminals must
// not differ from each other, and Decrypt must not differ from DecryptBytes.
// Two of this PR's confirmed findings were exactly that divergence: at the same
// cancellation timing DecryptBytes reported the cancellation while Decrypt
// reported ErrDecryptionFailed.
func TestCancellationVerdictIsTheSameFromEveryTerminal(t *testing.T) {
	// A 16-byte session key against the default AES-256-GCM block algorithm
	// fails validateKeySize, an encrypt stage that runs to completion before
	// its error can be seen.
	shortKeyEncryptor := xmlenc1.NewEncryptor().SessionKey(randKey(t, 16))

	// One EncryptedData whose single EncryptedKey unwraps cleanly under the
	// configured KEK but whose payload ciphertext is corrupt, so block
	// decryption is the stage that fails. Decrypt and DecryptBytes read the
	// SAME element, which is what makes their verdicts comparable.
	kek := randKey(t, 32)
	sessionKey := randKey(t, 32)
	corruptCipher, err := xmlenc1.EncryptBytesForTest(xmlenc1.AES256GCM, sessionKey, []byte(`<x>secret</x>`))
	require.NoError(t, err)
	corruptCipher[len(corruptCipher)-1] ^= 0xff
	wrapped, err := xmlenc1.AESKeyWrapForTest(kek, sessionKey)
	require.NoError(t, err)

	corruptDoc := mustParseXML(t, `<root/>`)
	corruptElem, err := xmlenc1.MarshalEncryptedDataForTest(corruptDoc, &xmlenc1.EncryptedData{
		EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
		EncryptedKeys: []*xmlenc1.EncryptedKey{{
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256KeyWrap},
			CipherValue:      wrapped,
		}},
		CipherValue: corruptCipher,
	})
	require.NoError(t, err)
	require.NoError(t, corruptDoc.DocumentElement().AddChild(corruptElem))
	corruptDecryptor := xmlenc1.NewDecryptor().KeyEncryptionKey(kek)

	// Each terminal is reduced to (context, document) → error. The document is
	// a parameter rather than something the terminal parses for itself so that
	// no ctx-carrying function here builds one against a different context.
	encryptElement := func(ctx context.Context, doc *helium.Document) error {
		_, err := shortKeyEncryptor.EncryptElement(ctx, doc.DocumentElement())
		return err
	}
	encryptContent := func(ctx context.Context, doc *helium.Document) error {
		_, err := shortKeyEncryptor.EncryptContent(ctx, doc.DocumentElement())
		return err
	}
	encryptBytes := func(ctx context.Context, doc *helium.Document) error {
		_, err := shortKeyEncryptor.EncryptBytes(ctx, doc, []byte("payload"))
		return err
	}
	decrypt := func(ctx context.Context, _ *helium.Document) error {
		_, err := corruptDecryptor.Decrypt(ctx, corruptElem)
		return err
	}
	decryptBytes := func(ctx context.Context, _ *helium.Document) error {
		_, err := corruptDecryptor.DecryptBytes(ctx, corruptElem)
		return err
	}

	type terminalCase struct {
		name string
		// livePolls is how many context polls the terminal makes before it
		// enters the stage this scenario cancels inside. The context stays
		// live for exactly that many, so the cancellation first becomes
		// visible on the poll that decides what the failed stage reports.
		// A negative value cancels the context before the call.
		livePolls int
		// doc is the tree an encrypt terminal works on, one per row so a
		// terminal that did mutate it cannot affect another row. The decrypt
		// terminals share corruptElem and ignore it.
		doc *helium.Document
		run func(ctx context.Context, doc *helium.Document) error
	}
	scenarios := []struct {
		name      string
		terminals []terminalCase
	}{
		{
			// Every terminal polls on entry, so none of them starts work.
			name: "a context cancelled before the call",
			terminals: []terminalCase{
				{name: "EncryptElement", livePolls: -1, doc: mustParseXML(t, `<root><child/></root>`), run: encryptElement},
				{name: "EncryptContent", livePolls: -1, doc: mustParseXML(t, `<root><child/></root>`), run: encryptContent},
				{name: "EncryptBytes", livePolls: -1, doc: mustParseXML(t, `<root/>`), run: encryptBytes},
				{name: "Decrypt", livePolls: -1, run: decrypt},
				{name: "DecryptBytes", livePolls: -1, run: decryptBytes},
			},
		},
		{
			// Session-key validation runs to completion, so its KeySizeError
			// must not be what a cancelled caller is told. The poll budgets
			// differ only because each terminal reaches encryptPlaintext by a
			// different route: EncryptBytes polls on entry and again inside
			// encryptPlaintext; EncryptElement adds a post-serialization poll;
			// EncryptContent adds one more for its single child, because
			// serializeChildren observes the context per child.
			name: "a cancellation landing while the session key is validated",
			terminals: []terminalCase{
				{name: "EncryptElement", livePolls: 3, doc: mustParseXML(t, `<root><child/></root>`), run: encryptElement},
				{name: "EncryptContent", livePolls: 4, doc: mustParseXML(t, `<root><child/></root>`), run: encryptContent},
				{name: "EncryptBytes", livePolls: 2, doc: mustParseXML(t, `<root/>`), run: encryptBytes},
			},
		},
		{
			// Block decryption runs to completion too. Both terminals poll on
			// entry, after the parse, and at the top of the candidate loop;
			// the fourth poll is DecryptBytes' post-resolution poll and
			// finishDecrypt's entry poll respectively, after which each enters
			// decryptCipherValue.
			name: "a cancellation landing while block decryption runs",
			terminals: []terminalCase{
				{name: "Decrypt", livePolls: 4, run: decrypt},
				{name: "DecryptBytes", livePolls: 4, run: decryptBytes},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			for _, terminal := range scenario.terminals {
				t.Run(terminal.name, func(t *testing.T) {
					ctx := context.Context(newCancelAfterErrCalls(terminal.livePolls))
					if terminal.livePolls < 0 {
						cancelled, cancel := context.WithCancel(t.Context())
						cancel()
						ctx = cancelled
					}
					require.ErrorIs(t, terminal.run(ctx, terminal.doc), context.Canceled)
				})
			}
		})
	}
}

// TestDecryptObservesContextPerContentNode pins the interior of the walk that
// collects a TypeContent payload's nodes. The node count is chosen by whoever
// wrote the ciphertext, so a walk that observed the context only at its ends
// would run to the end of an attacker-sized input after the caller cancelled
// and hand back every node with a nil error.
func TestDecryptObservesContextPerContentNode(t *testing.T) {
	const children = 512

	sessionKey := randKey(t, 32)
	doc := mustParseXML(t, `<root/>`)
	plaintext := []byte(strings.Repeat(`<a/>`, children))
	cipher, err := xmlenc1.EncryptBytesForTest(xmlenc1.AES256GCM, sessionKey, plaintext)
	require.NoError(t, err)
	elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, &xmlenc1.EncryptedData{
		Type:             xmlenc1.TypeContent,
		EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
		CipherValue:      cipher,
	})
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(elem))
	decryptor := xmlenc1.NewDecryptor().SessionKey(sessionKey)

	total := newPollCounter()
	nodes, err := decryptor.Decrypt(total, elem)
	require.NoError(t, err)
	require.Len(t, nodes, children)

	// The inner plaintext parse polls the context too, and its share grows
	// with the payload. Measure it against the same parser, context node, and
	// plaintext the decrypt used, so what remains is the decrypt's own
	// polling rather than a number copied from the parser's internals.
	parse := newPollCounter()
	_, err = xmlenc1.HardenedParserForTest().ParseInNodeContext(parse, elem.Parent(), plaintext)
	require.NoError(t, err)
	require.GreaterOrEqual(t, total.calls()-parse.calls(), children,
		"the TypeContent node walk must observe the context once per collected node")

	// Cancel one poll into that walk. Everything before it has already
	// succeeded — the payload is decrypted and the plaintext is parsed — so
	// only the walk itself can report the cancellation.
	ctx := newCancelAfterErrCalls(total.calls() - children)
	nodes, err = decryptor.Decrypt(ctx, elem)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, nodes)
}

// TestEncryptContentObservesContextPerChild pins the interior of the walk that
// serializes the caller's subtree. The child count is the caller's, so the
// walk's polling must grow with it rather than being fixed at its ends.
func TestEncryptContentObservesContextPerChild(t *testing.T) {
	pollsFor := func(t *testing.T, children int) int {
		t.Helper()
		doc := mustParseXML(t, `<root>`+strings.Repeat(`<a/>`, children)+`</root>`)
		counter := newPollCounter()
		_, err := xmlenc1.NewEncryptor().SessionKey(randKey(t, 32)).EncryptContent(counter, doc.DocumentElement())
		require.NoError(t, err)
		return counter.calls()
	}

	const (
		few  = 4
		many = 260
	)
	require.GreaterOrEqual(t, pollsFor(t, many)-pollsFor(t, few), many-few,
		"serializeChildren must observe the context once per child")
}

// pollCounter is a live context that counts how often its error is consulted.
// It implements context.Context directly (no embedding) to satisfy the
// containedctx linter, and needs no parent: it never cancels and the code under
// test reads no context values.
type pollCounter struct {
	mu sync.Mutex
	n  int
}

func newPollCounter() *pollCounter { return &pollCounter{} }

func (c *pollCounter) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *pollCounter) Done() <-chan struct{}       { return nil }
func (c *pollCounter) Value(any) any               { return nil }

func (c *pollCounter) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
	return nil
}

func (c *pollCounter) calls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
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
