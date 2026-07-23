package xmldsig1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"
)

// certSignerDERPath is the fs path a test FSReferenceResolver serves and the
// matching ds:RetrievalMethod URI references.
const certSignerDERPath = "certs/signer.der"

// selfSignedCert returns a throwaway self-signed certificate and its DER bytes.
func selfSignedCert(t *testing.T) (*x509.Certificate, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(12345),
		Subject:      pkix.Name{CommonName: "retrieval-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert, der
}

func TestResolveRetrievalMethodExternalRawX509(t *testing.T) {
	cert, der := selfSignedCert(t)
	fsys := fstest.MapFS{certSignerDERPath: {Data: der}}

	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="certs/signer.der" Type="`+TypeRawX509Certificate+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{referenceResolver: FSReferenceResolver(fsys)}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
	require.NoError(t, err)
	require.Len(t, data.X509Certificates, 1)
	require.Equal(t, cert.Raw, data.X509Certificates[0].Raw)
}

func TestResolveRetrievalMethodExternalX509Data(t *testing.T) {
	cert, der := selfSignedCert(t)
	x509Data := `<ds:X509Data xmlns:ds="` + NamespaceDSig + `"><ds:X509Certificate>` +
		base64.StdEncoding.EncodeToString(der) + `</ds:X509Certificate></ds:X509Data>`
	fsys := fstest.MapFS{"keyinfo/data.xml": {Data: []byte(x509Data)}}

	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="keyinfo/data.xml" Type="`+TypeX509Data+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{referenceResolver: FSReferenceResolver(fsys)}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
	require.NoError(t, err)
	require.Len(t, data.X509Certificates, 1)
	require.Equal(t, cert.Raw, data.X509Certificates[0].Raw)
}

func TestResolveRetrievalMethodNoResolverFailsClosed(t *testing.T) {
	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="certs/signer.der" Type="`+TypeRawX509Certificate+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
	require.ErrorIs(t, err, ErrReferenceNotFound)
	require.Empty(t, data.X509Certificates)
}

// retrievalChainKeyInfo builds a ds:KeyInfo whose top-level ds:RetrievalMethod
// starts a chain of n RetrievalMethods, each pointing at the next by same-document
// id, with the final link resolving a terminal ds:X509Data certificate. The first
// link is a top-level KeyInfo child; the remaining links and the terminal
// X509Data live inside a ds:Object so only one RetrievalMethod is a direct KeyInfo
// child, matching the verify-time layout. n counts the RetrievalMethod links: the
// top-level RetrievalMethod is processed at depth 0, so a chain of n links reaches
// depth n-1.
func retrievalChainKeyInfo(n int, certB64 string) string {
	var b strings.Builder
	b.WriteString(`<ds:KeyInfo xmlns:ds="` + NamespaceDSig + `">`)
	firstTarget := "#cert"
	if n > 1 {
		firstTarget = "#rm2"
	}
	b.WriteString(`<ds:RetrievalMethod URI="` + firstTarget + `" Type="` + TypeX509Data + `"/>`)
	b.WriteString(`<ds:Object>`)
	for i := 2; i <= n; i++ {
		target := "#cert"
		if i < n {
			target = "#rm" + strconv.Itoa(i+1)
		}
		b.WriteString(`<ds:RetrievalMethod Id="rm` + strconv.Itoa(i) + `" URI="` + target + `" Type="` + TypeX509Data + `"/>`)
	}
	b.WriteString(`<ds:X509Data Id="cert"><ds:X509Certificate>` + certB64 + `</ds:X509Certificate></ds:X509Data>`)
	b.WriteString(`</ds:Object></ds:KeyInfo>`)
	return b.String()
}

// TestRetrievalMethodChainDepthCap pins the ds:RetrievalMethod chain depth cap at
// exactly maxRetrievalMethodDepth (5) links: a chain of five RetrievalMethods
// resolves its terminal certificate, while a chain of six fails closed with
// ErrRetrievalMethodLoop before the sixth link is dereferenced. The top-level
// RetrievalMethod enters at depth 0 and link N is processed at depth N-1, so the
// depth >= maxRetrievalMethodDepth guard must reject the sixth link (depth 5) and
// admit the fifth (depth 4).
func TestRetrievalMethodChainDepthCap(t *testing.T) {
	cert, der := selfSignedCert(t)
	certB64 := base64.StdEncoding.EncodeToString(der)

	t.Run("five links succeed", func(t *testing.T) {
		doc := mustParse(t, retrievalChainKeyInfo(5, certB64))
		cfg := &verifierConfig{}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
		require.NoError(t, err)
		require.Len(t, data.X509Certificates, 1)
		require.Equal(t, cert.Raw, data.X509Certificates[0].Raw)
	})

	t.Run("six links fail", func(t *testing.T) {
		doc := mustParse(t, retrievalChainKeyInfo(6, certB64))
		cfg := &verifierConfig{}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
		require.ErrorIs(t, err, ErrRetrievalMethodLoop)
		require.Empty(t, data.X509Certificates)
	})
}

func TestResolveRetrievalMethodLoopRejected(t *testing.T) {
	// A RetrievalMethod that references itself by id must fail closed rather than
	// dereferencing an unbounded chain.
	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod Id="rm1" URI="#rm1" Type="`+TypeX509Data+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
	require.ErrorIs(t, err, ErrRetrievalMethodLoop)
}

func TestResolveRetrievalMethodForeignNamespaceIgnored(t *testing.T) {
	// A foreign-namespace RetrievalMethod look-alike must not steer key retrieval,
	// even with no resolver configured (it must simply be skipped).
	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`" xmlns:evil="urn:evil"><evil:RetrievalMethod URI="certs/signer.der" Type="`+TypeRawX509Certificate+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
	require.NoError(t, err)
	require.Empty(t, data.X509Certificates)
}

// oversizeResolver returns one byte more than the 64 MiB resolver cap, standing
// in for a caller-supplied ReferenceResolver that ignores the cap the shipped
// FSReferenceResolver enforces on itself.
type oversizeResolver struct{}

func (oversizeResolver) ResolveReference(_ context.Context, _ string) ([]byte, error) {
	return make([]byte, maxReferenceBytes+1), nil
}

// TestRetrievalMethodCapsCustomResolverResult proves the 64 MiB cap is enforced
// at the RetrievalMethod resolution site, not only inside FSReferenceResolver: a
// custom resolver returning an over-cap result fails closed with
// ErrReferenceTooLarge before the octets reach x509.ParseCertificate.
func TestRetrievalMethodCapsCustomResolverResult(t *testing.T) {
	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="certs/big.der" Type="`+TypeRawX509Certificate+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{referenceResolver: oversizeResolver{}}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
	require.ErrorIs(t, err, ErrReferenceTooLarge)
	require.Empty(t, data.X509Certificates)
}

// TestRetrievalMethodResolvesObjectInsideVerifiedSignature proves a same-document
// RetrievalMethod resolves an X509Data carried inside the Signature's own Object
// when the Signature is attached beneath the document (the verify-time layout).
// Passing the attached Signature as an extra resolution root would double-count
// the target — once through the document walk and again through the Signature
// subtree — and fail spuriously with ErrAmbiguousReference; resolving against the
// document only finds it exactly once.
func TestRetrievalMethodResolvesObjectInsideVerifiedSignature(t *testing.T) {
	cert, der := selfSignedCert(t)
	certB64 := base64.StdEncoding.EncodeToString(der)
	doc := mustParse(t, `<Root xmlns:ds="`+NamespaceDSig+`"><ds:Signature><ds:KeyInfo Id="ki">`+
		`<ds:RetrievalMethod URI="#cert-data" Type="`+TypeX509Data+`"/></ds:KeyInfo>`+
		`<ds:Object><ds:X509Data Id="cert-data"><ds:X509Certificate>`+certB64+
		`</ds:X509Certificate></ds:X509Data></ds:Object></ds:Signature></Root>`)
	kis := findElementsByIDUnder(doc.DocumentElement(), "ki")
	require.Len(t, kis, 1)
	cfg := &verifierConfig{}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, kis[0], data)
	require.NoError(t, err)
	require.Len(t, data.X509Certificates, 1)
	require.Equal(t, cert.Raw, data.X509Certificates[0].Raw)
}

