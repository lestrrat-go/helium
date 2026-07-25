// Package xmlenc1 implements W3C XML Encryption 1.1.
package xmlenc1

const (
	// NamespaceXMLEnc is the XML Encryption namespace.
	NamespaceXMLEnc = "http://www.w3.org/2001/04/xmlenc#"

	// NamespaceXMLEnc11 is the XML Encryption 1.1 namespace.
	NamespaceXMLEnc11 = "http://www.w3.org/2009/xmlenc11#"

	// NamespaceDSig is the XML Digital Signatures namespace (for KeyInfo).
	NamespaceDSig = "http://www.w3.org/2000/09/xmldsig#"

	// NamespaceDSigMore contains the additional XML Digital Signature
	// algorithm identifiers used by XML Encryption.
	NamespaceDSigMore = "http://www.w3.org/2001/04/xmldsig-more#"

	// NamespaceDSig11 is the XML Digital Signature 1.1 namespace.
	NamespaceDSig11 = "http://www.w3.org/2009/xmldsig11#"
)

// Block encryption algorithm URIs.
const (
	AES128CBC = NamespaceXMLEnc + "aes128-cbc"
	AES256CBC = NamespaceXMLEnc + "aes256-cbc"
	AES128GCM = NamespaceXMLEnc + "aes128-gcm"
	AES256GCM = NamespaceXMLEnc + "aes256-gcm"

	// AES128GCM11 and related constants are XML Encryption 1.1 GCM
	// identifiers. The 1.1 identifiers are distinct from the legacy XML
	// Encryption namespace identifiers above.
	AES128GCM11 = NamespaceXMLEnc11 + "aes128-gcm"
	AES192GCM11 = NamespaceXMLEnc11 + "aes192-gcm"
	AES256GCM11 = NamespaceXMLEnc11 + "aes256-gcm"
)

// Key transport algorithm URIs.
const (
	RSAOAEP   = NamespaceXMLEnc + "rsa-oaep-mgf1p"
	RSAOAEP11 = NamespaceXMLEnc11 + "rsa-oaep"
)

// Key wrapping algorithm URIs.
const (
	AES128KeyWrap = NamespaceXMLEnc + "kw-aes128"
	AES192KeyWrap = NamespaceXMLEnc + "kw-aes192"
	AES256KeyWrap = NamespaceXMLEnc + "kw-aes256"
)

// Digest algorithm URIs (for RSA-OAEP 1.1).
const (
	DigestSHA1   = NamespaceDSig + "sha1"
	DigestSHA256 = NamespaceXMLEnc + "sha256"
	DigestSHA384 = NamespaceXMLEnc + "sha384"
	// DigestSHA384DSigMore is the XMLDSig-more SHA-384 URI.
	DigestSHA384DSigMore = NamespaceDSigMore + "sha384"
	DigestSHA512         = NamespaceXMLEnc + "sha512"
)

// MGF algorithm URIs.
const (
	MGFSHA1   = NamespaceXMLEnc11 + "mgf1sha1"
	MGFSHA256 = NamespaceXMLEnc11 + "mgf1sha256"
	MGFSHA384 = NamespaceXMLEnc11 + "mgf1sha384"
	MGFSHA512 = NamespaceXMLEnc11 + "mgf1sha512"
)

// Key agreement and key derivation algorithm URIs.
const (
	ECDHES    = NamespaceXMLEnc11 + "ECDH-ES"
	ConcatKDF = NamespaceXMLEnc11 + "ConcatKDF"
)

// Encryption type URIs.
const (
	TypeElement = NamespaceXMLEnc + "Element"
	TypeContent = NamespaceXMLEnc + "Content"
)

// Namespace prefixes used when constructing elements.
const (
	nsPrefixEnc  = "xenc"
	nsPrefixDSig = "ds"
)
