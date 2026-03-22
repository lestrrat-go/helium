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

	if v := getAttr(root, lexicon.AttrPrefer); v != "" {
		cat.Prefer = icatalog.ParsePrefer(v)
	}

	parseEntries(ctx, root, cat.Prefer, baseURI, &cat.Entries, eh)

	return cat, nil
}

// parseEntries walks child elements of parent and appends catalog entries.
func parseEntries(ctx context.Context, parent *helium.Element, prefer icatalog.Prefer, baseURI string, entries *[]icatalog.Entry, eh helium.ErrorHandler) {
	if v := getAttrNS(parent, lexicon.AttrBase, lexicon.NamespaceXML); v != "" {
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
		if v := getAttrNS(elem, lexicon.AttrBase, lexicon.NamespaceXML); v != "" {
			elemBase = icatalog.ResolveURI(baseURI, v)
		}

		elemPrefer := prefer
		if v := getAttr(elem, lexicon.AttrPrefer); v != "" {
			elemPrefer = icatalog.ParsePrefer(v)
		}

		switch localName {
		case lexicon.ElemPublic:
			pubID := icatalog.NormalizePublicID(getAttr(elem, lexicon.AttrPublicID))
			uri := icatalog.ResolveURI(elemBase, getAttr(elem, lexicon.AttrURI))
			if pubID == "" || uri == "" {
				catalogMissingAttr(ctx, eh, localName, pubID, lexicon.AttrPublicID, uri, lexicon.AttrURI)
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type:   icatalog.EntryPublic,
					Name:   pubID,
					URL:    uri,
					Prefer: elemPrefer,
				})
			}
		case lexicon.ElemSystem:
			sysID := getAttr(elem, lexicon.AttrSystemID)
			uri := icatalog.ResolveURI(elemBase, getAttr(elem, lexicon.AttrURI))
			if sysID == "" || uri == "" {
				catalogMissingAttr(ctx, eh, localName, sysID, lexicon.AttrSystemID, uri, lexicon.AttrURI)
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntrySystem,
					Name: sysID,
					URL:  uri,
				})
			}
		case lexicon.ElemRewriteSystem:
			startString := getAttr(elem, lexicon.AttrSystemIDStartString)
			prefix := icatalog.ResolveURI(elemBase, getAttr(elem, lexicon.AttrRewritePrefix))
			if startString == "" || prefix == "" {
				catalogMissingAttr(ctx, eh, localName, startString, lexicon.AttrSystemIDStartString, prefix, lexicon.AttrRewritePrefix)
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntryRewriteSystem,
					Name: startString,
					URL:  prefix,
				})
			}
		case lexicon.ElemDelegatePublic:
			startString := icatalog.NormalizePublicID(getAttr(elem, lexicon.AttrPublicIDStartString))
			catFile := icatalog.ResolveURI(elemBase, getAttr(elem, lexicon.AttrCatalog))
			if startString == "" || catFile == "" {
				catalogMissingAttr(ctx, eh, localName, startString, lexicon.AttrPublicIDStartString, catFile, lexicon.AttrCatalog)
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type:   icatalog.EntryDelegatePublic,
					Name:   startString,
					URL:    catFile,
					Prefer: elemPrefer,
				})
			}
		case lexicon.ElemDelegateSystem:
			startString := getAttr(elem, lexicon.AttrSystemIDStartString)
			catFile := icatalog.ResolveURI(elemBase, getAttr(elem, lexicon.AttrCatalog))
			if startString == "" || catFile == "" {
				catalogMissingAttr(ctx, eh, localName, startString, lexicon.AttrSystemIDStartString, catFile, lexicon.AttrCatalog)
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntryDelegateSystem,
					Name: startString,
					URL:  catFile,
				})
			}
		case lexicon.ElemURI:
			name := getAttr(elem, lexicon.AttrName)
			uri := icatalog.ResolveURI(elemBase, getAttr(elem, lexicon.AttrURI))
			if name == "" || uri == "" {
				catalogMissingAttr(ctx, eh, localName, name, lexicon.AttrName, uri, lexicon.AttrURI)
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntryURI,
					Name: name,
					URL:  uri,
				})
			}
		case lexicon.ElemRewriteURI:
			startString := getAttr(elem, lexicon.AttrURIStartString)
			prefix := icatalog.ResolveURI(elemBase, getAttr(elem, lexicon.AttrRewritePrefix))
			if startString == "" || prefix == "" {
				catalogMissingAttr(ctx, eh, localName, startString, lexicon.AttrURIStartString, prefix, lexicon.AttrRewritePrefix)
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntryRewriteURI,
					Name: startString,
					URL:  prefix,
				})
			}
		case lexicon.ElemDelegateURI:
			startString := getAttr(elem, lexicon.AttrURIStartString)
			catFile := icatalog.ResolveURI(elemBase, getAttr(elem, lexicon.AttrCatalog))
			if startString == "" || catFile == "" {
				catalogMissingAttr(ctx, eh, localName, startString, lexicon.AttrURIStartString, catFile, lexicon.AttrCatalog)
			} else {
				*entries = append(*entries, icatalog.Entry{
					Type: icatalog.EntryDelegateURI,
					Name: startString,
					URL:  catFile,
				})
			}
		case lexicon.ElemNextCatalog:
			catFile := icatalog.ResolveURI(elemBase, getAttr(elem, lexicon.AttrCatalog))
			if catFile == "" {
				eh.Handle(ctx, fmt.Errorf("%s entry missing %s attribute", localName, lexicon.AttrCatalog))
			} else {
				if !icatalog.HasNextCatalog(*entries, catFile) {
					*entries = append(*entries, icatalog.Entry{
						Type: icatalog.EntryNextCatalog,
						URL:  catFile,
					})
				}
			}
		case lexicon.ElemGroup:
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
	attr, ok := elem.FindAttribute(helium.LocalNamePredicate(name))
	if !ok {
		return ""
	}
	return attr.Value()
}

// getAttrNS returns the value of the attribute with the given local name
// and namespace URI.
func getAttrNS(elem *helium.Element, name, nsURI string) string {
	attr, ok := elem.FindAttribute(helium.NSPredicate{Local: name, NamespaceURI: nsURI})
	if !ok {
		return ""
	}
	return attr.Value()
}
