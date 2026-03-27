package xinclude_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xinclude"
)

type fuzzResolver struct {
	files map[string][]byte
}

func (r fuzzResolver) Resolve(href, _ string) (io.ReadCloser, error) {
	data, ok := r.files[href]
	if !ok {
		return nil, errors.New("not found")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func FuzzProcess(f *testing.F) {
	f.Add(
		[]byte(`<?xml version="1.0"?><root xmlns:xi="http://www.w3.org/2001/XInclude"><xi:include href="included.xml"/></root>`),
		[]byte(`<?xml version="1.0"?><included><child/></included>`),
		[]byte(`hello world`),
		false,
		false,
	)
	f.Add(
		[]byte(`<?xml version="1.0"?><root xmlns:xi="http://www.w3.org/2001/XInclude"><xi:include href="included.txt" parse="text"/></root>`),
		[]byte(`<?xml version="1.0"?><ignored/>`),
		[]byte(`text include`),
		true,
		true,
	)

	f.Fuzz(func(t *testing.T, docData, xmlData, textData []byte, noMarkers, noBaseFixup bool) {
		if len(docData) > 1<<20 || len(xmlData) > 1<<20 || len(textData) > 1<<16 {
			return
		}

		doc, err := helium.NewParser().Parse(t.Context(), docData)
		if err != nil {
			return
		}

		proc := xinclude.NewProcessor().
			BaseURI("file:///fuzz/root.xml").
			Resolver(fuzzResolver{
				files: map[string][]byte{
					"included.xml": xmlData,
					"included.txt": textData,
					"nested.xml":   xmlData,
				},
			})
		if noMarkers {
			proc = proc.NoXIncludeMarkers()
		}
		if noBaseFixup {
			proc = proc.NoBaseFixup()
		}

		_, _ = proc.Process(t.Context(), doc)
	})
}
