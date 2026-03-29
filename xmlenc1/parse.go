package xmlenc1

import (
	"encoding/base64"
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

// parseEncryptedData parses an EncryptedData element.
func parseEncryptedData(elem *helium.Element) (*EncryptedData, error) {
	ed := &EncryptedData{}
	ed.ID, _ = elem.GetAttribute("Id")
	ed.Type, _ = elem.GetAttribute("Type")

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch localName(e) {
		case "EncryptionMethod":
			em, err := parseEncryptionMethod(e)
			if err != nil {
				return nil, err
			}
			ed.EncryptionMethod = em
		case "KeyInfo":
			if err := parseKeyInfoForEncryption(e, ed); err != nil {
				return nil, err
			}
		case "CipherData":
			cv, err := parseCipherData(e)
			if err != nil {
				return nil, err
			}
			ed.CipherValue = cv
		}
	}

	if ed.CipherValue == nil {
		return nil, fmt.Errorf("%w: missing CipherData/CipherValue", ErrMalformedEncrypted)
	}

	return ed, nil
}

func parseKeyInfoForEncryption(elem *helium.Element, ed *EncryptedData) error {
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if localName(e) == "EncryptedKey" {
			ek, err := parseEncryptedKey(e)
			if err != nil {
				return err
			}
			ed.EncryptedKey = ek
			return nil
		}
	}
	return nil
}

// parseEncryptedKey parses an EncryptedKey element.
func parseEncryptedKey(elem *helium.Element) (*EncryptedKey, error) {
	ek := &EncryptedKey{}
	ek.ID, _ = elem.GetAttribute("Id")
	ek.Recipient, _ = elem.GetAttribute("Recipient")

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch localName(e) {
		case "EncryptionMethod":
			em, err := parseEncryptionMethod(e)
			if err != nil {
				return nil, err
			}
			ek.EncryptionMethod = em
		case "CipherData":
			cv, err := parseCipherData(e)
			if err != nil {
				return nil, err
			}
			ek.CipherValue = cv
		case "CarriedKeyName":
			ek.CarriedKeyName = textContent(e)
		}
	}

	return ek, nil
}

func parseEncryptionMethod(elem *helium.Element) (*EncryptionMethod, error) {
	em := &EncryptionMethod{}
	alg, ok := elem.GetAttribute("Algorithm")
	if !ok {
		return nil, fmt.Errorf("%w: EncryptionMethod missing Algorithm", ErrMalformedEncrypted)
	}
	em.Algorithm = alg

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch localName(e) {
		case "DigestMethod":
			em.DigestMethod, _ = e.GetAttribute("Algorithm")
		case "MGF":
			em.MGFAlgorithm, _ = e.GetAttribute("Algorithm")
		case "OAEPparams":
			text := textContent(e)
			decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(text))
			if err != nil {
				return nil, fmt.Errorf("%w: invalid OAEPparams: %v", ErrMalformedEncrypted, err)
			}
			em.OAEPParams = decoded
		}
	}

	return em, nil
}

func parseCipherData(elem *helium.Element) ([]byte, error) {
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if localName(e) == "CipherValue" {
			text := strings.TrimSpace(textContent(e))
			decoded, err := base64.StdEncoding.DecodeString(text)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid CipherValue: %v", ErrMalformedEncrypted, err)
			}
			return decoded, nil
		}
	}
	return nil, fmt.Errorf("%w: missing CipherValue", ErrMalformedEncrypted)
}

// localName returns the local name of an element, stripping any prefix.
func localName(e *helium.Element) string {
	name := e.Name()
	for i := range len(name) {
		if name[i] == ':' {
			return name[i+1:]
		}
	}
	return name
}

// textContent returns the concatenated text content of an element.
func textContent(e *helium.Element) string {
	var sb []byte
	for child := e.FirstChild(); child != nil; child = child.NextSibling() {
		sb = append(sb, child.Content()...)
	}
	return string(sb)
}