// TestReviewSiblingRetrievalMethodsSharingURI proves two independent top-level
// ds:RetrievalMethod siblings may target the same same-document URI without the
// second being misreported as a loop: the visited-URI set is scoped per
// top-level chain, so each sibling resolves its shared #cert target and both
// certificates are collected.
func TestReviewSiblingRetrievalMethodsSharingURI(t *testing.T) {
	cert, der := selfSignedCert(t)
	certB64 := base64.StdEncoding.EncodeToString(der)
	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`">`+
		`<ds:RetrievalMethod URI="#cert" Type="`+TypeX509Data+`"/>`+
		`<ds:RetrievalMethod URI="#cert" Type="`+TypeX509Data+`"/>`+
		`<ds:X509Data Id="cert"><ds:X509Certificate>`+certB64+
		`</ds:X509Certificate></ds:X509Data></ds:KeyInfo>`)
	cfg := &verifierConfig{}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
	require.NoError(t, err)
	require.Len(t, data.X509Certificates, 2)
	require.Equal(t, cert.Raw, data.X509Certificates[0].Raw)
	require.Equal(t, cert.Raw, data.X509Certificates[1].Raw)
}

// TestReviewUnsupportedWrapperTypeFailsClosed proves a top-level
// ds:RetrievalMethod whose own Type is unsupported (advisory but not one of the
// recognized forms) fails closed even when its URI points at a chained
// RetrievalMethod that would otherwise resolve to a valid certificate. Without
// validating the wrapper's Type before the target-RetrievalMethod recursion, the
// unsupported wrapper Type would be silently accepted. An absent wrapper Type
// stays permissive, so the same chain resolves when the wrapper carries no Type.
func TestReviewUnsupportedWrapperTypeFailsClosed(t *testing.T) {
	cert, der := selfSignedCert(t)
	certB64 := base64.StdEncoding.EncodeToString(der)

	// The chained target RetrievalMethod and the terminal X509Data live inside a
	// ds:Object so only the wrapper is a top-level KeyInfo child; resolveReference
	// still finds them by id anywhere in the document.
	t.Run("unsupported wrapper type rejected", func(t *testing.T) {
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`">`+
			`<ds:RetrievalMethod URI="#next" Type="urn:unsupported"/>`+
			`<ds:Object><ds:RetrievalMethod Id="next" URI="#cert" Type="`+TypeX509Data+`"/>`+
			`<ds:X509Data Id="cert"><ds:X509Certificate>`+certB64+
			`</ds:X509Certificate></ds:X509Data></ds:Object></ds:KeyInfo>`)
		cfg := &verifierConfig{}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
		require.Empty(t, data.X509Certificates)
	})

	t.Run("absent wrapper type still follows chain", func(t *testing.T) {
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`">`+
			`<ds:RetrievalMethod URI="#next"/>`+
			`<ds:Object><ds:RetrievalMethod Id="next" URI="#cert" Type="`+TypeX509Data+`"/>`+
			`<ds:X509Data Id="cert"><ds:X509Certificate>`+certB64+
			`</ds:X509Certificate></ds:X509Data></ds:Object></ds:KeyInfo>`)
		cfg := &verifierConfig{}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
		require.NoError(t, err)
		require.Len(t, data.X509Certificates, 1)
		require.Equal(t, cert.Raw, data.X509Certificates[0].Raw)
	})
}

