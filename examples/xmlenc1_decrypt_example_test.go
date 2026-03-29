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

func Example_xmlenc1_decrypt() {
	// Start with a document, encrypt an element, then decrypt it.
	const src = `<Response><Assertion>secret</Assertion></Response>`

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

	assertion, _ := helium.AsNode[*helium.Element](doc.DocumentElement().FirstChild())
	edElem, err := xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256CBC).
		KeyTransportAlgorithm(xmlenc1.RSAOAEP).
		RecipientPublicKey(&key.PublicKey).
		EncryptElement(context.Background(), assertion)
	if err != nil {
		fmt.Printf("encrypt error: %s\n", err)
		return
	}

	// Decrypt returns the original node(s). The caller decides whether
	// to re-insert them into the tree or process them standalone.
	decryptor := xmlenc1.NewDecryptor().PrivateKey(key)
	nodes, err := decryptor.Decrypt(context.Background(), edElem)
	if err != nil {
		fmt.Printf("decrypt error: %s\n", err)
		return
	}

	out, _ := helium.WriteString(nodes[0])
	fmt.Println(strings.Contains(out, "secret"))
	// Output:
	// true
}
