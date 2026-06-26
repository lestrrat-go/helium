package xmlenc1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

// encryptConfig holds the configuration for an Encryptor.
type encryptConfig struct {
	blockAlgorithm   string
	keyTransport     string
	recipientPubKey  *rsa.PublicKey
	sessionKey       []byte
	oaepDigest       string
	oaepMGF          string
	oaepParams       []byte
	keyWrapAlgorithm string
	keyEncryptionKey []byte
	allowLegacyCBC   bool
}

// DefaultBlockAlgorithm is the block encryption algorithm an Encryptor
// uses when no BlockAlgorithm is set. It is authenticated AES-256-GCM.
const DefaultBlockAlgorithm = AES256GCM

// Encryptor encrypts XML elements or content. It uses clone-on-write
// semantics: each builder method returns a new Encryptor and the original
// is never mutated.
type Encryptor struct {
	cfg *encryptConfig
}

// NewEncryptor creates a new Encryptor with default settings.
func NewEncryptor() Encryptor {
	return Encryptor{cfg: &encryptConfig{}}
}

func (e Encryptor) clone() Encryptor {
	if e.cfg == nil {
		return Encryptor{cfg: &encryptConfig{}}
	}
	cp := *e.cfg
	return Encryptor{cfg: &cp}
}

// BlockAlgorithm sets the block encryption algorithm URI. If never set,
// the Encryptor defaults to [DefaultBlockAlgorithm] (authenticated
// AES-256-GCM).
//
// Selecting an AES-CBC algorithm (AES128CBC / AES256CBC) additionally
// requires AllowLegacyCBC(true): CBC under XML Encryption 1.0 is
// unauthenticated and padding-oracle-prone, so emitting new CBC
// ciphertext is gated behind an explicit opt-in. Without it, encryption
// returns [ErrCBCEncryptionRequiresOptIn].
func (e Encryptor) BlockAlgorithm(uri string) Encryptor {
	e = e.clone()
	e.cfg.blockAlgorithm = uri
	return e
}

// AllowLegacyCBC opts the Encryptor in to emitting unauthenticated
// AES-CBC ciphertext when a CBC BlockAlgorithm is selected.
//
// The Encryptor defaults to authenticated AES-GCM. AES-CBC under XML
// Encryption 1.0 is unauthenticated and vulnerable to padding-oracle
// attacks (Jager/Somorovsky 2011); XML Encryption 1.1 deprecated it in
// favor of AES-GCM. Set this to true only when you must produce
// ciphertext for a legacy recipient that cannot accept AES-GCM. This
// does not affect decryption (see Decryptor.AllowUnauthenticatedCBC).
func (e Encryptor) AllowLegacyCBC(v bool) Encryptor {
	e = e.clone()
	e.cfg.allowLegacyCBC = v
	return e
}

// KeyTransportAlgorithm sets the key transport algorithm URI.
func (e Encryptor) KeyTransportAlgorithm(uri string) Encryptor {
	e = e.clone()
	e.cfg.keyTransport = uri
	return e
}

// RecipientPublicKey sets the recipient's RSA public key for key transport.
func (e Encryptor) RecipientPublicKey(key *rsa.PublicKey) Encryptor {
	e = e.clone()
	e.cfg.recipientPubKey = key
	return e
}

// SessionKey sets a pre-existing session key. If not set, a random key
// is generated.
func (e Encryptor) SessionKey(key []byte) Encryptor {
	e = e.clone()
	e.cfg.sessionKey = append([]byte(nil), key...)
	return e
}

// OAEPDigest sets the digest algorithm for RSA-OAEP 1.1.
func (e Encryptor) OAEPDigest(uri string) Encryptor {
	e = e.clone()
	e.cfg.oaepDigest = uri
	return e
}

// OAEPMGF sets the MGF algorithm for RSA-OAEP 1.1.
func (e Encryptor) OAEPMGF(uri string) Encryptor {
	e = e.clone()
	e.cfg.oaepMGF = uri
	return e
}

// OAEPParams sets the OAEP parameters.
func (e Encryptor) OAEPParams(params []byte) Encryptor {
	e = e.clone()
	e.cfg.oaepParams = append([]byte(nil), params...)
	return e
}

// KeyWrapAlgorithm sets the key wrapping algorithm URI.
func (e Encryptor) KeyWrapAlgorithm(uri string) Encryptor {
	e = e.clone()
	e.cfg.keyWrapAlgorithm = uri
	return e
}

