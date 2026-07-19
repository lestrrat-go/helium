package helium

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/lestrrat-go/helium/internal/iofs"
	"github.com/lestrrat-go/helium/push"
	"github.com/lestrrat-go/helium/sax"
)

// pseudoRootName is the internal element name used for the synthetic root
// wrapping fragment content during entity/external-subset parsing.
const pseudoRootName = "pseudoroot"

// readerReturningErr always returns its stored error (and no bytes). It
// re-delivers a non-EOF read error that arrived together with the EBCDIC-sniff
// prefix bytes, so the error is surfaced only AFTER the buffered prefix drains
// (honoring the io.Reader contract that data returned alongside a non-EOF error
// must be processed before the error), instead of being lost.
type readerReturningErr struct{ err error }

func (r readerReturningErr) Read([]byte) (int, error) { return 0, r.err }

type stopFuncKey struct{}

// StopParser tells the parser to stop at the next opportunity. Call this
// from any SAX callback to abort parsing early. The parse functions will
// return the partial document built so far with a nil error.
func StopParser(ctx context.Context) {
	if ctx == nil {
		return
	}
	if fn, _ := ctx.Value(stopFuncKey{}).(func()); fn != nil {
		fn()
	}
}

// parserConfig holds the mutable configuration behind a Parser.
type parserConfig struct {
	sax            sax.SAX2Handler
	charBufferSize int
	options        parseOption
	baseURI        string
	catalog        CatalogResolver
	fsys           fs.FS
	maxDepth       int
	maxExtDTDSize  int
	maxNameLength  int
	maxEntityAmpl  int
	maxCMDepth     int
	maxNodeContent int
	errorHandler   ErrorHandler
	xincludeProc   XIncludeProcessor
}

// XIncludeProcessor performs XInclude substitution on a parsed document,
// replacing xi:include elements with the content they reference. It is satisfied
// by xinclude.Processor.
//
// The parser cannot import the xinclude package directly (xinclude depends on
// helium), so a configured processor is injected through this interface with
// [Parser.XInclude]. Process returns the number of substitutions made.
type XIncludeProcessor interface {
	Process(ctx context.Context, doc *Document) (int, error)
}

// Parser holds configuration for XML parsing (libxml2: xmlParserCtxt).
// It uses clone-on-write semantics: each builder method returns
// a new Parser sharing the underlying config until mutation.
//
// The zero value is usable and behaves exactly like [NewParser]: a
// `var p Parser` parses with the same secure defaults (see NewParser) and can
// head an option-method chain. NewParser remains the explicit, self-documenting
// way to construct one.
type Parser struct {
	cfg *parserConfig
}

// defaultMaxDepth is the element-nesting limit applied by [NewParser]. It bounds
// recursion/stack growth from hostile deeply-nested input. Callers parsing
// legitimately deep documents can raise it with [Parser.MaxDepth] or disable the
// check entirely with a negative value (MaxDepth(-1)).
const defaultMaxDepth = 256

// NewParser creates a new Parser with secure defaults suited to untrusted input:
//
//   - external entities and DTDs are not loaded ([Parser.BlockXXE] is on);
//   - network access is forbidden ([Parser.AllowNetwork] is off);
//   - no filesystem is exposed ([Parser.FS] defaults to a deny-all FS, so even a
//     document that does reach a loader cannot open host paths);
//   - element nesting is capped at defaultMaxDepth ([Parser.MaxDepth]).
//
// Entity substitution, external-DTD loading, XInclude, and DTD validation are
// likewise off by default. Opt back in explicitly per setting, e.g.
// NewParser().BlockXXE(false).LoadExternalDTD(true).FS(helium.PermissiveFS()).
func NewParser() Parser {
	return Parser{cfg: newParserConfig()}
}

// newParserConfig builds the default parser configuration — the secure defaults
// documented on [NewParser]. It is also what a zero-value Parser resolves to
// (see [Parser.normalized]), so `var p Parser` parses identically to
// NewParser().
func newParserConfig() *parserConfig {
	cfg := &parserConfig{
		sax:  NewTreeBuilder(),
		fsys: iofs.DenyAll{},
	}
	cfg.options.Set(parseNoXXE)
	cfg.options.Set(parseNoNet)
	return cfg
}

// normalized resolves a zero-value Parser (nil config) to one carrying the same
// secure defaults NewParser installs, so `var p Parser` behaves like
// NewParser(). An already-configured Parser is returned unchanged. The returned
// config is shared, not cloned — safe because every method that MUTATES the
// config goes through clone first.
func (p Parser) normalized() Parser {
	if p.cfg == nil {
		return Parser{cfg: newParserConfig()}
	}
	return p
}

// PermissiveFS returns an [fs.FS] that opens any path via [os.Open] — absolute,
// relative, or containing "..", anywhere on the host filesystem, without
// enforcing [fs.ValidPath]. It restores helium's historical unsandboxed loading
// behavior, which is NOT the default: a parser from [NewParser] loads no
// external resources at all (see [Parser.FS]).
//
// Pass it explicitly to opt back into host filesystem access:
//
//	doc, err := helium.NewParser().
//		BlockXXE(false).
//		LoadExternalDTD(true).
//		FS(helium.PermissiveFS()).
//		Parse(ctx, data)
//
// Prefer a confined [fs.FS] rooted at a trusted directory over PermissiveFS when
// the document's external references are known.
func PermissiveFS() fs.FS {
	return iofs.PermissiveRoot{}
}

