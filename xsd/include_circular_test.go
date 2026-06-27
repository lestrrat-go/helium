package xsd_test

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// uriReadFS is a minimal in-memory fs.FS keyed by the canonical URI names the
// xsd compiler hands to its loader, used to exercise the resolver/in-memory
// Compile path. fs.ReadFile prefers ReadFile when available, so only that
// method needs real backing.
type uriReadFS map[string][]byte

func (uriReadFS) Open(string) (fs.File, error) { return nil, fs.ErrNotExist }

func (f uriReadFS) ReadFile(name string) ([]byte, error) {
	data, ok := f[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return data, nil
}

// A circular xs:include chain (main -> inc -> main) must compile cleanly: the
// re-include of the top-level schema has to be recognized as already-loaded
// rather than re-parsed. Before the fix includeVisited only contained documents
// pulled in via loadInclude/loadRedefine, so the back-reference to main re-parsed
// it and re-registered its declarations, producing spurious duplicate-component
// errors. CompileFile seeds includeVisited with the root's resolved key to close
// the cycle.
func TestCompileFile_CircularInclude(t *testing.T) {
	const ns = "urn:c"

	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.xsd")
	require.NoError(t, os.WriteFile(mainPath, []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="`+ns+`" targetNamespace="`+ns+`">
  <xs:include schemaLocation="inc.xsd"/>
  <xs:element name="root" type="t:LeafType"/>
</xs:schema>`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "inc.xsd"), []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="`+ns+`">
  <xs:include schemaLocation="main.xsd"/>
  <xs:complexType name="LeafType">
    <xs:sequence>
      <xs:element name="x" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`), 0o600))

	schema, err := xsd.NewCompiler().FS(helium.PermissiveFS()).CompileFile(t.Context(), mainPath)
	require.NoError(t, err, "circular include back to the root schema must compile without duplicate-component errors")
	require.NotNil(t, schema)
}

// A malformed identity-constraint declared in an INCLUDED schema must be reported
// under the INCLUDED schema's filename (where its line number is meaningful), not
// the INCLUDING schema's. reportIDCStructureError previously used c.filename (the
// top-level file) instead of c.diagSource() (the currently-included file), so an
// included malformed IDC was cited under main.xsd with inc.xsd's line.
func TestCompileFile_IncludedMalformedIDCFilename(t *testing.T) {
	const ns = "urn:t"

	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.xsd")
	incPath := filepath.Join(dir, "inc.xsd")
	require.NoError(t, os.WriteFile(mainPath, []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="`+ns+`" targetNamespace="`+ns+`">
  <xs:include schemaLocation="inc.xsd"/>
  <xs:element name="root" type="t:RootType"/>
</xs:schema>`), 0o600))
	// inc.xsd hosts an element whose xs:key puts <field> before <selector> — an
	// order violation in (annotation?, (selector, field+)).
	require.NoError(t, os.WriteFile(incPath, []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="`+ns+`">
  <xs:element name="host">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="k"><xs:field xpath="@id"/><xs:selector xpath="item"/></xs:key>
  </xs:element>
  <xs:complexType name="RootType">
    <xs:sequence>
      <xs:element ref="host"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`), 0o600))

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err := xsd.NewCompiler().Label("main.xsd").FS(helium.PermissiveFS()).ErrorHandler(collector).CompileFile(t.Context(), mainPath)
	require.Error(t, err, "malformed included IDC must fail compilation")
	_ = collector.Close()

	var b strings.Builder
	for _, e := range collector.Errors() {
		b.WriteString(e.Error())
		b.WriteString("\n")
	}
	errs := b.String()
	// The include's display location is path.Join(dir(main label), "inc.xsd"); the
	// main label has no directory, so it is the bare "inc.xsd".
	incLabel := path.Join(path.Dir("main.xsd"), "inc.xsd")
	require.Contains(t, errs, incLabel+":1", "malformed IDC must be cited under the included schema's filename")
	require.Contains(t, errs, "The content is not valid. Expected is (annotation?, (selector, field+)).")
	require.NotContains(t, errs, "main.xsd:", "must not be cited under the including schema's filename")
}

