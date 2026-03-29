package xmlenc1

import (
	"encoding/base64"

	helium "github.com/lestrrat-go/helium"
)

// marshalEncryptedData builds the EncryptedData DOM element tree.
func marshalEncryptedData(doc *helium.Document, ed *EncryptedData) (*helium.Element, error) {
	root := doc.CreateElement("EncryptedData")
	if err := root.DeclareNamespace(nsPrefixEnc, NamespaceXMLEnc); err != nil {
		return nil, err
	}
	if err := root.SetActiveNamespace(nsPrefixEnc, NamespaceXMLEnc); err != nil {
		return nil, err
	}
	if ed.Type != "" {
		if err := root.SetLiteralAttribute("Type", ed.Type); err != nil {
			return nil, err
		}
	}
	if ed.ID != "" {
		if err := root.SetLiteralAttribute("Id", ed.ID); err != nil {
			return nil, err
		}
	}

	// EncryptionMethod
	if ed.EncryptionMethod != nil {
		em, err := marshalEncryptionMethod(doc, ed.EncryptionMethod)
		if err != nil {
			return nil, err
		}
		if err := root.AddChild(em); err != nil {
			return nil, err
		}
	}

	// KeyInfo with EncryptedKey
	if ed.EncryptedKey != nil {
		keyInfo := doc.CreateElement("KeyInfo")
		if err := keyInfo.DeclareNamespace(nsPrefixDSig, NamespaceDSig); err != nil {
			return nil, err
		}
		if err := keyInfo.SetActiveNamespace(nsPrefixDSig, NamespaceDSig); err != nil {
			return nil, err
		}

		ek, err := marshalEncryptedKey(doc, ed.EncryptedKey)
		if err != nil {
			return nil, err
		}
		if err := keyInfo.AddChild(ek); err != nil {
			return nil, err
		}
		if err := root.AddChild(keyInfo); err != nil {
			return nil, err
		}
	}

	// CipherData
	cd, err := marshalCipherData(doc, ed.CipherValue)
	if err != nil {
		return nil, err
	}
	return root, root.AddChild(cd)
}

// marshalEncryptedKey builds the EncryptedKey DOM element tree.
func marshalEncryptedKey(doc *helium.Document, ek *EncryptedKey) (*helium.Element, error) {
	root := doc.CreateElement("EncryptedKey")
	if err := root.SetActiveNamespace(nsPrefixEnc, NamespaceXMLEnc); err != nil {
		return nil, err
	}
	if ek.ID != "" {
		if err := root.SetLiteralAttribute("Id", ek.ID); err != nil {
			return nil, err
		}
	}

	if ek.EncryptionMethod != nil {
		em, err := marshalEncryptionMethod(doc, ek.EncryptionMethod)
		if err != nil {
			return nil, err
		}
		if err := root.AddChild(em); err != nil {
			return nil, err
		}
	}

	cd, err := marshalCipherData(doc, ek.CipherValue)
	if err != nil {
		return nil, err
	}
	return root, root.AddChild(cd)
}

func marshalEncryptionMethod(doc *helium.Document, em *EncryptionMethod) (*helium.Element, error) {
	elem := doc.CreateElement("EncryptionMethod")
	if err := elem.SetActiveNamespace(nsPrefixEnc, NamespaceXMLEnc); err != nil {
		return nil, err
	}
	if err := elem.SetLiteralAttribute("Algorithm", em.Algorithm); err != nil {
		return nil, err
	}

	if em.DigestMethod != "" {
		dm := doc.CreateElement("DigestMethod")
		if err := dm.SetActiveNamespace(nsPrefixDSig, NamespaceDSig); err != nil {
			return nil, err
		}
		if err := dm.SetLiteralAttribute("Algorithm", em.DigestMethod); err != nil {
			return nil, err
		}
		if err := elem.AddChild(dm); err != nil {
			return nil, err
		}
	}

	if em.MGFAlgorithm != "" {
		mgf := doc.CreateElement("MGF")
		if err := mgf.SetActiveNamespace(nsPrefixEnc, NamespaceXMLEnc11); err != nil {
			return nil, err
		}
		if err := mgf.SetLiteralAttribute("Algorithm", em.MGFAlgorithm); err != nil {
			return nil, err
		}
		if err := elem.AddChild(mgf); err != nil {
			return nil, err
		}
	}

	if len(em.OAEPParams) > 0 {
		params := doc.CreateElement("OAEPparams")
		if err := params.SetActiveNamespace(nsPrefixEnc, NamespaceXMLEnc); err != nil {
			return nil, err
		}
		encoded := base64.StdEncoding.EncodeToString(em.OAEPParams)
		if err := params.AddChild(doc.CreateText([]byte(encoded))); err != nil {
			return nil, err
		}
		if err := elem.AddChild(params); err != nil {
			return nil, err
		}
	}

	return elem, nil
}

func marshalCipherData(doc *helium.Document, cipherValue []byte) (*helium.Element, error) {
	cd := doc.CreateElement("CipherData")
	if err := cd.SetActiveNamespace(nsPrefixEnc, NamespaceXMLEnc); err != nil {
		return nil, err
	}

	cv := doc.CreateElement("CipherValue")
	if err := cv.SetActiveNamespace(nsPrefixEnc, NamespaceXMLEnc); err != nil {
		return nil, err
	}

	encoded := base64.StdEncoding.EncodeToString(cipherValue)
	if err := cv.AddChild(doc.CreateText([]byte(encoded))); err != nil {
		return nil, err
	}

	return cd, cd.AddChild(cv)
}
