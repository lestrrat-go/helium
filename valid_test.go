package helium_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestExtSubsetLookup_ElementInExtSubset(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dtdPath := filepath.Join(dir, "ext.dtd")
	require.NoError(t, os.WriteFile(dtdPath, []byte(`<!ELEMENT root (child)>
<!ELEMENT child EMPTY>
<!ATTLIST child role CDATA #REQUIRED>`), 0600))

	xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + dtdPath + `">
<root><child role="main"/></root>`

	p := helium.NewParser().DTDLoad(true).DTDValid(true)
	_, err := p.Parse(t.Context(), []byte(xml))
	require.NoError(t, err, "validation should pass when declarations are in extSubset")
}

func TestExtSubsetLookup_EntityInExtSubset(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dtdPath := filepath.Join(dir, "ext.dtd")
	require.NoError(t, os.WriteFile(dtdPath, []byte(`<!ELEMENT root (#PCDATA)>
<!ENTITY extEnt "hello">`), 0600))

	xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + dtdPath + `">
<root/>`

	p := helium.NewParser().DTDLoad(true)
	doc, err := p.Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	ent, found := doc.GetEntity("extEnt")
	require.True(t, found, "entity in extSubset should be found")
	require.Equal(t, "hello", string(ent.Content()))
}

func TestExtSubsetLookup_AttributeInExtSubset(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dtdPath := filepath.Join(dir, "ext.dtd")
	require.NoError(t, os.WriteFile(dtdPath, []byte(`<!ELEMENT root (child)>
<!ELEMENT child EMPTY>
<!ATTLIST child role CDATA #REQUIRED>`), 0600))

	xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + dtdPath + `">
<root><child/></root>`

	p := helium.NewParser().DTDLoad(true).DTDValid(true)
	_, err := p.Parse(t.Context(), []byte(xml))
	require.Error(t, err, "missing REQUIRED attribute from extSubset should fail")
	require.Contains(t, err.Error(), "attribute role is required")
}

func TestExtSubsetLookup_StandaloneYesPreventsExtSubset(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dtdPath := filepath.Join(dir, "ext.dtd")
	require.NoError(t, os.WriteFile(dtdPath, []byte(`<!ELEMENT root (child)>
<!ELEMENT child EMPTY>
<!ENTITY extEnt "hello">`), 0600))

	xml := `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE root SYSTEM "` + dtdPath + `">
<root><child/></root>`

	p := helium.NewParser().DTDLoad(true)
	doc, err := p.Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	_, found := doc.GetEntity("extEnt")
	require.False(t, found, "standalone=yes should prevent extSubset entity lookup")
}

func TestEnumerationAttributeValidation(t *testing.T) {
	t.Parallel()

	t.Run("valid value accepted", func(t *testing.T) {
		t.Parallel()

		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root color (red|green|blue) #REQUIRED>
]>
<root color="green"/>`
		p := helium.NewParser().DTDValid(true).DTDAttr(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
	})

	t.Run("invalid value rejected", func(t *testing.T) {
		t.Parallel()

		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root color (red|green|blue) #REQUIRED>
]>
<root color="yellow"/>`
		p := helium.NewParser().DTDValid(true).DTDAttr(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.Error(t, err)
		require.Contains(t, err.Error(), "not among the enumerated set")
	})

	t.Run("default value used when absent", func(t *testing.T) {
		t.Parallel()

		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root color (red|green|blue) "red">
]>
<root/>`
		p := helium.NewParser().DTDValid(true).DTDAttr(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
	})
}

func TestEntityAttributeValidation(t *testing.T) {
	t.Parallel()

	t.Run("valid unparsed entity", func(t *testing.T) {
		t.Parallel()

		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!NOTATION gif SYSTEM "image/gif">
  <!ENTITY logo SYSTEM "logo.gif" NDATA gif>
  <!ATTLIST root img ENTITY #REQUIRED>
]>
<root img="logo"/>`
		p := helium.NewParser().DTDValid(true).DTDAttr(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
	})

	t.Run("undeclared entity", func(t *testing.T) {
		t.Parallel()

		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root img ENTITY #REQUIRED>
]>
<root img="noSuchEntity"/>`
		p := helium.NewParser().DTDValid(true).DTDAttr(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.Error(t, err)
		require.Contains(t, err.Error(), "undeclared entity")
	})

	t.Run("wrong entity type (internal)", func(t *testing.T) {
		t.Parallel()

		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ENTITY internalEnt "hello">
  <!ATTLIST root img ENTITY #REQUIRED>
]>
<root img="internalEnt"/>`
		p := helium.NewParser().DTDValid(true).DTDAttr(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.Error(t, err)
		require.Contains(t, err.Error(), "not unparsed")
	})
}

func TestEntitiesAttributeValidation(t *testing.T) {
	t.Parallel()

	t.Run("valid multiple unparsed entities", func(t *testing.T) {
		t.Parallel()

		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!NOTATION gif SYSTEM "image/gif">
  <!ENTITY logo1 SYSTEM "logo1.gif" NDATA gif>
  <!ENTITY logo2 SYSTEM "logo2.gif" NDATA gif>
  <!ATTLIST root imgs ENTITIES #REQUIRED>
]>
<root imgs="logo1 logo2"/>`
		p := helium.NewParser().DTDValid(true).DTDAttr(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
	})

	t.Run("one undeclared entity", func(t *testing.T) {
		t.Parallel()

		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!NOTATION gif SYSTEM "image/gif">
  <!ENTITY logo1 SYSTEM "logo1.gif" NDATA gif>
  <!ATTLIST root imgs ENTITIES #REQUIRED>
]>
<root imgs="logo1 noSuchEntity"/>`
		p := helium.NewParser().DTDValid(true).DTDAttr(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.Error(t, err)
		require.Contains(t, err.Error(), "undeclared entity")
	})
}

func TestNotationAttributeValidation(t *testing.T) {
	t.Parallel()

	t.Run("valid notation", func(t *testing.T) {
		t.Parallel()

		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!NOTATION gif SYSTEM "image/gif">
  <!NOTATION png SYSTEM "image/png">
  <!ATTLIST root fmt NOTATION (gif|png) #REQUIRED>
]>
<root fmt="gif"/>`
		p := helium.NewParser().DTDValid(true).DTDAttr(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
	})

	t.Run("undeclared notation", func(t *testing.T) {
		t.Parallel()

		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!NOTATION gif SYSTEM "image/gif">
  <!ATTLIST root fmt NOTATION (gif|png) #REQUIRED>
]>
<root fmt="png"/>`
		p := helium.NewParser().DTDValid(true).DTDAttr(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.Error(t, err)
		require.Contains(t, err.Error(), "undeclared notation")
	})
}
