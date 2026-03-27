package xslt3_test

import (
	"io"
	"os"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
)

type fuzzURIResolver struct{}

func (fuzzURIResolver) Resolve(string) (io.ReadCloser, error) {
	return nil, os.ErrNotExist
}

type fuzzPackageResolver struct{}

func (fuzzPackageResolver) ResolvePackage(string, string) (io.ReadCloser, string, error) {
	return nil, "", os.ErrNotExist
}

const fuzzStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:value-of select="name(/*)"/></out></xsl:template>
</xsl:stylesheet>`

const fuzzSource = `<?xml version="1.0"?><root><item>1</item></root>`

func FuzzCompile(f *testing.F) {
	f.Add([]byte(fuzzStylesheet))
	f.Add([]byte(`<?xml version="1.0"?><xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"><xsl:template match="/"><out/></xsl:template></xsl:stylesheet>`))
	f.Add([]byte(``))
	f.Add([]byte(`not a stylesheet`))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			return
		}

		doc, err := helium.NewParser().Parse(t.Context(), data)
		if err != nil {
			return
		}

		_, _ = xslt3.NewCompiler().
			BaseURI("file:///fuzz/main.xsl").
			URIResolver(fuzzURIResolver{}).
			PackageResolver(fuzzPackageResolver{}).
			Compile(t.Context(), doc)
	})
}

func FuzzTransform(f *testing.F) {
	f.Add([]byte(fuzzStylesheet))
	f.Add([]byte(`<?xml version="1.0"?><xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"><xsl:template match="/"><out>ok</out></xsl:template></xsl:stylesheet>`))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			return
		}

		styleDoc, err := helium.NewParser().Parse(t.Context(), data)
		if err != nil {
			return
		}

		ss, err := xslt3.NewCompiler().
			BaseURI("file:///fuzz/main.xsl").
			URIResolver(fuzzURIResolver{}).
			PackageResolver(fuzzPackageResolver{}).
			Compile(t.Context(), styleDoc)
		if err != nil {
			return
		}

		sourceDoc, err := helium.NewParser().Parse(t.Context(), []byte(fuzzSource))
		if err != nil {
			t.Fatalf("parse source doc: %v", err)
		}

		_, _ = ss.Transform(sourceDoc).Serialize(t.Context())
	})
}