// The resolver/in-memory Compile path (no filesystem root path, e.g. xslt3
// loading a schema through its URIResolver) must close the same circular include
// cycle as CompileFile. Compile derives the root key from the document's own URL
// (or a full-URI BaseDir); without it a back-reference (main -> inc -> main)
// re-parses main and emits spurious duplicate-component errors.
func TestCompile_CircularInclude_URIBase(t *testing.T) {
	const ns = "urn:c"

	mainBytes := []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="` + ns + `" targetNamespace="` + ns + `">
  <xs:include schemaLocation="inc.xsd"/>
  <xs:element name="root" type="t:LeafType"/>
</xs:schema>`)
	incBytes := []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` + ns + `">
  <xs:include schemaLocation="main.xsd"/>
  <xs:complexType name="LeafType">
    <xs:sequence>
      <xs:element name="x" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`)

	const mainURI = "mem:/schemas/main.xsd"
	fsys := uriReadFS{
		mainURI:                mainBytes,
		"mem:/schemas/inc.xsd": incBytes,
	}

	// Parse main with its own URI as base so doc.URL() carries the canonical
	// location the compiler seeds the cycle guard from.
	doc, err := helium.NewParser().BaseURI(mainURI).Parse(t.Context(), mainBytes)
	require.NoError(t, err)
	require.Equal(t, mainURI, doc.URL())

	schema, err := xsd.NewCompiler().
		BaseDir(mainURI).
		FS(fsys).
		Compile(t.Context(), doc)
	require.NoError(t, err, "circular include back to the root schema must compile cleanly via Compile")
	require.NotNil(t, schema)
}

// The resolver/in-memory Compile path with a RELATIVE doc.URL() and a local
// (non-URI) BaseDir must also close the cycle. A nested back-reference resolves
// its relative schemaLocation against the including schema's base dir, so the
// root's key is BaseDir+doc.URL() ("schemas/main.xsd"), not the bare relative
// URL ("main.xsd"). Seeding the guard from the bare URL would miss the cycle and
// re-parse the root into duplicate components.
func TestCompile_CircularInclude_RelativeURLLocalBase(t *testing.T) {
	const ns = "urn:c"

	mainBytes := []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="` + ns + `" targetNamespace="` + ns + `">
  <xs:include schemaLocation="inc.xsd"/>
  <xs:element name="root" type="t:LeafType"/>
</xs:schema>`)
	incBytes := []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` + ns + `">
  <xs:include schemaLocation="main.xsd"/>
  <xs:complexType name="LeafType">
    <xs:sequence>
      <xs:element name="x" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`)

	fsys := uriReadFS{
		"schemas/main.xsd": mainBytes,
		"schemas/inc.xsd":  incBytes,
	}

	// doc.URL() is the relative "main.xsd"; BaseDir is the local "schemas" dir.
	doc, err := helium.NewParser().BaseURI("main.xsd").Parse(t.Context(), mainBytes)
	require.NoError(t, err)
	require.Equal(t, "main.xsd", doc.URL())

	schema, err := xsd.NewCompiler().
		BaseDir("schemas").
		FS(fsys).
		Compile(t.Context(), doc)
	require.NoError(t, err, "circular include with relative URL + local BaseDir must compile cleanly")
	require.NotNil(t, schema)
}

// The Compile path must also close the cycle when doc.URL() ALREADY carries the
// BaseDir prefix (an already-resolved fs key like "schemas/main.xsd" under
// BaseDir("schemas")). Seeding the guard by blindly resolving doc.URL() against
// BaseDir would double the prefix ("schemas/schemas/main.xsd"), miss the cycle,
// and re-parse the root into spurious duplicate components. The seed must equal
// the key the nested back-reference computes ("schemas/main.xsd"), which it does
// because rootSchemaKey recognizes doc.URL() already sits under BaseDir and seeds
// it unchanged rather than re-joining it onto BaseDir.
func TestCompile_CircularInclude_AlreadyResolvedURLLocalBase(t *testing.T) {
	const ns = "urn:c"

	mainBytes := []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="` + ns + `" targetNamespace="` + ns + `">
  <xs:include schemaLocation="inc.xsd"/>
  <xs:element name="root" type="t:LeafType"/>
</xs:schema>`)
	incBytes := []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` + ns + `">
  <xs:include schemaLocation="main.xsd"/>
  <xs:complexType name="LeafType">
    <xs:sequence>
      <xs:element name="x" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`)

	fsys := uriReadFS{
		"schemas/main.xsd": mainBytes,
		"schemas/inc.xsd":  incBytes,
	}

	// doc.URL() is the already-resolved fs key "schemas/main.xsd"; BaseDir is the
	// same "schemas" dir, so the URL already includes the BaseDir prefix.
	doc, err := helium.NewParser().BaseURI("schemas/main.xsd").Parse(t.Context(), mainBytes)
	require.NoError(t, err)
	require.Equal(t, "schemas/main.xsd", doc.URL())

	schema, err := xsd.NewCompiler().
		BaseDir("schemas").
		FS(fsys).
		Compile(t.Context(), doc)
	require.NoError(t, err, "circular include with an already-BaseDir-relative URL must not double-resolve and must compile cleanly")
	require.NotNil(t, schema)
}