// TestRetrievalMethodRejectsUnsupportedTransform proves a RetrievalMethod's
// ds:Transforms are inspected and applied, not ignored: an unsupported transform
// fails closed with ErrUnsupportedTransform for both external and same-document
// targets, rather than silently accepting the resolved certificate.
func TestRetrievalMethodRejectsUnsupportedTransform(t *testing.T) {
	t.Run("external", func(t *testing.T) {
		_, der := selfSignedCert(t)
		fsys := fstest.MapFS{certSignerDERPath: {Data: der}}
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="certs/signer.der" Type="`+TypeRawX509Certificate+`">`+
			`<ds:Transforms><ds:Transform Algorithm="urn:unsupported"/></ds:Transforms></ds:RetrievalMethod></ds:KeyInfo>`)
		cfg := &verifierConfig{referenceResolver: FSReferenceResolver(fsys)}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
		require.Empty(t, data.X509Certificates)
	})

	t.Run("same-document", func(t *testing.T) {
		_, der := selfSignedCert(t)
		certB64 := base64.StdEncoding.EncodeToString(der)
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="#cert" Type="`+TypeRawX509Certificate+`">`+
			`<ds:Transforms><ds:Transform Algorithm="urn:unsupported"/></ds:Transforms></ds:RetrievalMethod>`+
			`<ds:SPKIData Id="cert">`+certB64+`</ds:SPKIData></ds:KeyInfo>`)
		cfg := &verifierConfig{}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
		require.Empty(t, data.X509Certificates)
	})
}

