package catalog

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
)

func FuzzLoad(f *testing.F) {
	f.Add([]byte(`<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <public publicId="-//Example//DTD Doc//EN" uri="doc.dtd"/>
  <uri name="urn:example:item" uri="items/example.xml"/>
</catalog>`), "-//Example//DTD Doc//EN", "", "urn:example:item")
	f.Add([]byte(`<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog" prefer="system">
  <system systemId="http://example.com/doc.dtd" uri="doc.dtd"/>
  <rewriteURI uriStartString="http://example.com/" rewritePrefix="/tmp/example/"/>
</catalog>`), "", "http://example.com/doc.dtd", "http://example.com/test.xml")
	f.Add([]byte(``), "", "", "")
	f.Add([]byte(`not a catalog`), "pub", "sys", "uri")

	f.Fuzz(func(t *testing.T, data []byte, pubID, sysID, uri string) {
		if len(data) > 1<<20 || len(pubID) > 4096 || len(sysID) > 4096 || len(uri) > 4096 {
			return
		}

		cat, err := loadFromBytes(t.Context(), data, "file:///fuzz/catalog.xml", helium.NilErrorHandler{})
		if err != nil {
			return
		}

		_ = cat.Resolve(pubID, sysID)
		_ = cat.ResolveURI(uri)
	})
}
