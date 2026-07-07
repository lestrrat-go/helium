package xslt3

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/internal/uripath"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

// paramKeyToClark converts an XPath map key to the Clark notation string
// used internally for parameter lookup. QName keys are expanded to {uri}local;
// non-QName keys (xs:string) pass through AtomicToString unchanged.
func paramKeyToClark(key xpath3.AtomicValue) (string, error) {
	if q, ok := key.Value.(xpath3.QNameValue); ok {
		if q.URI != "" {
			return helium.ClarkName(q.URI, q.Local), nil
		}
		return q.Local, nil
	}
	return xpath3.AtomicToString(key)
}

func (ec *execContext) xsltFunctionsNS() map[xpath3.QualifiedName]xpath3.Function {
	if ec.cachedFnsNS != nil {
		return ec.cachedFnsNS
	}
	ec.cachedFnsNS = make(map[xpath3.QualifiedName]xpath3.Function, len(ec.stylesheet.functions)+1)

	// Register XSLT document() in the fn: namespace so fn:document() works.
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: fnNameDocument}] = &xsltFunc{min: 1, max: 2, fn: ec.fnDocument}

	// Override fn:doc to preserve source document identity and apply
	// xsl:strip-space rules to loaded documents.
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "doc"}] = &xsltFunc{min: 1, max: 1, fn: ec.fnDoc}

	// Register XSLT built-in functions in the fn: namespace so they are
	// discoverable via function-lookup with an explicit namespace.
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: funcSystemProperty}] =
		&xsltFunc{min: 1, max: 1, fn: ec.fnSystemProperty}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: funcAvailableSystemProperties}] =
		&xsltFunc{min: 0, max: 0, fn: ec.fnAvailableSystemProperties}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: fnNameCurrentOutputURI}] =
		&xsltFunc{min: 0, max: 0, fn: ec.fnCurrentOutputURI}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "stream-available"}] =
		&xsltFunc{min: 1, max: 1, fn: ec.fnStreamAvailable}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "unparsed-entity-uri"}] =
		&xsltFunc{min: 1, max: 2, fn: ec.fnUnparsedEntityURI}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "unparsed-entity-public-id"}] =
		&xsltFunc{min: 1, max: 2, fn: ec.fnUnparsedEntityPublicID}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "key"}] =
		&xsltFunc{min: 2, max: 3, fn: ec.fnKey}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "generate-id"}] =
		&xsltFunc{min: 0, max: 1, fn: ec.fnGenerateID}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: fnNameElementAvailable}] =
		&xsltFunc{min: 1, max: 1, fn: ec.fnElementAvailable}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: fnNameFunctionAvailable}] =
		&xsltFunc{min: 1, max: 2, fn: ec.fnFunctionAvailable}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: fnNameTypeAvailable}] =
		&xsltFunc{min: 1, max: 1, fn: ec.fnTypeAvailable}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: fnNameCurrent}] =
		&xsltFunc{min: 0, max: 0, fn: ec.fnCurrent}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: fnNameCurrentGroup}] =
		&xsltFunc{min: 0, max: 0, fn: ec.fnCurrentGroup, noDynRef: true, dynRefError: errCodeXTDE1061}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: fnNameCurrentGroupingKey}] =
		&xsltFunc{min: 0, max: 0, fn: ec.fnCurrentGroupingKey, noDynRef: true, dynRefError: errCodeXTDE1071}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "current-merge-group"}] =
		&xsltFunc{min: 0, max: 1, fn: ec.fnCurrentMergeGroup}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "current-merge-key"}] =
		&xsltFunc{min: 0, max: 0, fn: ec.fnCurrentMergeKey}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "regex-group"}] =
		&regexGroupFunc{ec: ec}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "accumulator-before"}] =
		&xsltFunc{min: 1, max: 1, fn: func(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
			return ec.accumulatorLookup(ctx, args, "accumulator-before", func() (map[helium.Node]map[string]xpath3.Sequence, map[helium.Node]map[string]error) {
				return ec.accumulatorBeforeByNode, ec.accumulatorBeforeErrorByNode
			})
		}}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "accumulator-after"}] =
		&xsltFunc{min: 1, max: 1, fn: func(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
			return ec.accumulatorLookup(ctx, args, "accumulator-after", func() (map[helium.Node]map[string]xpath3.Sequence, map[helium.Node]map[string]error) {
				return ec.accumulatorAfterByNode, ec.accumulatorAfterErrorByNode
			})
		}}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: fnNameCopyOf}] =
		&xsltFunc{min: 0, max: 1, fn: ec.fnCopyOf}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: funcSnapshot}] =
		&xsltFunc{min: 0, max: 1, fn: ec.fnSnapshot}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: fnNameTransform}] =
		&xsltFunc{min: 1, max: 1, fn: ec.fnTransform}

	ec.registerSchemaConstructors(ec.cachedFnsNS)

	// Per spec §20.3.3, function-lookup within a package must return the
	// original (pre-override) function, not the override. Register a
	// custom function-lookup that handles this when in a package context.
	if ec.currentPackage != nil {
		ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "function-lookup"}] =
			&xsltFunc{min: 2, max: 2, fn: ec.fnFunctionLookupPackage}
	}

	if ec.currentPackage != nil {
		// Per-package function scope: all functions from the current
		// package (including private), plus public functions from
		// packages it uses.
		for _, def := range ec.currentPackage.functions {
			if def.Visibility == visAbstract {
				continue // abstract functions have no implementation
			}
			ec.registerUserFunc(def)
		}
		for _, usedPkg := range ec.currentPackage.usedPackages {
			for _, def := range usedPkg.functions {
				if def.Visibility == visPublic || def.Visibility == visFinal || def.Visibility == "" {
					ec.registerUserFunc(def)
				}
			}
		}
	} else {
		for _, def := range ec.stylesheet.functions {
			ec.registerUserFunc(def)
		}
	}

	return ec.cachedFnsNS
}

