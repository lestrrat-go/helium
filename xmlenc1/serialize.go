package xmlenc1

import (
	"encoding/base64"
	"encoding/hex"

	helium "github.com/lestrrat-go/helium"
)

// marshalEncryptedData builds the EncryptedData DOM element tree.
func marshalEncryptedData(doc *helium.Document, ed *encryptedData) (*helium.Element, error) {
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
func marshalEncryptedKey(doc *helium.Document, ek *encryptedKey) (*helium.Element, error) {
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

	// XML Encryption 1.1 carries the key agreement in a ds:KeyInfo child,
	// between EncryptionMethod and CipherData.
	if ek.AgreementMethod != nil {
		keyInfo, err := marshalAgreementKeyInfo(doc, ek.AgreementMethod)
		if err != nil {
			return nil, err
		}
		if err := root.AddChild(keyInfo); err != nil {
			return nil, err
		}
	}

	cd, err := marshalCipherData(doc, ek.CipherValue)
	if err != nil {
		return nil, err
	}
	return root, root.AddChild(cd)
}

func marshalEncryptionMethod(doc *helium.Document, em *encryptionMethod) (*helium.Element, error) {
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

	// Child order follows the xenc-schema EncryptionMethodType content model:
	// a sequence of xenc:KeySize?, then xenc:OAEPparams?, then
	// <any namespace='##other' minOccurs='0' maxOccurs='unbounded'/>.
	// ds:DigestMethod and xenc11:MGF are foreign-namespace children reachable
	// only through that trailing wildcard, so they come after xenc:OAEPparams.
	// The wildcard does not make the order lax, for two reasons: ##other
	// excludes the xenc namespace, so it can never match xenc:OAEPparams; and
	// while xmlenc-core1 §3.1 asks only for laxly schema valid output, its own
	// note bounds that allowance to what xsd:ANY admits, so it excuses what
	// those foreign children contain rather than where the declared sequence
	// puts them. This package emits no xenc:KeySize, so the sequence is
	// OAEPparams first and the two foreign children after it.
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

// marshalAgreementKeyInfo builds the ds:KeyInfo subtree that carries an
// XML Encryption 1.1 key agreement inside an EncryptedKey:
//
//	<ds:KeyInfo>
//	  <xenc:AgreementMethod Algorithm="...ECDH-ES">
//	    <xenc11:KeyDerivationMethod Algorithm="...ConcatKDF">
//	      <xenc11:ConcatKDFParams ...><ds:DigestMethod Algorithm="..."/></xenc11:ConcatKDFParams>
//	    </xenc11:KeyDerivationMethod>
//	    <xenc:OriginatorKeyInfo>
//	      <ds:KeyValue><dsig11:ECKeyValue>...</dsig11:ECKeyValue></ds:KeyValue>
//	    </xenc:OriginatorKeyInfo>
//	  </xenc:AgreementMethod>
//	</ds:KeyInfo>
func marshalAgreementKeyInfo(doc *helium.Document, agreement *agreementMethod) (*helium.Element, error) {
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

	am, err := doc.CreateElement("AgreementMethod")
	if err != nil {
		return nil, err
	}
	if err := am.SetActiveNamespace(nsPrefixEnc, NamespaceXMLEnc); err != nil {
		return nil, err
	}
	if err := am.SetAttribute("Algorithm", agreement.Algorithm); err != nil {
		return nil, err
	}

	if agreement.KeyDerivationMethod != nil {
		kdm, err := marshalKeyDerivationMethod(doc, agreement.KeyDerivationMethod)
		if err != nil {
			return nil, err
		}
		if err := am.AddChild(kdm); err != nil {
			return nil, err
		}
	}

	if agreement.OriginatorKey != nil {
		oki, err := marshalOriginatorKeyInfo(doc, agreement.OriginatorKey)
		if err != nil {
			return nil, err
		}
		if err := am.AddChild(oki); err != nil {
			return nil, err
		}
	}

	return keyInfo, keyInfo.AddChild(am)
}

func marshalKeyDerivationMethod(doc *helium.Document, method *keyDerivationMethod) (*helium.Element, error) {
	kdm, err := doc.CreateElement("KeyDerivationMethod")
	if err != nil {
		return nil, err
	}
	if err := kdm.DeclareNamespace(nsPrefixEnc11, NamespaceXMLEnc11); err != nil {
		return nil, err
	}
	if err := kdm.SetActiveNamespace(nsPrefixEnc11, NamespaceXMLEnc11); err != nil {
		return nil, err
	}
	if err := kdm.SetAttribute("Algorithm", method.Algorithm); err != nil {
		return nil, err
	}
	if method.ConcatKDF == nil {
		return kdm, nil
	}

	params, err := marshalConcatKDFParams(doc, method.ConcatKDF)
	if err != nil {
		return nil, err
	}
	return kdm, kdm.AddChild(params)
}

func marshalConcatKDFParams(doc *helium.Document, params *ConcatKDFParams) (*helium.Element, error) {
	elem, err := doc.CreateElement("ConcatKDFParams")
	if err != nil {
		return nil, err
	}
	if err := elem.SetActiveNamespace(nsPrefixEnc11, NamespaceXMLEnc11); err != nil {
		return nil, err
	}

	// The OtherInfo fields are hexBinary bit strings: the first octet is the
	// count of unused trailing bits, so the parser can reconstruct a value
	// that is not a whole number of octets. An absent field is omitted
	// entirely rather than written as an empty attribute.
	for _, field := range []struct {
		name       string
		value      []byte
		unusedBits uint8
	}{
		{name: "AlgorithmID", value: params.AlgorithmID, unusedBits: params.algorithmIDUnusedBits},
		{name: "PartyUInfo", value: params.PartyUInfo, unusedBits: params.partyUInfoUnusedBits},
		{name: "PartyVInfo", value: params.PartyVInfo, unusedBits: params.partyVInfoUnusedBits},
		{name: "SuppPubInfo", value: params.SuppPubInfo, unusedBits: params.suppPubInfoUnusedBits},
		{name: "SuppPrivInfo", value: params.SuppPrivInfo, unusedBits: params.suppPrivInfoUnusedBits},
	} {
		if len(field.value) == 0 {
			continue
		}
		encoded := hex.EncodeToString(append([]byte{field.unusedBits}, field.value...))
		if err := elem.SetAttribute(field.name, encoded); err != nil {
			return nil, err
		}
	}

	dm, err := doc.CreateElement("DigestMethod")
	if err != nil {
		return nil, err
	}
	if err := dm.SetActiveNamespace(nsPrefixDSig, NamespaceDSig); err != nil {
		return nil, err
	}
	if err := dm.SetAttribute("Algorithm", params.DigestMethod); err != nil {
		return nil, err
	}
	return elem, elem.AddChild(dm)
}

func marshalOriginatorKeyInfo(doc *helium.Document, key *ecKeyValue) (*helium.Element, error) {
	curveURI, err := ecdhURIForCurve(key.curve)
	if err != nil {
		return nil, err
	}

	oki, err := doc.CreateElement("OriginatorKeyInfo")
	if err != nil {
		return nil, err
	}
	if err := oki.SetActiveNamespace(nsPrefixEnc, NamespaceXMLEnc); err != nil {
		return nil, err
	}

	keyValue, err := doc.CreateElement("KeyValue")
	if err != nil {
		return nil, err
	}
	if err := keyValue.SetActiveNamespace(nsPrefixDSig, NamespaceDSig); err != nil {
		return nil, err
	}

	ecKeyValueElem, err := doc.CreateElement("ECKeyValue")
	if err != nil {
		return nil, err
	}
	if err := ecKeyValueElem.DeclareNamespace(nsPrefixDSig11, NamespaceDSig11); err != nil {
		return nil, err
	}
	if err := ecKeyValueElem.SetActiveNamespace(nsPrefixDSig11, NamespaceDSig11); err != nil {
		return nil, err
	}

	namedCurve, err := doc.CreateElement("NamedCurve")
	if err != nil {
		return nil, err
	}
	if err := namedCurve.SetActiveNamespace(nsPrefixDSig11, NamespaceDSig11); err != nil {
		return nil, err
	}
	if err := namedCurve.SetAttribute("URI", curveURI); err != nil {
		return nil, err
	}
	if err := ecKeyValueElem.AddChild(namedCurve); err != nil {
		return nil, err
	}

	publicKey, err := doc.CreateElement("PublicKey")
	if err != nil {
		return nil, err
	}
	if err := publicKey.SetActiveNamespace(nsPrefixDSig11, NamespaceDSig11); err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(key.PublicKey)
	if err := publicKey.AddChild(doc.CreateText([]byte(encoded))); err != nil {
		return nil, err
	}
	if err := ecKeyValueElem.AddChild(publicKey); err != nil {
		return nil, err
	}

	if err := keyValue.AddChild(ecKeyValueElem); err != nil {
		return nil, err
	}
	return oki, oki.AddChild(keyValue)
}
