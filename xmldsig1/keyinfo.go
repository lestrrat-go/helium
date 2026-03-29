package xmldsig1

import (
	"context"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"math/big"

	helium "github.com/lestrrat-go/helium"
)

// KeySource provides keys for signature verification.
type KeySource interface {
	ResolveKey(ctx context.Context, keyInfo *KeyInfoData, alg string) (any, error)
}

// KeySourceFunc adapts a function to the KeySource interface.
type KeySourceFunc func(ctx context.Context, keyInfo *KeyInfoData, alg string) (any, error)

func (f KeySourceFunc) ResolveKey(ctx context.Context, keyInfo *KeyInfoData, alg string) (any, error) {
	return f(ctx, keyInfo, alg)
}

// StaticKey returns a KeySource that always returns the given key.
func StaticKey(key any) KeySource {
	return KeySourceFunc(func(_ context.Context, _ *KeyInfoData, _ string) (any, error) {
		return key, nil
	})
}

// X509CertKeySource returns a KeySource that extracts the public key from
// a trusted X.509 certificate. This is the common SAML pattern.
func X509CertKeySource(cert *x509.Certificate) KeySource {
	return KeySourceFunc(func(_ context.Context, _ *KeyInfoData, _ string) (any, error) {
		return cert.PublicKey, nil
	})
}

// KeyInfoData holds parsed KeyInfo content for verification.
type KeyInfoData struct {
	X509Certificates []*x509.Certificate
	RSAKeyValue      *RSAKeyValueData
	ECKeyValue       *ECKeyValueData
}

// RSAKeyValueData holds parsed RSAKeyValue content.
type RSAKeyValueData struct {
	Modulus  *big.Int
	Exponent int
}

// ECKeyValueData holds parsed ECKeyValue content.
type ECKeyValueData struct {
	Curve elliptic.Curve
	X, Y  *big.Int
}

// KeyInfoBuilder configures how the KeyInfo element is constructed during signing.
type KeyInfoBuilder interface {
	BuildKeyInfo(ctx context.Context, doc *helium.Document, key any) (*helium.Element, error)
}

// x509DataKeyInfo builds KeyInfo containing X509Data with certificate chain.
type x509DataKeyInfo struct {
	certs []*x509.Certificate
}

// X509DataKeyInfo returns a KeyInfoBuilder that includes X509Data containing
// the given certificates.
func X509DataKeyInfo(certs ...*x509.Certificate) KeyInfoBuilder {
	return &x509DataKeyInfo{certs: certs}
}

func (b *x509DataKeyInfo) BuildKeyInfo(_ context.Context, doc *helium.Document, _ any) (*helium.Element, error) {
	keyInfo := doc.CreateElement("KeyInfo")
	if err := keyInfo.DeclareNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, err
	}
	if err := keyInfo.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, err
	}

	x509Data := doc.CreateElement("X509Data")
	if err := x509Data.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, err
	}
	if err := keyInfo.AddChild(x509Data); err != nil {
		return nil, err
	}

	for _, cert := range b.certs {
		certElem := doc.CreateElement("X509Certificate")
		if err := certElem.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
			return nil, err
		}
		encoded := base64.StdEncoding.EncodeToString(cert.Raw)
		if err := certElem.AddChild(doc.CreateText([]byte(encoded))); err != nil {
			return nil, err
		}
		if err := x509Data.AddChild(certElem); err != nil {
			return nil, err
		}
	}
	return keyInfo, nil
}

// rsaKeyValueKeyInfo builds KeyInfo containing RSAKeyValue.
type rsaKeyValueKeyInfo struct{}

// RSAKeyValueKeyInfo returns a KeyInfoBuilder that includes RSAKeyValue
// derived from the signing key.
func RSAKeyValueKeyInfo() KeyInfoBuilder {
	return &rsaKeyValueKeyInfo{}
}

func (b *rsaKeyValueKeyInfo) BuildKeyInfo(_ context.Context, doc *helium.Document, key any) (*helium.Element, error) {
	var pub *rsa.PublicKey
	switch k := key.(type) {
	case *rsa.PrivateKey:
		pub = &k.PublicKey
	case *rsa.PublicKey:
		pub = k
	default:
		return nil, fmt.Errorf("%w: expected RSA key, got %T", ErrKeyMismatch, key)
	}

	keyInfo := doc.CreateElement("KeyInfo")
	if err := keyInfo.DeclareNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, err
	}
	if err := keyInfo.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, err
	}

	keyValue := doc.CreateElement("KeyValue")
	if err := keyValue.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, err
	}
	if err := keyInfo.AddChild(keyValue); err != nil {
		return nil, err
	}

	rsaKV := doc.CreateElement("RSAKeyValue")
	if err := rsaKV.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, err
	}
	if err := keyValue.AddChild(rsaKV); err != nil {
		return nil, err
	}

	modElem := doc.CreateElement("Modulus")
	if err := modElem.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, err
	}
	modText := base64.StdEncoding.EncodeToString(pub.N.Bytes())
	if err := modElem.AddChild(doc.CreateText([]byte(modText))); err != nil {
		return nil, err
	}
	if err := rsaKV.AddChild(modElem); err != nil {
		return nil, err
	}

	expElem := doc.CreateElement("Exponent")
	if err := expElem.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, err
	}
	expBytes := big.NewInt(int64(pub.E)).Bytes()
	expText := base64.StdEncoding.EncodeToString(expBytes)
	if err := expElem.AddChild(doc.CreateText([]byte(expText))); err != nil {
		return nil, err
	}
	if err := rsaKV.AddChild(expElem); err != nil {
		return nil, err
	}

	return keyInfo, nil
}

