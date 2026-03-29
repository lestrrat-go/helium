package xmlenc1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
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
			return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
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
		// Replace the element with EncryptedData.
		if pe, ok := helium.AsNode[*helium.Element](elem.Parent()); ok {
			if err := pe.AddChild(edElem); err != nil {
				return nil, err
			}
			helium.UnlinkNode(elem)
		} else if pd, ok := helium.AsNode[*helium.Document](elem.Parent()); ok {
			if err := pd.AddChild(edElem); err != nil {
				return nil, err
			}
			helium.UnlinkNode(elem)
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
	privateKey       *rsa.PrivateKey
	keyEncryptionKey []byte
	sessionKey       []byte
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
	plaintext, err := blockDecrypt(ed.EncryptionMethod.Algorithm, sessionKey, ed.CipherValue)
	if err != nil {
		return nil, err
	}

	// Parse decrypted XML.
	isContent := ed.Type == TypeContent

	var nodes []helium.Node
	if isContent {
		// Content may be multiple children; wrap in a temporary root.
		wrapped := "<_wrap>" + string(plaintext) + "</_wrap>"
		tmpDoc, err := helium.NewParser().Parse(ctx, []byte(wrapped))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
		}
		root := tmpDoc.DocumentElement()
		for child := root.FirstChild(); child != nil; child = child.NextSibling() {
			nodes = append(nodes, child)
		}
	} else {
		tmpDoc, err := helium.NewParser().Parse(ctx, plaintext)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
		}
		nodes = append(nodes, tmpDoc.DocumentElement())
	}

	return nodes, nil
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
		return aesKeyUnwrap(cfg.keyEncryptionKey, ek.CipherValue)
	default:
		return nil, &UnsupportedAlgorithmError{Algorithm: alg}
	}
}

