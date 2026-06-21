package heliumcmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/helium/catalog"
	"github.com/stretchr/testify/require"
)

func writeCatalogFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	return p
}

func loadChainCatalog(t *testing.T, path string) *catalog.Catalog {
	t.Helper()
	cat, err := catalog.Load(context.Background(), path)
	require.NoError(t, err)
	return cat
}

// A catalog break in an early catalog must STOP the chain: the search must not
// fall through to a later catalog that would otherwise resolve the identifier.
// The first catalog delegates the systemId to a sub-catalog with no matching
// entry (a "cut"); the second catalog DOES map it. The break must win.
func TestCatalogChainStopsOnBreak(t *testing.T) {
	dir := t.TempDir()

	const sysID = "http://example.com/test.dtd"

	// Sub-catalog reached via delegateSystem that has NO matching entry, which
	// produces a catalog break ("delegates tried, all failed").
	subEmpty := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
</catalog>`
	subEmptyFile := writeCatalogFile(t, dir, "sub-empty.xml", subEmpty)

	firstCat := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <delegateSystem systemIdStartString="http://example.com/" catalog="` + subEmptyFile + `"/>
</catalog>`
	firstFile := writeCatalogFile(t, dir, "first.xml", firstCat)

	// Second catalog DOES map the systemId. Without honoring the break the chain
	// would wrongly resolve here.
	secondCat := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <system systemId="` + sysID + `" uri="file:///should/not/be/used.dtd"/>
</catalog>`
	secondFile := writeCatalogFile(t, dir, "second.xml", secondCat)

	chain := catalogChain{loadChainCatalog(t, firstFile), loadChainCatalog(t, secondFile)}

	got := chain.Resolve(context.Background(), "", sysID)
	require.Equal(t, "", got, "catalog break in first catalog must stop the chain")
}

// A plain no-match (no break) in an early catalog MUST continue to later
// catalogs in the chain.
func TestCatalogChainContinuesOnPlainNoMatch(t *testing.T) {
	dir := t.TempDir()

	const sysID = "http://example.com/test.dtd"

	emptyCat := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
</catalog>`
	emptyFile := writeCatalogFile(t, dir, "empty.xml", emptyCat)

	usefulCat := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <system systemId="` + sysID + `" uri="file:///resolved.dtd"/>
</catalog>`
	usefulFile := writeCatalogFile(t, dir, "useful.xml", usefulCat)

	chain := catalogChain{loadChainCatalog(t, emptyFile), loadChainCatalog(t, usefulFile)}

	got := chain.Resolve(context.Background(), "", sysID)
	require.Equal(t, "file:///resolved.dtd", got, "plain no-match must continue to later catalogs")
}

// The same break/no-match distinction must hold for ResolveURI.
func TestCatalogChainURIStopsOnBreak(t *testing.T) {
	dir := t.TempDir()

	const uri = "http://example.com/asset"

	subEmpty := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
</catalog>`
	subEmptyFile := writeCatalogFile(t, dir, "sub-empty.xml", subEmpty)

	firstCat := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <delegateURI uriStartString="http://example.com/" catalog="` + subEmptyFile + `"/>
</catalog>`
	firstFile := writeCatalogFile(t, dir, "first.xml", firstCat)

	secondCat := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <uri name="` + uri + `" uri="file:///should/not/be/used"/>
</catalog>`
	secondFile := writeCatalogFile(t, dir, "second.xml", secondCat)

	chain := catalogChain{loadChainCatalog(t, firstFile), loadChainCatalog(t, secondFile)}

	got := chain.ResolveURI(context.Background(), uri)
	require.Equal(t, "", got, "catalog break in first catalog must stop the URI chain")
}
