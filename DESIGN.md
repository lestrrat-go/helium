# RelaxNG Validation — Design Document

## Goal

Implement RELAX NG (XML) schema validation for helium, matching libxml2's `relaxng.c` behavior. Given the same `.rng` schema and `.xml` instance, helium must produce the same validation output (error messages, pass/fail status) as libxml2.

Scope: XML syntax only (not compact syntax). DOM-based validation only (not streaming/progressive).

## Public API

```go
package relaxng

// Grammar is a compiled RELAX NG schema.
type Grammar struct {
    compileErrors   string
    compileWarnings string
    // internal: start pattern, named definitions, etc.
}

func (g *Grammar) CompileErrors() string
func (g *Grammar) CompileWarnings() string

// Compile compiles a RELAX NG document into a Grammar.
func Compile(doc *helium.Document, opts ...CompileOption) (*Grammar, error)

// CompileFile reads and compiles a RELAX NG file into a Grammar.
func CompileFile(path string, opts ...CompileOption) (*Grammar, error)

// Validate validates a document against a compiled grammar.
// Returns validation output in libxml2-compatible format.
func Validate(doc *helium.Document, grammar *Grammar, opts ...ValidateOption) string
```

### Options

```go
type CompileOption func(*compileConfig)
type ValidateOption func(*validateConfig)

func WithSchemaFilename(name string) CompileOption  // for error messages
func WithFilename(name string) ValidateOption        // for error messages
```

## Internal Architecture

### Data Model (`grammar.go`)

Core types modeling the RELAX NG pattern language:

```go
type PatternKind int
const (
    PatternEmpty PatternKind = iota
    PatternNotAllowed
    PatternText
    PatternElement
    PatternAttribute
    PatternGroup
    PatternInterleave
    PatternChoice
    PatternOptional
    PatternZeroOrMore
    PatternOneOrMore
    PatternRef
    PatternParentRef
    PatternExternalRef
    PatternData
    PatternValue
    PatternList
    PatternMixed
    PatternDefine
    PatternStart
)

type Pattern struct {
    Kind     PatternKind
    Name     string        // element/attribute local name
    NS       string        // namespace URI
    Value    string        // for value patterns
    DataType *DataType     // for data patterns
    Content  *Pattern      // child pattern
    Attrs    *Pattern      // attribute patterns
    Next     *Pattern      // sibling in group/choice/interleave
    NameClass *NameClass   // for element/attribute name matching
    Params   []*Param      // for data patterns
    Ref      string        // define name for ref patterns
}

type NameClassKind int
const (
    NCName NameClassKind = iota
    NCAnyName
    NCNsName
    NCChoice
    NCExcept
)

type NameClass struct {
    Kind    NameClassKind
    Name    string     // for NCName
    NS      string     // for NCName, NCNsName
    Left    *NameClass // for NCChoice
    Right   *NameClass // for NCChoice, NCExcept
}

type DataType struct {
    Library string // datatype library URI
    Type    string // type name
}

type Param struct {
    Name  string
    Value string
}
```

### Compilation (`parse.go`)

Multi-phase schema compilation:

1. **Parse**: Walk the RNG DOM document, build pattern tree
2. **Include resolution**: Process `<include>` and `<externalRef>` with recursion guard
3. **Reference resolution**: Resolve `<ref>` patterns to `<define>` patterns
4. **Simplification**: Flatten redundant patterns (choice with notAllowed, etc.)
5. **Semantic checks**: Detect cycles, validate pattern usage rules

Key internal type:
```go
type compiler struct {
    grammar   *Grammar
    baseDir   string
    filename  string
    defines   map[string]*Pattern  // named patterns
    refs      map[string][]*Pattern // unresolved refs
    errors    strings.Builder
    warnings  strings.Builder
    includeStack []string          // recursion guard
    includeLimit int               // max depth (default 1000)
}
```

### Validation (`validate.go`)

Tree-walk validation against compiled grammar:

```go
type validator struct {
    grammar  *Grammar
    doc      *helium.Document
    filename string
    errors   strings.Builder
    valid    bool
}

type validState struct {
    node     helium.Node
    seq      helium.Node  // next sibling to validate
    attrs    []*helium.Attribute
    attrLeft int
}
```

Validation algorithm: recursive pattern matching against DOM tree.
- `validateElement`: match element name, validate attributes, validate content
- `validateAttribute`: match attribute name and value
- `validateContent`: walk children matching against content pattern
- `validateData`/`validateValue`: XSD datatype validation (reuse from xmlschema)
- `validateInterleave`: unordered content matching with partition groups

