package helium_test

import (
	"bytes"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestFuzzParseRoundtripRepro contains regression tests for crashes and
// incorrect behaviour discovered by the FuzzParseRoundtrip fuzzer.
func TestFuzzParseRoundtripRepro(t *testing.T) {
	t.Run("empty_local_after_colon_in_attr", func(t *testing.T) {
		// Attribute name "A:" has an empty local part after the colon.
		// parseNmtoken returned ("", nil) for zero-length input, so
		// parseQName accepted an empty local name. The writer then
		// emitted `=""` which is not valid XML.
		data := []byte(`<A A:=""/>`)
		p := helium.NewParser()
		doc, err := p.Parse(t.Context(), data)
		if err != nil {
			return // correctly rejected
		}

		var buf bytes.Buffer
		w := helium.NewWriter()
		err = w.WriteDoc(&buf, doc)
		require.NoError(t, err)

		_, err = p.Parse(t.Context(), buf.Bytes())
		require.NoError(t, err)
	})
	t.Run("invalid_utf8_in_attr_value", func(t *testing.T) {
		// Attribute value contains a truncated UTF-8 sequence (\xd4 without
		// a continuation byte). The parser must reject this as invalid.
		data := []byte("<root A!\"×\xd4\"></root>")
		p := helium.NewParser()
		doc, err := p.Parse(t.Context(), data)
		if err != nil {
			return // correctly rejected
		}

		var buf bytes.Buffer
		w := helium.NewWriter()
		err = w.WriteDoc(&buf, doc)
		require.NoError(t, err)

		_, err = p.Parse(t.Context(), buf.Bytes())
		require.NoError(t, err)
	})
	t.Run("invalid_qname_local_in_attr", func(t *testing.T) {
		p := helium.NewParser()

		for _, data := range [][]byte{
			[]byte(`<root a:0="x"/>`),
			[]byte(`<root a:-="x"/>`),
			[]byte(`<root a:.="x"/>`),
		} {
			_, err := p.Parse(t.Context(), data)
			require.Error(t, err, "input %q should be rejected", data)
		}
	})
}

func FuzzParse(f *testing.F) {
	f.Add([]byte(`<?xml version="1.0"?><root/>`))
	f.Add([]byte(`<root xmlns="http://example.com" xmlns:ns="http://ns.example.com"><ns:child attr="value">text</ns:child></root>`))
	f.Add([]byte(`<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE root SYSTEM "test.dtd"><root><![CDATA[data]]></root>`))
	f.Add([]byte(`<root><!-- comment --><?pi target?></root>`))
	f.Add([]byte(``))
	f.Add([]byte(`not xml`))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			return
		}
		_, _ = helium.NewParser().Parse(t.Context(), data)
	})
}

func FuzzParseRoundtrip(f *testing.F) {
	f.Add([]byte(`<?xml version="1.0"?><root/>`))
	f.Add([]byte(`<root xmlns="http://example.com"><child attr="val">text</child></root>`))
	f.Add([]byte(`<?xml version="1.0"?><root><a><b><c>deep</c></b></a></root>`))

	p := helium.NewParser()
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			return
		}
		doc, err := p.Parse(t.Context(), data)
		if err != nil {
			return
		}

		var buf bytes.Buffer
		w := helium.NewWriter()
		err = w.WriteDoc(&buf, doc)
		require.NoError(t, err)

		_, err = p.Parse(t.Context(), buf.Bytes())
		require.NoError(t, err)
	})
}