// xsltEvaluateFunctionsNS returns the namespaced function map for use with
// xsl:evaluate. Per XSLT 3.0 §20.3, user-defined stylesheet functions
// (xsl:function) are NOT available in xsl:evaluate unless they are
// explicitly declared as public or final.
func (ec *execContext) xsltEvaluateFunctionsNS() map[xpath3.QualifiedName]xpath3.Function {
	all := ec.xsltFunctionsNS()
	// Collect QNames of user functions that are NOT explicitly public/final
	excluded := make(map[xpath3.QualifiedName]struct{})
	fns := ec.stylesheet.functions
	if ec.currentPackage != nil {
		fns = ec.currentPackage.functions
	}
	for _, def := range fns {
		vis := def.Visibility
		if vis == visPublic || vis == visFinal {
			continue // explicitly public → available in xsl:evaluate
		}
		excluded[def.Name] = struct{}{}
	}
	result := make(map[xpath3.QualifiedName]xpath3.Function, len(all))
	for k, v := range all {
		if _, skip := excluded[k]; skip {
			continue
		}
		result[k] = v
	}
	return result
}

// registerUserFunc registers an XSL user function into cachedFnsNS,
// handling multi-arity overloads by wrapping them in xslMultiArityFunc.
func (ec *execContext) registerUserFunc(def *xslFunction) {
	qn := def.Name
	uf := &xslUserFunc{def: def, ec: ec}
	if existing, ok := ec.cachedFnsNS[qn]; ok {
		if maf, ok := existing.(*xslMultiArityFunc); ok {
			maf.addVariant(uf)
		} else {
			maf := &xslMultiArityFunc{minArity: existing.MinArity(), maxArity: existing.MaxArity()}
			if euf, ok := existing.(*xslUserFunc); ok {
				maf.variants = append(maf.variants, euf)
			}
			maf.addVariant(uf)
			ec.cachedFnsNS[qn] = maf
		}
	} else {
		ec.cachedFnsNS[qn] = uf
	}
}

// fnFunctionLookupPackage is a package-aware implementation of function-lookup
// per spec §20.3.3. Within a package, function-lookup must return the original
// (pre-override) function definition, not the override. This implementation
// looks up the function, and if it's an overridden xsl:function, substitutes
// the original definition.
func (ec *execContext) fnFunctionLookupPackage(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	// Delegate to the standard function-lookup implementation first
	result, err := xpath3.CallFunctionLookup(ctx, args)
	if err != nil {
		return nil, err
	}
	if result == nil || result.Len() == 0 {
		return result, nil
	}
	// Extract the QName and arity from the arguments to look up the
	// original function definition in the current package.
	fi, ok := result.Get(0).(xpath3.FunctionItem)
	if !ok {
		return result, nil
	}
	qn := xpath3.QualifiedName{URI: fi.Namespace, Name: fi.Name}
	fk := funcKey{Name: qn, Arity: fi.Arity}
	pkgFn, exists := ec.currentPackage.functions[fk]
	if !exists {
		return result, nil // not a package function
	}
	if pkgFn.OriginalFunc == nil {
		return result, nil // not overridden
	}
	origDef := pkgFn.OriginalFunc
	if origDef.Visibility == visAbstract {
		// Original is abstract — no implementation to look up
		return nil, nil //nolint:nilnil
	}
	origUF := &xslUserFunc{def: origDef, ec: ec}
	origFI := xpath3.FunctionItem{
		Arity:     fi.Arity,
		Name:      fi.Name,
		Namespace: fi.Namespace,
		Invoke:    origUF.Call,
	}
	return xpath3.ItemSlice{origFI}, nil
}

// findXSLFunction finds an xsl:function by QName and arity (-1 = any).
func (ec *execContext) findXSLFunction(qn xpath3.QualifiedName, arity int) *xslFunction {
	if arity < 0 {
		// Any arity: just check if any overload exists
		for fk, def := range ec.stylesheet.functions {
			if fk.Name == qn {
				return def
			}
		}
		return nil
	}
	fk := funcKey{Name: qn, Arity: arity}
	return ec.stylesheet.functions[fk]
}

// findXSLFunctionByArity finds an xsl:function by QName and exact arity.
func (ec *execContext) findXSLFunctionByArity(qn xpath3.QualifiedName, arity int) *xslFunction {
	fk := funcKey{Name: qn, Arity: arity}
	return ec.stylesheet.functions[fk]
}