### Error Formatting (`errors.go`)

libxml2-compatible error format:
```
{file}:{line}: element {name}: Relax-NG validity error : {message}
{file} fails to validate
```

or:
```
{file} validates
```

Schema compilation errors:
```
Relax-NG parser error : {message}
Relax-NG schema {file} failed to compile
```

## Scope / Non-goals

**In scope:**
- XML syntax RELAX NG schemas
- All pattern types: element, attribute, group, interleave, choice, optional, zeroOrMore, oneOrMore, mixed, text, empty, notAllowed, data, value, list, ref, define, start, grammar, externalRef, include, parentRef
- Name classes: name, anyName, nsName, choice, except
- XSD datatype library (`http://www.w3.org/2001/XMLSchema-datatypes`)
- Built-in token datatype library
- Include/externalRef with recursion guard
- Grammar combine (choice/interleave for multiple start/define)
- libxml2-compatible error messages

**Out of scope (initial implementation):**
- Compact syntax (`.rnc`)
- Streaming/progressive validation
- RELAX NG Compatibility Annotations (a:defaultValue, etc.)
- Custom datatype libraries beyond XSD built-ins

## Dependencies

- `github.com/lestrrat-go/helium` — DOM types, parser
- `github.com/lestrrat-go/helium/xmlschema` — may reuse XSD datatype validation logic (or extract shared types)

## Task List

### Phase 1: Skeleton + Basic Patterns (target: ~60 tests)

1. **RNG1**: Package skeleton — `relaxng.go` (public API), `options.go`, `grammar.go` (data model), `errors.go`
2. **RNG2**: Add relaxng section to `testdata/libxml2/generate.sh`, run it to import test data
3. **RNG3**: Golden file test harness — `relaxng_test.go` with auto-discovery, skip list
4. **RNG4**: Schema parser — `parse.go`: parse `<element>`, `<attribute>`, `<text>`, `<empty>`, `<notAllowed>`, `<group>`, `<choice>`, `<optional>`, `<zeroOrMore>`, `<oneOrMore>`, `<mixed>`
5. **RNG5**: Name class parsing — `<name>`, `<anyName>`, `<nsName>`, `<choice>` (in name class), `<except>`
6. **RNG6**: Grammar/start/define/ref — parse `<grammar>`, `<start>`, `<define>`, `<ref>` with reference resolution
7. **RNG7**: Basic validation engine — `validate.go`: element matching, attribute matching, content model walking (group, choice, optional, zeroOrMore, oneOrMore, text, empty)
8. **RNG8**: Error formatting matching libxml2 output

### Phase 2: Advanced Patterns (target: ~120 tests)

9. **RNG9**: Interleave validation — partition-based unordered content matching
10. **RNG10**: `<data>` pattern — XSD datatype validation (integer, float, string, boolean, date, etc.)
11. **RNG11**: `<value>` pattern — exact value matching with datatype context
12. **RNG12**: `<list>` pattern — whitespace-separated value list validation
13. **RNG13**: Include resolution — `<include>` with override of `<define>`/`<start>`, recursion guard
14. **RNG14**: ExternalRef resolution — `<externalRef>` loading and merging
15. **RNG15**: Grammar combine — `combine="choice"` and `combine="interleave"` for multiple `<start>` and `<define>`
16. **RNG16**: `<parentRef>` — parent grammar reference resolution

### Phase 3: Edge Cases + Polish (target: ~159 tests)

17. **RNG17**: Namespace-aware element/attribute matching (ns attribute on patterns)
18. **RNG18**: Schema simplification (flatten redundant patterns, notAllowed propagation)
19. **RNG19**: Cycle detection in references
20. **RNG20**: Broken schema handling (XML parse errors, schema compilation errors)
21. **RNG21**: Large/complex schema support (libvirt.rng, docbook.rng, OpenDocument, ISO XMP)
22. **RNG22**: Data type parameters (`<param>`)
23. **RNG23**: Fix remaining test failures, minimize skip list

## Reference

- [RELAX NG Specification](https://relaxng.org/spec-20011203.html)
- [RELAX NG Tutorial](https://relaxng.org/tutorial-20011203.html)
- libxml2 source: `testdata/libxml2/source/relaxng.c` (~14000 lines)
- libxml2 header: `testdata/libxml2/source/include/libxml/relaxng.h`
- libxml2 test runner: `testdata/libxml2/source/runtest.c` (lines 3625-3806)
- Test schemas: `testdata/libxml2/source/test/relaxng/` (105 .rng files)
- Expected results: `testdata/libxml2/source/result/relaxng/` (159 .err files)