// DirFS returns an [fs.FS], for use with [Parser.FS], that opens external
// resources only at or below root. Unlike [PermissiveFS] it refuses any name
// that resolves outside root, so even with external loading enabled an
// attacker-supplied SYSTEM identifier ("/etc/passwd", "../../secret") cannot
// disclose arbitrary local files, and a non-file URI scheme (http, https, ...)
// is rejected so the FS never reaches the network.
//
// It is the general confined-FS adapter: root may be ANY trusted directory, not
// only the document's own directory. The parser resolves a relative SYSTEM id
// against the document's base URI into an absolute path; DirFS serves that
// absolute name directly when it lies within root, so — unlike a bare
// [os.DirFS] or [os.Root.FS] passed to [Parser.FS] — it does not depend on the
// parser's base-relative retry (which only recovers the document-directory
// case). Pass it to opt into confined host access:
//
//	doc, err := helium.NewParser().
//		BlockXXE(false).
//		LoadExternalDTD(true).
//		FS(helium.DirFS("/trusted/dtds")).
//		Parse(ctx, data)
//
// Confinement is enforced with [os.Root] (os.OpenRoot, Go 1.24+): a "../"- or
// absolute-path escape above root is rejected, AND an in-root symlink pointing
// outside root is refused. DirFS is therefore both path-escape-safe and a
// symlink sandbox — stronger than [os.DirFS], which follows an in-root symlink
// out of its root. A relative root is resolved against the process working
// directory when DirFS is called; if that resolution fails (the working
// directory is unavailable) the FS fails closed — every open returns the error
// rather than resolving against a working directory current at open time.
func DirFS(root string) fs.FS {
	return iofs.NewConfinedDir(root)
}

func (p Parser) clone() Parser {
	p = p.normalized()
	cp := *p.cfg
	return Parser{cfg: &cp}
}

// --- Flag methods (each sets/clears the corresponding bit) ---

// RecoverOnError controls whether the parser attempts to recover from
// well-formedness errors and returns a partial document.
// libxml2: XML_PARSE_RECOVER
// Default: false
func (p Parser) RecoverOnError(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseRecover)
		return p
	}
	p.cfg.options.Set(parseRecover)
	return p
}

// SubstituteEntities controls whether entity references are replaced
// with their substitution text during parsing.
// libxml2: XML_PARSE_NOENT
// Default: false
func (p Parser) SubstituteEntities(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseNoEnt)
		return p
	}
	p.cfg.options.Set(parseNoEnt)
	return p
}

// LoadExternalDTD controls whether the parser loads the external DTD subset.
// libxml2: XML_PARSE_DTDLOAD
// Default: false
func (p Parser) LoadExternalDTD(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseDTDLoad)
		return p
	}
	p.cfg.options.Set(parseDTDLoad)
	return p
}

// DefaultDTDAttributes controls whether the parser adds default attributes
// defined in the DTD. The external subset is loaded when default-attribute
// application, external-DTD loading, or DTD validation is requested; the three
// intents are independent, so call order does not matter and turning this off
// does not leave loading stuck on.
// libxml2: XML_PARSE_DTDATTR
// Default: false
func (p Parser) DefaultDTDAttributes(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseDTDAttr)
		return p
	}
	p.cfg.options.Set(parseDTDAttr)
	return p
}

// ValidateDTD controls whether the parser validates the document against
// its DTD after parsing. The external subset is loaded when DTD validation,
// external-DTD loading, or default-attribute application is requested; the
// three intents are independent, so call order does not matter and turning
// this off does not leave loading stuck on.
// libxml2: XML_PARSE_DTDVALID
// Default: false
func (p Parser) ValidateDTD(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseDTDValid)
		return p
	}
	p.cfg.options.Set(parseDTDValid)
	return p
}

// SuppressErrors controls whether error reports from the parser are
// suppressed. When true, the SAX error callback is not invoked.
// libxml2: XML_PARSE_NOERROR
// Default: false
func (p Parser) SuppressErrors(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseNoError)
		return p
	}
	p.cfg.options.Set(parseNoError)
	return p
}

// SuppressWarnings controls whether warning reports from the parser are
// suppressed. When true, the SAX warning callback is not invoked.
// libxml2: XML_PARSE_NOWARNING
// Default: false
func (p Parser) SuppressWarnings(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseNoWarning)
		return p
	}
	p.cfg.options.Set(parseNoWarning)
	return p
}

// PedanticErrors controls whether the parser reports pedantic warnings
// for minor specification violations.
// libxml2: XML_PARSE_PEDANTIC
// Default: false
func (p Parser) PedanticErrors(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parsePedantic)
		return p
	}
	p.cfg.options.Set(parsePedantic)
	return p
}

// StripBlanks controls whether whitespace-only text nodes are removed
// from the resulting DOM tree.
// libxml2: XML_PARSE_NOBLANKS
// Default: false
func (p Parser) StripBlanks(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseNoBlanks)
		return p
	}
	p.cfg.options.Set(parseNoBlanks)
	return p
}

// AllowNetwork controls whether the parser may reach an external resource (an
// external DTD subset or external parsed entity) whose reference names a network
// URI — an http, https, or ftp scheme.
//
// helium has NO built-in network loader: every external-resource load goes
// through the configured [fs.FS] (see [Parser.FS]). This flag does not itself
// fetch anything — it only gates a guard in front of that fs.FS:
//
//   - AllowNetwork(false) (the default) refuses any network-scheme reference with
//     [ErrNetworkAccessForbidden] before it reaches the fs.FS.
//   - AllowNetwork(true) lifts that guard, so a network-scheme reference is
//     handed to the configured fs.FS like any other name. An actual network
//     fetch happens only if that fs.FS itself opens network URIs; the default
//     deny-all FS and [PermissiveFS] (os.Open) do not, so AllowNetwork(true)
//     alone loads nothing over the network.
//
// libxml2: XML_PARSE_NONET (semantics inverted — libxml2 sets that flag to
// *forbid* network access, whereas AllowNetwork(true) *permits* it).
// Default: false (network-scheme references are refused).
func (p Parser) AllowNetwork(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Set(parseNoNet)
		return p
	}
	p.cfg.options.Clear(parseNoNet)
	return p
}

