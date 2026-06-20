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
}

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

// BlockAlgorithm sets the block encryption algorithm URI.
func (e Encryptor) BlockAlgorithm(uri string) Encryptor {
	e = e.clone()
	e.cfg.blockAlgorithm = uri
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
	if cfg.blockAlgorithm == "" {
		return nil, fmt.Errorf("%w: block algorithm not set", ErrMissingConfig)
	}

	hasKeyTransport := cfg.recipientPubKey != nil && cfg.keyTransport != ""
	hasKeyWrap := len(cfg.keyEncryptionKey) > 0 && cfg.keyWrapAlgorithm != ""
	hasSessionKey := len(cfg.sessionKey) > 0

	if !hasKeyTransport && !hasKeyWrap && !hasSessionKey {
		return nil, fmt.Errorf("%w: no key transport, key wrap, or session key configured", ErrMissingConfig)
	}

	// Get or generate session key.
	keySize, err := keySizeForAlgorithm(cfg.blockAlgorithm)
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
	if err := validateKeySize(cfg.blockAlgorithm, sessionKey); err != nil {
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
	cipherValue, err := blockEncrypt(cfg.blockAlgorithm, sessionKey, []byte(plaintext))
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
	ed := &EncryptedData{
		Type:             encType,
		EncryptionMethod: &EncryptionMethod{Algorithm: cfg.blockAlgorithm},
		EncryptedKey:     encKey,
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

	// Obtain session key.
	sessionKey, err := resolveSessionKey(cfg, ed)
	if err != nil {
		return nil, err
	}

	// Block decrypt.
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
	isContent := ed.Type == TypeContent

	var nodes []helium.Node
	if isContent {
		// Content may be multiple children; wrap in a temporary root.
		// The decrypted fragment may reference namespace prefixes that
		// were declared on an ancestor of the EncryptedData element (the
		// content's original location), not inside the ciphertext itself.
		// Re-declare those in-scope namespaces on the wrapper so the
		// prefixes resolve; otherwise valid content fails to parse.
		wrapped := "<_wrap" + inScopeNamespaceAttrs(elem) + ">" + string(plaintext) + "</_wrap>"
		tmpDoc, err := parser.Parse(ctx, []byte(wrapped))
		if err != nil {
			return nil, ErrDecryptionFailed
		}
		root := tmpDoc.DocumentElement()
		for child := root.FirstChild(); child != nil; child = child.NextSibling() {
			nodes = append(nodes, child)
		}
	} else {
		tmpDoc, err := parser.Parse(ctx, plaintext)
		if err != nil {
			return nil, ErrDecryptionFailed
		}
		nodes = append(nodes, tmpDoc.DocumentElement())
	}

	return nodes, nil
}

// inScopeNamespaceAttrs returns the serialized xmlns attributes for all
// namespace declarations in scope at elem, collected by walking elem and its
// ancestors (a nearer declaration shadows a farther one for the same prefix).
// The result is a leading-space-prefixed string suitable for splicing into an
// element start tag, e.g. ` xmlns:saml="urn:..."`. URIs are XML-attribute
// escaped because the recovered plaintext (and thus its declared namespaces)
// is attacker-controlled.
func inScopeNamespaceAttrs(elem *helium.Element) string {
	seen := map[string]bool{}
	var b strings.Builder
	var cur helium.Node = elem
	for cur != nil {
		if e, ok := cur.(*helium.Element); ok {
			for _, ns := range e.Namespaces() {
				prefix := ns.Prefix()
				if seen[prefix] {
					continue
				}
				seen[prefix] = true
				if prefix == "" {
					b.WriteString(` xmlns="`)
				} else {
					b.WriteString(` xmlns:`)
					b.WriteString(prefix)
					b.WriteString(`="`)
				}
				b.WriteString(escapeNamespaceURI(ns.URI()))
				b.WriteString(`"`)
			}
		}
		cur = cur.Parent()
	}
	return b.String()
}

// escapeNamespaceURI escapes a namespace URI for inclusion in a double-quoted
// XML attribute value.
func escapeNamespaceURI(uri string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return replacer.Replace(uri)
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

func resolveSessionKey(cfg *decryptConfig, ed *EncryptedData) ([]byte, error) {
	if len(cfg.sessionKey) > 0 {
		return cfg.sessionKey, nil
	}

	if ed.EncryptedKey == nil {
		return nil, ErrMissingKey
	}

	ek := ed.EncryptedKey
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