// KeyEncryptionKey sets the key encryption key for AES key wrapping.
func (e Encryptor) KeyEncryptionKey(kek []byte) Encryptor {
	e = e.clone()
	e.cfg.keyEncryptionKey = append([]byte(nil), kek...)
	return e
}

// EncryptElement encrypts an entire element, replacing it in the tree
// with an EncryptedData element. Returns the EncryptedData element.
func (e Encryptor) EncryptElement(ctx context.Context, elem *helium.Element) (*helium.Element, error) {
	return encrypt(ctx, e.cfg, elem, TypeElement)
}

// EncryptContent encrypts the content of an element, replacing the
// children with an EncryptedData element. Returns the EncryptedData element.
func (e Encryptor) EncryptContent(ctx context.Context, elem *helium.Element) (*helium.Element, error) {
	return encrypt(ctx, e.cfg, elem, TypeContent)
}

func encrypt(_ context.Context, cfg *encryptConfig, elem *helium.Element, encType string) (*helium.Element, error) {
	// Secure by default: an unset block algorithm uses authenticated
	// AES-256-GCM rather than refusing or falling back to CBC.
	blockAlgorithm := cfg.blockAlgorithm
	if blockAlgorithm == "" {
		blockAlgorithm = DefaultBlockAlgorithm
	}

	// Emitting new unauthenticated CBC ciphertext requires an explicit
	// opt-in. Decryption of existing CBC ciphertext is unaffected.
	switch blockAlgorithm {
	case AES128CBC, AES256CBC:
		if !cfg.allowLegacyCBC {
			return nil, ErrCBCEncryptionRequiresOptIn
		}
	}

	hasKeyTransport := cfg.recipientPubKey != nil && cfg.keyTransport != ""
	hasKeyWrap := len(cfg.keyEncryptionKey) > 0 && cfg.keyWrapAlgorithm != ""
	hasSessionKey := len(cfg.sessionKey) > 0

	if !hasKeyTransport && !hasKeyWrap && !hasSessionKey {
		return nil, fmt.Errorf("%w: no key transport, key wrap, or session key configured", ErrMissingConfig)
	}

	// Get or generate session key.
	keySize, err := keySizeForAlgorithm(blockAlgorithm)
	if err != nil {
		return nil, err
	}

	sessionKey := cfg.sessionKey
	if len(sessionKey) == 0 {
		sessionKey = make([]byte, keySize)
		if _, err := io.ReadFull(rand.Reader, sessionKey); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
		}
	}

	// Bind the declared block-algorithm URI to the session-key length so
	// a user-supplied SessionKey cannot make us emit, e.g., an AES-256
	// URI while actually encrypting with AES-128.
	if err := validateKeySize(blockAlgorithm, sessionKey); err != nil {
		return nil, err
	}

	// Serialize the plaintext.
	var plaintext string
	if encType == TypeElement {
		plaintext, err = helium.WriteString(elem)
	} else {
		plaintext, err = serializeChildren(elem)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	// Block encrypt.
	cipherValue, err := blockEncrypt(blockAlgorithm, sessionKey, []byte(plaintext))
	if err != nil {
		return nil, err
	}

	// Encrypt session key.
	var encKey *EncryptedKey
	if hasKeyTransport {
		encKeyBytes, err := encryptSessionKey(cfg.keyTransport, cfg.recipientPubKey, sessionKey, cfg.oaepDigest, cfg.oaepMGF, cfg.oaepParams)
		if err != nil {
			return nil, err
		}
		encKey = &EncryptedKey{
			EncryptionMethod: &EncryptionMethod{
				Algorithm:    cfg.keyTransport,
				DigestMethod: cfg.oaepDigest,
				MGFAlgorithm: cfg.oaepMGF,
				OAEPParams:   cfg.oaepParams,
			},
			CipherValue: encKeyBytes,
		}
	} else if hasKeyWrap {
		// Bind the declared key-wrap URI to the KEK length so a 16-byte
		// KEK cannot make us emit a kw-aes256 URI while wrapping with
		// AES-128.
		if err := validateKeySize(cfg.keyWrapAlgorithm, cfg.keyEncryptionKey); err != nil {
			return nil, err
		}
		wrappedKey, err := aesKeyWrap(cfg.keyEncryptionKey, sessionKey)
		if err != nil {
			return nil, err
		}
		encKey = &EncryptedKey{
			EncryptionMethod: &EncryptionMethod{Algorithm: cfg.keyWrapAlgorithm},
			CipherValue:      wrappedKey,
		}
	}

	// Build EncryptedData.
	var encKeys []*EncryptedKey
	if encKey != nil {
		encKeys = []*EncryptedKey{encKey}
	}
	ed := &EncryptedData{
		Type:             encType,
		EncryptionMethod: &EncryptionMethod{Algorithm: blockAlgorithm},
		EncryptedKeys:    encKeys,
		CipherValue:      cipherValue,
	}

	doc := elem.OwnerDocument()
	edElem, err := marshalEncryptedData(doc, ed)
	if err != nil {
		return nil, err
	}

	// Replace in tree.
	if encType == TypeElement {
		// Replace the element with EncryptedData in place, preserving the
		// original element's position among its siblings.
		if err := elem.Replace(edElem); err != nil {
			return nil, err
		}
	} else {
		// Replace children with EncryptedData.
		removeChildren(elem)
		if err := elem.AddChild(edElem); err != nil {
			return nil, err
		}
	}

	return edElem, nil
}