// CleanNamespaces controls whether redundant namespace declarations are
// removed from the resulting DOM tree.
// libxml2: XML_PARSE_NSCLEAN
// Default: false
func (p Parser) CleanNamespaces(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseNsClean)
		return p
	}
	p.cfg.options.Set(parseNsClean)
	return p
}

// MergeCDATA controls whether CDATA sections are merged into adjacent
// text nodes instead of being represented as separate CDATA nodes.
// libxml2: XML_PARSE_NOCDATA
// Default: false
func (p Parser) MergeCDATA(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseNoCDATA)
		return p
	}
	p.cfg.options.Set(parseNoCDATA)
	return p
}

// FixBaseURIs controls whether xml:base URIs are fixed up during
// XInclude processing. When set to false, xml:base attributes are
// not adjusted on included content.
// libxml2: XML_PARSE_NOBASEFIX (note: semantics are inverted — libxml2
// sets this flag to *disable* fixup, whereas FixBaseURIs(false)
// disables it)
// Default: true (xml:base URIs are fixed up)
func (p Parser) FixBaseURIs(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Set(parseNoBaseFix)
		return p
	}
	p.cfg.options.Clear(parseNoBaseFix)
	return p
}

// Default values for the per-limit parser knobs. Each [Parser] limit method
// treats a zero argument as "use the default" and a negative argument as
// "no limit".
const (
	// DefaultMaxNameLength is the default cap on the length, in bytes, of a
	// single element / attribute / namespace-prefix / NCName token.
	DefaultMaxNameLength = 50000
	// DefaultMaxEntityAmplification is the default entity-expansion
	// amplification factor: the parser rejects a document whose total
	// expanded entity bytes exceed this multiple of the input size (beyond a
	// fixed baseline). The 1 GiB absolute expansion ceiling applies
	// regardless of this factor.
	DefaultMaxEntityAmplification = 5
	// DefaultMaxContentModelDepth is the default cap on the nesting depth of a
	// DTD element content-model declaration (the parenthesized groups in
	// <!ELEMENT ...>).
	DefaultMaxContentModelDepth = 128
	// DefaultMaxNodeContentSize is the default cap, in bytes, on a single
	// indivisible content run — a CDATA section, comment body,
	// processing-instruction body, or character-data run. Each such construct
	// maps to a single SAX event / DOM node and cannot be chunked, so an
	// oversized one is a memory-amplification vector on untrusted input. The
	// 10 MiB value mirrors the intent of libxml2's XML_MAX_TEXT_LENGTH.
	DefaultMaxNodeContentSize = 10 << 20
)

// MaxNameLength sets the maximum length, in bytes, of a single element,
// attribute, namespace-prefix, or NCName token. A value of zero (the default)
// uses [DefaultMaxNameLength] (50000); a negative value removes the limit.
// Removing the limit lets a hostile document allocate very large name tokens,
// so do so only for trusted input.
func (p Parser) MaxNameLength(n int) Parser {
	p = p.clone()
	p.cfg.maxNameLength = n
	return p
}

// MaxEntityAmplification sets the maximum entity-expansion amplification
// factor: the parser rejects a document whose cumulative expanded entity
// bytes exceed n times the input size (past a fixed baseline), the guard
// against "billion laughs" style attacks. A value of zero (the default) uses
// [DefaultMaxEntityAmplification] (5); a negative value disables the ratio
// check. The 1 GiB absolute expansion ceiling is always enforced, even when
// the ratio check is disabled.
func (p Parser) MaxEntityAmplification(n int) Parser {
	p = p.clone()
	p.cfg.maxEntityAmpl = n
	return p
}

// MaxContentModelDepth sets the maximum nesting depth of a DTD element
// content-model declaration (the parenthesized groups in <!ELEMENT ...>). A
// value of zero (the default) uses [DefaultMaxContentModelDepth] (128); a
// negative value removes the limit.
func (p Parser) MaxContentModelDepth(n int) Parser {
	p = p.clone()
	p.cfg.maxCMDepth = n
	return p
}

// MaxNodeContentSize sets the maximum size, in bytes, of a single indivisible
// content run: a CDATA section, comment body, processing-instruction body,
// character-data run, or attribute value. Each maps to a single SAX event / DOM
// node (or attribute) and cannot be chunked, so an oversized one on untrusted
// input is a memory-amplification vector. The cap fires during accumulation —
// the parse fails with
// [ErrNodeContentTooLarge] the moment a run exceeds it, before the whole run is
// buffered. The same cap also bounds a single contiguous run of XML whitespace
// (a blank skip — in the prolog/epilogue, between declarations in an external
// DTD subset, or inside an INCLUDE conditional section), since an unbounded
// whitespace run would otherwise grow the cursor buffer without limit; an
// over-cap blank run likewise fails with [ErrNodeContentTooLarge]. A value of
// zero (the default) uses [DefaultMaxNodeContentSize] (10 MiB); a negative value
// removes both the node-content and the blank-run limit. Removing the limit lets
// a hostile document drive unbounded memory use, so do so only for trusted
// input.
//
// A streaming SAX consumer that configured [Parser.CharBufferSize] receives
// character data in bounded chunks and is not subject to this cap (its memory is
// already bounded); the cap still applies to its CDATA, comment, and PI runs.
func (p Parser) MaxNodeContentSize(n int) Parser {
	p = p.clone()
	p.cfg.maxNodeContent = n
	return p
}

// IgnoreEncoding controls whether the parser ignores the encoding
// declaration inside the document and uses the transport-level encoding
// instead.
// libxml2: XML_PARSE_IGNORE_ENC
// Default: false
func (p Parser) IgnoreEncoding(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseIgnoreEnc)
		return p
	}
	p.cfg.options.Set(parseIgnoreEnc)
	return p
}