// xsltEvaluateFunctions returns XSLT built-in functions available in
// xsl:evaluate dynamic context. Per XSLT 3.0 section 20.3, current()
// is excluded.
func (ec *execContext) xsltEvaluateFunctions() map[string]xpath3.Function {
	fns := ec.xsltFunctions()
	evalFns := make(map[string]xpath3.Function, len(fns))
	for k, v := range fns {
		switch k {
		case "current", "system-property", "current-output-uri", "available-system-properties":
			continue
		}
		evalFns[k] = v
	}
	return evalFns
}

type transformDepthKey struct{}

const maxTransformDepth = 10

// resultDocCollector implements ResultDocumentHandler for fn:transform,
// collecting secondary result documents and their output definitions into maps.
type resultDocCollector struct {
	results    map[string]*helium.Document
	outputDefs map[string]*OutputDef
}

func (c resultDocCollector) HandleResultDocument(href string, doc *helium.Document, outDef *OutputDef) error {
	c.results[href] = doc
	if outDef != nil && c.outputDefs != nil {
		c.outputDefs[href] = outDef
	}
	return nil
}

// resolveRelativeURI resolves a reference against a base URI.
//
// Absoluteness is decided with [xsd.URIScheme] (RFC 3986): an absolute-URI ref
// (it carries its own scheme, e.g. "urn:shared", "file:/modules/m.xsl") is
// returned unchanged and must never be filepath.Join'ed onto a local base
// (which would corrupt it into "/styles/urn:shared"). A relative ref against a
// URI base is resolved with RFC 3986 semantics (scheme/authority preserved);
// against a local filesystem base it keeps historical filepath.Join handling.
// Resolution of the URI cases is delegated to the shared canonical
// [xsd.ResolveSchemaURI] helper.
func resolveRelativeURI(base, ref string) string {
	if xsd.URIScheme(ref) != "" || xsd.URIScheme(base) != "" {
		resolved, err := xsd.ResolveSchemaURI(ref, base)
		if err != nil {
			return ref
		}
		return resolved
	}
	// Local filesystem base: resolve with forward-slash (path) semantics so the
	// result uses '/' on every OS; on Windows filepath.Dir/Join would emit '\'.
	return uripath.JoinLocalBaseDir(path.Dir(uripath.ToSlash(base)), ref)
}

// resolveStylesheetLocation resolves an fn:transform stylesheet-location loc
// against the current stylesheet base URI.
//
// Absoluteness is decided with [xsd.URIScheme] (RFC 3986), not filepath.IsAbs:
// when base is a URI (it has a scheme), a filepath-absolute or root-relative
// loc such as "/inner.xsl" must be resolved against the base scheme/authority
// (mem://pkg/main.xsl + /inner.xsl -> mem://pkg/inner.xsl) rather than passed
// through verbatim. Only a purely-local absolute path against a local base is
// left unchanged.
func resolveStylesheetLocation(base, loc string) string {
	if base == "" {
		return loc
	}
	// uripath.IsAbsolutePath recognizes both POSIX- and Windows-absolute shapes
	// regardless of GOOS, so a purely-local absolute loc against a local base is
	// left unchanged on every OS.
	if xsd.URIScheme(base) != "" || !uripath.IsAbsolutePath(loc) {
		return resolveRelativeURI(base, loc)
	}
	return loc
}

// newNestedCompiler creates a Compiler pre-configured with the same
// resolver, package resolver, and import schemas that were used to
// compile this stylesheet, so that fn:transform nested compiles
// behave consistently with top-level compilation.
func (ss *Stylesheet) newNestedCompiler() Compiler {
	c := NewCompiler()
	if ss.uriResolver != nil {
		c = c.URIResolver(ss.uriResolver)
	}
	if ss.packageResolver != nil {
		c = c.PackageResolver(ss.packageResolver)
	}
	if len(ss.compilerImportSchemas) > 0 {
		c = c.ImportSchemas(ss.compilerImportSchemas...)
	}
	if ss.maxResourceBytes != 0 {
		c = c.MaxResourceBytes(ss.maxResourceBytes)
	}
	if ss.allowExternalEntities {
		c = c.AllowExternalEntities(true)
	}
	if ss.parser != nil {
		c = c.Parser(*ss.parser)
	}
	return c
}

// transformFnConfig carries the resolvers, resource caps, parser, and
// external-entity posture that the fn:transform implementation needs. It
// decouples the shared implementation (run) from the running execution context,
// so both the in-stylesheet path (ec.fnTransform) and the standalone
// TransformFunction feed the same logic.
type transformFnConfig struct {
	// baseURI resolves a relative stylesheet-location.
	baseURI string
	// nestedCompiler is the pre-configured compiler used to compile the
	// dynamically-loaded stylesheet/package/stylesheet-text.
	nestedCompiler Compiler
	// stylesheetResolver reads an explicit stylesheet-location.
	stylesheetResolver URIResolver
	// packageResolver resolves a package-name/package-version.
	packageResolver PackageResolver
	// importSchemas are pre-compiled schemas passed to the nested compiler
	// (standalone construction only; the in-stylesheet path folds these into
	// nestedCompiler via newNestedCompiler).
	importSchemas []*xsd.Schema
	// innerURIResolver serves fn:doc/document()/fn:unparsed-text evaluated
	// inside the dynamically-run stylesheet, and external-entity loading during
	// its parse.
	innerURIResolver xpath3.URIResolver
	// httpClient is the opt-in HTTP client for the inner transform's resource
	// retrieval.
	httpClient *http.Client
	// maxResourceBytes is the per-resource read cap (0 = default, <0 = unbounded).
	maxResourceBytes int64
	// allowExternalEntities opts into the legacy permissive external-entity parse.
	allowExternalEntities bool
	// parseParser is the base parser governing parse policy for the nested
	// stylesheet parse (nil = hardened default).
	parseParser *helium.Parser
	// entityLoader loads external entities referenced during the nested
	// stylesheet parse.
	entityLoader externalEntityLoader
}