func serializeChildren(elem *helium.Element) (string, error) {
	var sb strings.Builder
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		s, err := helium.WriteString(child)
		if err != nil {
			return "", err
		}
		sb.WriteString(s)
	}
	return sb.String(), nil
}

func removeChildren(elem *helium.Element) {
	for child := elem.FirstChild(); child != nil; {
		next := child.NextSibling()
		if mn, ok := child.(helium.MutableNode); ok {
			helium.UnlinkNode(mn)
		}
		child = next
	}
}

// decryptConfig holds the configuration for a Decryptor.
type decryptConfig struct {
	privateKey              *rsa.PrivateKey
	keyEncryptionKey        []byte
	sessionKey              []byte
	allowUnauthenticatedCBC bool
}

// Decryptor decrypts XML EncryptedData elements. It uses clone-on-write
// semantics.
type Decryptor struct {
	cfg *decryptConfig
}

// NewDecryptor creates a new Decryptor.
func NewDecryptor() Decryptor {
	return Decryptor{cfg: &decryptConfig{}}
}

func (d Decryptor) clone() Decryptor {
	if d.cfg == nil {
		return Decryptor{cfg: &decryptConfig{}}
	}
	cp := *d.cfg
	return Decryptor{cfg: &cp}
}

// PrivateKey sets the RSA private key for key transport decryption.
func (d Decryptor) PrivateKey(key *rsa.PrivateKey) Decryptor {
	d = d.clone()
	d.cfg.privateKey = key
	return d
}

// KeyEncryptionKey sets the key for AES key unwrapping.
func (d Decryptor) KeyEncryptionKey(kek []byte) Decryptor {
	d = d.clone()
	d.cfg.keyEncryptionKey = append([]byte(nil), kek...)
	return d
}

// SessionKey sets a pre-shared session key directly.
func (d Decryptor) SessionKey(key []byte) Decryptor {
	d = d.clone()
	d.cfg.sessionKey = append([]byte(nil), key...)
	return d
}

// AllowUnauthenticatedCBC opts the Decryptor in to decrypting AES-CBC
// ciphertexts. AES-CBC under XML Encryption 1.0 is unauthenticated and
// vulnerable to padding-oracle attacks (Jager/Somorovsky 2011); XML
// Encryption 1.1 deprecated CBC in favor of AES-GCM.
//
// By default the Decryptor refuses CBC and returns
// [ErrCBCRequiresOptIn]. Set this to true only if you must accept
// legacy CBC ciphertexts AND you have verified that decryption errors
// are not exposed to remote attackers (e.g. by surfacing the same
// generic error for every failure path and never timing-sidechannel
// distinguishing them).
func (d Decryptor) AllowUnauthenticatedCBC(v bool) Decryptor {
	d = d.clone()
	d.cfg.allowUnauthenticatedCBC = v
	return d
}

// Decrypt decrypts an EncryptedData element and returns the decrypted nodes.
func (d Decryptor) Decrypt(ctx context.Context, elem *helium.Element) ([]helium.Node, error) {
	return decryptElement(ctx, d.cfg, elem)
}