// BlockXXE controls whether loading of external entities and DTDs is
// blocked, preventing XML External Entity (XXE) attacks.
// libxml2: XML_PARSE_NOXXE
// Default: true (external entity/DTD loading is blocked)
func (p Parser) BlockXXE(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseNoXXE)
		return p
	}
	p.cfg.options.Set(parseNoXXE)
	return p
}

// SkipIDs controls whether ID attribute interning is skipped during
// parsing. When true, the parser does not build the ID table.
// libxml2: XML_PARSE_SKIP_IDS
// Default: false
func (p Parser) SkipIDs(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseSkipIDs)
		return p
	}
	p.cfg.options.Set(parseSkipIDs)
	return p
}

// LenientXMLDecl relaxes XML declaration parsing so that the version,
// encoding, and standalone pseudo-attributes may appear in any order.
// Per the XML spec (section 2.8) the order MUST be version, encoding,
// standalone, but some real-world producers emit them differently.
// This is a helium extension not present in libxml2.
// Default: false
func (p Parser) LenientXMLDecl(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseLenientXMLDecl)
		return p
	}
	p.cfg.options.Set(parseLenientXMLDecl)
	return p
}

// --- Non-flag configuration ---

// SAXHandler sets the SAX2 event handler that receives parse events. The
// default (installed by [NewParser]) is a [TreeBuilder], which assembles the
// events into the *Document that Parse returns.
//
// Like every Parser option, this replaces — it does not merge: the value from
// the most recent call wins, and each call returns a new Parser (clone-on-write)
// leaving the receiver unchanged.
//
// Passing nil restores the default [TreeBuilder] that [NewParser] installs, so
// [Parser.Parse] still returns a built *Document. This mirrors [Parser.FS],
// where nil likewise restores the default, and keeps a caller that forwards an
// optional handler straight through from silently losing all output. To consume
// events without building a tree, pass a handler that ignores them rather than
// nil.
func (p Parser) SAXHandler(s sax.SAX2Handler) Parser {
	p = p.clone()
	if s == nil {
		s = NewTreeBuilder()
	}
	p.cfg.sax = s
	return p
}

// BaseURI sets the document's base URI, used for resolving relative
// references such as external DTD system identifiers.
func (p Parser) BaseURI(uri string) Parser {
	p = p.clone()
	p.cfg.baseURI = uri
	return p
}

// CharBufferSize sets a TARGET maximum number of bytes delivered in a single
// Characters or IgnorableWhitespace SAX callback. When size <= 0 (the
// default), all character data is delivered in one call. When size > 0,
// data longer than size bytes is split into chunks of at most size bytes,
// always respecting UTF-8 character boundaries.
//
// size is a target, not a hard cap, in two documented cases:
//   - A single UTF-8 rune wider than size is delivered whole rather than split
//     into invalid fragments, so that one callback may exceed size.
//   - To keep memory bounded, an all-whitespace run that exceeds the internal
//     pending-whitespace budget is delivered as Characters (not as
//     IgnorableWhitespace); only abnormally large pure-blank runs are affected.
func (p Parser) CharBufferSize(size int) Parser {
	p = p.clone()
	p.cfg.charBufferSize = size
	return p
}

// MaxDepth sets the maximum element nesting depth allowed during parsing. When
// depth is greater than zero, the parser returns an error if the input document
// contains elements nested deeper than this limit. A value of zero (the default)
// uses the 256-level cap ([NewParser] applies it to bound recursion from hostile
// input); a negative value removes the cap. Removing the cap lets hostile
// deeply-nested input drive unbounded recursion/stack growth, so do so only for
// trusted input.
//
// This matches the convention of the other parser limit options
// ([Parser.MaxNameLength], [Parser.MaxEntityAmplification],
// [Parser.MaxContentModelDepth], [Parser.MaxNodeContentSize],
// [Parser.MaxExternalDTDBytes]): zero selects the documented default, a negative
// value disables the limit.
func (p Parser) MaxDepth(depth int) Parser {
	p = p.clone()
	p.cfg.maxDepth = depth
	return p
}

// MaxExternalDTDBytes sets the maximum number of bytes read from an external
// DTD subset (see [LoadExternalDTD], [ValidateDTD], [DefaultDTDAttributes]).
// The cap is enforced against the actual number of bytes read, guarding
// against hostile or pathological sources (e.g. /dev/zero) that could
// otherwise exhaust memory before any entity or parse limits apply. A value of
// zero (the default) uses [MaxExternalDTDSize] (10 MiB); a negative value
// removes the cap. Removing the cap lets a hostile source drive unbounded
// memory use, so do so only for trusted input.
func (p Parser) MaxExternalDTDBytes(n int) Parser {
	p = p.clone()
	p.cfg.maxExtDTDSize = n
	return p
}

// Catalog sets an XML Catalog for resolving external entity identifiers
// (public/system IDs) during parsing. When set, the parser consults the
// catalog before attempting to load external DTDs and entities.
func (p Parser) Catalog(c CatalogResolver) Parser {
	p = p.clone()
	p.cfg.catalog = c
	return p
}