// compileURIResolverAdapter adapts an xpath3.URIResolver (ResolveURI) to the
// compile-time xslt3 URIResolver (Resolve), so a single resolver supplied to
// TransformFunction serves both stylesheet-module loading and fn:doc.
type compileURIResolverAdapter struct {
	r xpath3.URIResolver
}

func (a compileURIResolverAdapter) Resolve(uri string) (io.ReadCloser, error) {
	return a.r.ResolveURI(uri)
}

// resolverEntityLoader is an externalEntityLoader backed by an
// xpath3.URIResolver / HTTP client pair, used by the standalone fn:transform
// path when parsing a dynamically-loaded stylesheet.
type resolverEntityLoader struct {
	resolver   xpath3.URIResolver
	httpClient *http.Client
	limit      int64
}

func (l resolverEntityLoader) retrieve(ctx context.Context, uri string) ([]byte, error) {
	return retrieveBytesVia(ctx, uri, l.resolver, l.httpClient, l.limit)
}

// TransformOption configures the fn:transform implementation returned by
// [TransformFunction].
type TransformOption interface {
	applyTransform(*transformFnConfig)
}

type transformOptionFunc func(*transformFnConfig)

func (f transformOptionFunc) applyTransform(c *transformFnConfig) { f(c) }

// WithTransformURIResolver sets the resolver used to read a stylesheet-location
// and to resolve resources (fn:doc/document()/fn:unparsed-text and external
// entities) referenced while compiling and running the transformed stylesheet.
// Resource access is opt-in: without a resolver, stylesheet-location and fn:doc
// fail (FOXT0003 / FODC0002).
func WithTransformURIResolver(r xpath3.URIResolver) TransformOption {
	return transformOptionFunc(func(c *transformFnConfig) { c.innerURIResolver = r })
}

// WithTransformPackageResolver sets the resolver used for the package-name /
// package-version options.
func WithTransformPackageResolver(r PackageResolver) TransformOption {
	return transformOptionFunc(func(c *transformFnConfig) { c.packageResolver = r })
}

// WithTransformHTTPClient sets an opt-in HTTP client for retrieving http(s)
// resources inside the transformed stylesheet.
func WithTransformHTTPClient(client *http.Client) TransformOption {
	return transformOptionFunc(func(c *transformFnConfig) { c.httpClient = client })
}

// WithTransformBaseURI sets the base URI against which a relative
// stylesheet-location (and stylesheet-text base) is resolved.
func WithTransformBaseURI(uri string) TransformOption {
	return transformOptionFunc(func(c *transformFnConfig) { c.baseURI = uri })
}

// WithTransformMaxResourceBytes sets the per-resource read cap (0 = default,
// negative = unbounded).
func WithTransformMaxResourceBytes(n int64) TransformOption {
	return transformOptionFunc(func(c *transformFnConfig) { c.maxResourceBytes = n })
}

// WithTransformAllowExternalEntities opts into the legacy permissive parse of
// the dynamically-loaded stylesheet and runtime documents (resolver-mediated
// external entity / DTD loading). Default false: XXE is blocked.
func WithTransformAllowExternalEntities(v bool) TransformOption {
	return transformOptionFunc(func(c *transformFnConfig) { c.allowExternalEntities = v })
}

// WithTransformParser sets the base parser governing parse policy for the
// dynamically-loaded stylesheet and runtime source parses.
func WithTransformParser(p helium.Parser) TransformOption {
	return transformOptionFunc(func(c *transformFnConfig) {
		pp := p
		c.parseParser = &pp
	})
}

// WithTransformImportSchemas supplies pre-compiled schemas to the nested
// compiler for xsl:import-schema resolution.
func WithTransformImportSchemas(schemas ...*xsd.Schema) TransformOption {
	return transformOptionFunc(func(c *transformFnConfig) { c.importSchemas = schemas })
}

