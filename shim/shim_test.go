package shim_test

import (
	"bytes"
	stdxml "encoding/xml"
	"fmt"
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

func TestRawTokenSequenceMatchesStdlib(t *testing.T) {
	input := []byte(`<root><a x="1">t</a><!--c--><?pi d?></root>`)

	stdDec := stdxml.NewDecoder(bytes.NewReader(input))
	shimDec := shim.NewDecoder(bytes.NewReader(input))

	var stdSeq []string
	for {
		tok, err := stdDec.RawToken()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		stdSeq = append(stdSeq, tokenRepr(tok))
	}

	var shimSeq []string
	for {
		tok, err := shimDec.RawToken()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		shimSeq = append(shimSeq, tokenRepr(tok))
	}

	require.Equal(t, stdSeq, shimSeq, "RawToken sequence mismatch")
}

func TestSkipBehaviorMatchesStdlib(t *testing.T) {
	input := []byte(`<root><skip><a/><b>txt</b></skip><keep/></root>`)

	stdDec := stdxml.NewDecoder(bytes.NewReader(input))
	shimDec := shim.NewDecoder(bytes.NewReader(input))

	consumeUntilStart := func(next func() (stdxml.Token, error), name string) {
		for {
			tok, err := next()
			require.NoError(t, err)
			se, ok := tok.(stdxml.StartElement)
			if ok && se.Name.Local == name {
				return
			}
		}
	}

	consumeUntilStart(stdDec.Token, "skip")
	consumeUntilStart(func() (stdxml.Token, error) { return shimDec.Token() }, "skip")

	require.NoError(t, stdDec.Skip())
	require.NoError(t, shimDec.Skip())

	var stdNext, shimNext string
	for {
		tok, err := stdDec.Token()
		require.NoError(t, err)
		if se, ok := tok.(stdxml.StartElement); ok {
			stdNext = se.Name.Local
			break
		}
	}
	for {
		tok, err := shimDec.Token()
		require.NoError(t, err)
		if se, ok := tok.(shim.StartElement); ok {
			shimNext = se.Name.Local
			break
		}
	}

	require.Equal(t, "keep", stdNext)
	require.Equal(t, stdNext, shimNext)
}

func TestInputOffsetAndPosMatchStdlib(t *testing.T) {
	input := []byte("<root>abc</root>")
	stdDec := stdxml.NewDecoder(bytes.NewReader(input))
	shimDec := shim.NewDecoder(bytes.NewReader(input))

	for i := 0; i < 2; i++ {
		_, err := stdDec.Token()
		require.NoError(t, err)
		_, err = shimDec.Token()
		require.NoError(t, err)
	}

	require.Equal(t, stdDec.InputOffset(), shimDec.InputOffset(), "InputOffset mismatch")

	stdLine, stdCol := stdDec.InputPos()
	shimLine, shimCol := shimDec.InputPos()
	require.Equal(t, stdLine, shimLine, "InputPos line mismatch")
	require.Equal(t, stdCol, shimCol, "InputPos col mismatch")
}

func TestEncoderEncodeAndEncodeElementMatchStdlib(t *testing.T) {
	type item struct {
		XMLName stdxml.Name `xml:"item"`
		Value   string      `xml:",chardata"`
	}

	val := item{Value: "hello"}

	var stdBuf bytes.Buffer
	stdEnc := stdxml.NewEncoder(&stdBuf)
	require.NoError(t, stdEnc.Encode(val))
	require.NoError(t, stdEnc.EncodeElement(item{Value: "world"}, stdxml.StartElement{Name: stdxml.Name{Local: "x"}}))
	require.NoError(t, stdEnc.Flush())

	var shimBuf bytes.Buffer
	shimEnc := shim.NewEncoder(&shimBuf)
	require.NoError(t, shimEnc.Encode(val))
	require.NoError(t, shimEnc.EncodeElement(item{Value: "world"}, stdxml.StartElement{Name: stdxml.Name{Local: "x"}}))
	require.NoError(t, shimEnc.Flush())

	require.Equal(t, stdBuf.Bytes(), shimBuf.Bytes(), "Encode/EncodeElement output mismatch")
}

func TestEncoderIndentMatchesStdlib(t *testing.T) {
	type root struct {
		XMLName stdxml.Name `xml:"root"`
		Child   string      `xml:"child"`
	}

	v := root{Child: "v"}

	var stdBuf bytes.Buffer
	stdEnc := stdxml.NewEncoder(&stdBuf)
	stdEnc.Indent("", "  ")
	require.NoError(t, stdEnc.Encode(v))
	require.NoError(t, stdEnc.Flush())

	var shimBuf bytes.Buffer
	shimEnc := shim.NewEncoder(&shimBuf)
	shimEnc.Indent("", "  ")
	require.NoError(t, shimEnc.Encode(v))
	require.NoError(t, shimEnc.Flush())

	require.Equal(t, stdBuf.Bytes(), shimBuf.Bytes(), "Indent output mismatch")
}

func tokenRepr(tok stdxml.Token) string {
	switch v := tok.(type) {
	case stdxml.StartElement:
		return fmt.Sprintf("S:%s attrs=%v", v.Name.Local, v.Attr)
	case stdxml.EndElement:
		return fmt.Sprintf("E:%s", v.Name.Local)
	case stdxml.CharData:
		return fmt.Sprintf("C:%s", string(v))
	case stdxml.Comment:
		return fmt.Sprintf("M:%s", string(v))
	case stdxml.ProcInst:
		return fmt.Sprintf("P:%s:%s", v.Target, string(v.Inst))
	case stdxml.Directive:
		return fmt.Sprintf("D:%s", string(v))
	default:
		return fmt.Sprintf("%T", tok)
	}
}