// FS sets the [fs.FS] used to load external resources referenced by the
// document — external DTDs ([LoadExternalDTD]) and external entities
// resolved through [TreeBuilder.ResolveEntity].
//
// The default (and what a nil value restores) is a deny-all FS that refuses
// every open: a parser from [NewParser] loads no external resources from the
// host filesystem. To opt into host access, pass [PermissiveFS] (any os.Open
// path) or — preferably — a confined [fs.FS] rooted at a trusted directory. For
// a confined FS, prefer [DirFS] (or [os.Root.FS], os.OpenRoot, Go 1.24+): it
// refuses any open that escapes the root through a symlink. [os.DirFS] blocks
// "../"- and absolute-path escape but FOLLOWS an in-root symlink out of the
// root, so it is path-escape-safe but not a symlink sandbox. [DirFS]
// additionally serves an in-root ABSOLUTE name directly, so it can be rooted at
// any trusted directory, not only the document's own directory (the
// base-relative retry below recovers only the document-directory case).
//
// A relative SYSTEM id is resolved against the document's base URI, which is
// absolute whenever one is set (e.g. [Parser.ParseFile] uses the file's
// absolute path). The parser first hands the FS that resolved name — absolute,
// and possibly a "file:" URI — which an FS that enforces [fs.ValidPath]
// ([os.DirFS], [os.Root.FS], [testing/fstest.MapFS]) rejects. On that rejection
// it retries with the name made relative to the document base's directory, so a
// confined FS rooted at the document's own directory resolves the reference
// (including a nested resource in a subdirectory, which relativizes against the
// document root). The retry is a validated fs.ValidPath, so a "../"- or
// absolute-path escape above the FS root is disqualified (a leading "/" or a
// surviving ".."); a network-scheme name is refused before either attempt.
// [PermissiveFS], which loads any os.Open path, is served by the first
// (absolute) attempt and is unaffected.
func (p Parser) FS(fsys fs.FS) Parser {
	p = p.clone()
	if fsys == nil {
		fsys = iofs.DenyAll{}
	}
	p.cfg.fsys = fsys
	return p
}

// ErrorHandler sets the handler that receives individual errors produced
// during DTD validation ([ValidateDTD]); the returned error from Parse is
// [ErrDTDValidationFailed] on failure. The handler is not consulted for
// well-formedness or namespace errors — those surface only as the error
// returned from Parse.
//
// The handler is retained by reference and shared by every Parse call on the
// returned Parser (and on any Parser derived from it by further configuration),
// so it must tolerate the reuse and concurrency the caller subjects the Parser
// to. Passing a nil handler is allowed and means "discard": at validation time
// a nil handler is treated as [NilErrorHandler]. Parser values are immutable, so
// calling ErrorHandler again returns a new Parser carrying the replacement
// handler and leaves the original unchanged.
//
// If the handler implements [io.Closer] it is closed once at the end of each
// Parse that performs DTD validation, so a Closer handler is not meant to be
// shared across such Parse calls.
func (p Parser) ErrorHandler(h ErrorHandler) Parser {
	p = p.clone()
	p.cfg.errorHandler = h
	return p
}

// XInclude enables XInclude substitution during parsing. The supplied processor
// — typically [xinclude.NewProcessor] configured with a resolver — is run over
// the parsed document before it is returned, so xi:include elements are already
// expanded in the result. When DTD validation is also requested ([ValidateDTD]),
// it validates the expanded tree. Passing a nil interface value disables
// XInclude processing. (A caller-constructed typed-nil — a nil pointer of the
// caller's own [XIncludeProcessor] implementation — is the standard Go typed-nil
// footgun and is unsupported; xinclude.Processor is a value type and cannot be
// typed-nil.)
//
// XInclude is off by default. Because the parser (and the processor's own
// default) resolves no filesystem, grant access explicitly on the processor,
// e.g. xinclude.NewProcessor().Resolver(xinclude.NewFSResolver(fsys)).
func (p Parser) XInclude(proc XIncludeProcessor) Parser {
	p = p.clone()
	p.cfg.xincludeProc = proc
	return p
}

func (p Parser) closeHandler() {
	if p.cfg != nil && p.cfg.errorHandler != nil {
		if cl, ok := p.cfg.errorHandler.(io.Closer); ok {
			_ = cl.Close()
		}
	}
}

// finalize runs post-parse steps on a successfully built document: XInclude
// substitution when a processor was injected via [Parser.XInclude], followed by
// DTD validation when [Parser.ValidateDTD] is set. XInclude runs first so that
// DTD validation, if requested, sees the expanded tree.
func (p Parser) finalize(ctx context.Context, doc *Document) (*Document, error) {
	if doc == nil {
		return doc, nil
	}
	// The disable signal is a nil interface value (see [Parser.XInclude] and the
	// nil case in TestParserXIncludeInjection). A caller-supplied typed-nil — a
	// nil pointer of the caller's own implementation — is caller-constructed
	// nil-ness this package does not guard against; normalizing it away would be a
	// reflective guard against caller API misuse.
	if p.cfg.xincludeProc != nil {
		if _, err := p.cfg.xincludeProc.Process(ctx, doc); err != nil {
			// A cancelled/timed-out post-parse step follows Parse's contract:
			// return the context error with a nil document, never a partial tree.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			return doc, err
		}
	}
	if p.cfg.options.IsSet(parseDTDValid) {
		handler := p.cfg.errorHandler
		if handler == nil {
			handler = NilErrorHandler{}
		}
		err := validateDocument(ctx, doc, handler)
		p.closeHandler()
		if err != nil {
			return doc, err
		}
	}
	return doc, nil
}

// --- Terminal methods ---

