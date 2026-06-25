package heliumcmd_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/internal/cli/heliumcmd"
	"github.com/stretchr/testify/require"
)

// deepXML returns a document nested n elements deep: <a>...<a/>...</a>.
func deepXML(n int) string {
	return strings.Repeat("<a>", n-1) + "<a/>" + strings.Repeat("</a>", n-1)
}

func executeXPath(t *testing.T, xml string, args ...string) (string, int) {
	t.Helper()
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", xml)
	var errBuf bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &errBuf)
	ctx = heliumcmd.WithStdinTTY(ctx, true)
	full := append([]string{cmdXPath, "count(//a)"}, append(args, xmlFile)...)
	code := heliumcmd.Execute(ctx, full)
	return errBuf.String(), code
}

func TestXPathMaxDepth(t *testing.T) {
	// 300-deep document exceeds the parser's default 256 cap.
	deep := deepXML(300)

	t.Run("default cap rejects deep input", func(t *testing.T) {
		_, code := executeXPath(t, deep)
		require.NotEqual(t, heliumcmd.ExitOK, code,
			"the default 256 depth cap must reject 300-deep nesting")
	})

	t.Run("--max-depth 0 disables the cap", func(t *testing.T) {
		errOut, code := executeXPath(t, deep, flagMaxDepth, "0")
		require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)
	})

	t.Run("--max-depth 2 rejects a 3-deep doc", func(t *testing.T) {
		_, code := executeXPath(t, deepXML(3), flagMaxDepth, "2")
		require.NotEqual(t, heliumcmd.ExitOK, code)
	})

	t.Run("invalid argument rejected", func(t *testing.T) {
		errOut, code := executeXPath(t, deepXML(3), flagMaxDepth, "-1")
		require.NotEqual(t, heliumcmd.ExitOK, code)
		require.Contains(t, errOut, "--max-depth: invalid argument")
	})
}

func TestXSDValidateMaxDepth(t *testing.T) {
	// Recursive type: <a> optionally contains a nested <a>, so any depth of
	// well-formed <a> nesting validates once parsing succeeds.
	const schema = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" type="aType"/>
  <xs:complexType name="aType">
    <xs:sequence>
      <xs:element name="a" type="aType" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	run := func(t *testing.T, xml string, args ...string) (string, int) {
		t.Helper()
		dir := t.TempDir()
		schemaFile := writeFile(t, dir, "schema.xsd", schema)
		xmlFile := writeFile(t, dir, "doc.xml", xml)
		var errBuf bytes.Buffer
		ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &errBuf)
		ctx = heliumcmd.WithStdinTTY(ctx, true)
		full := append([]string{cmdXSD, cmdValidate}, append(args, schemaFile, xmlFile)...)
		code := heliumcmd.Execute(ctx, full)
		return errBuf.String(), code
	}

	deep := deepXML(300)

	t.Run("default cap rejects deep input", func(t *testing.T) {
		_, code := run(t, deep)
		require.Equal(t, heliumcmd.ExitErr, code,
			"the default 256 depth cap must fail the parse before validation")
	})

	t.Run("--max-depth 0 disables the cap", func(t *testing.T) {
		errOut, code := run(t, deep, flagMaxDepth, "0")
		require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)
	})

	t.Run("--max-depth 2 rejects a 3-deep doc", func(t *testing.T) {
		_, code := run(t, deepXML(3), flagMaxDepth, "2")
		require.Equal(t, heliumcmd.ExitErr, code)
	})
}

// TestLintHugeLiftsDepthCap verifies that --huge removes the default element
// depth cap (along with the other internal limits), so deeply nested input
// that the default would reject parses successfully.
func TestLintHugeLiftsDepthCap(t *testing.T) {
	deep := deepXML(300)

	_, _, code := executeLintStdin(t, deep, "--noout")
	require.NotEqual(t, heliumcmd.ExitOK, code,
		"default 256 depth cap must reject 300-deep input")

	_, errOut, code := executeLintStdin(t, deep, "--noout", "--huge")
	require.Equal(t, heliumcmd.ExitOK, code,
		"--huge must lift the depth cap; stderr: %s", errOut)
}

// TestLintHugeMaxDepthOrderIndependent verifies that --huge and --max-depth
// produce the same result regardless of flag order: an explicit --max-depth
// (the more specific flag) wins over --huge's limit-lifting.
func TestLintHugeMaxDepthOrderIndependent(t *testing.T) {
	const depth3 = `<a><b><c/></b></a>` // nesting depth 3

	t.Run("--max-depth before --huge still caps", func(t *testing.T) {
		_, _, code := executeLintStdin(t, depth3, "--noout", "--max-depth", "2", "--huge")
		require.NotEqual(t, heliumcmd.ExitOK, code,
			"explicit --max-depth 2 must win over --huge regardless of order")
	})

	t.Run("--huge before --max-depth still caps", func(t *testing.T) {
		_, _, code := executeLintStdin(t, depth3, "--noout", "--huge", "--max-depth", "2")
		require.NotEqual(t, heliumcmd.ExitOK, code)
	})
}
