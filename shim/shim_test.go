package shim_test

import (
	"bytes"
	stdxml "encoding/xml"
	"io"
	"testing"

	"github.com/lestrrat-go/helium/shim"
	"github.com/stretchr/testify/require"
)

func TestHeaderMatchesStdlib(t *testing.T) {
	require.Equal(t, stdxml.Header, shim.Header, "Header mismatch")
}

func TestHTMLEntityExposed(t *testing.T) {
	got, ok := shim.HTMLEntity["amp"]
	require.True(t, ok, "HTMLEntity should contain amp")
	require.Equal(t, "&", got, "HTMLEntity['amp'] mismatch")
}

func TestEscapeTextMatchesStdlib(t *testing.T) {
	input := []byte(`a&<b>"`)

	var stdBuf bytes.Buffer
	require.NoError(t, stdxml.EscapeText(&stdBuf, input), "stdlib EscapeText failed")

	var shimBuf bytes.Buffer
	require.NoError(t, shim.EscapeText(&shimBuf, input), "shim EscapeText failed")

	require.Equal(t, stdBuf.Bytes(), shimBuf.Bytes(), "EscapeText mismatch")
}

func TestNewDecoderToken(t *testing.T) {
	dec := shim.NewDecoder(bytes.NewReader([]byte(`<root/>`)))
	require.NotNil(t, dec, "NewDecoder returned nil")

	tok, err := dec.Token()
	require.NoError(t, err, "Token failed")
	se, ok := tok.(shim.StartElement)
	require.True(t, ok, "expected first token StartElement, got %T", tok)
	require.Equal(t, "root", se.Name.Local, "start element mismatch")
}

func TestNewTokenDecoder(t *testing.T) {
	src := stdxml.NewDecoder(bytes.NewReader([]byte(`<root/>`)))
	dec := shim.NewTokenDecoder(src)
	require.NotNil(t, dec, "NewTokenDecoder returned nil")

	tok, err := dec.Token()
	require.NoError(t, err, "Token failed")
	_, ok := tok.(shim.StartElement)
	require.True(t, ok, "expected StartElement, got %T", tok)
}

func TestNewEncoderEncodeTokenFlush(t *testing.T) {
	var stdBuf bytes.Buffer
	stdEnc := stdxml.NewEncoder(&stdBuf)
	requireEncodeTokenSequence(t, stdEnc)

	var shimBuf bytes.Buffer
	shimEnc := shim.NewEncoder(&shimBuf)
	requireEncodeTokenSequence(t, shimEnc)

	require.Equal(t, stdBuf.Bytes(), shimBuf.Bytes(), "encoded output mismatch")
}

func requireEncodeTokenSequence(t *testing.T, enc interface {
	EncodeToken(stdxml.Token) error
	Flush() error
}) {
	t.Helper()
	require.NoError(t, enc.EncodeToken(stdxml.StartElement{Name: stdxml.Name{Local: "root"}}), "EncodeToken(start) failed")
	require.NoError(t, enc.EncodeToken(stdxml.CharData([]byte("hello"))), "EncodeToken(chardata) failed")
	require.NoError(t, enc.EncodeToken(stdxml.EndElement{Name: stdxml.Name{Local: "root"}}), "EncodeToken(end) failed")
	require.NoError(t, enc.Flush(), "Flush failed")
}

func TestCopyTokenCopiesCharData(t *testing.T) {
	orig := shim.CharData([]byte("abc"))
	copied, ok := shim.CopyToken(orig).(shim.CharData)
	require.True(t, ok, "expected copied token type CharData")
	orig[0] = 'z'
	require.Equal(t, "abc", string(copied), "CopyToken should deep-copy CharData")
}

func TestMarshalMarshalIndentUnmarshalBasic(t *testing.T) {
	type payload struct {
		XMLName shim.Name `xml:"book"`
		ID      string    `xml:"id,attr"`
		Title   string    `xml:"title"`
	}

	in := payload{ID: "b1", Title: "hello"}

	stdOut, stdErr := stdxml.Marshal(in)
	shimOut, shimErr := shim.Marshal(in)
	require.Equal(t, stdErr != nil, shimErr != nil, "Marshal error mismatch: stdlib=%v shim=%v", stdErr, shimErr)
	if stdErr == nil {
		require.Equal(t, stdOut, shimOut, "Marshal output mismatch")
	}

	stdIndentOut, stdIndentErr := stdxml.MarshalIndent(in, "", "  ")
	shimIndentOut, shimIndentErr := shim.MarshalIndent(in, "", "  ")
	require.Equal(t, stdIndentErr != nil, shimIndentErr != nil, "MarshalIndent error mismatch: stdlib=%v shim=%v", stdIndentErr, shimIndentErr)
	if stdIndentErr == nil {
		require.Equal(t, stdIndentOut, shimIndentOut, "MarshalIndent output mismatch")
	}

	var out payload
	require.NoError(t, shim.Unmarshal(shimOut, &out), "Unmarshal failed")
	require.Equal(t, in.ID, out.ID, "Unmarshal ID mismatch")
	require.Equal(t, in.Title, out.Title, "Unmarshal Title mismatch")
}

func TestAPICompilesWithInterfaces(t *testing.T) {
	var _ io.Reader = bytes.NewReader(nil)
	_ = shim.NewDecoder(bytes.NewReader(nil))
	_ = shim.NewEncoder(io.Discard)
}