// Parse parses XML from a byte slice and returns the resulting Document
// (libxml2: xmlParseDoc / xmlParseMemory).
//
// When [ValidateDTD] is enabled and the document fails validation, the
// returned error is [ErrDTDValidationFailed] and the document is still
// returned. Individual validation errors are delivered to the [ErrorHandler]
// configured via [Parser.ErrorHandler].
//
// Cancellation: if ctx is cancelled or its deadline is exceeded, Parse aborts
// and returns the context error (matched by [errors.Is] against
// [context.Canceled] / [context.DeadlineExceeded]) with a nil Document — never
// a partial tree. Because Parse reads from an in-memory byte slice there is no
// blocking read, so cancellation is always observed promptly: the parser checks
// the context between parse steps and between cursor refills.
func (p Parser) Parse(ctx context.Context, b []byte) (*Document, error) { //nolint:contextcheck
	if ctx == nil {
		ctx = context.Background()
	}

	p = p.normalized()

	pctx := &parserCtx{rawInput: b, baseURI: p.cfg.baseURI}
	if err := pctx.init(p.cfg, bytes.NewReader(b)); err != nil {
		return nil, err
	}
	defer func() {
		// Release the parser context; any error is intentionally ignored so it
		// does not override the main return error.
		_ = pctx.release()
	}()

	if err := pctx.parseDocument(ctx); err != nil {
		if errors.Is(err, errParserStopped) {
			return pctx.doc, nil
		}
		// A cancelled or timed-out parse is not a recoverable parse error:
		// return the context error with a nil document, never a partial tree.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		if p.cfg.options.IsSet(parseRecover) {
			// ParseRecover: return the partial document along with the error
			return pctx.doc, err
		}
		return nil, err
	}

	// Post-parse steps (XInclude substitution, DTD validation) on the built tree.
	return p.finalize(ctx, pctx.doc)
}

// ParseReader parses XML from an io.Reader and returns the resulting Document
// (libxml2: xmlReadIO).
// This is identical to [Parse] but reads from a stream instead of a byte slice.
// See [Parse] for DTD validation error handling.
//
// EBCDIC encoding detection buffers only a bounded leading prefix (enough to
// recover the encoding name from the XML declaration) and then streams and
// decodes the remainder through the normal cursor pipeline, exactly like the
// non-EBCDIC path. Resident memory is bounded by the parser's incremental
// per-node content caps rather than by buffering the whole document, so a large
// finite EBCDIC document parses (under the same per-node limits [Parse] applies)
// while an unbounded stream is bounded by those caps. All inputs stream.
//
// Cancellation: context cancellation and deadlines are observed BETWEEN read
// operations and parse steps. The parser checks ctx before each cursor refill
// (read from r) and between parse steps, so a cancelled or timed-out context is
// honored as soon as the parser regains control, returning the context error
// (matched by [errors.Is] against [context.Canceled] /
// [context.DeadlineExceeded]) with a nil Document.
//
// A reader already blocked inside its own Read call cannot be interrupted
// generically: Go provides no way to unblock a Read in progress. Such a read is
// only interruptible if r itself honors the context or a deadline — for example
// a reader that sets a read deadline when ctx.Done() fires, or that returns from
// Read with an error on cancellation. If r can block indefinitely (e.g. a slow
// or never-returning network reader), wrap it so its Read observes ctx, or pass
// the already-read bytes to [Parse] instead.
func (p Parser) ParseReader(ctx context.Context, r io.Reader) (*Document, error) {
	// A generic reader has unknown size: pass -1 so the streaming path keeps
	// inputSize == 0 and the entity-amplification guard behaves as before.
	return p.parseReader(ctx, r, -1)
}