// TransformFunction returns an [xpath3.Function] implementing fn:transform()
// that can be registered on a standalone xpath3.Evaluator via
// Evaluator.Functions (in the fn: namespace, name "transform"), so callers that
// drive xpath3 directly — with no outer running stylesheet — can invoke
// fn:transform. It shares its implementation with the in-stylesheet fn:transform
// (ec.fnTransform); the deps the in-stylesheet path inherits from its execution
// context are supplied here explicitly through TransformOption values.
func TransformFunction(options ...TransformOption) xpath3.Function {
	cfg := &transformFnConfig{}
	for _, o := range options {
		o.applyTransform(cfg)
	}
	// Adapt the xpath3 resolver to a compile-time module resolver for the
	// nested compile and stylesheet-location reads.
	if cfg.innerURIResolver != nil && cfg.stylesheetResolver == nil {
		cfg.stylesheetResolver = compileURIResolverAdapter{r: cfg.innerURIResolver}
	}
	c := NewCompiler()
	if cfg.stylesheetResolver != nil {
		c = c.URIResolver(cfg.stylesheetResolver)
	}
	if cfg.packageResolver != nil {
		c = c.PackageResolver(cfg.packageResolver)
	}
	if len(cfg.importSchemas) > 0 {
		c = c.ImportSchemas(cfg.importSchemas...)
	}
	c = c.MaxResourceBytes(cfg.maxResourceBytes).AllowExternalEntities(cfg.allowExternalEntities)
	if cfg.parseParser != nil {
		c = c.Parser(*cfg.parseParser)
	}
	cfg.nestedCompiler = c
	if cfg.entityLoader == nil {
		cfg.entityLoader = resolverEntityLoader{resolver: cfg.innerURIResolver, httpClient: cfg.httpClient, limit: cfg.maxResourceBytes}.retrieve
	}
	return &xsltFunc{min: 1, max: 1, fn: cfg.run}
}

// fnTransform implements fn:transform() for the in-stylesheet path. It builds a
// transformFnConfig from the running execution context — inheriting the outer
// stylesheet's resolvers, resource cap, parser, and external-entity posture —
// and delegates to the shared implementation (run), so the in-stylesheet and
// standalone (TransformFunction) paths run identical logic.
func (ec *execContext) fnTransform(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	cfg := &transformFnConfig{
		baseURI:               ec.stylesheet.baseURI,
		nestedCompiler:        ec.stylesheet.newNestedCompiler().MaxResourceBytes(ec.resourceLimit()).AllowExternalEntities(ec.allowExternalEntities()),
		stylesheetResolver:    ec.stylesheet.uriResolver,
		packageResolver:       ec.stylesheet.packageResolver,
		maxResourceBytes:      ec.resourceLimit(),
		allowExternalEntities: ec.allowExternalEntities(),
		parseParser:           ec.injectedParser(),
		entityLoader:          ec.retrieveDocumentBytes,
	}
	if ec.transformConfig != nil {
		cfg.innerURIResolver = ec.transformConfig.uriResolver
		cfg.httpClient = ec.transformConfig.httpClient
	}
	return cfg.run(ctx, args)
}

