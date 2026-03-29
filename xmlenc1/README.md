# xmlenc1

The `xmlenc1` package implements W3C XML Encryption 1.1 for helium documents.

Import path: `github.com/lestrrat-go/helium/xmlenc1`

<!-- INCLUDE(examples/xmlenc1_encrypt_element_example_test.go) -->
```go
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

func Example_xmlenc1_encrypt_element() {
  // Parse a document containing sensitive data. In SAML, this would
  // be an Assertion element inside a Response.
  const src = `<Response><Assertion>sensitive user data</Assertion></Response>`

  doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
  if err != nil {
    fmt.Printf("parse error: %s\n", err)
    return
  }

  // Generate an RSA key pair. In production, use the recipient's
  // public key (e.g., the SP's certificate in SAML).
  key, err := rsa.GenerateKey(rand.Reader, 2048)
  if err != nil {
    fmt.Printf("keygen error: %s\n", err)
    return
  }

  // Encrypt the Assertion element. The Encryptor:
  // 1. Generates a random AES session key
  // 2. Encrypts the serialized element with AES-128-GCM
  // 3. Wraps the session key with RSA-OAEP
  // 4. Replaces the element in the tree with <EncryptedData>
  assertion, ok := helium.AsNode[*helium.Element](doc.DocumentElement().FirstChild())
  if !ok {
    fmt.Println("assertion not found")
    return
  }

  _, err = xmlenc1.NewEncryptor().
    BlockAlgorithm(xmlenc1.AES128GCM).
    KeyTransportAlgorithm(xmlenc1.RSAOAEP).
    RecipientPublicKey(&key.PublicKey).
    EncryptElement(context.Background(), assertion)
  if err != nil {
    fmt.Printf("encrypt error: %s\n", err)
    return
  }

  out, _ := helium.WriteString(doc)

  // The plaintext is gone; only EncryptedData remains.
  fmt.Println(strings.Contains(out, "sensitive user data"))
  fmt.Println(strings.Contains(out, "EncryptedData"))
  // Output:
  // false
  // true
}
```
source: [examples/xmlenc1_encrypt_element_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/xmlenc1_encrypt_element_example_test.go)
<!-- END INCLUDE -->

<!-- INCLUDE(examples/xmlenc1_decrypt_example_test.go) -->
```go
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
```
source: [examples/xmlenc1_decrypt_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/xmlenc1_decrypt_example_test.go)
<!-- END INCLUDE -->