// parseReader parses XML from an io.Reader. srcSize is the known size of the
// underlying source in bytes, or a negative value when the size is unknown.
// When a non-negative size is supplied (e.g. by ParseFile, which can stat the
// file) it seeds parserCtx.inputSize so the entity-amplification guard matches
// the byte-slice path in Parse, where inputSize is set from the slice length.
func (p Parser) parseReader(ctx context.Context, r io.Reader, srcSize int64) (*Document, error) { //nolint:contextcheck
	if ctx == nil {
		ctx = context.Background()
	}

	p = p.normalized()

	// Honor an already-cancelled context BEFORE touching r: the EBCDIC sniff
	// below reads from r, and r may be a non-context-aware reader that blocks
	// indefinitely. Checking ctx first preserves the "cancelled before any
	// blocking read" contract — a cancellation observed up front never depends
	// on the reader making progress.
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Read the leading bytes to detect EBCDIC, which is not ASCII-compatible
	// and whose detection/decode requires the original raw bytes (the
	// streaming path cannot replay them). When detected, read the whole input
	// and route through the byte-slice path so behavior matches Parse exactly.
	//
	// Read the head with a small loop (not bufio.Peek) so ctx is re-checked
	// between reads: a slow reader that delivers the prefix one byte at a time
	// is interrupted as soon as the parser regains control, instead of blocking
	// inside Peek until the whole prefix arrives. The scratch buffer is larger
	// than the EBCDIC prefix so a reader that returns more bytes than requested
	// in one Read is captured in full rather than truncated; all captured bytes
	// are prepended to the remaining stream so the non-EBCDIC path is
	// unaffected, and any non-EOF error returned alongside the bytes is
	// preserved and surfaced once the buffered head drains.
	var scratch [512]byte
	var hn int
	var perr error
	sniffZeroProgress := 0
	for hn < len(patEBCDIC) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		m, err := r.Read(scratch[hn:])
		hn += m
		if err != nil {
			perr = err
			break
		}
		if m == 0 {
			// A single (0, nil) read is legal: a slow producer may return no data
			// and no error while it waits for more input. Retry a bounded number of
			// CONSECUTIVE empty reads (mirroring the cursor fill loops'
			// maxZeroProgressReads guard) so a transient empty read does not
			// truncate the sniff prefix below the invariant EBCDIC pattern and
			// silently misclassify an EBCDIC stream as non-EBCDIC. A reader stuck at
			// (0, nil) forever fails fast with io.ErrNoProgress instead of hanging.
			sniffZeroProgress++
			if sniffZeroProgress >= maxSniffZeroProgressReads {
				return nil, io.ErrNoProgress
			}
			continue
		}
		sniffZeroProgress = 0
	}
	// Copy the sniffed bytes off the stack scratch buffer so head may be grown
	// (EBCDIC prefix extension below) and safely referenced for the lifetime of
	// the parse.
	head := append([]byte(nil), scratch[:hn]...)

	// Unifying invariant for both sniff paths: bytes returned alongside a
	// non-EOF read error MUST be parsed before the error is surfaced (the
	// io.Reader contract). io.EOF is the normal terminator and is never
	// re-surfaced. A non-EOF perr is therefore carried forward, not returned
	// early, so the head bytes are processed first.

	// Detect EBCDIC regardless of whether the head read ended with io.EOF: an
	// io.Reader may legally return all of its bytes together with io.EOF in a
	// single Read. EBCDIC is not ASCII-compatible, so its XML declaration cannot
	// be parsed at byte level; the encoding name is instead recovered by
	// ExtractEBCDICEncoding, which scans the invariant-translated XML
	// declaration in the first ~200 bytes. We therefore buffer ONLY a bounded
	// prefix (ebcdicEncodingSniffMax) here — enough for that scan — and then
	// STREAM-decode the remainder through the normal cursor pipeline. The
	// parser's incremental per-node content caps bound resident memory exactly
	// as on the non-EBCDIC path, so a large finite EBCDIC document parses (with
	// the same per-node limits Parse([]byte) applies) while a hostile,
	// never-ending stream is bounded by those caps instead of being buffered
	// whole into memory before parsing begins.
	ebcdic := hn >= len(patEBCDIC) && bytes.Equal(head[:len(patEBCDIC)], patEBCDIC)
	if ebcdic && perr == nil {
		// Extend the buffered prefix to ebcdicEncodingSniffMax so the encoding
		// declaration is available, re-checking ctx BEFORE each Read so a
		// cancellation observed after detection aborts promptly without waiting
		// on a slow or stalled stream. A short read / EOF before the cap is fine:
		// ExtractEBCDICEncoding works with whatever prefix arrived, falling back
		// to IBM-037 when no declaration is present.
		var chunk [ebcdicEncodingSniffMax]byte
		extZeroProgress := 0
		for len(head) < ebcdicEncodingSniffMax {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			n, rerr := r.Read(chunk[:ebcdicEncodingSniffMax-len(head)])
			head = append(head, chunk[:n]...)
			if rerr != nil {
				perr = rerr
				break
			}
			if n == 0 {
				// A single (0, nil) read is legal here too: treating it as
				// end-of-sniff would leave the prefix too short for
				// ExtractEBCDICEncoding to find the encoding declaration, so the
				// parser would default to IBM-037 and never re-switch to the
				// declared EBCDIC variant. Retry a bounded number of CONSECUTIVE
				// empty reads (mirroring the cursor fill loops' maxZeroProgressReads
				// guard) so a transient empty read does not prematurely truncate the
				// sniff prefix, while keeping the prefix bounded (the loop still
				// stops at ebcdicEncodingSniffMax — no unbounded buffering). A reader
				// stuck at (0, nil) forever fails fast with io.ErrNoProgress.
				extZeroProgress++
				if extZeroProgress >= maxSniffZeroProgressReads {
					return nil, io.ErrNoProgress
				}
				continue
			}
			extZeroProgress = 0
		}
	}

	// Reconstruct the stream: head bytes + remainder. io.EOF means the whole
	// document is in head; a non-EOF perr arrived alongside the head bytes, so
	// replay the head then a sticky error-reader that surfaces the error only
	// after those bytes drain (errReader-replay) — never discarding valid bytes.
	var stream io.Reader
	switch {
	case perr != nil && perr != io.EOF:
		// Replay the head bytes, then re-deliver the sticky read error. For
		// len(head) == 0 the head reader yields nothing, so only the error surfaces.
		stream = io.MultiReader(bytes.NewReader(head), readerReturningErr{err: perr})
	case perr == io.EOF:
		// No tail remains in r; replay only the buffered head.
		stream = bytes.NewReader(head)
	case len(head) > 0:
		stream = io.MultiReader(bytes.NewReader(head), r)
	default:
		stream = r
	}

	pctx := &parserCtx{baseURI: p.cfg.baseURI}
	if ebcdic {
		// EBCDIC: rawInput is the bounded sniff prefix used by
		// ExtractEBCDICEncoding; ebcdicStream tells parseDocument to decode the
		// live prefix+remainder cursor in place rather than reset it from
		// rawInput (which is only a prefix here, not the whole document).
		pctx.rawInput = head
		pctx.ebcdicStream = true
		// rawInput is only the sniff prefix here, so init would seed inputSize
		// with the prefix length rather than the real document size. Count the
		// bytes the cursor actually pulls from the reconstructed stream (prefix +
		// remainder) so the entity-amplification guard compares against the real
		// consumed size — matching Parse([]byte), where inputSize is the full
		// slice length — instead of falsely rejecting a large internal entity
		// referenced once. See entityCheckLimits.
		counter := &countingReader{r: stream}
		pctx.ebcdicConsumed = counter
		stream = counter
	}
	if err := pctx.init(p.cfg, stream); err != nil {
		return nil, err
	}
	// init seeds inputSize from rawInput (nil here, so 0). When the caller
	// knows the source size, set it so the amplification guard isn't tripped
	// for a large internal entity referenced only once.
	if srcSize >= 0 {
		pctx.inputSize = srcSize
	}
	defer func() {
		// Release the parser context; any error is intentionally ignored so it
		// does not override the main return error.
		_ = pctx.release()
	}()

	if err := pctx.parseDocument(ctx); err != nil {
		if errors.Is(err, errParserStopped) {
			return pctx.doc, nil
		}
		// A cancelled or timed-out parse is not a recoverable parse error:
		// return the context error with a nil document, never a partial tree.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		if p.cfg.options.IsSet(parseRecover) {
			return pctx.doc, err
		}
		return nil, err
	}

	return p.finalize(ctx, pctx.doc)
}

