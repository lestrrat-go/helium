package heliumcmd_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/internal/cli/heliumcmd"
	"github.com/stretchr/testify/require"
)

// TestCLIExternalLoadingOptIn locks recommendation-A behavior: bare lint is
// safe-by-default (no external DTD loading), while a loading flag transparently
// lifts the XXE block and installs a permissive FS so the DTD is read.
func TestCLIExternalLoadingOptIn(t *testing.T) {
	const dtd = `<!ELEMENT doc (#PCDATA)>` + "\n" + `<!ATTLIST doc status CDATA "active">`
	const doc = `<?xml version="1.0"?>` + "\n" + `<!DOCTYPE doc SYSTEM "note.dtd">` + "\n" + `<doc>hi</doc>`

	t.Run("external DTD not loaded without flags", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "note.dtd", dtd)
		xml := writeFile(t, dir, "doc.xml", doc)

		out, errOut, code := executeLintFile(t, xml)
		require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)
		require.NotContains(t, out, `status="active"`,
			"bare lint must not load the external DTD")
	})

	t.Run("loading flags load the external DTD", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "note.dtd", dtd)
		xml := writeFile(t, dir, "doc.xml", doc)

		out, errOut, code := executeLintFile(t, xml, "--loaddtd", "--dtdattr")
		require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)
		require.Contains(t, out, `status="active"`,
			"--loaddtd --dtdattr must lift the XXE block and load the DTD")
	})
}

// TestCLIMaxDepth locks the --max-depth flag and the default depth cap.
func TestCLIMaxDepth(t *testing.T) {
	const depth3 = `<a><b><c/></b></a>`
	deep := strings.Repeat("<a>", 300) + strings.Repeat("</a>", 300)

	t.Run("--max-depth rejects over the limit", func(t *testing.T) {
		_, _, code := executeLintStdin(t, depth3, "--noout", "--max-depth", "2")
		require.NotEqual(t, heliumcmd.ExitOK, code)
	})

	t.Run("--max-depth allows at the limit", func(t *testing.T) {
		_, errOut, code := executeLintStdin(t, depth3, "--noout", "--max-depth", "3")
		require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)
	})

	t.Run("--max-depth 0 disables the cap", func(t *testing.T) {
		_, errOut, code := executeLintStdin(t, deep, "--noout", "--max-depth", "0")
		require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)
	})

	t.Run("default depth cap rejects deep input", func(t *testing.T) {
		_, _, code := executeLintStdin(t, deep, "--noout")
		require.NotEqual(t, heliumcmd.ExitOK, code,
			"the default 256 depth cap must reject 300-deep nesting")
	})
}