// parseKeyInfo extracts key information from a ds:KeyInfo element.
func parseKeyInfo(keyInfoElem *helium.Element) (*KeyInfoData, error) {
	data := &KeyInfoData{}
	for child := keyInfoElem.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch localName(elem) {
		case "X509Data":
			if err := parseX509Data(elem, data); err != nil {
				return nil, err
			}
		case "KeyValue":
			if err := parseKeyValue(elem, data); err != nil {
				return nil, err
			}
		}
	}
	return data, nil
}

func parseX509Data(elem *helium.Element, data *KeyInfoData) error {
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		certElem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if localName(certElem) != "X509Certificate" {
			continue
		}
		text := textContent(certElem)
		derBytes, err := base64.StdEncoding.DecodeString(text)
		if err != nil {
			return fmt.Errorf("%w: invalid X509Certificate base64: %v", ErrInvalidKeyInfo, err)
		}
		cert, err := x509.ParseCertificate(derBytes)
		if err != nil {
			return fmt.Errorf("%w: invalid X509Certificate: %v", ErrInvalidKeyInfo, err)
		}
		data.X509Certificates = append(data.X509Certificates, cert)
	}
	return nil
}

func parseKeyValue(elem *helium.Element, data *KeyInfoData) error {
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		kvElem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch localName(kvElem) {
		case "RSAKeyValue":
			return parseRSAKeyValue(kvElem, data)
		case "ECKeyValue":
			return parseECKeyValue(kvElem, data)
		}
	}
	return nil
}

func parseRSAKeyValue(elem *helium.Element, data *KeyInfoData) error {
	kv := &RSAKeyValueData{}
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		text := textContent(e)
		decoded, err := base64.StdEncoding.DecodeString(text)
		if err != nil {
			return fmt.Errorf("%w: invalid RSAKeyValue base64: %v", ErrInvalidKeyInfo, err)
		}
		switch localName(e) {
		case "Modulus":
			kv.Modulus = new(big.Int).SetBytes(decoded)
		case "Exponent":
			kv.Exponent = int(new(big.Int).SetBytes(decoded).Int64())
		}
	}
	data.RSAKeyValue = kv
	return nil
}

func parseECKeyValue(elem *helium.Element, data *KeyInfoData) error {
	kv := &ECKeyValueData{}
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch localName(e) {
		case "NamedCurve":
			uri, _ := e.GetAttribute("URI")
			switch uri {
			case "urn:oid:1.2.840.10045.3.1.7":
				kv.Curve = elliptic.P256()
			case "urn:oid:1.3.132.0.34":
				kv.Curve = elliptic.P384()
			default:
				return fmt.Errorf("%w: unsupported EC curve: %s", ErrInvalidKeyInfo, uri)
			}
		case "PublicKey":
			text := textContent(e)
			decoded, err := base64.StdEncoding.DecodeString(text)
			if err != nil {
				return fmt.Errorf("%w: invalid ECKeyValue base64: %v", ErrInvalidKeyInfo, err)
			}
			if kv.Curve == nil {
				return fmt.Errorf("%w: ECKeyValue missing NamedCurve", ErrInvalidKeyInfo)
			}
			kv.X, kv.Y = elliptic.Unmarshal(kv.Curve, decoded)
			if kv.X == nil {
				return fmt.Errorf("%w: invalid EC public key point", ErrInvalidKeyInfo)
			}
		}
	}
	data.ECKeyValue = kv
	return nil
}

// localName returns the local name of an element, stripping any prefix.
func localName(e *helium.Element) string {
	name := e.Name()
	if i := len(name) - 1; i >= 0 {
		for j := 0; j <= i; j++ {
			if name[j] == ':' {
				return name[j+1:]
			}
		}
	}
	return name
}

// textContent returns the concatenated text content of an element.
func textContent(e *helium.Element) string {
	var sb []byte
	for child := e.FirstChild(); child != nil; child = child.NextSibling() {
		sb = append(sb, child.Content()...)
	}
	return string(sb)
}