// ParseFile reads and parses an XML file. The document's URL is set to the
// absolute path of the file, and the file path is used as the base URI for
// relative URI resolution during parsing.
func (p Parser) ParseFile(ctx context.Context, path string) (*Document, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("helium: failed to resolve path %q: %w", path, err)
	}
	f, err := os.Open(path) //nolint:gosec // path is caller-supplied
	if err != nil {
		return nil, fmt.Errorf("helium: failed to read %q: %w", path, err)
	}
	defer f.Close()

	// Pass the file size so the entity-amplification guard uses the real input
	// size, matching Parse([]byte). Stat failure falls back to unknown (-1).
	srcSize := int64(-1)
	if fi, statErr := f.Stat(); statErr == nil {
		srcSize = fi.Size()
	}

	// Mirror Parse/ParseReader: a recoverable parse error (ParseRecover) or a
	// DTD-validation failure returns the partial document ALONGSIDE the error, so
	// ParseFile must not discard a non-nil doc. Set the source URL on any document
	// produced (recovered or complete); only a nil doc returns as nil.
	doc, err := p.BaseURI(abs).parseReader(ctx, f, srcSize)
	if doc != nil {
		doc.SetURL(abs)
	}
	return doc, err
}

// ParseInNodeContext parses an XML fragment in the context of an existing
// node. The node provides in-scope namespace declarations and document-level
// DTD/entity context. Returns the first node of the parsed fragment list
// (siblings linked via NextSibling). The returned nodes are not attached
// to any parent.
func (p Parser) ParseInNodeContext(ctx context.Context, node Node, data []byte) (Node, error) { //nolint:contextcheck
	if ctx == nil {
		ctx = context.Background()
	}

	p = p.normalized()

	// Reject both a literal nil interface and a typed-nil pointer (e.g. the
	// *Element that Document.DocumentElement returns for a rootless document)
	// with the matchable ErrNilNode, before the type switch below dereferences
	// the node and panics on a typed nil.
	if isNilNode(node) {
		return nil, ErrNilNode
	}

	// Walk up to the nearest element or document node.
	var ctxElem *Element
	var doc *Document
	cur := node
	for cur != nil {
		switch v := cur.(type) {
		case *Document:
			doc = v
			goto found
		case *Element:
			ctxElem = v
			doc = v.doc
			goto found
		}
		cur = cur.Parent()
	}
	return nil, errors.New("no element or document context found")

found:
	if doc == nil {
		doc = NewDocument("1.0", "", StandaloneImplicitNo)
	}

	newctx := &parserCtx{}
	if err := newctx.init(p.cfg, bytes.NewReader(data)); err != nil {
		return nil, err
	}
	defer func() {
		// Release the parser context; any error is intentionally ignored so it
		// does not override the main return error.
		_ = newctx.release()
	}()

	// Save the document's children and restore them afterward.
	fc := doc.FirstChild()
	lc := doc.LastChild()
	setFirstChild(doc, nil)
	setLastChild(doc, nil)
	defer func() {
		setFirstChild(doc, fc)
		setLastChild(doc, lc)
	}()

	newctx.doc = doc

	// Push in-scope namespaces from the context element into the parser's
	// namespace stack so that the fragment can resolve prefixed names.
	if ctxElem != nil {
		nsList := collectInScopeNamespaces(ctxElem)
		for _, ns := range nsList {
			newctx.pushNS(ns.Prefix(), ns.URI())
		}
	}

	// Create pseudoroot element, push to node stack.
	newRoot := doc.CreateElement(pseudoRootName)
	newctx.pushNodeEntry(nodeEntry{local: pseudoRootName, qname: pseudoRootName, synthetic: true})
	newctx.elem = newRoot
	if err := doc.AddChild(newRoot); err != nil {
		return nil, err
	}

	if err := newctx.switchEncoding(); err != nil {
		return nil, err
	}
	innerCtx := withParserCtx(ctx, newctx)
	innerCtx = sax.WithDocumentLocator(innerCtx, newctx)
	innerCtx = context.WithValue(innerCtx, stopFuncKey{}, newctx.stop)
	if err := newctx.parseContent(innerCtx); err != nil {
		// errParserStopped is a benign stop (helium.StopParser); any other
		// error, including context cancellation, propagates with a nil result.
		if !errors.Is(err, errParserStopped) {
			return nil, err
		}
	}

	// Extract children from pseudoroot.
	if child := doc.FirstChild(); child != nil {
		if grandchild := child.FirstChild(); grandchild != nil {
			for e := grandchild; e != nil; e = e.NextSibling() {
				e.(MutableNode).SetTreeDoc(doc) //nolint:forcetypeassert
				e.baseDocNode().parent = nil
			}
			return grandchild, nil
		}
	}

	return nil, nil //nolint:nilnil
}

// collectInScopeNamespaces walks up from elem collecting all namespace
// declarations. Inner declarations shadow outer ones (closer to elem wins).
func collectInScopeNamespaces(elem *Element) []*Namespace {
	seen := map[string]bool{}
	var result []*Namespace
	var cur Node = elem
	for cur != nil {
		if e, ok := cur.(*Element); ok {
			for _, ns := range e.Namespaces() {
				if !seen[ns.Prefix()] {
					seen[ns.Prefix()] = true
					result = append(result, ns)
				}
			}
		}
		cur = cur.Parent()
	}
	return result
}

// PushParser provides an incremental XML parsing interface
// (libxml2: xmlParserCtxt in push mode).
// Data is pushed via Push or Write, and the parser processes tokens as
// they become available in a background goroutine. Call [PushParser.Close]
// to signal end-of-input and retrieve the parsed Document.
type PushParser = push.Parser[*Document]

// NewPushParser creates a PushParser using the given Parser's configuration.
// The parser runs in a background goroutine, reading from the internal
// stream as data is pushed.
func (p Parser) NewPushParser(ctx context.Context) *PushParser {
	return push.New[*Document](ctx, p)
}
