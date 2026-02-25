# HTML Parser Design

## Goal

HTML 4.01 parser matching libxml2's `HTMLparser.c` behavior. SAX1-style callbacks (no namespaces). Produces `helium.Document` with `HTMLDocumentNode` type.

## Public API

```go
package html

func Parse(data []byte, opts ...Option) (*helium.Document, error)
func ParseFile(filename string, opts ...Option) (*helium.Document, error)
func ParseWithSAX(data []byte, handler SAXHandler, opts ...Option) error
```

## SAXHandler interface (SAX1, no namespaces)

```go
type SAXHandler interface {
    SetDocumentLocator(loc DocumentLocator) error
    StartDocument() error
    EndDocument() error
    StartElement(name string, attrs []Attribute) error
    EndElement(name string) error
    Characters(ch []byte) error
    CDataBlock(value []byte) error
    Comment(value []byte) error
    InternalSubset(name, externalID, systemID string) error
    ProcessingInstruction(target, data string) error
    IgnorableWhitespace(ch []byte) error
    Error(msg string, args ...interface{}) error
    Warning(msg string, args ...interface{}) error
}
```

## Internal architecture

- Element table: ~90 HTML4 elements with end-tag rules, void/inline flags, data mode
- Auto-close rules: ~200 entries (oldTag → newTag implies close)
- Entity table: 252 HTML4 named entities
- Tokenizer: start tags, end tags, attributes, comments, DOCTYPE, char data
- Parser: main loop with auto-close logic, implicit html/head/body insertion
- Tree builder: SAXHandler implementation that builds DOM

## Scope

Phase 1 (MVP): ~15/47 SAX golden tests. Common elements, auto-close, implicit insertion, script/style raw content, entity expansion.

## Non-goals (Phase 1)

- HTML5 entity table (2000+ entries)
- Full HTML serializer
- Encoding detection from meta tags
- Complex misnesting recovery
- Error reporting matching libxml2 format

## Dependencies

- `helium` (DOM types: Document, Element, Text, Comment, etc.)
- `helium/sax` (DocumentLocator interface reuse)

## Reference

- libxml2: `testdata/libxml2/source/HTMLparser.c`
- Element table: lines 502-904
- Auto-close rules: lines 914-1164
- Test data: `testdata/libxml2/source/test/HTML/`, `result/HTML/`
