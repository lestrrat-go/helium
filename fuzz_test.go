package helium_test

import (
	"bytes"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func FuzzParse(f *testing.F) {
	f.Add([]byte(`<?xml version="1.0"?><root/>`))
	f.Add([]byte(`<root xmlns="http://example.com" xmlns:ns="http://ns.example.com"><ns:child attr="value">text</ns:child></root>`))
	f.Add([]byte(`<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE root SYSTEM "test.dtd"><root><![CDATA[data]]></root>`))
	f.Add([]byte(`<root><!-- comment --><?pi target?></root>`))
	f.Add([]byte(``))
	f.Add([]byte(`not xml`))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = helium.Parse(t.Context(), data)
	})
}

func FuzzParseRoundtrip(f *testing.F) {
	f.Add([]byte(`<?xml version="1.0"?><root/>`))
	f.Add([]byte(`<root xmlns="http://example.com"><child attr="val">text</child></root>`))
	f.Add([]byte(`<?xml version="1.0"?><root><a><b><c>deep</c></b></a></root>`))

	f.Fuzz(func(t *testing.T, data []byte) {
		doc, err := helium.Parse(t.Context(), data)
		if err != nil {
			return
		}

		var buf bytes.Buffer
		w := helium.NewWriter()
		err = w.WriteDoc(&buf, doc)
		require.NoError(t, err)

		_, err = helium.Parse(t.Context(), buf.Bytes())
		require.NoError(t, err)
	})
}
