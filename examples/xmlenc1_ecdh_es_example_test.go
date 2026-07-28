package examples_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
)

func Example_xmlenc1_ecdh_es() {
	const src = `<Response><Assertion>sensitive user data</Assertion></Response>`

	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("failed to parse document: %s\n", err)
		return
	}

	// The recipient's static EC key pair. In production only the public
	// half is known to the sender.
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Printf("failed to generate key: %s\n", err)
		return
	}

	assertion, ok := helium.AsNode[*helium.Element](doc.DocumentElement().FirstChild())
	if !ok {
		fmt.Println("assertion not found")
		return
	}

	// ECDH-ES agrees a shared secret with a per-message ephemeral key,
	// derives a key-encryption key from it with ConcatKDF, and wraps the
	// session key under that. KeyWrapAlgorithm picks the AES Key Wrap
	// variant; no key-encryption key is supplied, because it is derived.
	edElem, err := xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256GCM11).
		KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
		RecipientECPublicKey(&key.PublicKey).
		EncryptElement(context.Background(), assertion)
	if err != nil {
		fmt.Printf("failed to encrypt: %s\n", err)
		return
	}

	encrypted, err := helium.WriteString(doc)
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Println(strings.Contains(encrypted, "sensitive user data"))
	fmt.Println(strings.Contains(encrypted, "AgreementMethod"))

	// The recipient reaches the same shared secret from the ephemeral
	// public key carried in the AgreementMethod.
	nodes, err := xmlenc1.NewDecryptor().
		ECPrivateKey(key).
		Decrypt(context.Background(), edElem)
	if err != nil {
		fmt.Printf("failed to decrypt: %s\n", err)
		return
	}

	decrypted, err := helium.WriteString(nodes[0])
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Println(strings.Contains(decrypted, "sensitive user data"))
	// Output:
	// false
	// true
	// true
}
