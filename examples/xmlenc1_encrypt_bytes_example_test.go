package examples_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
)

func Example_xmlenc1_encrypt_bytes() {
	// A payload that is not XML at all. XML Encryption allows this: an
	// EncryptedData whose plaintext is neither an element nor element
	// content simply carries no Type attribute.
	payload := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}

	// EncryptBytes needs a document to own the element it builds, but it
	// does not attach the element anywhere.
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<Envelope/>`))
	if err != nil {
		fmt.Printf("failed to parse document: %s\n", err)
		return
	}

	// A key-encryption key wraps the randomly generated session key.
	kek := make([]byte, 32)
	if _, err := rand.Read(kek); err != nil {
		fmt.Printf("failed to generate key: %s\n", err)
		return
	}

	edElem, err := xmlenc1.NewEncryptor().
		KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
		KeyEncryptionKey(kek).
		EncryptBytes(context.Background(), doc, payload)
	if err != nil {
		fmt.Printf("failed to encrypt: %s\n", err)
		return
	}

	// The caller decides where the EncryptedData goes; here it becomes the
	// sole child of the envelope.
	if err := doc.DocumentElement().AddChild(edElem); err != nil {
		fmt.Printf("failed to insert EncryptedData: %s\n", err)
		return
	}

	// DecryptBytes is the inverse: it returns the octets without trying to
	// parse them as XML.
	got, err := xmlenc1.NewDecryptor().
		KeyEncryptionKey(kek).
		DecryptBytes(context.Background(), edElem)
	if err != nil {
		fmt.Printf("failed to decrypt: %s\n", err)
		return
	}

	fmt.Println(bytes.Equal(payload, got))
	// Output:
	// true
}