func decryptElement(ctx context.Context, cfg *decryptConfig, elem *helium.Element) ([]helium.Node, error) {
	ed, err := parseEncryptedData(elem)
	if err != nil {
		return nil, err
	}

	// Validate the declared content Type up front. An omitted Type defaults
	// to Element (preserving prior behavior); any other non-empty value,
	// including unknown URIs, is rejected rather than silently treated as
	// Element.
	var isContent bool
	switch ed.Type {
	case "", TypeElement:
		isContent = false
	case TypeContent:
		isContent = true
	default:
		return nil, fmt.Errorf("%w: unsupported EncryptedData Type %q", ErrMalformedEncrypted, ed.Type)
	}

	// Validate the EncryptionMethod and CBC opt-in once, up front: these
	// describe the block cipher and are independent of which session-key
	// candidate ultimately succeeds.
	if ed.EncryptionMethod == nil {
		return nil, fmt.Errorf("%w: missing EncryptionMethod", ErrMalformedEncrypted)
	}
	alg := ed.EncryptionMethod.Algorithm
	switch alg {
	case AES128CBC, AES256CBC:
		if !cfg.allowUnauthenticatedCBC {
			return nil, ErrCBCRequiresOptIn
		}
	}

	// A pre-shared session key, when configured, is the sole candidate.
	if len(cfg.sessionKey) > 0 {
		return finishDecrypt(ctx, ed, elem, alg, isContent, cfg.sessionKey)
	}

	keys := ed.effectiveEncryptedKeys()
	if len(keys) == 0 {
		return nil, ErrMissingKey
	}

	// A document may carry several EncryptedKey candidates (one per
	// recipient), and an attacker can prepend a junk EncryptedKey that
	// unwraps cleanly under the recipient's key while wrapping the WRONG
	// session key. Carry each candidate all the way through block
	// decryption AND plaintext parse/shape validation, accepting only a
	// candidate that fully succeeds — never stop at one that merely
	// unwraps/transports. This supports multi-recipient documents and
	// stops a bogus-but-valid EncryptedKey from masking the real one.
	var lastErr error
	for _, ek := range keys {
		sessionKey, err := resolveSessionKeyFromEncryptedKey(cfg, ek)
		if err != nil {
			lastErr = preferInformativeErr(lastErr, err)
			continue
		}
		nodes, err := finishDecrypt(ctx, ed, elem, alg, isContent, sessionKey)
		if err != nil {
			lastErr = preferInformativeErr(lastErr, err)
			continue
		}
		return nodes, nil
	}
	return nil, lastErr
}

// preferInformativeErr keeps the most informative error across EncryptedKey
// candidates. A non-applicable candidate (one whose algorithm needs a key the
// caller did not supply) yields ErrMissingKey, which carries no signal about
// why decryption truly failed. A real failure from an applicable candidate
// (bad unwrap, wrong session key, malformed plaintext) must not be masked by
// a later ErrMissingKey. So: the first non-ErrMissingKey error wins and is
// never overwritten by a subsequent ErrMissingKey.
func preferInformativeErr(existing, candidate error) error {
	if existing == nil {
		return candidate
	}
	if errors.Is(existing, ErrMissingKey) && !errors.Is(candidate, ErrMissingKey) {
		return candidate
	}
	return existing
}

