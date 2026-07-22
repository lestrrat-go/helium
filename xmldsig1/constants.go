// Package xmldsig1 implements W3C XML Digital Signatures 1.1.
package xmldsig1

const (
	// NamespaceDSig is the XML Digital Signatures namespace.
	NamespaceDSig = "http://www.w3.org/2000/09/xmldsig#"

	// NamespaceDSig11 is the XML Digital Signatures 1.1 namespace.
	NamespaceDSig11 = "http://www.w3.org/2009/xmldsig11#"

	// NamespaceDSigMore is the xmldsig-more namespace. RFC 4050 places its
	// legacy ECDSAKeyValue (and its DomainParameters/NamedCurve/PublicKey
	// children) in this namespace, distinct from both the core xmldsig#
	// namespace and the XML-Signature 1.1 xmldsig11# namespace.
	NamespaceDSigMore = "http://www.w3.org/2001/04/xmldsig-more#"
)

// Signature algorithm URIs.
const (
	AlgRSASHA1     = "http://www.w3.org/2000/09/xmldsig#rsa-sha1"
	AlgRSASHA224   = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha224"
	AlgRSASHA256   = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
	AlgRSASHA384   = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha384"
	AlgRSASHA512   = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha512"
	AlgECDSASHA1   = "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha1"
	AlgECDSASHA224 = "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha224"
	AlgECDSASHA256 = "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256"
	AlgECDSASHA384 = "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha384"
	AlgECDSASHA512 = "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha512"
	AlgHMACSHA1    = "http://www.w3.org/2000/09/xmldsig#hmac-sha1"
	AlgHMACSHA224  = "http://www.w3.org/2001/04/xmldsig-more#hmac-sha224"
	AlgHMACSHA256  = "http://www.w3.org/2001/04/xmldsig-more#hmac-sha256"
	AlgHMACSHA384  = "http://www.w3.org/2001/04/xmldsig-more#hmac-sha384"
	AlgHMACSHA512  = "http://www.w3.org/2001/04/xmldsig-more#hmac-sha512"
	AlgEd25519     = "http://www.w3.org/2021/04/xmldsig-more#eddsa-ed25519"
	// AlgDSASHA1 is DSA-SHA1. It is verify-only (signing is not supported) and
	// SHA-1-weak, so it is rejected on verify unless Verifier.AllowSHA1(true).
	AlgDSASHA1 = "http://www.w3.org/2000/09/xmldsig#dsa-sha1"
)

// Digest algorithm URIs.
const (
	DigestSHA1   = "http://www.w3.org/2000/09/xmldsig#sha1"
	DigestSHA224 = "http://www.w3.org/2001/04/xmldsig-more#sha224"
	DigestSHA256 = "http://www.w3.org/2001/04/xmlenc#sha256"
	DigestSHA384 = "http://www.w3.org/2001/04/xmldsig-more#sha384"
	DigestSHA512 = "http://www.w3.org/2001/04/xmlenc#sha512"
)

// Canonicalization method URIs.
const (
	C14N10            = "http://www.w3.org/TR/2001/REC-xml-c14n-20010315"
	C14N10Comments    = "http://www.w3.org/TR/2001/REC-xml-c14n-20010315#WithComments"
	ExcC14N10         = "http://www.w3.org/2001/10/xml-exc-c14n#"
	ExcC14N10Comments = "http://www.w3.org/2001/10/xml-exc-c14n#WithComments"
	C14N11URI         = "http://www.w3.org/2006/12/xml-c14n11"
	C14N11Comments    = "http://www.w3.org/2006/12/xml-c14n11#WithComments"
)

// Transform URIs.
const (
	TransformEnvelopedSignature = "http://www.w3.org/2000/09/xmldsig#enveloped-signature"
	TransformXPath              = "http://www.w3.org/TR/1999/REC-xpath-19991116"
	// TransformBase64 is the base64 decode transform (XMLDSig core §6.6.2). Its
	// input node-set's XPath 1.0 string-value is base64-decoded and the decoded
	// octets are digested directly, with no canonicalization applied afterward.
	// It is verify-only: signing has no typed Transform for it and the sign
	// preflight rejects it fail-closed.
	TransformBase64 = "http://www.w3.org/2000/09/xmldsig#base64"
)

// Namespace prefix used when constructing signature elements.
const nsPrefix = "ds"

// Type URIs for Reference elements.
const (
	TypeObject    = "http://www.w3.org/2000/09/xmldsig#Object"
	TypeManifest  = "http://www.w3.org/2000/09/xmldsig#Manifest"
	TypeSignProps = "http://www.w3.org/2000/09/xmldsig#SignatureProperties"
)