// The Compile path must close the cycle when doc.URL() sits NESTED under BaseDir
// by more than one segment ("schemas/root/main.xsd" under BaseDir("schemas")).
// A nested include's back-reference resolves its relative schemaLocation against
// the including schema's base dir, landing on the root's FULL resolved key
// "schemas/root/main.xsd" — NOT "schemas/main.xsd". Seeding the guard from only
// doc.URL()'s basename joined onto BaseDir would drop the "root/" segment, miss
// the back-reference, and re-parse the root into spurious duplicate components.
func TestCompile_CircularInclude_NestedUnderBaseDir(t *testing.T) {
	const ns = "urn:c"

	mainBytes := []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="` + ns + `" targetNamespace="` + ns + `">
  <xs:include schemaLocation="root/inc.xsd"/>
  <xs:element name="root" type="t:LeafType"/>
</xs:schema>`)
	incBytes := []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` + ns + `">
  <xs:include schemaLocation="main.xsd"/>
  <xs:complexType name="LeafType">
    <xs:sequence>
      <xs:element name="x" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`)

	fsys := uriReadFS{
		"schemas/root/main.xsd": mainBytes,
		"schemas/root/inc.xsd":  incBytes,
	}

	// doc.URL() is nested two segments under BaseDir: "schemas/root/main.xsd"
	// while BaseDir is "schemas". The root's include of "root/inc.xsd" resolves
	// to "schemas/root/inc.xsd"; inc's back-include of "main.xsd" resolves to
	// "schemas/root/main.xsd", which the seed must match exactly.
	doc, err := helium.NewParser().BaseURI("schemas/root/main.xsd").Parse(t.Context(), mainBytes)
	require.NoError(t, err)
	require.Equal(t, "schemas/root/main.xsd", doc.URL())

	schema, err := xsd.NewCompiler().
		BaseDir("schemas").
		FS(fsys).
		Compile(t.Context(), doc)
	require.NoError(t, err, "circular include with a doc URL nested under BaseDir must not drop directory segments and must compile cleanly")
	require.NotNil(t, schema)
}

// The Compile path must close the cycle when doc.URL() carries a directory
// ("schemas/main.xsd") and NO BaseDir is configured. With an unset BaseDir,
// nested includes resolve against "" (the directory embedded in doc.URL() is
// NOT a base), so a back-reference to the root lands on the root's FULL relative
// URL "schemas/main.xsd" — not its basename "main.xsd". Seeding the guard with
// the basename would (a) miss the back-reference and re-parse the root into
// spurious duplicate components, and (b) wrongly skip a DISTINCT real include of
// "main.xsd". Here the root includes both "inc.xsd" (which back-references
// "schemas/main.xsd") and a separate top-level "main.xsd" that must still load.
func TestCompile_CircularInclude_DirURLNoBase(t *testing.T) {
	const ns = "urn:c"

	// Root lives at "schemas/main.xsd"; it includes inc.xsd (cycles back to the
	// root) and a DISTINCT top-level main.xsd (defines TopType). root uses a type
	// from each, so both includes must resolve.
	mainBytes := []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="` + ns + `" targetNamespace="` + ns + `">
  <xs:include schemaLocation="inc.xsd"/>
  <xs:include schemaLocation="main.xsd"/>
  <xs:element name="root" type="t:LeafType"/>
  <xs:element name="root2" type="t:TopType"/>
</xs:schema>`)
	incBytes := []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` + ns + `">
  <xs:include schemaLocation="schemas/main.xsd"/>
  <xs:complexType name="LeafType">
    <xs:sequence>
      <xs:element name="x" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`)
	// Distinct top-level main.xsd: a real include that must NOT be skipped.
	topMainBytes := []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` + ns + `">
  <xs:complexType name="TopType">
    <xs:sequence>
      <xs:element name="y" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`)

	const rootKey = "schemas/main.xsd"
	fsys := uriReadFS{
		// The back-reference key: present so a broken guard re-parses it into
		// spurious duplicates (the bug) rather than failing with not-found.
		rootKey:   mainBytes,
		"inc.xsd": incBytes,
		// The distinct real include keyed by its bare basename.
		"main.xsd": topMainBytes,
	}

	// doc.URL() carries the directory; the compiler has NO BaseDir configured.
	doc, err := helium.NewParser().BaseURI(rootKey).Parse(t.Context(), mainBytes)
	require.NoError(t, err)
	require.Equal(t, rootKey, doc.URL())

	schema, err := xsd.NewCompiler().
		FS(fsys).
		Compile(t.Context(), doc)
	require.NoError(t, err, "dir URL with no BaseDir must guard the cycle yet still load a distinct real main.xsd include")
	require.NotNil(t, schema)
}