// TestRetrievalMethodAppliesSupportedTransform proves a supported transform
// pipeline is applied before Type interpretation: a same-document c14n transform
// canonicalizes the target X509Data subtree before it is reparsed, and an
// external base64 transform decodes the retrieved octets before the certificate
// is parsed.
func TestRetrievalMethodAppliesSupportedTransform(t *testing.T) {
	t.Run("same-document c14n", func(t *testing.T) {
		cert, der := selfSignedCert(t)
		certB64 := base64.StdEncoding.EncodeToString(der)
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="#x509" Type="`+TypeX509Data+`">`+
			`<ds:Transforms><ds:Transform Algorithm="`+C14N10+`"/></ds:Transforms></ds:RetrievalMethod>`+
			`<ds:X509Data Id="x509"><ds:X509Certificate>`+certB64+`</ds:X509Certificate></ds:X509Data></ds:KeyInfo>`)
		cfg := &verifierConfig{}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
		require.NoError(t, err)
		require.Len(t, data.X509Certificates, 1)
		require.Equal(t, cert.Raw, data.X509Certificates[0].Raw)
	})

	t.Run("external base64", func(t *testing.T) {
		cert, der := selfSignedCert(t)
		b64File := base64.StdEncoding.EncodeToString(der)
		fsys := fstest.MapFS{"certs/signer.b64": {Data: []byte(b64File)}}
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="certs/signer.b64" Type="`+TypeRawX509Certificate+`">`+
			`<ds:Transforms><ds:Transform Algorithm="`+TransformBase64+`"/></ds:Transforms></ds:RetrievalMethod></ds:KeyInfo>`)
		cfg := &verifierConfig{referenceResolver: FSReferenceResolver(fsys)}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
		require.NoError(t, err)
		require.Len(t, data.X509Certificates, 1)
		require.Equal(t, cert.Raw, data.X509Certificates[0].Raw)
	})

	t.Run("same-document multiphase", func(t *testing.T) {
		cert, der := selfSignedCert(t)
		certB64 := base64.StdEncoding.EncodeToString(der)
		transforms := `<ds:Transforms>` +
			`<ds:Transform Algorithm="` + C14N10 + `"/>` +
			`<ds:Transform Algorithm="` + TransformXSLT + `"><xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0"/></ds:Transform>` +
			`<ds:Transform Algorithm="` + TransformXPath + `"><ds:XPath>true()</ds:XPath></ds:Transform>` +
			`<ds:Transform Algorithm="` + C14N10 + `"/>` +
			`</ds:Transforms>`
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="#x509" Type="`+TypeX509Data+`">`+
			transforms+`</ds:RetrievalMethod><ds:X509Data Id="x509"><ds:X509Certificate>`+certB64+`</ds:X509Certificate></ds:X509Data></ds:KeyInfo>`)
		transformer := &pipelineRecordingTransformer{}
		cfg := &verifierConfig{xsltTransformer: transformer}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
		require.NoError(t, err)
		require.Len(t, transformer.snapshot(), 1)
		require.Len(t, data.X509Certificates, 1)
		require.Equal(t, cert.Raw, data.X509Certificates[0].Raw)
	})
}

// xsltTransformElem is a ds:Transforms carrying a single XSLT transform whose
// xsl:stylesheet is an identity transform. It is well-formed and parses, so the
// RetrievalMethod path must decide whether to apply or reject it — never ignore
// it while still accepting the resolved certificate.
const xsltTransformElem = `<ds:Transforms><ds:Transform Algorithm="` + TransformXSLT + `">` +
	`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">` +
	`<xsl:template match="/"><xsl:copy-of select="."/></xsl:template>` +
	`</xsl:stylesheet></ds:Transform></ds:Transforms>`

// TestRetrievalMethodXSLTFailsClosed proves an XSLT transform on a
// RetrievalMethod is NOT silently ignored: with no XSLTTransformer configured it
// fails closed with ErrUnsupportedTransform on both the external and
// same-document branches, and the resolved certificate is never merged. This is
// the same fail-closed posture the Reference paths apply to an XSLT transform.
func TestRetrievalMethodXSLTFailsClosed(t *testing.T) {
	t.Run("external", func(t *testing.T) {
		_, der := selfSignedCert(t)
		fsys := fstest.MapFS{certSignerDERPath: {Data: der}}
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">`+
			`<ds:RetrievalMethod URI="certs/signer.der" Type="`+TypeRawX509Certificate+`">`+xsltTransformElem+`</ds:RetrievalMethod></ds:KeyInfo>`)
		cfg := &verifierConfig{referenceResolver: FSReferenceResolver(fsys)}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
		require.Empty(t, data.X509Certificates)
	})

	t.Run("same-document", func(t *testing.T) {
		_, der := selfSignedCert(t)
		certB64 := base64.StdEncoding.EncodeToString(der)
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">`+
			`<ds:RetrievalMethod URI="#x509" Type="`+TypeX509Data+`">`+xsltTransformElem+`</ds:RetrievalMethod>`+
			`<ds:X509Data Id="x509"><ds:X509Certificate>`+certB64+`</ds:X509Certificate></ds:X509Data></ds:KeyInfo>`)
		cfg := &verifierConfig{}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
		require.Empty(t, data.X509Certificates)
	})
}

