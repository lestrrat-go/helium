package xmlenc1

import (
	"context"
	"crypto/ecdh"
	"crypto/ecdsa"
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
	recipientECPub   *ecdsa.PublicKey
	kdfParams        *ConcatKDFParams
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

// config returns the Encryptor's configuration, substituting NewEncryptor
// defaults when the Encryptor was constructed directly (a zero-value
// Encryptor{} whose cfg is nil). This mirrors clone's nil handling so the
// encrypt terminals never dereference a nil cfg: a zero-value Encryptor
// encrypts as a default Encryptor with no key source, returning
// ErrMissingConfig rather than panicking.
func (e Encryptor) config() *encryptConfig {
	if e.cfg == nil {
		return &encryptConfig{}
	}
	return e.cfg
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

// KeyTransportAlgorithm sets the key transport algorithm URI. Together with
// RecipientPublicKey it selects RSA key transport, one of the mechanisms
// that protect the session key; [ErrConflictingKeyConfig] states how many of
// them an Encryptor may configure.
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

// SessionKey sets a pre-existing session key. An empty or nil key counts as
// not set: a random key of the length the block algorithm requires is
// generated per encryption, and that is also what an Encryptor that never
// calls SessionKey does.
//
// The key is still protected by whichever mechanism is configured (key
// transport or key wrapping); supplying it does not skip that step. A
// non-empty key's length must match the block algorithm exactly, or
// encryption fails with a [KeySizeError] rather than silently encrypting at
// a weaker strength than the emitted @Algorithm claims. An empty or nil key
// never reaches that check, because the generated key is used instead.
//
// A non-empty key with no protection mechanism configured emits no
// EncryptedKey, and the recipient must already hold this key. An empty or
// nil key with no protection mechanism configured leaves nothing configured
// at all, so encryption fails with [ErrMissingConfig].
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

// KeyWrapAlgorithm sets the AES Key Wrap algorithm URI. It names the wrap
// itself, not which mechanism performs it: with KeyEncryptionKey it selects
// AES key wrapping under the supplied key, and with RecipientECPublicKey it
// is the wrap applied to the key ECDH-ES derives. Those are two different
// mechanisms, and [ErrConflictingKeyConfig] states how many of them an
// Encryptor may configure.
func (e Encryptor) KeyWrapAlgorithm(uri string) Encryptor {
	e = e.clone()
	e.cfg.keyWrapAlgorithm = uri
	return e
}

// KeyEncryptionKey sets the key encryption key for AES key wrapping.
// Together with KeyWrapAlgorithm it selects that mechanism, one of the
// mechanisms that protect the session key; [ErrConflictingKeyConfig] states
// how many of them an Encryptor may configure.
func (e Encryptor) KeyEncryptionKey(kek []byte) Encryptor {
	e = e.clone()
	e.cfg.keyEncryptionKey = append([]byte(nil), kek...)
	return e
}

// RecipientECPublicKey sets the recipient's elliptic-curve public key and
// selects XML Encryption 1.1 ECDH-ES key agreement. It is the encrypt-side
// counterpart of Decryptor.ECPrivateKey, and supports P-256, P-384, and
// P-521.
//
// ECDH-ES derives the key-encryption key rather than taking one, so
// KeyWrapAlgorithm still selects the AES Key Wrap variant applied to the
// session key, while KeyEncryptionKey belongs to the separate AES key
// wrapping mechanism; [ErrConflictingKeyConfig] states how many mechanisms an
// Encryptor may configure. Each encryption generates a fresh ephemeral key
// pair, and the EncryptedKey carries its public half in an
// xenc:AgreementMethod.
//
// The key derivation is ConcatKDF; [Encryptor.KeyDerivationParams] controls
// it.
func (e Encryptor) RecipientECPublicKey(key *ecdsa.PublicKey) Encryptor {
	e = e.clone()
	e.cfg.recipientECPub = key
	return e
}

// KeyDerivationParams sets the ConcatKDF parameters used by ECDH-ES key
// agreement. It has no effect without RecipientECPublicKey.
//
// The five OtherInfo fields are concatenated into the KDF input exactly as
// given, so the recipient must derive with identical values; they travel on
// the wire in the emitted xenc11:ConcatKDFParams. A nil params, or one with
// an empty DigestMethod, falls back to SHA-256 with empty OtherInfo.
// [ConcatKDFParams] states the size limit the five OtherInfo fields share.
// An encryption whose params name a DigestMethod and exceed that limit fails
// rather than emitting a document no hardened recipient would accept. Params
// with an empty DigestMethod take the fallback above instead: their OtherInfo
// is discarded before any derivation, so it is never measured against the
// limit and never reaches the wire.
//
// The parameters are copied, byte slices included, so mutating the caller's
// arrays afterwards cannot change what a later encryption derives or emits.
func (e Encryptor) KeyDerivationParams(params *ConcatKDFParams) Encryptor {
	e = e.clone()
	e.cfg.kdfParams = params.clone()
	return e
}

// EncryptElement encrypts an entire element, replacing it in the tree
// with an EncryptedData element. Returns the EncryptedData element.
//
// This mutates the document: elem is unlinked from its position and the
// EncryptedData takes its place among the siblings. Decryptor.Decrypt is
// not the mirror image — it leaves the tree alone and returns the nodes.
func (e Encryptor) EncryptElement(ctx context.Context, elem *helium.Element) (*helium.Element, error) {
	return encrypt(ctx, e.config(), elem, TypeElement)
}

// EncryptContent encrypts the content of an element, replacing the
// children with an EncryptedData element. Returns the EncryptedData element.
//
// This mutates the document: every child of elem is unlinked and the
// EncryptedData becomes its only child. elem itself stays in place.
func (e Encryptor) EncryptContent(ctx context.Context, elem *helium.Element) (*helium.Element, error) {
	return encrypt(ctx, e.config(), elem, TypeContent)
}

// EncryptBytes encrypts arbitrary octets and returns a detached
// EncryptedData element owned by doc. It is the counterpart of
// Decryptor.DecryptBytes: together they cover the payloads that are not an
// XML element or element content, which W3C xmlenc-core1 §3.1 admits by
// leaving @Type absent. The returned element carries no Type attribute for
// exactly that reason, and no tree is modified — the caller decides where
// to insert it.
func (e Encryptor) EncryptBytes(ctx context.Context, doc *helium.Document, plaintext []byte) (*helium.Element, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("%w: EncryptBytes requires a document to own the EncryptedData element", ErrMissingConfig)
	}
	cfg := e.config()
	resolved, err := resolveEncryptConfig(cfg)
	if err != nil {
		return nil, err
	}
	return encryptPlaintext(ctx, cfg, resolved, doc, plaintext, "")
}

func encrypt(ctx context.Context, cfg *encryptConfig, elem *helium.Element, encType string) (*helium.Element, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	// Resolve the configuration before touching the payload: no payload can
	// make a misconfigured Encryptor or an unusable recipient key work, so the
	// errors resolveEncryptConfig decides must be what the caller sees even
	// when the plaintext also fails to serialize.
	resolved, err := resolveEncryptConfig(cfg)
	if err != nil {
		return nil, err
	}

	// Serialize the plaintext. Element encrypts the element itself;
	// Content encrypts its children.
	var plaintext string
	if encType == TypeElement {
		plaintext, err = helium.WriteString(elem)
	} else {
		plaintext, err = serializeChildren(elem)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	edElem, err := encryptPlaintext(ctx, cfg, resolved, elem.OwnerDocument(), []byte(plaintext), encType)
	if err != nil {
		return nil, err
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	// Replace in tree.
	if encType == TypeElement {
		// Replace the element with EncryptedData in place, preserving the
		// original element's position among its siblings.
		if err := elem.Replace(edElem); err != nil {
			return nil, err
		}
		return edElem, nil
	}

	// Replace children with EncryptedData.
	removeChildren(elem)
	if err := elem.AddChild(edElem); err != nil {
		return nil, err
	}
	return edElem, nil
}

// resolvedEncryptConfig is what resolveEncryptConfig decides: the effective
// block algorithm, which of the three mutually exclusive mechanisms protects
// the session key, and the recipient ECDH key when key agreement is the one
// selected. No mechanism being selected means the caller supplied the session
// key directly.
type resolvedEncryptConfig struct {
	blockAlgorithm  string
	hasKeyTransport bool
	hasKeyAgreement bool
	hasKeyWrap      bool
	recipientECDH   *ecdh.PublicKey
}

// resolveEncryptConfig decides five parts of an Encryptor's configuration: it
// defaults the block algorithm, enforces the CBC opt-in, requires a key
// source, rejects a second key-protection mechanism, and resolves the
// ECDH-ES recipient key. Every entry point calls it before any payload work,
// so none of those five is masked by a later failure to serialize or encrypt
// the plaintext. Session-key length is not one of them: it is bound to the
// block algorithm in encryptPlaintext, once the session key exists.
func resolveEncryptConfig(cfg *encryptConfig) (resolvedEncryptConfig, error) {
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
			return resolvedEncryptConfig{}, ErrCBCEncryptionRequiresOptIn
		}
	}

	hasKeyTransport := cfg.recipientPubKey != nil && cfg.keyTransport != ""
	hasKeyAgreement := cfg.recipientECPub != nil && cfg.keyWrapAlgorithm != ""
	hasKeyWrap := len(cfg.keyEncryptionKey) > 0 && cfg.keyWrapAlgorithm != ""
	hasSessionKey := len(cfg.sessionKey) > 0

	if !hasKeyTransport && !hasKeyAgreement && !hasKeyWrap && !hasSessionKey {
		return resolvedEncryptConfig{}, fmt.Errorf("%w: no key transport, key agreement, key wrap, or session key configured", ErrMissingConfig)
	}

	if err := conflictingKeyProtection(cfg, hasKeyTransport, hasKeyAgreement, hasKeyWrap); err != nil {
		return resolvedEncryptConfig{}, err
	}

	resolved := resolvedEncryptConfig{
		blockAlgorithm:  blockAlgorithm,
		hasKeyTransport: hasKeyTransport,
		hasKeyAgreement: hasKeyAgreement,
		hasKeyWrap:      hasKeyWrap,
	}
	// Resolve the recipient key only when key agreement is the mechanism
	// actually in use, so an unusable curve cannot fail an encryption that
	// would never have touched that key.
	if !hasKeyAgreement {
		return resolved, nil
	}
	recipientECDH, err := ecdhRecipientKey(cfg.recipientECPub)
	if err != nil {
		return resolvedEncryptConfig{}, err
	}
	resolved.recipientECDH = recipientECDH
	return resolved, nil
}

// conflictingKeyProtection returns an error naming both mechanisms when two
// of the three ways to protect the session key are configured at once, and
// nil otherwise. encryptPlaintext emits a single EncryptedKey and dispatches
// transport, then agreement, then wrap, so accepting a pair would silently
// discard the loser: a recipient holding only the discarded key then fails to
// decrypt far from the real mistake. A SessionKey alongside a single
// mechanism is not a pair — it supplies the key that mechanism protects.
//
// Agreement and wrap share the same KeyWrapAlgorithm URI, so that message
// names the two setters rather than two algorithms.
func conflictingKeyProtection(cfg *encryptConfig, hasKeyTransport, hasKeyAgreement, hasKeyWrap bool) error {
	switch {
	case hasKeyTransport && hasKeyAgreement:
		return fmt.Errorf("%w: key transport (%q) and ECDH-ES key agreement (%q) are both configured; configure exactly one", ErrConflictingKeyConfig, cfg.keyTransport, cfg.keyWrapAlgorithm)
	case hasKeyTransport && hasKeyWrap:
		return fmt.Errorf("%w: key transport (%q) and key wrap (%q) are both configured; configure exactly one", ErrConflictingKeyConfig, cfg.keyTransport, cfg.keyWrapAlgorithm)
	case hasKeyAgreement && hasKeyWrap:
		return fmt.Errorf("%w: ECDH-ES key agreement (RecipientECPublicKey derives the key-encryption key) and key wrap (KeyEncryptionKey supplies one) are both configured for key wrap %q; configure exactly one", ErrConflictingKeyConfig, cfg.keyWrapAlgorithm)
	}
	return nil
}

// encryptPlaintext performs the whole encryption pipeline over already
// serialized plaintext and an already resolved configuration: it obtains the
// session key, binds its length to the block algorithm, block-encrypts,
// protects the session key, and marshals the EncryptedData element into doc.
// It never touches the tree, so both the element/content and the raw-octet
// entry points share identical crypto handling.
func encryptPlaintext(ctx context.Context, cfg *encryptConfig, resolved resolvedEncryptConfig, doc *helium.Document, plaintext []byte, encType string) (*helium.Element, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	blockAlgorithm := resolved.blockAlgorithm

	// Get or generate session key.
	keySize, err := keySizeForAlgorithm(paramBlockAlgorithm, blockAlgorithm)
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
	if err := validateKeySize(paramBlockAlgorithm, blockAlgorithm, paramSessionKey, sessionKey); err != nil {
		return nil, err
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	// Block encrypt.
	cipherValue, err := blockEncrypt(blockAlgorithm, sessionKey, plaintext)
	if err != nil {
		return nil, err
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	// Encrypt session key.
	var encKey *EncryptedKey
	if resolved.hasKeyTransport {
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
	} else if resolved.hasKeyAgreement {
		// resolveEncryptConfig has already rejected any other mechanism
		// alongside key agreement, so this branch derives the key-encryption
		// key from an ephemeral exchange and wraps the session key under
		// cfg.keyWrapAlgorithm.
		encKey, err = encryptECDHSessionKey(resolved.recipientECDH, cfg.keyWrapAlgorithm, effectiveKDFParams(cfg.kdfParams), sessionKey)
		if err != nil {
			return nil, err
		}
	} else if resolved.hasKeyWrap {
		// Bind the declared key-wrap URI to the KEK length so a 16-byte
		// KEK cannot make us emit a kw-aes256 URI while wrapping with
		// AES-128.
		if err := validateKeySize(paramKeyWrap, cfg.keyWrapAlgorithm, paramKEK, cfg.keyEncryptionKey); err != nil {
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
	if err := contextErr(ctx); err != nil {
		return nil, err
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

	edElem, err := marshalEncryptedData(doc, ed)
	if err != nil {
		return nil, err
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
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

// DefaultMaxEncryptedKeys bounds how many <EncryptedKey> candidates a
// Decryptor will trial-decrypt for a single EncryptedData when
// MaxEncryptedKeys is not set. An unbounded count is a CPU amplification
// (DoS) vector. [Decryptor.MaxEncryptedKeys] documents the per-candidate
// cost this bounds. The default mirrors jwx's WithMaxRecipients (100),
// which is generous for real multi-recipient documents yet caps
// amplification.
const DefaultMaxEncryptedKeys = 100

// DefaultMaxEncryptedKeyBytes bounds the total decoded <EncryptedKey>
// ciphertext of one EncryptedData when [Decryptor.MaxEncryptedKeyBytes] is
// not set, which owns what the budget covers and when it is charged.
//
// 64 KiB fits [DefaultMaxEncryptedKeys] recipients at 512 bytes each — an
// RSA-4096 wrapped key, the largest in ordinary use — and still leaves 14 KiB
// spare. Real wrapped keys are usually far smaller: 24 to 40 bytes for AES
// key wrap, 256 bytes for RSA-2048.
const DefaultMaxEncryptedKeyBytes = 64 << 10

// DefaultMaxCipherValueBytes bounds the decoded EncryptedData CipherValue
// payload when Decryptor.MaxCipherValueBytes is not set. It matches helium's
// default maximum size for an individual XML content node and prevents a
// payload split across CDATA nodes from bypassing that parser-level limit.
const DefaultMaxCipherValueBytes = 10 << 20

// decryptConfig holds the configuration for a Decryptor.
type decryptConfig struct {
	privateKey              *rsa.PrivateKey
	ecPrivateKey            *ecdsa.PrivateKey
	keyEncryptionKey        []byte
	sessionKey              []byte
	allowUnauthenticatedCBC bool
	maxEncryptedKeys        int
	maxEncryptedKeyBytes    int
	maxCipherValueBytes     int
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

// config returns the Decryptor's configuration, substituting NewDecryptor
// defaults when the Decryptor was constructed directly (a zero-value
// Decryptor{} whose cfg is nil). This mirrors clone's nil handling so the
// decrypt terminals never dereference a nil cfg: a zero-value Decryptor
// decrypts as a default Decryptor with no key, returning ErrMissingKey
// rather than panicking.
func (d Decryptor) config() *decryptConfig {
	if d.cfg == nil {
		return &decryptConfig{}
	}
	return d.cfg
}

// PrivateKey sets the RSA private key for key transport decryption.
//
// PrivateKey, ECPrivateKey, and KeyEncryptionKey may all be set at once, so a
// single Decryptor handles documents protected different ways.
// [Decryptor.MaxEncryptedKeys] states which of the three an EncryptedKey
// candidate uses. A non-empty [Decryptor.SessionKey] makes all three inert.
func (d Decryptor) PrivateKey(key *rsa.PrivateKey) Decryptor {
	d = d.clone()
	d.cfg.privateKey = key
	return d
}

// ECPrivateKey sets the elliptic-curve private key used for XML Encryption
// 1.1 ECDH-ES key agreement.
func (d Decryptor) ECPrivateKey(key *ecdsa.PrivateKey) Decryptor {
	d = d.clone()
	d.cfg.ecPrivateKey = key
	return d
}

// KeyEncryptionKey sets the key for AES key unwrapping.
func (d Decryptor) KeyEncryptionKey(kek []byte) Decryptor {
	d = d.clone()
	d.cfg.keyEncryptionKey = append([]byte(nil), kek...)
	return d
}

// SessionKey sets a pre-shared session key directly. As on the Encryptor, an
// empty or nil key counts as not set, and decryption falls back to the
// EncryptedKey candidates.
//
// A non-empty key is not a preference among keys; it is an early return.
// Decrypt and DecryptBytes take it as the session key and return before
// candidate selection, per-candidate validation, and per-candidate key
// resolution, none of which runs. The [Decryptor.MaxEncryptedKeys] cap is
// applied ahead of that return and still holds. Every consequence follows
// from that one fact:
//
//   - PrivateKey, ECPrivateKey, and KeyEncryptionKey have no effect.
//   - An EncryptedKey that only candidate selection would reject — a
//     missing EncryptionMethod, an unsupported algorithm URI, an algorithm
//     whose key the caller never configured — does not fail the decrypt.
//
// Set it only when the session key is known out of band.
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

// MaxEncryptedKeys caps the number of <EncryptedKey> candidates the
// Decryptor will trial-decrypt for a single EncryptedData. A document packed
// with junk EncryptedKey elements is a CPU amplification (DoS) vector, so
// the cap is enforced before any candidate crypto runs.
//
// A candidate's branch — which key it uses and what it costs — is dispatched
// on its AgreementMethod first and on its declared algorithm second. An
// EncryptedKey carrying an AgreementMethod takes the key-agreement branch and
// uses [Decryptor.ECPrivateKey]; its declared algorithm is the AES key-wrap
// URI applied to the agreed key and does not choose the branch. Only a
// supported ECDH-ES agreement URI then reaches the full cost of a key
// agreement, a ConcatKDF derivation, and an AES key unwrap; any other
// agreement URI is rejected before all three. Without an AgreementMethod the
// declared algorithm decides: an RSA-OAEP URI uses [Decryptor.PrivateKey] and
// costs a private-key decrypt, an AES key-wrap URI uses
// [Decryptor.KeyEncryptionKey] and costs a plain key unwrap. A candidate
// whose branch needs a key the caller never configured costs no crypto at all
// and yields [ErrMissingKey].
//
// Zero (the default) uses [DefaultMaxEncryptedKeys]; a negative value
// removes the limit (matching helium's MaxDepth convention). A document
// exceeding the effective cap fails with [ErrTooManyEncryptedKeys], in every
// key configuration: the cap is applied to the parsed candidate list before
// the [Decryptor.SessionKey] early return.
func (d Decryptor) MaxEncryptedKeys(n int) Decryptor {
	d = d.clone()
	d.cfg.maxEncryptedKeys = n
	return d
}

// MaxEncryptedKeyBytes caps the total decoded <EncryptedKey> ciphertext, in
// bytes, that decrypting one EncryptedData will hold. [Decryptor.MaxEncryptedKeys]
// bounds how many candidates a document may carry; this bounds how large they
// may be together, which that count alone does not.
//
// The budget is charged while the document is read, before each candidate's
// CipherValue is assembled or decoded, and both are skipped once the running
// total would exceed it. A CipherValue the base64 decoder would reject is
// charged what that rejected decode costs, so malformed ciphertext cannot buy
// work the budget was set to deny. Only <EncryptedKey> ciphertext counts. The
// EncryptedData payload is charged separately by MaxCipherValueBytes.
//
// What the budget bounds is memory held for a candidate, not the length of the
// text it was written as. A CipherValue may carry XML whitespace between its
// characters and may be spread over any number of text and CDATA nodes, none
// of which changes the bytes it decodes to; that text is counted where it lies
// and never gathered into a value of its own, so an unbounded amount of it
// costs nothing beyond reading it.
//
// Zero (the default) uses [DefaultMaxEncryptedKeyBytes]; a negative value
// removes the limit (matching helium's MaxDepth convention). A document over
// the effective budget fails with [ErrEncryptedKeyBytesExceeded], in every key
// configuration: the budget is charged during parsing, so it holds ahead of
// both the candidate loop and the [Decryptor.SessionKey] early return.
func (d Decryptor) MaxEncryptedKeyBytes(n int) Decryptor {
	d = d.clone()
	d.cfg.maxEncryptedKeyBytes = n
	return d
}

// MaxCipherValueBytes caps the decoded EncryptedData CipherValue payload, in
// bytes, that decrypting one EncryptedData will hold. This is independent of
// MaxEncryptedKeyBytes, which bounds only the wrapped session-key candidates.
//
// The budget is charged while the document is read, before the payload
// CipherValue is assembled or decoded. It measures decoded octets, so XML
// whitespace and splitting across text or CDATA nodes cannot bypass it.
//
// Zero (the default) uses DefaultMaxCipherValueBytes; a negative value removes
// the limit. A document over the effective budget fails with
// ErrCipherValueBytesExceeded before block decryption or plaintext parsing.
func (d Decryptor) MaxCipherValueBytes(n int) Decryptor {
	d = d.clone()
	d.cfg.maxCipherValueBytes = n
	return d
}

// Decrypt decrypts an EncryptedData element and returns the decrypted nodes.
//
// Unlike Encryptor.EncryptElement and Encryptor.EncryptContent, which splice
// EncryptedData into the tree, Decrypt does NOT modify the document: elem
// stays exactly where it is and the returned nodes are detached. Restoring
// the original document is the caller's decision — call elem.Replace with
// the single node for a TypeElement payload, or remove elem and insert the
// nodes at its position for TypeContent.
//
// The plaintext is parsed with DTD loading, external entity resolution, and
// network access disabled, in the in-scope-namespace context of elem's
// parent, so prefixes declared only on an ancestor resolve correctly.
//
// A TypeContent payload yields its children; a TypeElement payload (the
// default when @Type is absent) must yield exactly one element node. An
// unrecognized @Type is rejected as malformed — use DecryptBytes for a
// payload that is not XML.
func (d Decryptor) Decrypt(ctx context.Context, elem *helium.Element) ([]helium.Node, error) {
	return decryptElement(ctx, d.config(), elem)
}

// DecryptBytes decrypts an EncryptedData element and returns its plaintext
// octets without parsing them as XML. It is used for EncryptedData values whose
// Type is an application-defined binary payload.
func (d Decryptor) DecryptBytes(ctx context.Context, elem *helium.Element) ([]byte, error) {
	return decryptBytes(ctx, d.config(), elem)
}

// checkEncryptedKeyCap fails closed when an EncryptedData carries more
// EncryptedKey candidates than the Decryptor's effective limit (zero => the
// default, negative => unlimited), before any candidate crypto runs: an
// attacker who can pack a document with junk <EncryptedKey> elements gets CPU
// amplification. Decryptor.MaxEncryptedKeys documents the per-candidate cost
// this bounds.
//
// Both decrypt entry points apply it to the parsed candidate list before they
// consult the configured keys, so the bound holds however the caller supplies
// the session key.
func checkEncryptedKeyCap(cfg *decryptConfig, candidates int) error {
	maxKeys := cfg.maxEncryptedKeys
	if maxKeys == 0 {
		maxKeys = DefaultMaxEncryptedKeys
	}
	if maxKeys >= 0 && candidates > maxKeys {
		return fmt.Errorf("%w: %d candidates exceed the limit of %d; raise or remove it with Decryptor.MaxEncryptedKeys", ErrTooManyEncryptedKeys, candidates, maxKeys)
	}
	return nil
}

// encryptedKeyBudget is the running state of the cumulative EncryptedKey
// ciphertext allowance while one EncryptedData is parsed. Decryptor.MaxEncryptedKeyBytes
// documents the budget; this type only spends it.
type encryptedKeyBudget struct {
	// limit is the effective allowance in bytes. Negative means unlimited.
	limit     int
	remaining int
}

// payloadCipherValueBudget is the per-EncryptedData allowance for the
// ciphertext payload. Unlike encryptedKeyBudget, it is spent once.
type payloadCipherValueBudget struct {
	limit int
}

type cipherValueBudget interface {
	charge(int) error
}

// newEncryptedKeyBudget resolves a Decryptor's configured budget (zero => the
// default, negative => unlimited) into the state the parse path spends.
func newEncryptedKeyBudget(cfg *decryptConfig) *encryptedKeyBudget {
	limit := cfg.maxEncryptedKeyBytes
	if limit == 0 {
		limit = DefaultMaxEncryptedKeyBytes
	}
	return &encryptedKeyBudget{limit: limit, remaining: limit}
}

func newPayloadCipherValueBudget(cfg *decryptConfig) *payloadCipherValueBudget {
	limit := cfg.maxCipherValueBytes
	if limit == 0 {
		limit = DefaultMaxCipherValueBytes
	}
	return &payloadCipherValueBudget{limit: limit}
}

// charge deducts n bytes from the remaining allowance, failing closed when
// they do not fit. Callers charge what the decode would cost BEFORE building
// anything from the value, so an oversized CipherValue is rejected without
// being assembled or decoded.
//
// A nil budget is unlimited for parser-only test helpers.
func (b *encryptedKeyBudget) charge(n int) error {
	if b == nil || b.limit < 0 {
		return nil
	}
	if n > b.remaining {
		return fmt.Errorf("%w: EncryptedKey ciphertext exceeds the total of %d bytes; raise or remove it with Decryptor.MaxEncryptedKeyBytes", ErrEncryptedKeyBytesExceeded, b.limit)
	}
	b.remaining -= n
	return nil
}

func (b *payloadCipherValueBudget) charge(n int) error {
	if b == nil || b.limit < 0 {
		return nil
	}
	if n > b.limit {
		return fmt.Errorf("%w: payload ciphertext exceeds the limit of %d bytes; raise or remove it with Decryptor.MaxCipherValueBytes", ErrCipherValueBytesExceeded, b.limit)
	}
	return nil
}

func decryptBytes(ctx context.Context, cfg *decryptConfig, elem *helium.Element) ([]byte, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	ed, err := parseEncryptedData(elem, newEncryptedKeyBudget(cfg), newPayloadCipherValueBudget(cfg))
	if err != nil {
		return nil, err
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
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

	keys := ed.effectiveEncryptedKeys()
	if err := checkEncryptedKeyCap(cfg, len(keys)); err != nil {
		return nil, err
	}

	if len(cfg.sessionKey) > 0 {
		plaintext, err := decryptCipherValue(ed, alg, cfg.sessionKey)
		if err != nil {
			return nil, err
		}
		if err := contextErr(ctx); err != nil {
			return nil, err
		}
		return plaintext, nil
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("%w: EncryptedData carries no EncryptedKey; set Decryptor.SessionKey to supply the key directly", ErrMissingKey)
	}

	sessionKeySize, err := keySizeForAlgorithm(paramBlockAlgorithm, alg)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, ek := range keys {
		if err := contextErr(ctx); err != nil {
			return nil, err
		}
		sessionKey, err := resolveSessionKeyFromEncryptedKey(cfg, ek, sessionKeySize)
		if err != nil {
			// Key resolution cannot be interrupted once entered, so a
			// cancellation that lands while it runs is only observable
			// here. Let it win over the candidate's own error: the caller
			// asked to stop, and the candidate error would otherwise be
			// reported as the reason decryption failed.
			if cerr := contextErr(ctx); cerr != nil {
				return nil, cerr
			}
			lastErr = preferInformativeErr(lastErr, err)
			continue
		}
		if err := contextErr(ctx); err != nil {
			return nil, err
		}
		plaintext, err := decryptCipherValue(ed, alg, sessionKey)
		if err != nil {
			if cerr := contextErr(ctx); cerr != nil {
				return nil, cerr
			}
			lastErr = preferInformativeErr(lastErr, err)
			continue
		}
		if err := contextErr(ctx); err != nil {
			return nil, err
		}
		return plaintext, nil
	}
	return nil, lastErr
}

func decryptElement(ctx context.Context, cfg *decryptConfig, elem *helium.Element) ([]helium.Node, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	ed, err := parseEncryptedData(elem, newEncryptedKeyBudget(cfg), newPayloadCipherValueBudget(cfg))
	if err != nil {
		return nil, err
	}
	if err := contextErr(ctx); err != nil {
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

	keys := ed.effectiveEncryptedKeys()
	if err := checkEncryptedKeyCap(cfg, len(keys)); err != nil {
		return nil, err
	}

	// A pre-shared session key, when configured, returns here — before
	// candidate selection, per-candidate validation, and per-candidate key
	// resolution. Decryptor.SessionKey documents what that skips.
	if len(cfg.sessionKey) > 0 {
		return finishDecrypt(ctx, ed, elem, alg, isContent, cfg.sessionKey)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("%w: EncryptedData carries no EncryptedKey; set Decryptor.SessionKey to supply the key directly", ErrMissingKey)
	}

	// The declared block algorithm fixes the session-key length, and with it
	// the exact length of a valid AES key-wrap ciphertext. Resolve it once,
	// before the loop, so every candidate's unwrap is length-bound.
	sessionKeySize, err := keySizeForAlgorithm(paramBlockAlgorithm, alg)
	if err != nil {
		return nil, err
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
		// Poll the caller's deadline between candidates so a cancellation
		// interrupts the per-candidate key-resolution work rather than
		// running to completion over every EncryptedKey.
		if err := contextErr(ctx); err != nil {
			return nil, err
		}
		sessionKey, err := resolveSessionKeyFromEncryptedKey(cfg, ek, sessionKeySize)
		if err != nil {
			// Key resolution cannot be interrupted once entered, so a
			// cancellation that lands while it runs is only observable
			// here. It must win over the candidate's own error, exactly as
			// the finishDecrypt branch below lets a cancellation win.
			if cerr := contextErr(ctx); cerr != nil {
				return nil, cerr
			}
			lastErr = preferInformativeErr(lastErr, err)
			continue
		}
		nodes, err := finishDecrypt(ctx, ed, elem, alg, isContent, sessionKey)
		if err != nil {
			if isContextErr(err) {
				return nil, err
			}
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
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	plaintext, err := decryptCipherValue(ed, alg, sessionKey)
	if err != nil {
		return nil, err
	}
	if err := contextErr(ctx); err != nil {
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
		if isContextErr(err) {
			return nil, err
		}
		return nil, ErrDecryptionFailed
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
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

func decryptCipherValue(ed *EncryptedData, alg string, sessionKey []byte) ([]byte, error) {
	plaintext, err := blockDecrypt(alg, sessionKey, ed.CipherValue)
	if err == nil {
		return plaintext, nil
	}
	// Squash all decryption errors to the same sentinel — and crucially, the
	// same string — so callers cannot distinguish "bad padding" from "bad
	// cipher" from "downstream parse" when CBC is in use. GCM authenticates
	// so this collapse is safe there too.
	if errors.Is(err, ErrDecryptionFailed) || errors.Is(err, ErrInvalidPadding) {
		return nil, ErrDecryptionFailed
	}
	return nil, err
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

// resolveSessionKeyFromEncryptedKey recovers the session key one EncryptedKey
// candidate protects. sessionKeySize is the length the EncryptedData's
// declared block algorithm requires; the key-wrap branches pass it to
// aesKeyUnwrap, which owns what it binds.
func resolveSessionKeyFromEncryptedKey(cfg *decryptConfig, ek *EncryptedKey, sessionKeySize int) ([]byte, error) {
	if ek.EncryptionMethod == nil {
		return nil, fmt.Errorf("%w: EncryptedKey missing EncryptionMethod", ErrMalformedEncrypted)
	}

	// Every "no key" path below names the key kind the candidate needs and
	// the setter that supplies it. The information is derived from the
	// document's own declared algorithm, which the caller can already read,
	// so it leaks nothing an attacker does not have — and without it every
	// key kind fails with the same unactionable "no decryption key
	// available".
	alg := ek.EncryptionMethod.Algorithm
	if ek.AgreementMethod != nil {
		if cfg.ecPrivateKey == nil {
			return nil, fmt.Errorf("%w: EncryptedKey uses key agreement %q; set Decryptor.ECPrivateKey", ErrMissingKey, ek.AgreementMethod.Algorithm)
		}
		return decryptECDHSessionKey(cfg.ecPrivateKey, ek, sessionKeySize)
	}
	switch alg {
	case RSAOAEP, RSAOAEP11:
		if cfg.privateKey == nil {
			return nil, fmt.Errorf("%w: EncryptedKey uses RSA key transport %q; set Decryptor.PrivateKey", ErrMissingKey, alg)
		}
		return decryptSessionKey(alg, cfg.privateKey, ek.CipherValue,
			ek.EncryptionMethod.DigestMethod, ek.EncryptionMethod.MGFAlgorithm, ek.EncryptionMethod.OAEPParams)
	case AES128KeyWrap, AES192KeyWrap, AES256KeyWrap:
		if len(cfg.keyEncryptionKey) == 0 {
			return nil, fmt.Errorf("%w: EncryptedKey uses AES key wrap %q; set Decryptor.KeyEncryptionKey", ErrMissingKey, alg)
		}
		// Bind the declared key-wrap URI to the KEK length so a 16-byte
		// KEK is not silently accepted as AES-128 against a kw-aes256
		// declaration. The unwrapped session key is in turn validated
		// against the data-encryption algorithm in blockDecrypt.
		if err := validateKeySize(paramKeyWrap, alg, paramKEK, cfg.keyEncryptionKey); err != nil {
			return nil, err
		}
		return aesKeyUnwrap(cfg.keyEncryptionKey, ek.CipherValue, sessionKeySize)
	default:
		// Classify under the decrypt path while preserving the typed
		// error in the chain for errors.As, consistent with the
		// decryptSessionKey wrapping above.
		return nil, fmt.Errorf("%w: %w", ErrDecryptionFailed, &UnsupportedAlgorithmError{Parameter: paramKeyTransport, Algorithm: alg})
	}
}
