package xslt3_test

import (
	"bytes"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func TestCharacterMapReplacementSkipsNormalization(t *testing.T) {
	decomposed := "e\u0301"
	composed := "é"
	doc, err := helium.NewParser().Parse(t.Context(), []byte("<out>x"+decomposed+"</out>"))
	require.NoError(t, err)

	var out bytes.Buffer
	require.NoError(t, xslt3.SerializeResult(&out, doc, &xslt3.OutputDef{
		Method:            "xml",
		OmitDeclaration:   true,
		NormalizationForm: "NFC",
		ResolvedCharMap:   map[rune]string{'x': decomposed},
	}))

	require.Equal(t, "<out>"+decomposed+composed+"</out>", out.String())
}

func TestCharacterMapNormalizationKeepsRawDOEContent(t *testing.T) {
	doc := helium.NewDefaultDocument()
	root, err := doc.CreateElement("out")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	// Build the historical metadata-shaped raw bytes without keeping the old
	// protocol spelling as a source literal.
	rawStart := string([]byte{0, 'C', 'M', 'S', 'T', 'A', 'R', 'T', 0})
	rawEnd := string([]byte{0, 'C', 'M', 'E', 'N', 'D', 0})
	decomposed := "e\u0301"
	composed := "é"
	require.NoError(t, root.AddChild(doc.CreatePI("disable-output-escaping", "")))
	require.NoError(t, root.AddChild(doc.CreateText([]byte(rawStart+decomposed+rawEnd))))
	require.NoError(t, root.AddChild(doc.CreatePI("enable-output-escaping", "")))
	require.NoError(t, root.AddChild(doc.CreateText([]byte("x"))))

	var out bytes.Buffer
	require.NoError(t, xslt3.SerializeResult(&out, doc, &xslt3.OutputDef{
		Method:            "xml",
		OmitDeclaration:   true,
		NormalizationForm: "NFC",
		ResolvedCharMap:   map[rune]string{'x': decomposed},
	}))

	// Caller-provided raw content is ordinary output and normalizes to NFC.
	// Only the character-map replacement remains decomposed.
	require.Equal(t, "<out>"+rawStart+composed+rawEnd+decomposed+"</out>", out.String())
}

func TestCharacterMapNormalizationPreservesRawDOEMarkup(t *testing.T) {
	doc := helium.NewDefaultDocument()
	root, err := doc.CreateElement("out")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	decomposed := "e\u0301"
	composed := "é"
	raw := `<inner a="` + decomposed + `">` + decomposed + `</inner>`
	require.NoError(t, root.AddChild(doc.CreatePI("disable-output-escaping", "")))
	require.NoError(t, root.AddChild(doc.CreateText([]byte(raw))))
	require.NoError(t, root.AddChild(doc.CreatePI("enable-output-escaping", "")))

	var out bytes.Buffer
	require.NoError(t, xslt3.SerializeResult(&out, doc, &xslt3.OutputDef{
		Method:            "xml",
		OmitDeclaration:   true,
		NormalizationForm: "NFC",
		ResolvedCharMap:   map[rune]string{'x': "x"},
	}))

	require.Equal(t, `<out><inner a="`+composed+`">`+composed+`</inner></out>`, out.String())
}
