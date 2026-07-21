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
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/xmlbase64"
)

// KeySource provides keys for signature verification.
type KeySource interface {
	ResolveKey(ctx context.Context, keyInfo *KeyInfoData, alg string) (any, error)
}

// KeySourceFunc adapts a function to the KeySource interface.
type KeySourceFunc func(ctx context.Context, keyInfo *KeyInfoData, alg string) (any, error)

func (f KeySourceFunc) ResolveKey(ctx context.Context, keyInfo *KeyInfoData, alg string) (any, error) {
	// A typed-nil KeySourceFunc (e.g. var ks KeySourceFunc; NewVerifier(ks))
	// survives the interface!=nil check in verifySignature, so guard the nil
	// func here and return a typed error instead of panicking on the call.
	if f == nil {
		return nil, ErrNoKeySource
	}
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
		// A nil certificate (e.g. a per-request registry lookup that misses on an
		// unknown issuer) would panic on cert.PublicKey below. Fail closed with a
		// typed key-source error at resolve time instead, mirroring the nil-func
		// guard in KeySourceFunc.ResolveKey.
		if cert == nil {
			return nil, fmt.Errorf("%w: nil certificate", ErrNoKeySource)
		}
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
	// With no certificates the loop below runs zero times and would emit a
	// schema-invalid empty <X509Data>. Reject that up front so signing fails
	// loudly instead of producing an empty X509Data.
	if len(b.certs) == 0 {
		return nil, fmt.Errorf("%w: X509DataKeyInfo requires at least one certificate", ErrInvalidKeyInfo)
	}

	keyInfo, err := doc.CreateElement("KeyInfo")
	if err != nil {
		return nil, err
	}
	if err := keyInfo.DeclareNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, err
	}
	if err := keyInfo.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, err
	}

	x509Data, err := doc.CreateElement("X509Data")
	if err != nil {
		return nil, err
	}
	if err := x509Data.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, err
	}
	if err := keyInfo.AddChild(x509Data); err != nil {
		return nil, err
	}

	for _, cert := range b.certs {
		certElem, err := doc.CreateElement("X509Certificate")
		if err != nil {
			return nil, err
		}
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

	keyInfo, err := doc.CreateElement("KeyInfo")
	if err != nil {
		return nil, err
	}
	if err := keyInfo.DeclareNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, err
	}
	if err := keyInfo.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, err
	}

	keyValue, err := doc.CreateElement("KeyValue")
	if err != nil {
		return nil, err
	}
	if err := keyValue.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, err
	}
	if err := keyInfo.AddChild(keyValue); err != nil {
		return nil, err
	}

	rsaKV, err := doc.CreateElement("RSAKeyValue")
	if err != nil {
		return nil, err
	}
	if err := rsaKV.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, err
	}
	if err := keyValue.AddChild(rsaKV); err != nil {
		return nil, err
	}

	modElem, err := doc.CreateElement("Modulus")
	if err != nil {
		return nil, err
	}
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

	expElem, err := doc.CreateElement("Exponent")
	if err != nil {
		return nil, err
	}
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
		// Core KeyInfo children (X509Data, KeyValue) live only in the core
		// XML-Signature namespace. Matching on local name alone would let a
		// foreign-namespace look-alike (e.g. <evil:X509Data>) supply an
		// attacker-chosen verification key, so require the core namespace. The
		// 1.1 xmldsig11# namespace is for new 1.1 elements and must not satisfy
		// this check.
		if !isDSigCoreNS(elem) {
			continue
		}
		switch domutil.LocalName(elem) {
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
		// X509Certificate is a core XML-Signature element; a foreign-namespace
		// look-alike must not supply a verification certificate.
		if !isDSigCoreNS(certElem) {
			continue
		}
		if domutil.LocalName(certElem) != "X509Certificate" {
			continue
		}
		derBytes, err := xmlbase64.DecodeString(domutil.TextContent(certElem))
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
		switch domutil.LocalName(kvElem) {
		case "RSAKeyValue":
			// RSAKeyValue is a core XML-Signature element; reject
			// foreign-namespace look-alikes.
			if !isDSigCoreNS(kvElem) {
				continue
			}
			return parseRSAKeyValue(kvElem, data)
		case "ECKeyValue":
			// ECKeyValue is an XML-Signature 1.1 element, so it lives in the
			// xmldsig11# namespace rather than the core namespace. Require that
			// exact namespace and reject foreign-namespace look-alikes.
			if !isDSig11NS(kvElem) {
				continue
			}
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
		// Modulus and Exponent are core XML-Signature elements; reject
		// foreign-namespace look-alikes before consuming their base64 content.
		if !isDSigCoreNS(e) {
			continue
		}
		decoded, err := xmlbase64.DecodeString(domutil.TextContent(e))
		if err != nil {
			return fmt.Errorf("%w: invalid RSAKeyValue base64: %v", ErrInvalidKeyInfo, err)
		}
		switch domutil.LocalName(e) {
		case "Modulus":
			kv.Modulus = new(big.Int).SetBytes(decoded)
		case "Exponent":
			exp := new(big.Int).SetBytes(decoded)
			const maxInt = int(^uint(0) >> 1)
			if exp.Sign() <= 0 || !exp.IsInt64() || exp.Int64() > int64(maxInt) {
				return fmt.Errorf("%w: RSAKeyValue Exponent out of range", ErrInvalidKeyInfo)
			}
			kv.Exponent = int(exp.Int64())
		}
	}
	// An RSAKeyValue requires BOTH Modulus and Exponent (XML-Signature core).
	// Refuse to emit a partial key (e.g. Exponent-only, or a Modulus whose
	// element was skipped as a foreign-namespace look-alike): such material must
	// never reach the KeySource.
	if kv.Modulus == nil || kv.Exponent == 0 {
		return fmt.Errorf("%w: RSAKeyValue requires both Modulus and Exponent", ErrInvalidKeyInfo)
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
		// NamedCurve and PublicKey are XML-Signature 1.1 elements; require the
		// xmldsig11# namespace and reject foreign-namespace look-alikes.
		if !isDSig11NS(e) {
			continue
		}
		switch domutil.LocalName(e) {
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
			decoded, err := xmlbase64.DecodeString(domutil.TextContent(e))
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
	// An ECKeyValue requires both the NamedCurve and the PublicKey point
	// (XML-Signature 1.1). Refuse to emit a partial key (e.g. NamedCurve-only):
	// such material must never reach the KeySource.
	if kv.Curve == nil || kv.X == nil || kv.Y == nil {
		return fmt.Errorf("%w: ECKeyValue requires both NamedCurve and PublicKey", ErrInvalidKeyInfo)
	}
	data.ECKeyValue = kv
	return nil
}

// localName returns the local name of an element, stripping any prefix.