// TestRetrievalMethodRawCertCountsAgainstBudget proves a retrieved
// rawX509Certificate is charged to the per-Verify decoded-byte budget exactly
// like an inline ds:X509Certificate, so a RetrievalMethod cannot fetch and parse
// certificate octets that escape MaxDecodedBytes. A budget whose decoded cap is
// far below a certificate's DER length trips ErrResourceLimitExceeded.
func TestRetrievalMethodRawCertCountsAgainstBudget(t *testing.T) {
	tinyBudget := func() *verifyBudget {
		return &verifyBudget{maxRefs: 1024, maxEntries: 256, maxDecoded: 8}
	}

	t.Run("external", func(t *testing.T) {
		_, der := selfSignedCert(t)
		require.Greater(t, len(der), 8)
		fsys := fstest.MapFS{certSignerDERPath: {Data: der}}
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="certs/signer.der" Type="`+TypeRawX509Certificate+`"/></ds:KeyInfo>`)
		cfg := &verifierConfig{referenceResolver: FSReferenceResolver(fsys)}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), tinyBudget(), cfg, doc, doc.DocumentElement(), data)
		require.ErrorIs(t, err, ErrResourceLimitExceeded)
		require.Empty(t, data.X509Certificates)
	})

	t.Run("same-document", func(t *testing.T) {
		_, der := selfSignedCert(t)
		certB64 := base64.StdEncoding.EncodeToString(der)
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="#raw" Type="`+TypeRawX509Certificate+`"/>`+
			`<ds:SPKIData Id="raw">`+certB64+`</ds:SPKIData></ds:KeyInfo>`)
		cfg := &verifierConfig{}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), tinyBudget(), cfg, doc, doc.DocumentElement(), data)
		require.ErrorIs(t, err, ErrResourceLimitExceeded)
		require.Empty(t, data.X509Certificates)
	})
}

