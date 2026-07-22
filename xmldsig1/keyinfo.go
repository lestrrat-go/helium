package xmldsig1

import (
	"bytes"
	"context"
	"crypto"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"math/big"
	"slices"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/xmlbase64"
)

// KeySource provides keys for signature verification.
//
// SECURITY: the keyInfo passed to ResolveKey is parsed from the document's
// ds:KeyInfo BEFORE the signature is verified, so it is entirely
// attacker-controlled (see [KeyInfoData]). A KeySource MUST decide trust itself:
// select the verification key by matching keyInfo against a trust store, a
// pinned key, or a validated certificate chain. It MUST NOT blindly return an
// embedded X509Certificate's public key or a KeyValue as the verification key —
// doing so lets an attacker present a signature made with their own key and have
// it verify. keyInfo is a selector into trusted key material, never the key
// material itself. [StaticKey] and [X509CertKeySource] ignore keyInfo and return
// a key the caller already trusts, which is the safe pattern; a custom KeySource
// that consults keyInfo carries the trust decision.
type KeySource interface {
	// ResolveKey returns the verification key for a signature. keyInfo is the
	// document's parsed, UNTRUSTED KeyInfo (nil when the Signature carries no
	// KeyInfo); alg is the SignatureMethod algorithm URI. See the [KeySource]
	// contract: match keyInfo against trusted material rather than trusting it.
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

// KeyByNameSource returns a KeySource that maps a ds:KeyName to a key. The
// KeyInfo's KeyNames are tried in document order and the first name present in
// keys wins; a KeyInfo with no matching KeyName (including one that carries no
// KeyName at all) fails closed with ErrNoKeySource. A ds:KeyName is an opaque,
// producer-chosen label, so the caller owns the name→key mapping and the trust
// decision that a named key is acceptable.
func KeyByNameSource(keys map[string]any) KeySource {
	return KeySourceFunc(func(_ context.Context, keyInfo *KeyInfoData, _ string) (any, error) {
		if keyInfo == nil {
			return nil, ErrNoKeySource
		}
		for _, name := range keyInfo.KeyNames {
			key, ok := keys[name]
			if ok {
				return key, nil
			}
		}
		return nil, ErrNoKeySource
	})
}

// X509CertPoolKeySource returns a KeySource that selects a certificate from the
// given set by matching the verification-side KeyInfo against it, and returns
// the matched certificate's PublicKey. Selector strength is applied across the
// WHOLE pool, strongest first, so a strong match on a later certificate is never
// masked by a weak match on an earlier one:
//
//   - first, an exact raw-DER match against a ds:X509Certificate in the KeyInfo,
//     over every certificate in the pool;
//   - then a ds:X509SKI match against the certificate's SubjectKeyId;
//   - then a ds:X509IssuerSerial match against the certificate's Issuer and
//     SerialNumber;
//   - finally a ds:X509SubjectName match against the certificate's Subject.
//
// Pool order is preserved only within a single selector class. The raw-DER and
// SubjectKeyId paths are exact and reliable. The IssuerSerial and SubjectName
// paths compare Go's pkix.Name.String() rendering, which is NOT RFC 2253
// canonical, so DName matching is best-effort — prefer supplying the certificate
// whose SKI or raw bytes the signature carries. Selecting a certificate never
// establishes trust: the returned public key is still subject to the same
// out-of-band trust decision as any other verification key.
func X509CertPoolKeySource(certs ...*x509.Certificate) KeySource {
	return KeySourceFunc(func(_ context.Context, keyInfo *KeyInfoData, _ string) (any, error) {
		if keyInfo == nil {
			return nil, ErrNoKeySource
		}
		// Try the strongest selector class against every certificate before
		// falling to the next weaker one, so pool order only breaks ties within a
		// class. Iterating each certificate through all classes first (the naive
		// nesting) would instead let an earlier certificate's weak SubjectName
		// match win over a later certificate's exact raw-DER match.
		selectors := []func(*x509.Certificate, *KeyInfoData) bool{
			certMatchesRawDER, certMatchesSKI, certMatchesIssuerSerial, certMatchesSubjectName,
		}
		for _, matches := range selectors {
			for _, cert := range certs {
				if cert == nil {
					continue
				}
				if matches(cert, keyInfo) {
					return cert.PublicKey, nil
				}
			}
		}
		return nil, ErrNoKeySource
	})
}

// certMatchesRawDER reports whether the KeyInfo carries cert's exact DER bytes in
// a ds:X509Certificate. This is the strongest, exact selector.
func certMatchesRawDER(cert *x509.Certificate, keyInfo *KeyInfoData) bool {
	for _, kc := range keyInfo.X509Certificates {
		if kc != nil && bytes.Equal(kc.Raw, cert.Raw) {
			return true
		}
	}
	return false
}

// certMatchesSKI reports whether a ds:X509SKI matches cert's SubjectKeyId. Exact.
func certMatchesSKI(cert *x509.Certificate, keyInfo *KeyInfoData) bool {
	for _, ski := range keyInfo.X509SKIs {
		if len(cert.SubjectKeyId) > 0 && bytes.Equal(ski, cert.SubjectKeyId) {
			return true
		}
	}
	return false
}

// certMatchesIssuerSerial reports whether a ds:X509IssuerSerial matches cert's
// Issuer and SerialNumber. Best-effort: the issuer DName comparison uses Go's
// pkix.Name.String() rendering.
func certMatchesIssuerSerial(cert *x509.Certificate, keyInfo *KeyInfoData) bool {
	for _, is := range keyInfo.X509IssuerSerials {
		if cert.SerialNumber != nil && is.SerialNumber != nil &&
			cert.SerialNumber.Cmp(is.SerialNumber) == 0 &&
			cert.Issuer.String() == is.IssuerName {
			return true
		}
	}
	return false
}

// certMatchesSubjectName reports whether a ds:X509SubjectName matches cert's
// Subject. Best-effort: the DName comparison uses Go's pkix.Name.String()
// rendering, so it is the weakest selector.
func certMatchesSubjectName(cert *x509.Certificate, keyInfo *KeyInfoData) bool {
	return slices.Contains(keyInfo.X509SubjectNames, cert.Subject.String())
}

// KeyInfoData holds parsed KeyInfo content for verification.
//
// SECURITY: every field is parsed from the document's ds:KeyInfo, which is
// attacker-controlled and NOT authenticated by the signature — KeyInfo is
// resolved before the signature is checked. Treat these values as untrusted
// hints for selecting a key from trusted material (a trust store, a pinned key,
// a validated chain), never as the key material to verify with. In particular an
// embedded X509Certificate is not proof of anything on its own: an attacker can
// embed a certificate for a key they control. See the [KeySource] contract.
type KeyInfoData struct {
	KeyNames          []string
	X509Certificates  []*x509.Certificate
	X509SKIs          [][]byte
	X509IssuerSerials []*X509IssuerSerial
	X509SubjectNames  []string
	RSAKeyValue       *RSAKeyValueData
	ECKeyValue        *ECKeyValueData
	DSAKeyValue       *DSAKeyValueData
}

// X509IssuerSerial holds a parsed ds:X509IssuerSerial: the issuer distinguished
// name and certificate serial number. The library performs no DName
// canonicalization or matching; it extracts the values verbatim so a KeySource
// can select the corresponding certificate out of band.
type X509IssuerSerial struct {
	IssuerName   string
	SerialNumber *big.Int
}

// DSAKeyValueData holds parsed DSAKeyValue content (the P, Q, G, Y
// CryptoBinary parameters). A KeySource builds a *dsa.PublicKey from these.
type DSAKeyValueData struct {
	P, Q, G, Y *big.Int
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

// validate reports whether the configured certificate list can build a
// schema-valid <X509Data>: it must be non-empty (an empty list would emit a
// schema-invalid empty element) and contain no nil entry (a nil
// *x509.Certificate would panic on cert.Raw below). Shared by BuildKeyInfo and
// the SignEnveloping preflight so both reject the same inputs — the preflight
// before any caller content is moved into the <Object>, BuildKeyInfo as the
// single source of truth on every signing path.
func (b *x509DataKeyInfo) validate() error {
	if len(b.certs) == 0 {
		return fmt.Errorf("%w: X509DataKeyInfo requires at least one certificate", ErrInvalidKeyInfo)
	}
	for i, cert := range b.certs {
		if cert == nil {
			return fmt.Errorf("%w: X509DataKeyInfo certificate[%d] is nil", ErrInvalidKeyInfo, i)
		}
	}
	return nil
}

func (b *x509DataKeyInfo) BuildKeyInfo(_ context.Context, doc *helium.Document, _ any) (*helium.Element, error) {
	if err := b.validate(); err != nil {
		return nil, err
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
		// Fall back to an opaque crypto.Signer (HSM/KMS/PKCS#11) whose concrete
		// type is not *rsa.*, mirroring the signing path in signRSA. Read its
		// public key and require an *rsa.PublicKey so the emitted RSAKeyValue
		// matches the key that produced the signature.
		signer, ok := key.(crypto.Signer)
		if !ok {
			return nil, fmt.Errorf("%w: expected RSA key, got %T", ErrKeyMismatch, key)
		}
		rsaPub, ok := signer.Public().(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("%w: crypto.Signer public key is %T, not *rsa.PublicKey", ErrKeyMismatch, signer.Public())
		}
		pub = rsaPub
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

// parseKeyInfo extracts key information from a ds:KeyInfo element. budget bounds
// the parse-time work an attacker-controlled KeyInfo can force (a per-cert
// x509.ParseCertificate for every embedded certificate, plus base64 decodes) and
// carries ctx for in-loop cancellation polling.
func parseKeyInfo(ctx context.Context, budget *verifyBudget, keyInfoElem *helium.Element) (*KeyInfoData, error) {
	data := &KeyInfoData{}
	for child := keyInfoElem.FirstChild(); child != nil; child = child.NextSibling() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
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
		// Count each core KeyInfo child so a KeyInfo stuffed with many entries is
		// rejected before it drives repeated key-material parsing.
		if err := budget.addKeyInfoEntry(); err != nil {
			return nil, err
		}
		switch domutil.LocalName(elem) {
		case "KeyName":
			// A ds:KeyName is an opaque producer-chosen label; surface it verbatim
			// (whitespace-trimmed) for a KeySource to map to a key.
			data.KeyNames = append(data.KeyNames, strings.TrimSpace(domutil.TextContent(elem)))
		case "X509Data":
			if err := parseX509Data(ctx, budget, elem, data); err != nil {
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

func parseX509Data(ctx context.Context, budget *verifyBudget, elem *helium.Element, data *KeyInfoData) error {
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if err := ctx.Err(); err != nil {
			return err
		}
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		// X509Data children are core XML-Signature elements; a foreign-namespace
		// look-alike must not supply a verification certificate or selector.
		if !isDSigCoreNS(e) {
			continue
		}
		// Count each X509Data child (X509Certificate/X509SubjectName/
		// X509IssuerSerial) so a flood of certificates cannot drive an unbounded
		// number of x509.ParseCertificate calls.
		if err := budget.addKeyInfoEntry(); err != nil {
			return err
		}
		switch domutil.LocalName(e) {
		case "X509Certificate":
			derBytes, err := xmlbase64.DecodeString(domutil.TextContent(e))
			if err != nil {
				return fmt.Errorf("%w: invalid X509Certificate base64: %v", ErrInvalidKeyInfo, err)
			}
			if err := budget.consume(len(derBytes)); err != nil {
				return err
			}
			cert, err := x509.ParseCertificate(derBytes)
			if err != nil {
				return fmt.Errorf("%w: invalid X509Certificate: %v", ErrInvalidKeyInfo, err)
			}
			data.X509Certificates = append(data.X509Certificates, cert)
		case "X509SKI":
			// The X509SKI carries the raw DER SubjectKeyIdentifier octets,
			// base64-encoded. Decode to the raw bytes so a KeySource can compare
			// them against a certificate's SubjectKeyId; a decode error is
			// malformed key material and fails closed.
			ski, err := xmlbase64.DecodeString(domutil.TextContent(e))
			if err != nil {
				return fmt.Errorf("%w: invalid X509SKI base64: %v", ErrInvalidKeyInfo, err)
			}
			data.X509SKIs = append(data.X509SKIs, ski)
		case "X509SubjectName":
			data.X509SubjectNames = append(data.X509SubjectNames, domutil.TextContent(e))
		case "X509IssuerSerial":
			is, err := parseX509IssuerSerial(e)
			if err != nil {
				return err
			}
			data.X509IssuerSerials = append(data.X509IssuerSerials, is)
		}
	}
	return nil
}

// parseX509IssuerSerial extracts the issuer distinguished name and serial number
// from a ds:X509IssuerSerial. The values are extracted verbatim (no DName
// canonicalization) for out-of-band certificate selection by a KeySource.
func parseX509IssuerSerial(elem *helium.Element) (*X509IssuerSerial, error) {
	is := &X509IssuerSerial{}
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		// X509IssuerName and X509SerialNumber are core XML-Signature elements;
		// reject foreign-namespace look-alikes.
		if !isDSigCoreNS(e) {
			continue
		}
		switch domutil.LocalName(e) {
		case "X509IssuerName":
			is.IssuerName = domutil.TextContent(e)
		case "X509SerialNumber":
			serial, ok := new(big.Int).SetString(strings.TrimSpace(domutil.TextContent(e)), 10)
			if !ok {
				return nil, fmt.Errorf("%w: X509SerialNumber is not a decimal integer", ErrInvalidKeyInfo)
			}
			is.SerialNumber = serial
		}
	}
	if is.IssuerName == "" || is.SerialNumber == nil {
		return nil, fmt.Errorf("%w: X509IssuerSerial requires both X509IssuerName and X509SerialNumber", ErrInvalidKeyInfo)
	}
	return is, nil
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
		case "ECDSAKeyValue":
			// RFC 4050 legacy ECDSAKeyValue lives in the xmldsig-more#
			// namespace. Require that exact namespace and reject
			// foreign-namespace look-alikes.
			if !isDSigMoreNS(kvElem) {
				continue
			}
			return parseRFC4050ECDSAKeyValue(kvElem, data)
		case "DSAKeyValue":
			// DSAKeyValue is a core XML-Signature element; reject
			// foreign-namespace look-alikes.
			if !isDSigCoreNS(kvElem) {
				continue
			}
			return parseDSAKeyValue(kvElem, data)
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
			curve, err := curveForOID(uri)
			if err != nil {
				return err
			}
			kv.Curve = curve
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

// curveForOID maps a NamedCurve OID URN (used by both the XML-Signature 1.1
// ECKeyValue NamedCurve@URI and the RFC 4050 ECDSAKeyValue NamedCurve@URN) to a
// supported elliptic curve, rejecting an unrecognized curve with
// ErrInvalidKeyInfo so unknown key material never reaches the KeySource.
func curveForOID(oid string) (elliptic.Curve, error) {
	switch oid {
	case "urn:oid:1.2.840.10045.3.1.7":
		return elliptic.P256(), nil
	case "urn:oid:1.3.132.0.34":
		return elliptic.P384(), nil
	case "urn:oid:1.3.132.0.35":
		return elliptic.P521(), nil
	default:
		return nil, fmt.Errorf("%w: unsupported EC curve: %s", ErrInvalidKeyInfo, oid)
	}
}

// parseRFC4050ECDSAKeyValue parses an RFC 4050 ECDSAKeyValue (in the
// xmldsig-more# namespace) into an ECKeyValueData, so an RFC 4050 key surfaces
// through the same KeyInfoData.ECKeyValue as an XML-Signature 1.1 ECKeyValue.
// The curve comes from DomainParameters/NamedCurve@URN and the point from
// PublicKey/X and /Y, whose Value attributes are DECIMAL integer strings (RFC
// 4050 §2). This is verification input only; RFC 4050 emission is not supported.
func parseRFC4050ECDSAKeyValue(elem *helium.Element, data *KeyInfoData) error {
	kv := &ECKeyValueData{}
	var x, y *big.Int
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		// DomainParameters and PublicKey are RFC 4050 xmldsig-more# elements;
		// reject foreign-namespace look-alikes.
		if !isDSigMoreNS(e) {
			continue
		}
		switch domutil.LocalName(e) {
		case "DomainParameters":
			curve, err := parseRFC4050NamedCurve(e)
			if err != nil {
				return err
			}
			kv.Curve = curve
		case "PublicKey":
			px, py, err := parseRFC4050PublicKey(e)
			if err != nil {
				return err
			}
			x, y = px, py
		}
	}
	// An RFC 4050 ECDSAKeyValue requires both the curve and the public-key
	// point; refuse to emit a partial key.
	if kv.Curve == nil {
		return fmt.Errorf("%w: RFC 4050 ECDSAKeyValue missing DomainParameters/NamedCurve", ErrInvalidKeyInfo)
	}
	if x == nil || y == nil {
		return fmt.Errorf("%w: RFC 4050 ECDSAKeyValue missing PublicKey X/Y", ErrInvalidKeyInfo)
	}
	// Reject a point that is not on the named curve so invalid key material
	// never reaches the KeySource, mirroring elliptic.Unmarshal's on-curve check
	// for the 1.1 ECKeyValue path.
	if !kv.Curve.IsOnCurve(x, y) {
		return fmt.Errorf("%w: RFC 4050 ECDSAKeyValue public key point is not on the named curve", ErrInvalidKeyInfo)
	}
	kv.X, kv.Y = x, y
	data.ECKeyValue = kv
	return nil
}

// parseRFC4050NamedCurve resolves the curve from an RFC 4050 DomainParameters
// element via its NamedCurve@URN. Explicit (non-named) domain parameters are
// not supported.
func parseRFC4050NamedCurve(elem *helium.Element) (elliptic.Curve, error) {
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if !isDSigMoreNS(e) || domutil.LocalName(e) != "NamedCurve" {
			continue
		}
		urn, _ := e.GetAttribute("URN")
		return curveForOID(urn)
	}
	return nil, fmt.Errorf("%w: RFC 4050 DomainParameters missing NamedCurve", ErrInvalidKeyInfo)
}

// parseRFC4050PublicKey reads the decimal X and Y Value attributes from an RFC
// 4050 PublicKey element.
func parseRFC4050PublicKey(elem *helium.Element) (*big.Int, *big.Int, error) {
	var x, y *big.Int
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if !isDSigMoreNS(e) {
			continue
		}
		name := domutil.LocalName(e)
		if name != "X" && name != "Y" {
			continue
		}
		val, _ := e.GetAttribute("Value")
		n, ok := new(big.Int).SetString(strings.TrimSpace(val), 10)
		if !ok {
			return nil, nil, fmt.Errorf("%w: RFC 4050 PublicKey %s Value is not a decimal integer", ErrInvalidKeyInfo, name)
		}
		if name == "X" {
			x = n
			continue
		}
		y = n
	}
	return x, y, nil
}

// parseDSAKeyValue parses a core-namespace DSAKeyValue into a DSAKeyValueData.
// P, Q, G, and Y are base64 CryptoBinary values; the optional J, Seed, and
// PgenCounter are not needed for verification and are ignored. A KeySource
// builds a *dsa.PublicKey from the result. DSA is verify-only legacy interop.
func parseDSAKeyValue(elem *helium.Element, data *KeyInfoData) error {
	kv := &DSAKeyValueData{}
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		// P, Q, G, Y are core XML-Signature elements; reject foreign-namespace
		// look-alikes before consuming their base64 content.
		if !isDSigCoreNS(e) {
			continue
		}
		name := domutil.LocalName(e)
		var dst **big.Int
		switch name {
		case "P":
			dst = &kv.P
		case "Q":
			dst = &kv.Q
		case "G":
			dst = &kv.G
		case "Y":
			dst = &kv.Y
		default:
			continue
		}
		decoded, err := xmlbase64.DecodeString(domutil.TextContent(e))
		if err != nil {
			return fmt.Errorf("%w: invalid DSAKeyValue %s base64: %v", ErrInvalidKeyInfo, name, err)
		}
		*dst = new(big.Int).SetBytes(decoded)
	}
	// A DSAKeyValue requires P, Q, G, and Y; refuse to emit a partial key.
	if kv.P == nil || kv.Q == nil || kv.G == nil || kv.Y == nil {
		return fmt.Errorf("%w: DSAKeyValue requires P, Q, G, and Y", ErrInvalidKeyInfo)
	}
	data.DSAKeyValue = kv
	return nil
}

// localName returns the local name of an element, stripping any prefix.
