package shim_test

import (
	"bytes"
	stdxml "encoding/xml"
	"fmt"
	"io"
	"sync"
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

func TestUnmarshalTagSemanticsMatchStdlib(t *testing.T) {
	type anyNode struct {
		XMLName stdxml.Name
		Text    string `xml:",chardata"`
	}
	type payload struct {
		XMLName stdxml.Name `xml:"root"`
		ID      string      `xml:"id,attr"`
		Text    string      `xml:",chardata"`
		Inner   string      `xml:",innerxml"`
		PathVal string      `xml:"a>b>c"`
		Any     anyNode     `xml:",any"`
	}

	input := []byte(`<root id="x">prefix<a><b><c>path</c></b></a><inner><x>1</x></inner><extra>any</extra></root>`)

	var stdOut payload
	var shimOut payload
	require.NoError(t, stdxml.Unmarshal(input, &stdOut))
	require.NoError(t, shim.Unmarshal(input, &shimOut))

	require.Equal(t, stdOut.ID, shimOut.ID, "attr mismatch")
	require.Equal(t, stdOut.PathVal, shimOut.PathVal, "path tag mismatch")
	require.Equal(t, stdOut.Any.XMLName, shimOut.Any.XMLName, "any XMLName mismatch")
	require.Equal(t, stdOut.Any.Text, shimOut.Any.Text, "any text mismatch")
	require.Equal(t, stdOut.Inner, shimOut.Inner, "innerxml mismatch")
	require.Equal(t, stdOut.Text, shimOut.Text, "chardata mismatch")
}

func TestUnmarshalRepeatedConsistency(t *testing.T) {
	type payload struct {
		XMLName stdxml.Name `xml:"item"`
		ID      string      `xml:"id,attr"`
		Value   string      `xml:",chardata"`
	}

	input := []byte(`<item id="42">hello</item>`)

	var want payload
	require.NoError(t, stdxml.Unmarshal(input, &want))

	for i := 0; i < 100; i++ {
		var got payload
		require.NoError(t, shim.Unmarshal(input, &got))
		require.Equal(t, want, got)
	}
}

func TestUnmarshalConcurrentConsistency(t *testing.T) {
	type payload struct {
		XMLName stdxml.Name `xml:"item"`
		ID      string      `xml:"id,attr"`
		Value   string      `xml:",chardata"`
	}

	input := []byte(`<item id="42">hello</item>`)

	var want payload
	require.NoError(t, stdxml.Unmarshal(input, &want))

	const workers = 16
	const iterations = 50

	errCh := make(chan error, workers*iterations)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				var got payload
				if err := shim.Unmarshal(input, &got); err != nil {
					errCh <- err
					return
				}
				if got != want {
					errCh <- fmt.Errorf("unexpected output: got=%+v want=%+v", got, want)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		require.NoError(t, err)
	}
}

type hookAttr string

func (h *hookAttr) UnmarshalXMLAttr(attr stdxml.Attr) error {
	*h = hookAttr("attr:" + attr.Value)
	return nil
}

type hookText string

func (h *hookText) UnmarshalText(text []byte) error {
	*h = hookText("text:" + string(text))
	return nil
}

type hookElem string

func (h *hookElem) UnmarshalXML(dec *stdxml.Decoder, start stdxml.StartElement) error {
	var tmp struct {
		Text string `xml:",chardata"`
	}
	if err := dec.DecodeElement(&tmp, &start); err != nil {
		return err
	}
	*h = hookElem("elem:" + tmp.Text)
	return nil
}

func TestUnmarshalInterfaceHooksMatchStdlib(t *testing.T) {
	type payload struct {
		XMLName stdxml.Name `xml:"root"`
		ID      hookAttr    `xml:"id,attr"`
		Name    hookText    `xml:"name"`
		Item    hookElem    `xml:"item"`
	}

	input := []byte(`<root id="x"><name>n</name><item>v</item></root>`)

	var stdOut payload
	var shimOut payload
	require.NoError(t, stdxml.Unmarshal(input, &stdOut))
	require.NoError(t, shim.Unmarshal(input, &shimOut))
	require.Equal(t, stdOut, shimOut)
}

func TestUnmarshalNamespaceTagsMatchStdlib(t *testing.T) {
	type payload struct {
		XMLName stdxml.Name `xml:"urn:root root"`
		ID      string      `xml:"urn:attr id,attr"`
		Child   string      `xml:"urn:root child"`
	}

	input := []byte(`<root xmlns="urn:root" xmlns:a="urn:attr" a:id="42"><child>hello</child></root>`)

	var stdOut payload
	var shimOut payload
	require.NoError(t, stdxml.Unmarshal(input, &stdOut))
	require.NoError(t, shim.Unmarshal(input, &shimOut))
	require.Equal(t, stdOut, shimOut)
}

func TestUnmarshalEmbeddedFieldMatchStdlib(t *testing.T) {
	type Embedded struct {
		ID   string `xml:"id,attr"`
		Name string `xml:"name"`
	}
	type payload struct {
		XMLName stdxml.Name `xml:"root"`
		Embedded
	}

	input := []byte(`<root id="x"><name>n</name></root>`)

	var stdOut payload
	var shimOut payload
	require.NoError(t, stdxml.Unmarshal(input, &stdOut))
	require.NoError(t, shim.Unmarshal(input, &shimOut))
	require.Equal(t, stdOut, shimOut)
}

func TestUnmarshalEmbeddedConflictPrecedenceMatchStdlib(t *testing.T) {
	type Embedded struct {
		Name string `xml:"name"`
	}
	type payload struct {
		Embedded
		Name string `xml:"name"`
	}

	input := []byte(`<payload><name>v</name></payload>`)

	var stdOut payload
	stdErr := stdxml.Unmarshal(input, &stdOut)

	var shimOut payload
	shimErr := shim.Unmarshal(input, &shimOut)

	require.Equal(t, stdErr == nil, shimErr == nil)
	if stdErr == nil {
		require.Equal(t, stdOut, shimOut)
	}
}

func TestUnmarshalEmbeddedPointerMatchStdlib(t *testing.T) {
	type Embedded struct {
		ID string `xml:"id,attr"`
	}
	type payload struct {
		*Embedded
	}

	input := []byte(`<payload id="x"/>`)

	var stdOut payload
	var shimOut payload
	require.NoError(t, stdxml.Unmarshal(input, &stdOut))
	require.NoError(t, shim.Unmarshal(input, &shimOut))
	require.Equal(t, stdOut, shimOut)
}

func TestUnmarshalRecursiveEmbeddedDoesNotRecurseForever(t *testing.T) {
	type Recursive struct {
		Value string `xml:"value"`
		*Recursive
	}

	input := []byte(`<root><value>ok</value></root>`)

	var shimOut Recursive
	require.NoError(t, shim.Unmarshal(input, &shimOut))
	require.Equal(t, "ok", shimOut.Value)
}

func TestUnmarshalNamespacePathMatchStdlib(t *testing.T) {
	type payload struct {
		Value string `xml:"urn:root a>b"`
	}

	input := []byte(`<root xmlns="urn:root"><a><b>ok</b></a></root>`)

	var stdOut payload
	var shimOut payload
	require.NoError(t, stdxml.Unmarshal(input, &stdOut))
	require.NoError(t, shim.Unmarshal(input, &shimOut))
	require.Equal(t, stdOut, shimOut)
}

func TestUnmarshalNamespaceAttrMismatchMatchStdlib(t *testing.T) {
	type payload struct {
		ID string `xml:"urn:other id,attr"`
	}

	input := []byte(`<root xmlns:a="urn:attr" a:id="42"/>`)

	var stdOut payload
	var shimOut payload
	require.NoError(t, stdxml.Unmarshal(input, &stdOut))
	require.NoError(t, shim.Unmarshal(input, &shimOut))
	require.Equal(t, stdOut, shimOut)
}

func TestUnmarshalNumericOverflowMatchStdlib(t *testing.T) {
	type payload struct {
		I8  int8  `xml:"i8"`
		U8  uint8 `xml:"u8"`
		I16 int16 `xml:"i16"`
	}

	input := []byte(`<root><i8>128</i8><u8>999</u8><i16>12</i16></root>`)

	var stdOut payload
	stdErr := stdxml.Unmarshal(input, &stdOut)

	var shimOut payload
	shimErr := shim.Unmarshal(input, &shimOut)

	require.Equal(t, stdErr == nil, shimErr == nil)
}

func TestUnmarshalUnsignedNegativeMatchStdlib(t *testing.T) {
	type payload struct {
		U uint `xml:"u"`
	}

	input := []byte(`<root><u>-1</u></root>`)

	var stdOut payload
	stdErr := stdxml.Unmarshal(input, &stdOut)

	var shimOut payload
	shimErr := shim.Unmarshal(input, &shimOut)

	require.Equal(t, stdErr == nil, shimErr == nil)
}

func TestUnmarshalFloat32OverflowMatchStdlib(t *testing.T) {
	type payload struct {
		F float32 `xml:"f"`
	}

	input := []byte(`<root><f>1e40</f></root>`)

	var stdOut payload
	stdErr := stdxml.Unmarshal(input, &stdOut)

	var shimOut payload
	shimErr := shim.Unmarshal(input, &shimOut)

	require.Equal(t, stdErr == nil, shimErr == nil)
}

func TestUnmarshalUnsupportedFieldTypeMatchStdlib(t *testing.T) {
	type payload struct {
		M map[string]string `xml:"m"`
	}

	input := []byte(`<root><m><k>v</k></m></root>`)

	var stdOut payload
	stdErr := stdxml.Unmarshal(input, &stdOut)

	var shimOut payload
	shimErr := shim.Unmarshal(input, &shimOut)

	require.Equal(t, stdErr == nil, shimErr == nil)
}

func TestUnmarshalSliceElementsMatchStdlib(t *testing.T) {
	type payload struct {
		Items []string `xml:"item"`
	}

	input := []byte(`<root><item>a</item><item>b</item><item>c</item></root>`)

	var stdOut payload
	var shimOut payload
	require.NoError(t, stdxml.Unmarshal(input, &stdOut))
	require.NoError(t, shim.Unmarshal(input, &shimOut))
	require.Equal(t, stdOut, shimOut)
}

func TestUnmarshalSliceStructElementsMatchStdlib(t *testing.T) {
	type item struct {
		Value string `xml:",chardata"`
	}
	type payload struct {
		Items []item `xml:"item"`
	}

	input := []byte(`<root><item>a</item><item>b</item></root>`)

	var stdOut payload
	var shimOut payload
	require.NoError(t, stdxml.Unmarshal(input, &stdOut))
	require.NoError(t, shim.Unmarshal(input, &shimOut))
	require.Equal(t, stdOut, shimOut)
}

func TestDecoderDecodeMatchesStdlib(t *testing.T) {
	type item struct {
		XMLName stdxml.Name `xml:"item"`
		ID      string      `xml:"id,attr"`
		Value   string      `xml:",chardata"`
	}

	input := []byte(`<item id="42">hello</item>`)

	var stdItem item
	stdDec := stdxml.NewDecoder(bytes.NewReader(input))
	require.NoError(t, stdDec.Decode(&stdItem))

	var shimItem item
	shimDec := shim.NewDecoder(bytes.NewReader(input))
	require.NoError(t, shimDec.Decode(&shimItem))

	require.Equal(t, stdItem, shimItem, "Decode result mismatch")
}

func TestDecoderDecodeElementMatchesStdlib(t *testing.T) {
	type child struct {
		XMLName stdxml.Name `xml:"child"`
		Value   string      `xml:",chardata"`
	}

	input := []byte(`<root><child>one</child><child>two</child></root>`)

	stdDec := stdxml.NewDecoder(bytes.NewReader(input))
	shimDec := shim.NewDecoder(bytes.NewReader(input))

	consumeRootStart := func(next func() (stdxml.Token, error)) stdxml.StartElement {
		for {
			tok, err := next()
			require.NoError(t, err)
			if se, ok := tok.(stdxml.StartElement); ok && se.Name.Local == "root" {
				return se
			}
		}
	}
	_ = consumeRootStart(stdDec.Token)
	_ = consumeRootStart(func() (stdxml.Token, error) { return shimDec.Token() })

	nextChildStart := func(next func() (stdxml.Token, error)) stdxml.StartElement {
		for {
			tok, err := next()
			require.NoError(t, err)
			if se, ok := tok.(stdxml.StartElement); ok && se.Name.Local == "child" {
				return se
			}
		}
	}

	stdStart := nextChildStart(stdDec.Token)
	shimStart := nextChildStart(func() (stdxml.Token, error) { return shimDec.Token() })

	var stdChild child
	var shimChild child
	require.NoError(t, stdDec.DecodeElement(&stdChild, &stdStart))
	require.NoError(t, shimDec.DecodeElement(&shimChild, &shimStart))
	require.Equal(t, stdChild, shimChild, "DecodeElement first child mismatch")
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