// TestRetrievalMethodLenientKeyInfo proves LenientKeyInfo skips only an
// UNRESOLVABLE RetrievalMethod (no resolver / not found), while a
// resolved-but-invalid one still fails closed regardless of the setting.
func TestRetrievalMethodLenientKeyInfo(t *testing.T) {
	externalRM := `<ds:KeyInfo xmlns:ds="` + NamespaceDSig + `"><ds:RetrievalMethod URI="certs/signer.der" Type="` + TypeRawX509Certificate + `"/></ds:KeyInfo>`

	t.Run("unresolvable external fails hard by default", func(t *testing.T) {
		doc := mustParse(t, externalRM)
		cfg := &verifierConfig{} // no resolver, not lenient
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
		require.ErrorIs(t, err, ErrReferenceNotFound)
	})

	t.Run("unresolvable external skipped when lenient", func(t *testing.T) {
		doc := mustParse(t, externalRM)
		cfg := &verifierConfig{lenientKeyInfo: true} // no resolver
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
		require.NoError(t, err)
		require.Empty(t, data.X509Certificates)
	})

	t.Run("inline material preserved alongside a skipped RetrievalMethod", func(t *testing.T) {
		cert, _ := selfSignedCert(t)
		doc := mustParse(t, externalRM)
		cfg := &verifierConfig{lenientKeyInfo: true}
		// Simulate parseKeyInfo having already collected an inline certificate.
		data := &KeyInfoData{X509Certificates: []*x509.Certificate{cert}}

		err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
		require.NoError(t, err)
		require.Len(t, data.X509Certificates, 1)
		require.Equal(t, cert.Raw, data.X509Certificates[0].Raw)
	})

	t.Run("resolved but corrupt external still fails hard when lenient", func(t *testing.T) {
		fsys := fstest.MapFS{certSignerDERPath: {Data: []byte("not a certificate")}}
		doc := mustParse(t, externalRM)
		cfg := &verifierConfig{referenceResolver: FSReferenceResolver(fsys), lenientKeyInfo: true}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
	})

	t.Run("unsupported Type still fails hard when lenient", func(t *testing.T) {
		_, der := selfSignedCert(t)
		certB64 := base64.StdEncoding.EncodeToString(der)
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="#x509" Type="urn:bogus"/>`+
			`<ds:X509Data Id="x509"><ds:X509Certificate>`+certB64+`</ds:X509Certificate></ds:X509Data></ds:KeyInfo>`)
		cfg := &verifierConfig{lenientKeyInfo: true}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
	})
}

// TestResolveRetrievalMethodResolvedButNotXML proves a RESOLVED external
// X509Data resource whose octets are not well-formed XML is treated as
// resolved-but-corrupt (ErrInvalidKeyInfo), not as an unavailable resource
// (ErrReferenceNotFound). This keeps a garbage resource a hard failure and, under
// Verifier.LenientKeyInfo, non-skippable — leniency only skips the genuinely
// unavailable case.
func TestResolveRetrievalMethodResolvedButNotXML(t *testing.T) {
	fsys := fstest.MapFS{"keyinfo/data.xml": {Data: []byte("this is not <well-formed> xml")}}
	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="keyinfo/data.xml" Type="`+TypeX509Data+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{referenceResolver: FSReferenceResolver(fsys)}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), newVerifyBudget(cfg), cfg, doc, doc.DocumentElement(), data)
	require.ErrorIs(t, err, ErrInvalidKeyInfo)
	require.Empty(t, data.X509Certificates)
}
