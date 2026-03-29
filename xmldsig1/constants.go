// Package xmldsig1 implements W3C XML Digital Signatures 1.1.
package xmldsig1

const (
	// NamespaceDSig is the XML Digital Signatures namespace.
	NamespaceDSig = "http://www.w3.org/2000/09/xmldsig#"

	// NamespaceDSig11 is the XML Digital Signatures 1.1 namespace.
	NamespaceDSig11 = "http://www.w3.org/2009/xmldsig11#"
)

// Signature algorithm URIs.
const (
	AlgRSASHA1     = "http://www.w3.org/2000/09/xmldsig#rsa-sha1"
	AlgRSASHA256   = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
	AlgECDSASHA256 = "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256"
	AlgECDSASHA384 = "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha384"
	AlgHMACSHA1    = "http://www.w3.org/2000/09/xmldsig#hmac-sha1"
	AlgHMACSHA256  = "http://www.w3.org/2001/04/xmldsig-more#hmac-sha256"
	AlgEd25519     = "http://www.w3.org/2021/04/xmldsig-more#eddsa-ed25519"
)

// Digest algorithm URIs.
const (
	DigestSHA1   = "http://www.w3.org/2000/09/xmldsig#sha1"
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
)

// Namespace prefix used when constructing signature elements.
const nsPrefix = "ds"

// Type URIs for Reference elements.
const (
	TypeObject    = "http://www.w3.org/2000/09/xmldsig#Object"
	TypeManifest  = "http://www.w3.org/2000/09/xmldsig#Manifest"
	TypeSignProps = "http://www.w3.org/2000/09/xmldsig#SignatureProperties"
)
