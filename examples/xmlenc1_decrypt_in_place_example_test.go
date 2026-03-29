package examples_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
)

func Example_xmlenc1_decrypt_in_place() {
	// DecryptInPlace finds all EncryptedData elements in a document and
	// decrypts them, replacing each with the original plaintext nodes.
	// This is useful when processing SAML Responses that may contain
	// multiple encrypted assertions.
	const src = `<Response><Assertion>confidential</Assertion></Response>`

	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("parse error: %s\n", err)
		return
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("keygen error: %s\n", err)
		return
	}

	// Encrypt the Assertion element.
	assertion, _ := helium.AsNode[*helium.Element](doc.DocumentElement().FirstChild())
	_, err = xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES128GCM).
		KeyTransportAlgorithm(xmlenc1.RSAOAEP).
		RecipientPublicKey(&key.PublicKey).
		EncryptElement(context.Background(), assertion)
	if err != nil {
		fmt.Printf("encrypt error: %s\n", err)
		return
	}

	// The document now has EncryptedData instead of Assertion.
	encrypted, _ := helium.WriteString(doc)
	fmt.Println(strings.Contains(encrypted, "EncryptedData"))

	// DecryptInPlace walks the document, finds EncryptedData elements,
	// decrypts them, and replaces them with the plaintext.
	err = xmlenc1.NewDecryptor().PrivateKey(key).
		DecryptInPlace(context.Background(), doc)
	if err != nil {
		fmt.Printf("decrypt error: %s\n", err)
		return
	}

	decrypted, _ := helium.WriteString(doc)
	fmt.Println(strings.Contains(decrypted, "confidential"))
	// Output:
	// true
	// true
}