// finishDecrypt block-decrypts the CipherValue under a candidate session
// key, parses the recovered plaintext through a hardened parser, and
// validates its shape. It returns the decrypted nodes only when the entire
// pipeline succeeds, so a caller iterating session-key candidates can
// safely fall through to the next candidate on any error.
func finishDecrypt(ctx context.Context, ed *EncryptedData, elem *helium.Element, alg string, isContent bool, sessionKey []byte) ([]helium.Node, error) {
	plaintext, err := blockDecrypt(alg, sessionKey, ed.CipherValue)
	if err != nil {
		// Squash all decryption errors to the same sentinel — and
		// crucially, the same string — so callers cannot distinguish
		// "bad padding" from "bad cipher" from "downstream parse"
		// when CBC is in use. GCM authenticates so this collapse is
		// safe there too.
		if errors.Is(err, ErrDecryptionFailed) || errors.Is(err, ErrInvalidPadding) {
			return nil, ErrDecryptionFailed
		}
		return nil, err
	}

	// Parse decrypted XML through a hardened parser: no external DTD,
	// no XXE entity loading, no network. The plaintext is attacker-
	// controlled (it is the output of the attacker's ciphertext under
	// the recipient's key) and must not be allowed to fetch external
	// resources.
	parser := newHardenedInnerParser()

	// Both Element and Content replace EncryptedData at its position in the
	// tree, so the decrypted fragment must be parsed in the in-scope-namespace
	// context of EncryptedData's PARENT — not EncryptedData's own declarations,
	// which could wrongly shadow the parent context (e.g. an EncryptedData
	// carrying its own default xmlns would make unprefixed plaintext resolve in
	// the wrong namespace). Equally important, the plaintext may use a prefix
	// declared only on an ANCESTOR (a serialized <saml:NameID/> carries no
	// xmlns:saml of its own); parsing it as a standalone document would fail on
	// the unbound prefix. ParseInNodeContext resolves prefixes/default-ns
	// exactly as the replacement position requires and avoids any manual
	// splicing of attacker-controlled namespace strings.
	//
	// If EncryptedData is detached (no parent), there is no replacement
	// position whose in-scope namespaces should apply, and EncryptedData's
	// OWN declarations must not be used as context — a detached element
	// carrying its own default xmlns (e.g. the XML-Encryption namespace)
	// would otherwise wrongly shadow the decrypted fragment. Use a NEUTRAL
	// context (the owning document, or a fresh empty document) so no
	// spurious default-ns or prefix bindings leak into the fragment.
	contextNode := elem.Parent()
	if contextNode == nil {
		if doc := elem.OwnerDocument(); doc != nil {
			contextNode = doc
		} else {
			contextNode = helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		}
	}

	first, err := parser.ParseInNodeContext(ctx, contextNode, plaintext)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	var nodes []helium.Node
	if isContent {
		// Content may be multiple children.
		for child := first; child != nil; child = child.NextSibling() {
			nodes = append(nodes, child)
		}
		return nodes, nil
	}

	// Element must yield exactly one element node.
	var elemNode helium.Node
	for child := first; child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			return nil, ErrDecryptionFailed
		}
		if elemNode != nil {
			return nil, ErrDecryptionFailed
		}
		elemNode = child
	}
	if elemNode == nil {
		return nil, ErrDecryptionFailed
	}
	nodes = append(nodes, elemNode)

	return nodes, nil
}

// newHardenedInnerParser returns the helium parser used to parse the
// decrypted plaintext of an EncryptedData element. Decrypted bytes are
// attacker-controlled (an attacker who can submit ciphertexts to a
// decryption oracle can choose the recovered plaintext), so DTD loading,
// external entity resolution, and network access are all disabled.
func newHardenedInnerParser() helium.Parser {
	return helium.NewParser().
		BlockXXE(true).
		LoadExternalDTD(false).
		AllowNetwork(false)
}

func resolveSessionKeyFromEncryptedKey(cfg *decryptConfig, ek *EncryptedKey) ([]byte, error) {
	if ek.EncryptionMethod == nil {
		return nil, fmt.Errorf("%w: EncryptedKey missing EncryptionMethod", ErrMalformedEncrypted)
	}

	alg := ek.EncryptionMethod.Algorithm
	switch alg {
	case RSAOAEP, RSAOAEP11:
		if cfg.privateKey == nil {
			return nil, ErrMissingKey
		}
		return decryptSessionKey(alg, cfg.privateKey, ek.CipherValue,
			ek.EncryptionMethod.DigestMethod, ek.EncryptionMethod.MGFAlgorithm, ek.EncryptionMethod.OAEPParams)
	case AES128KeyWrap, AES256KeyWrap:
		if len(cfg.keyEncryptionKey) == 0 {
			return nil, ErrMissingKey
		}
		// Bind the declared key-wrap URI to the KEK length so a 16-byte
		// KEK is not silently accepted as AES-128 against a kw-aes256
		// declaration. The unwrapped session key is in turn validated
		// against the data-encryption algorithm in blockDecrypt.
		if err := validateKeySize(alg, cfg.keyEncryptionKey); err != nil {
			return nil, err
		}
		return aesKeyUnwrap(cfg.keyEncryptionKey, ek.CipherValue)
	default:
		// Classify under the decrypt path while preserving the typed
		// error in the chain for errors.As, consistent with the
		// decryptSessionKey wrapping above.
		return nil, fmt.Errorf("%w: %w", ErrDecryptionFailed, &UnsupportedAlgorithmError{Algorithm: alg})
	}
}
