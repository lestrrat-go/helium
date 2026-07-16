package helium_test

import (
	"bytes"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// netGateFS serves the same content for ANY name and records every name it is
// asked to open. It stands in for a caller-supplied, network-capable fs.FS: its
// Open never fails, so a network-scheme name reaches it unless the parser's
// AllowNetwork(false) gate refuses the load first.
type netGateFS struct {
	content []byte
	opened  *[]string
}

func (fsys netGateFS) Open(name string) (fs.File, error) {
	*fsys.opened = append(*fsys.opened, name)
	return &netGateFile{Reader: bytes.NewReader(fsys.content)}, nil
}

type netGateFile struct {
	*bytes.Reader
}

func (f *netGateFile) Stat() (fs.FileInfo, error) { return netGateInfo{size: f.Size()}, nil }
func (f *netGateFile) Close() error               { return nil }

type netGateInfo struct{ size int64 }

func (netGateInfo) Name() string       { return "resource" }
func (i netGateInfo) Size() int64      { return i.size }
func (netGateInfo) Mode() fs.FileMode  { return 0 }
func (netGateInfo) ModTime() time.Time { return time.Time{} }
func (netGateInfo) IsDir() bool        { return false }
func (netGateInfo) Sys() any           { return nil }

func openedNetwork(names []string) bool {
	for _, n := range names {
		low := strings.ToLower(n)
		if strings.HasPrefix(low, "http://") ||
			strings.HasPrefix(low, "https://") ||
			strings.HasPrefix(low, "ftp://") {
			return true
		}
	}
	return false
}

// A network-scheme external general entity must not reach the fs.FS when
// AllowNetwork is false (the default), and must reach it when AllowNetwork is
// true. The fs.FS here would happily serve any name, so the only thing that can
// keep the network id from being opened is the NONET scheme gate.
func TestAllowNetworkGatesExternalEntity(t *testing.T) {
	t.Parallel()

	doc := `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE root [<!ENTITY e SYSTEM "http://example.com/x.ent">]>` + "\n" +
		`<root>&e;</root>`
	entity := []byte(`<child>net</child>`)

	// Default (AllowNetwork off): the network entity must be refused before Open.
	var openedOff []string
	_, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		SubstituteEntities(true).
		FS(netGateFS{content: entity, opened: &openedOff}).
		Parse(t.Context(), []byte(doc))
	require.Error(t, err, "a network-scheme entity must be refused when AllowNetwork is off")
	require.False(t, openedNetwork(openedOff), "the network id must never reach the fs.FS when AllowNetwork is off")

	// AllowNetwork on: the entity load reaches the fs.FS and expands.
	var openedOn []string
	parsed, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		SubstituteEntities(true).
		AllowNetwork(true).
		FS(netGateFS{content: entity, opened: &openedOn}).
		Parse(t.Context(), []byte(doc))
	require.NoError(t, err, "a network-scheme entity must load when AllowNetwork is on")
	require.True(t, openedNetwork(openedOn), "the network id must reach the fs.FS when AllowNetwork is on")
	require.NotNil(t, parsed)
	root := parsed.DocumentElement()
	require.NotNil(t, root)
	child := root.FirstChild()
	require.NotNil(t, child, "the network entity replacement text must have expanded")
	require.Equal(t, "child", child.(*helium.Element).LocalName())
}

// A network-scheme external DTD subset must not reach the fs.FS when
// AllowNetwork is false, and must reach it when AllowNetwork is true.
func TestAllowNetworkGatesExternalDTD(t *testing.T) {
	t.Parallel()

	doc := `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE root SYSTEM "http://example.com/x.dtd">` + "\n" +
		`<root/>`
	dtd := []byte(`<!ELEMENT root EMPTY>`)

	var openedOff []string
	_, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		FS(netGateFS{content: dtd, opened: &openedOff}).
		Parse(t.Context(), []byte(doc))
	require.False(t, openedNetwork(openedOff), "the network DTD id must never reach the fs.FS when AllowNetwork is off")
	_ = err

	var openedOn []string
	_, err = helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		AllowNetwork(true).
		FS(netGateFS{content: dtd, opened: &openedOn}).
		Parse(t.Context(), []byte(doc))
	require.NoError(t, err)
	require.True(t, openedNetwork(openedOn), "the network DTD id must reach the fs.FS when AllowNetwork is on")
}

// A non-network (bare filename) external entity must load regardless of the
// AllowNetwork setting: the scheme gate only bites http/https/ftp.
func TestAllowNetworkAllowsLocalEntity(t *testing.T) {
	t.Parallel()

	doc := `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE root [<!ENTITY e SYSTEM "local.ent">]>` + "\n" +
		`<root>&e;</root>`
	entity := []byte(`<child>local</child>`)

	for _, allow := range []bool{false, true} {
		var opened []string
		p := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			SubstituteEntities(true).
			FS(netGateFS{content: entity, opened: &opened})
		p = p.AllowNetwork(allow)
		parsed, err := p.Parse(t.Context(), []byte(doc))
		require.NoErrorf(t, err, "a non-network entity must load with AllowNetwork(%v)", allow)
		require.NotNil(t, parsed)
		require.NotEmptyf(t, opened, "the local id must reach the fs.FS with AllowNetwork(%v)", allow)
		require.Falsef(t, openedNetwork(opened), "a bare filename is not a network id (AllowNetwork(%v))", allow)
	}
}
