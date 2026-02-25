package catalog

import (
	"fmt"
	"os"
	"path/filepath"

	helium "github.com/lestrrat-go/helium"
	icatalog "github.com/lestrrat-go/helium/internal/catalog"
)

// loader implements icatalog.Loader using helium's parser.
type loader struct{}

func (loader) Load(filename string) (*icatalog.Catalog, error) {
	return loadInternal(filename)
}

// Load parses an OASIS XML Catalog file and returns a Catalog.
func Load(filename string) (*Catalog, error) {
	ic, err := loadInternal(filename)
	if err != nil {
		return nil, err
	}
	return &Catalog{cat: ic}, nil
}

func loadInternal(filename string) (*icatalog.Catalog, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("catalog: failed to resolve path %q: %w", filename, err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("catalog: failed to read %q: %w", absPath, err)
	}

	return loadFromBytes(data, absPath)
}

func loadFromBytes(data []byte, baseURI string) (*icatalog.Catalog, error) {
	p := helium.NewParser()
	doc, err := p.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("catalog: failed to parse %q: %w", baseURI, err)
	}

	root := documentElement(doc)
	if root == nil {
		return nil, fmt.Errorf("catalog: no root element in %q", baseURI)
	}

	if root.URI() != icatalog.CatalogNamespace {
		return nil, fmt.Errorf("catalog: root element namespace %q is not %q in %q",
			root.URI(), icatalog.CatalogNamespace, baseURI)
	}

	cat := &icatalog.Catalog{
		Pref:    icatalog.PreferPublic, // default per OASIS spec
		BaseURI: baseURI,
		Ldr:     loader{},
	}

	if v := getAttr(root, "prefer"); v != "" {
		cat.Pref = icatalog.ParsePrefer(v)
	}

	parseEntries(root, cat.Pref, baseURI, &cat.Entries)

	return cat, nil
}

// parseEntries walks child elements of parent and appends catalog entries.
func parseEntries(parent *helium.Element, prefer icatalog.Prefer, baseURI string, entries *[]icatalog.Entry) {
	if v := getAttrNS(parent, "base", "http://www.w3.org/XML/1998/namespace"); v != "" {
		baseURI = icatalog.ResolveURI(baseURI, v)
	}

	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		elem := child.(*helium.Element)

		if elem.URI() != icatalog.CatalogNamespace {
			continue
		}

		localName := elem.LocalName()

		elemBase := baseURI
		if v := getAttrNS(elem, "base", "http://www.w3.org/XML/1998/namespace"); v != "" {
			elemBase = icatalog.ResolveURI(baseURI, v)
		}

		elemPrefer := prefer
		if v := getAttr(elem, "prefer"); v != "" {
			elemPrefer = icatalog.ParsePrefer(v)
		}

		switch localName {
		case "public":
			pubID := icatalog.NormalizePublicID(getAttr(elem, "publicId"))
			uri := icatalog.ResolveURI(elemBase, getAttr(elem, "uri"))
			if pubID != "" && uri != "" {
				*entries = append(*entries, icatalog.Entry{
					Typ:    icatalog.EntryPublic,
					Name:   pubID,
					URL:    uri,
					Prefer: elemPrefer,
				})
			}
		case "system":
			sysID := getAttr(elem, "systemId")
			uri := icatalog.ResolveURI(elemBase, getAttr(elem, "uri"))
			if sysID != "" && uri != "" {
				*entries = append(*entries, icatalog.Entry{
					Typ:  icatalog.EntrySystem,
					Name: sysID,
					URL:  uri,
				})
			}
		case "rewriteSystem":
			startString := getAttr(elem, "systemIdStartString")
			prefix := icatalog.ResolveURI(elemBase, getAttr(elem, "rewritePrefix"))
			if startString != "" && prefix != "" {
				*entries = append(*entries, icatalog.Entry{
					Typ:  icatalog.EntryRewriteSystem,
					Name: startString,
					URL:  prefix,
				})
			}
		case "delegatePublic":
			startString := icatalog.NormalizePublicID(getAttr(elem, "publicIdStartString"))
			catFile := icatalog.ResolveURI(elemBase, getAttr(elem, "catalog"))
			if startString != "" && catFile != "" {
				*entries = append(*entries, icatalog.Entry{
					Typ:    icatalog.EntryDelegatePublic,
					Name:   startString,
					URL:    catFile,
					Prefer: elemPrefer,
				})
			}
		case "delegateSystem":
			startString := getAttr(elem, "systemIdStartString")
			catFile := icatalog.ResolveURI(elemBase, getAttr(elem, "catalog"))
			if startString != "" && catFile != "" {
				*entries = append(*entries, icatalog.Entry{
					Typ:  icatalog.EntryDelegateSystem,
					Name: startString,
					URL:  catFile,
				})
			}
		case "uri":
			name := getAttr(elem, "name")
			uri := icatalog.ResolveURI(elemBase, getAttr(elem, "uri"))
			if name != "" && uri != "" {
				*entries = append(*entries, icatalog.Entry{
					Typ:  icatalog.EntryURI,
					Name: name,
					URL:  uri,
				})
			}
		case "rewriteURI":
			startString := getAttr(elem, "uriStartString")
			prefix := icatalog.ResolveURI(elemBase, getAttr(elem, "rewritePrefix"))
			if startString != "" && prefix != "" {
				*entries = append(*entries, icatalog.Entry{
					Typ:  icatalog.EntryRewriteURI,
					Name: startString,
					URL:  prefix,
				})
			}
		case "delegateURI":
			startString := getAttr(elem, "uriStartString")
			catFile := icatalog.ResolveURI(elemBase, getAttr(elem, "catalog"))
			if startString != "" && catFile != "" {
				*entries = append(*entries, icatalog.Entry{
					Typ:  icatalog.EntryDelegateURI,
					Name: startString,
					URL:  catFile,
				})
			}
		case "nextCatalog":
			catFile := icatalog.ResolveURI(elemBase, getAttr(elem, "catalog"))
			if catFile != "" {
				if !icatalog.HasNextCatalog(*entries, catFile) {
					*entries = append(*entries, icatalog.Entry{
						Typ: icatalog.EntryNextCatalog,
						URL: catFile,
					})
				}
			}
		case "group":
			parseEntries(elem, elemPrefer, elemBase, entries)
		}
	}
}

// documentElement returns the first child element of a Document.
func documentElement(doc *helium.Document) *helium.Element {
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == helium.ElementNode {
			return child.(*helium.Element)
		}
	}
	return nil
}

// getAttr returns the value of the attribute with the given local name.
func getAttr(elem *helium.Element, name string) string {
	for _, a := range elem.Attributes() {
		if a.LocalName() == name {
			return a.Value()
		}
	}
	return ""
}

// getAttrNS returns the value of the attribute with the given local name
// and namespace URI.
func getAttrNS(elem *helium.Element, name, nsURI string) string {
	for _, a := range elem.Attributes() {
		if a.LocalName() == name && a.URI() == nsURI {
			return a.Value()
		}
	}
	return ""
}
