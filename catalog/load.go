package catalog

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	helium "github.com/lestrrat-go/helium"
	icatalog "github.com/lestrrat-go/helium/internal/catalog"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// loader implements icatalog.Loader using helium's parser.
type loader struct {
	ctx          context.Context
	errorHandler helium.ErrorHandler
}

func (l loader) Load(filename string) (*icatalog.Catalog, error) {
	return loadInternal(l.ctx, filename, l.errorHandler)
}

// Load parses an OASIS XML Catalog file and returns a Catalog.
func Load(ctx context.Context, filename string, opts ...LoadOption) (*Catalog, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var cfg loadConfig
	for _, o := range opts {
		o(&cfg)
	}

	var eh helium.ErrorHandler
	if cfg.errorHandler != nil {
		eh = cfg.errorHandler
	} else {
		eh = helium.NilErrorHandler{}
	}

	ic, err := loadInternal(ctx, filename, eh)
	if err != nil {
		closeHandler(eh)
		return nil, err
	}

	closeHandler(eh)
	return &Catalog{cat: ic}, nil
}

func loadInternal(ctx context.Context, filename string, eh helium.ErrorHandler) (*icatalog.Catalog, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("catalog: failed to resolve path %q: %w", filename, err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("catalog: failed to read %q: %w", absPath, err)
	}

	return loadFromBytes(ctx, data, absPath, eh)
}

func loadFromBytes(ctx context.Context, data []byte, baseURI string, eh helium.ErrorHandler) (*icatalog.Catalog, error) {
	p := helium.NewParser()
	doc, err := p.Parse(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("catalog: failed to parse %q: %w", baseURI, err)
	}

	root := documentElement(doc)
	if root == nil {
		return nil, fmt.Errorf("catalog: no root element in %q", baseURI)
	}

	if root.URI() != lexicon.Catalog {
		return nil, fmt.Errorf("catalog: root element namespace %q is not %q in %q",
			root.URI(), lexicon.Catalog, baseURI)
	}

	cat := &icatalog.Catalog{
		Prefer:  icatalog.PreferPublic, // default per OASIS spec
		BaseURI: baseURI,
		Loader:  loader{ctx: ctx, errorHandler: eh},
	}

	if v := getAttr(root, "prefer"); v != "" {
		cat.Prefer = icatalog.ParsePrefer(v)
	}

	parseEntries(ctx, root, cat.Prefer, baseURI, &cat.Entries, eh)

	return cat, nil
}

// parseEntries walks child elements of parent and appends catalog entries.
func parseEntries(ctx context.Context, parent *helium.Element, prefer icatalog.Prefer, baseURI string, entries *[]icatalog.Entry, eh helium.ErrorHandler) {
	if v := getAttrNS(parent, "base", "http://www.w3.org/XML/1998/namespace"); v != "" {
		baseURI = icatalog.ResolveURI(baseURI, v)
	}

	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		elem := child.(*helium.Element)

		if elem.URI() != lexicon.Catalog {
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
			if pubID == "" || uri == "" {
				catalogMissingAttr(ctx, eh, localName, pubID, "publicId", uri, "uri")
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type:   icatalog.EntryPublic,
					Name:   pubID,
					URL:    uri,
					Prefer: elemPrefer,
				})
			}
		case "system":
			sysID := getAttr(elem, "systemId")
			uri := icatalog.ResolveURI(elemBase, getAttr(elem, "uri"))
			if sysID == "" || uri == "" {
				catalogMissingAttr(ctx, eh, localName, sysID, "systemId", uri, "uri")
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntrySystem,
					Name: sysID,
					URL:  uri,
				})
			}
		case "rewriteSystem":
			startString := getAttr(elem, "systemIdStartString")
			prefix := icatalog.ResolveURI(elemBase, getAttr(elem, "rewritePrefix"))
			if startString == "" || prefix == "" {
				catalogMissingAttr(ctx, eh, localName, startString, "systemIdStartString", prefix, "rewritePrefix")
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntryRewriteSystem,
					Name: startString,
					URL:  prefix,
				})
			}
		case "delegatePublic":
			startString := icatalog.NormalizePublicID(getAttr(elem, "publicIdStartString"))
			catFile := icatalog.ResolveURI(elemBase, getAttr(elem, "catalog"))
			if startString == "" || catFile == "" {
				catalogMissingAttr(ctx, eh, localName, startString, "publicIdStartString", catFile, "catalog")
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type:   icatalog.EntryDelegatePublic,
					Name:   startString,
					URL:    catFile,
					Prefer: elemPrefer,
				})
			}
		case "delegateSystem":
			startString := getAttr(elem, "systemIdStartString")
			catFile := icatalog.ResolveURI(elemBase, getAttr(elem, "catalog"))
			if startString == "" || catFile == "" {
				catalogMissingAttr(ctx, eh, localName, startString, "systemIdStartString", catFile, "catalog")
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntryDelegateSystem,
					Name: startString,
					URL:  catFile,
				})
			}
		case "uri":
			name := getAttr(elem, "name")
			uri := icatalog.ResolveURI(elemBase, getAttr(elem, "uri"))
			if name == "" || uri == "" {
				catalogMissingAttr(ctx, eh, localName, name, "name", uri, "uri")
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntryURI,
					Name: name,
					URL:  uri,
				})
			}
		case "rewriteURI":
			startString := getAttr(elem, "uriStartString")
			prefix := icatalog.ResolveURI(elemBase, getAttr(elem, "rewritePrefix"))
			if startString == "" || prefix == "" {
				catalogMissingAttr(ctx, eh, localName, startString, "uriStartString", prefix, "rewritePrefix")
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntryRewriteURI,
					Name: startString,
					URL:  prefix,
				})
			}
		case "delegateURI":
			startString := getAttr(elem, "uriStartString")
			catFile := icatalog.ResolveURI(elemBase, getAttr(elem, "catalog"))
			if startString == "" || catFile == "" {
				catalogMissingAttr(ctx, eh, localName, startString, "uriStartString", catFile, "catalog")
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntryDelegateURI,
					Name: startString,
					URL:  catFile,
				})
			}
		case "nextCatalog":
			catFile := icatalog.ResolveURI(elemBase, getAttr(elem, "catalog"))
			if catFile == "" {
				eh.Handle(ctx, fmt.Errorf("%s entry missing catalog attribute", localName))
			} else {
				if !icatalog.HasNextCatalog(*entries, catFile) {
					*entries = append(*entries, icatalog.Entry{
						Type: icatalog.EntryNextCatalog,
						URL:  catFile,
					})
				}
			}
		case "group":
			parseEntries(ctx, elem, elemPrefer, elemBase, entries, eh)
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

// catalogMissingAttr reports which required attributes are missing on a catalog entry.
func catalogMissingAttr(ctx context.Context, eh helium.ErrorHandler, elemName, val1, attr1, val2, attr2 string) {
	if val1 == "" {
		eh.Handle(ctx, fmt.Errorf("%s entry missing %s attribute", elemName, attr1))
	}
	if val2 == "" {
		eh.Handle(ctx, fmt.Errorf("%s entry missing %s attribute", elemName, attr2))
	}
}

// closeHandler closes the error handler if it implements io.Closer.
func closeHandler(eh helium.ErrorHandler) {
	if c, ok := eh.(io.Closer); ok {
		_ = c.Close()
	}
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