// run is the shared fn:transform implementation — it dynamically compiles and
// executes an XSLT stylesheet from the options map, using the resolvers and
// resource policy carried on cfg.
func (cfg *transformFnConfig) run(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	// Check recursion depth
	depth := 0
	if d, ok := ctx.Value(transformDepthKey{}).(int); ok {
		depth = d
	}
	if depth >= maxTransformDepth {
		return nil, dynamicError(errCodeFOXT0004, "fn:transform: maximum recursion depth (%d) exceeded", maxTransformDepth)
	}
	ctx = context.WithValue(ctx, transformDepthKey{}, depth+1)
	if len(args) != 1 || (args[0] == nil || sequence.Len(args[0]) != 1) {
		return nil, dynamicError(errCodeFOXT0001, "fn:transform requires a single map argument")
	}
	m, ok := args[0].Get(0).(xpath3.MapItem)
	if !ok {
		return nil, dynamicError(errCodeFOXT0001, "fn:transform argument must be a map")
	}

	// Extract option values from the map
	getStr := func(key string) string {
		k := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: key}
		seq, ok := m.Get(k)
		if !ok || seq == nil || sequence.Len(seq) == 0 {
			return ""
		}
		av, err := xpath3.AtomizeItem(seq.Get(0))
		if err != nil {
			return ""
		}
		s, err := xpath3.AtomicToString(av)
		if err != nil {
			return ""
		}
		return s
	}

	// getQNameStr extracts a string option that may be a QName value.
	// Unlike getStr, it preserves the namespace URI by producing Clark
	// notation {uri}local when the value is xs:QName.
	getQNameStr := func(key string) string {
		k := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: key}
		seq, ok := m.Get(k)
		if !ok || seq == nil || sequence.Len(seq) == 0 {
			return ""
		}
		av, err := xpath3.AtomizeItem(seq.Get(0))
		if err != nil {
			return ""
		}
		if av.TypeName == xpath3.TypeQName {
			if qv, ok := av.Value.(xpath3.QNameValue); ok {
				if qv.URI != "" {
					return helium.ClarkName(qv.URI, qv.Local)
				}
				return qv.Local
			}
		}
		s, err := xpath3.AtomicToString(av)
		if err != nil {
			return ""
		}
		return s
	}

	getSeq := func(key string) xpath3.Sequence {
		k := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: key}
		seq, ok := m.Get(k)
		if !ok {
			return nil
		}
		return seq
	}

	// Unhandled fn:transform options (processor-specific or optional):
	//   requested-properties, vendor-options, cache, post-process, serialization-params
	stylesheetLoc := getStr("stylesheet-location")
	packageName := getStr("package-name")
	packageVersion := getStr("package-version")
	initialTemplate := getStr("initial-template")
	initialMode := getStr("initial-mode")
	initialFunction := getQNameStr("initial-function")
	deliveryFormat := getStr("delivery-format")
	baseOutputURI := getStr("base-output-uri")
	initialMatchSel := getSeq("initial-match-selection")
	sourceNode := getSeq("source-node")
	stylesheetParamsSeq := getSeq("stylesheet-params")
	staticParamsSeq := getSeq("static-params")
	templateParamsSeq := getSeq("template-params")
	tunnelParamsSeq := getSeq("tunnel-params")
	functionParamsSeq := getSeq("function-params")

	// The nested compiler inherits the outer stylesheet's configuration (or, for
	// the standalone path, the TransformOption values), including the effective
	// per-resource read cap so that resources loaded while COMPILING the nested
	// stylesheet/package (its include/import/schema/param-doc reads) honor the
	// same MaxResourceBytes override rather than falling back to the default.
	nestedCompiler := cfg.nestedCompiler

	// Apply static-params from the options map to the nested compiler.
	// Static params affect both compile time (use-when, shadow attributes)
	// and runtime (param default values), so we collect them for both paths.
	var staticParamValues map[string]xpath3.Sequence
	if staticParamsSeq != nil && sequence.Len(staticParamsSeq) > 0 {
		if sm, ok := staticParamsSeq.Get(0).(xpath3.MapItem); ok {
			staticParamValues = make(map[string]xpath3.Sequence, sm.Size())
			_ = sm.ForEach(func(key xpath3.AtomicValue, value xpath3.Sequence) error {
				name, sErr := paramKeyToClark(key)
				if sErr != nil {
					return nil //nolint:nilerr // skip unparseable static param keys
				}
				nestedCompiler = nestedCompiler.SetStaticParameter(name, value)
				staticParamValues[name] = value
				return nil
			})
		}
	}

	// Compile the stylesheet
	var ss *Stylesheet
	if stylesheetLoc != "" {
		// Resolve relative to the current stylesheet base URI.
		loc := resolveStylesheetLocation(cfg.baseURI, stylesheetLoc)
		var data []byte
		baseURI := loc
		if cfg.stylesheetResolver == nil {
			return nil, dynamicError(errCodeFOXT0003, "fn:transform: cannot read stylesheet %q: no URIResolver configured (filesystem access is opt-in)", stylesheetLoc)
		}
		rc, resolveErr := cfg.stylesheetResolver.Resolve(loc)
		if resolveErr != nil {
			return nil, dynamicError(errCodeFOXT0003, "fn:transform: cannot resolve stylesheet %q: %v", stylesheetLoc, resolveErr)
		}
		var readErr error
		data, readErr = readResourceBounded(rc, cfg.maxResourceBytes)
		// Close right after reading rather than deferring: the rest of this
		// function parses, compiles and runs the stylesheet, and we must not
		// hold the source handle open across that work.
		_ = rc.Close()
		if readErr != nil {
			return nil, dynamicErrorCause(errCodeFOXT0003, readErr, "fn:transform: cannot read stylesheet %q: %v", stylesheetLoc, readErr)
		}
		doc, parseErr := parseStylesheetDocument(ctx, cfg.parseParser, data, baseURI, cfg.allowExternalEntities, cfg.entityLoader, cfg.maxResourceBytes)
		if parseErr != nil {
			return nil, dynamicError(errCodeFOXT0003, "fn:transform: cannot parse stylesheet %q: %v", stylesheetLoc, parseErr)
		}
		var compileErr error
		ss, compileErr = nestedCompiler.BaseURI(baseURI).Compile(ctx, doc)
		if compileErr != nil {
			return nil, dynamicErrorCause(errCodeFOXT0003, compileErr, "fn:transform: cannot compile stylesheet %q: %v", stylesheetLoc, compileErr)
		}
	} else if packageName != "" {
		// Resolve via package-name / package-version using the PackageResolver
		// stored on the compiled stylesheet (set at compile time).
		resolver := cfg.packageResolver
		if resolver == nil {
			return nil, dynamicError(errCodeFOXT0002, "fn:transform: package-name specified but no PackageResolver available")
		}
		rc, location, resolveErr := resolver.ResolvePackage(packageName, packageVersion)
		if resolveErr != nil {
			return nil, dynamicError(errCodeFOXT0003, "fn:transform: cannot resolve package %q (version %q): %v", packageName, packageVersion, resolveErr)
		}
		data, readErr := readResourceBounded(rc, cfg.maxResourceBytes)
		_ = rc.Close()
		if readErr != nil {
			return nil, dynamicErrorCause(errCodeFOXT0003, readErr, "fn:transform: cannot read package %q: %v", packageName, readErr)
		}
		doc, parseErr := parseStylesheetDocument(ctx, cfg.parseParser, data, location, cfg.allowExternalEntities, cfg.entityLoader, cfg.maxResourceBytes)
		if parseErr != nil {
			return nil, dynamicError(errCodeFOXT0003, "fn:transform: cannot parse package %q: %v", packageName, parseErr)
		}
		compiler := nestedCompiler
		if location != "" {
			compiler = compiler.BaseURI(location)
		}
		var compileErr error
		ss, compileErr = compiler.Compile(ctx, doc)
		if compileErr != nil {
			return nil, dynamicErrorCause(errCodeFOXT0003, compileErr, "fn:transform: cannot compile package %q: %v", packageName, compileErr)
		}
	} else if stylesheetText := getStr("stylesheet-text"); stylesheetText != "" {
		// stylesheet-text: the stylesheet source is supplied inline as a string.
		// Parse it (base URI = the static base URI) and compile.
		baseURI := cfg.baseURI
		doc, parseErr := parseStylesheetDocument(ctx, cfg.parseParser, []byte(stylesheetText), baseURI, cfg.allowExternalEntities, cfg.entityLoader, cfg.maxResourceBytes)
		if parseErr != nil {
			return nil, dynamicError(errCodeFOXT0003, "fn:transform: cannot parse stylesheet-text: %v", parseErr)
		}
		compiler := nestedCompiler
		if baseURI != "" {
			compiler = compiler.BaseURI(baseURI)
		}
		var compileErr error
		ss, compileErr = compiler.Compile(ctx, doc)
		if compileErr != nil {
			return nil, dynamicErrorCause(errCodeFOXT0003, compileErr, "fn:transform: cannot compile stylesheet-text: %v", compileErr)
		}
	} else {
		// Check for stylesheet-node
		ssNodeSeq := getSeq("stylesheet-node")
		if ssNodeSeq != nil && sequence.Len(ssNodeSeq) > 0 {
			if ni, ok := ssNodeSeq.Get(0).(xpath3.NodeItem); ok {
				// Find the document containing this node
				var doc *helium.Document
				n := ni.Node
				for n != nil {
					if d, ok := n.(*helium.Document); ok {
						doc = d
						break
					}
					n = n.Parent()
				}
				if doc == nil {
					return nil, dynamicError(errCodeFOXT0003, "fn:transform: stylesheet-node is not part of a document")
				}
				var compileErr error
				ss, compileErr = nestedCompiler.Compile(ctx, doc)
				if compileErr != nil {
					return nil, dynamicErrorCause(errCodeFOXT0003, compileErr, "fn:transform: cannot compile stylesheet: %v", compileErr)
				}
			}
		}
	}

	if ss == nil {
		return nil, dynamicError(errCodeFOXT0002, "fn:transform: no stylesheet specified (stylesheet-location, stylesheet-node, stylesheet-text, or package-name required)")
	}

	// Determine the source document
	var sourceDoc *helium.Document
	if sourceNode != nil && sequence.Len(sourceNode) > 0 {
		if ni, ok := sourceNode.Get(0).(xpath3.NodeItem); ok {
			n := ni.Node
			for n != nil {
				if d, ok := n.(*helium.Document); ok {
					sourceDoc = d
					break
				}
				n = n.Parent()
			}
		}
	}

	// Build a fresh transform config for the inner fn:transform call.
	// Inherit the outer Invocation's URIResolver and HTTPClient so that
	// fn:doc / fn:unparsed-text inside the inner transform see the same
	// opt-in resource access as the caller. Without this, secure-by-default
	// retrieval would refuse network/filesystem access even when the outer
	// Invocation enabled it.
	secondaryResults := make(map[string]*helium.Document)
	secondaryOutputDefs := make(map[string]*OutputDef)
	fnTransformCfg := &transformConfig{
		initialTemplate:  initialTemplate,
		initialMode:      initialMode,
		initialFunction:  initialFunction,
		baseOutputURI:    baseOutputURI,
		resultDocHandler: resultDocCollector{results: secondaryResults, outputDefs: secondaryOutputDefs},
	}
	fnTransformCfg.uriResolver = cfg.innerURIResolver
	fnTransformCfg.httpClient = cfg.httpClient
	// Inherit the outer Invocation's effective per-resource read cap so that
	// fn:doc / fn:unparsed-text / fn:json-doc inside the inner transform honor
	// the same MaxResourceBytes override. Without this the inner reads would
	// silently fall back to the default cap, ignoring Invocation.MaxResourceBytes.
	fnTransformCfg.maxResourceBytes = cfg.maxResourceBytes
	// Inherit the outer Invocation's external-entity opt-in so doc() /
	// xsl:source-document inside the nested transform see the same posture as the
	// caller. Without this the nested transform would force the secure (blocked)
	// parse even when the outer invocation opted in.
	fnTransformCfg.allowExternalEntities = cfg.allowExternalEntities
	// Inherit the injected base parser so nested-transform runtime parses use the
	// same parse policy as the caller.
	fnTransformCfg.parser = cfg.parseParser

	// Apply map-valued options from the fn:transform options map.
	for _, mp := range []struct {
		seq    xpath3.Sequence
		target *map[string]xpath3.Sequence
	}{
		{stylesheetParamsSeq, &fnTransformCfg.sequenceParams},
		{templateParamsSeq, &fnTransformCfg.initialTemplateParams},
		{tunnelParamsSeq, &fnTransformCfg.initialTemplateTunnel},
	} {
		if mp.seq == nil || sequence.Len(mp.seq) == 0 {
			continue
		}
		sm, ok := mp.seq.Get(0).(xpath3.MapItem)
		if !ok {
			continue
		}
		params := make(map[string]xpath3.Sequence, sm.Size())
		_ = sm.ForEach(func(key xpath3.AtomicValue, value xpath3.Sequence) error {
			name, sErr := paramKeyToClark(key)
			if sErr != nil {
				return nil //nolint:nilerr // skip unparseable param keys
			}
			params[name] = value
			return nil
		})
		*mp.target = params
	}

	// Merge static params as runtime params so the externally supplied
	// values override the compiled select="..." defaults at runtime.
	// Explicit stylesheet-params take precedence over static-params.
	if len(staticParamValues) > 0 {
		if fnTransformCfg.sequenceParams == nil {
			fnTransformCfg.sequenceParams = make(map[string]xpath3.Sequence, len(staticParamValues))
		}
		for name, val := range staticParamValues {
			if _, exists := fnTransformCfg.sequenceParams[name]; !exists {
				fnTransformCfg.sequenceParams[name] = val
			}
		}
	}

	// Apply function-params (array of sequences) for initial-function.
	if functionParamsSeq != nil && sequence.Len(functionParamsSeq) > 0 {
		if arr, ok := functionParamsSeq.Get(0).(xpath3.ArrayItem); ok {
			fnTransformCfg.initialFunctionParams = arr.Members()
		}
	}

	// Execute the transform
	var resultDoc *helium.Document
	var capturedItems xpath3.Sequence
	if initialMatchSel != nil && sequence.Len(initialMatchSel) > 0 {
		// initial-match-selection: when the selection is a single node, derive
		// the owning document so the source tree (and its accumulators, schema
		// annotations, etc.) is available during template execution.
		if sequence.Len(initialMatchSel) == 1 {
			if ni, ok := initialMatchSel.Get(0).(xpath3.NodeItem); ok {
				n := ni.Node
				for n != nil {
					if d, ok := n.(*helium.Document); ok {
						sourceDoc = d
						break
					}
					n = n.Parent()
				}
			}
		}
		// Route the selection through the normal executeTransform path via the
		// shared config so output-def resolution, initial-mode resolution,
		// global-context-item handling, result-document/message handlers, and
		// schema/static context all behave identically to a source-driven
		// transform.
		fnTransformCfg.initialMatchSelection = initialMatchSel
	}
	if sourceDoc == nil {
		sourceDoc = helium.NewDefaultDocument()
	}
	// Enable raw capture when delivery-format is "raw" so function items and
	// other non-node XDM values are preserved for raw delivery.
	if deliveryFormat == lexicon.OutputRaw {
		fnTransformCfg.rawCapture = true
	}
	var execErr error
	resultDoc, execErr = executeTransform(ctx, sourceDoc, ss, fnTransformCfg)
	if execErr != nil {
		return nil, execErr
	}
	if fnTransformCfg.rawCapture {
		capturedItems = fnTransformCfg.rawCapturedItems
	}

	// Build result map
	outputKey := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "output"}
	result := xpath3.MapItem{}

	switch deliveryFormat {
	case "raw":
		// Raw delivery: return the XDM items from the transformation.
		// When captured items are available (from raw capture mode), use
		// those directly — they may contain function items, maps, etc.
		// that cannot be represented as DOM children. Otherwise fall
		// back to extracting DOM children for backward compatibility.
		if capturedItems != nil && sequence.Len(capturedItems) > 0 {
			// Merge DOM children and captured non-node items.
			var seq xpath3.ItemSlice
			for child := range helium.Children(resultDoc) {
				seq = append(seq, xpath3.NodeItem{Node: child})
			}
			seq = append(seq, sequence.Materialize(capturedItems)...)
			result = result.Put(outputKey, seq)
		} else if resultDoc != nil {
			var seq xpath3.ItemSlice
			for child := range helium.Children(resultDoc) {
				seq = append(seq, xpath3.NodeItem{Node: child})
			}
			result = result.Put(outputKey, seq)
		}
		// Secondary results are returned as document nodes in raw mode.
		for href, doc := range secondaryResults {
			hrefKey := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: href}
			result = result.Put(hrefKey, xpath3.ItemSlice{xpath3.NodeItem{Node: doc}})
		}
	case "serialized":
		// Serialized delivery: serialize the result document to a string.
		if resultDoc != nil {
			outDef := fnTransformCfg.resolvedOutputDef
			var buf bytes.Buffer
			if err := SerializeResult(&buf, resultDoc, outDef); err != nil {
				return nil, dynamicError(errCodeFOXT0003, "fn:transform: serialization error: %v", err)
			}
			result = result.Put(outputKey, xpath3.SingleString(buf.String()))
		}
		// Serialize secondary results too.
		for href, doc := range secondaryResults {
			hrefKey := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: href}
			outDef := secondaryOutputDefs[href]
			if outDef == nil {
				outDef = ss.outputs[href]
			}
			if outDef == nil {
				outDef = ss.outputs[""]
			}
			var buf bytes.Buffer
			if err := SerializeResult(&buf, doc, outDef); err != nil {
				result = result.Put(hrefKey, xpath3.SingleString(""))
			} else {
				// Trim trailing newline added by the XML serializer's document
				// serialization so the string value matches spec expectations.
				result = result.Put(hrefKey, xpath3.SingleString(strings.TrimRight(buf.String(), "\n")))
			}
		}
	default:
		// Default: return the result document
		if resultDoc != nil {
			result = result.Put(outputKey, xpath3.ItemSlice{xpath3.NodeItem{Node: resultDoc}})
		}
		// Add secondary results as document nodes.
		for href, doc := range secondaryResults {
			hrefKey := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: href}
			result = result.Put(hrefKey, xpath3.ItemSlice{xpath3.NodeItem{Node: doc}})
		}
	}

	return xpath3.ItemSlice{result}, nil
}
