package xmlenc1

import (
	"encoding/base64"

	helium "github.com/lestrrat-go/helium"
)

// marshalEncryptedData builds the EncryptedData DOM element tree.
func marshalEncryptedData(doc *helium.Document, ed *EncryptedData) (*helium.Element, error) {
	root, err := doc.CreateElement("EncryptedData")
	if err != nil {
		return nil, err
	}
	if err := root.DeclareNamespace(nsPrefixEnc, NamespaceXMLEnc); err != nil {
		return nil, err
	}
	if err := root.SetActiveNamespace(nsPrefixEnc, NamespaceXMLEnc); err != nil {
		return nil, err
	}
	if ed.Type != "" {
		if err := root.SetAttribute("Type", ed.Type); err != nil {
			return nil, err
		}
	}
	if ed.ID != "" {
		if err := root.SetAttribute("Id", ed.ID); err != nil {
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

	// KeyInfo with one EncryptedKey per recipient. EncryptedKeys takes
	// precedence over the deprecated single EncryptedKey field.
	encKeys := ed.effectiveEncryptedKeys()
	if len(encKeys) > 0 {
		keyInfo, err := doc.CreateElement("KeyInfo")
		if err != nil {
			return nil, err
		}
		if err := keyInfo.DeclareNamespace(nsPrefixDSig, NamespaceDSig); err != nil {
			return nil, err
		}
		if err := keyInfo.SetActiveNamespace(nsPrefixDSig, NamespaceDSig); err != nil {
			return nil, err
		}

		for _, k := range encKeys {
			ek, err := marshalEncryptedKey(doc, k)
			if err != nil {
				return nil, err
			}
			if err := keyInfo.AddChild(ek); err != nil {
				return nil, err
			}
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
	root, err := doc.CreateElement("EncryptedKey")
	if err != nil {
		return nil, err
	}
	if err := root.SetActiveNamespace(nsPrefixEnc, NamespaceXMLEnc); err != nil {
		return nil, err
	}
	if ek.ID != "" {
		if err := root.SetAttribute("Id", ek.ID); err != nil {
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
	elem, err := doc.CreateElement("EncryptionMethod")
	if err != nil {
		return nil, err
	}
	if err := elem.SetActiveNamespace(nsPrefixEnc, NamespaceXMLEnc); err != nil {
		return nil, err
	}
	if err := elem.SetAttribute("Algorithm", em.Algorithm); err != nil {
		return nil, err
	}

	if em.DigestMethod != "" {
		dm, err := doc.CreateElement("DigestMethod")
		if err != nil {
			return nil, err
		}
		if err := dm.SetActiveNamespace(nsPrefixDSig, NamespaceDSig); err != nil {
			return nil, err
		}
		if err := dm.SetAttribute("Algorithm", em.DigestMethod); err != nil {
			return nil, err
		}
		if err := elem.AddChild(dm); err != nil {
			return nil, err
		}
	}

	if em.MGFAlgorithm != "" {
		mgf, err := doc.CreateElement("MGF")
		if err != nil {
			return nil, err
		}
		if err := mgf.SetActiveNamespace(nsPrefixEnc, NamespaceXMLEnc11); err != nil {
			return nil, err
		}
		if err := mgf.SetAttribute("Algorithm", em.MGFAlgorithm); err != nil {
			return nil, err
		}
		if err := elem.AddChild(mgf); err != nil {
			return nil, err
		}
	}

	if len(em.OAEPParams) > 0 {
		params, err := doc.CreateElement("OAEPparams")
		if err != nil {
			return nil, err
		}
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
	cd, err := doc.CreateElement("CipherData")
	if err != nil {
		return nil, err
	}
	if err := cd.SetActiveNamespace(nsPrefixEnc, NamespaceXMLEnc); err != nil {
		return nil, err
	}

	cv, err := doc.CreateElement("CipherValue")
	if err != nil {
		return nil, err
	}
	if err := cv.SetActiveNamespace(nsPrefixEnc, NamespaceXMLEnc); err != nil {
		return nil, err
	}

	encoded := base64.StdEncoding.EncodeToString(cipherValue)
	if err := cv.AddChild(doc.CreateText([]byte(encoded))); err != nil {
		return nil, err
	}

	return cd, cd.AddChild(cv)
}
