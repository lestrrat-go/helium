package xmlenc1

// EncryptionMethod represents the <EncryptionMethod> element.
type EncryptionMethod struct {
	Algorithm    string
	DigestMethod string // optional (for RSA-OAEP 1.1)
	MGFAlgorithm string // optional (for RSA-OAEP 1.1)
	OAEPParams   []byte // optional
}

// EncryptedData represents the <EncryptedData> element.
type EncryptedData struct {
	ID               string
	Type             string // TypeElement or TypeContent
	EncryptionMethod *EncryptionMethod
	EncryptedKey     *EncryptedKey // from KeyInfo
	CipherValue      []byte        // base64-decoded cipher bytes
}

// EncryptedKey represents the <EncryptedKey> element.
type EncryptedKey struct {
	ID               string
	Recipient        string
	EncryptionMethod *EncryptionMethod
	CipherValue      []byte // base64-decoded cipher bytes
	CarriedKeyName   string
}
