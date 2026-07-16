package heliumcmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
	"github.com/lestrrat-go/helium/catalog"
	henc "github.com/lestrrat-go/helium/internal/encoding"
	"github.com/lestrrat-go/helium/internal/uripath"
	"github.com/lestrrat-go/helium/xinclude"
	"github.com/lestrrat-go/helium/xpath1"
	"github.com/lestrrat-go/helium/xsd"
)

// Exit codes matching xmllint conventions.
const (
	ExitOK         = 0
	ExitErr        = 1
	ExitValidation = 3
	ExitReadFile   = 4
	ExitSchemaComp = 5
	ExitXPath      = 10
)

type config struct {
	parser helium.Parser

	doXInclude  bool
	noXIncNode  bool
	noBaseFixup bool
	dtdValid    bool
	c14nMode    int
	schemaFile  string
	xpathExpr   string
	catalogs    bool
	noCatalogs  bool
	pathDirs    string

	// loadExternal records that a flag requesting external DTD/entity loading
	// (--loaddtd, --valid, --dtdattr, --noent) was given. NewParser blocks
	// external loading by default; these flags are the user's explicit opt-in,
	// so the parser's XXE block is lifted and a permissive FS installed.
	loadExternal bool

	// huge records --huge; maxDepth records --max-depth (-1 = unset). Both are
	// applied once after argument parsing so the result is order-independent:
	// --huge lifts the limits, then an explicit --max-depth (the more specific
	// flag) re-imposes its cap and wins regardless of flag order.
	huge     bool
	maxDepth int

	noout      bool
	format     bool
	outputFile string
	encode     string
	pretty     int

	quiet   bool
	timing  bool
	dropdtd bool
	repeat  int

	maxInputBytes int64

	version bool
}

type namedInput struct {
	name  string
	stdin bool
}

type command struct {
	prog     string
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
	stdinTTY bool
}

func newCommandWithIO(prog string, stdin io.Reader, stdout, stderr io.Writer, stdinTTY bool) *command {
	return &command{
		prog:     prog,
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
		stdinTTY: stdinTTY,
	}
}

func (c *command) runContext(ctx context.Context, args []string) int {
	cfg, files := c.parseArgs(args)
	if cfg == nil {
		c.showUsage()
		return ExitErr
	}

	if cfg.version {
		c.showVersion()
		return ExitOK
	}

	if cfg.pretty >= 1 {
		cfg.format = true
	}

	// --quiet suppresses non-error informational output. Timing messages are
	// purely informational, so silence them; parser/validator warnings are
	// suppressed via the parser's own SuppressWarnings option.
	if cfg.quiet {
		cfg.timing = false
		cfg.parser = cfg.parser.SuppressWarnings(true)
	}

	var inputs []namedInput
	switch {
	case len(files) > 0:
		for _, f := range files {
			inputs = append(inputs, namedInput{name: f})
		}
	case !c.stdinTTY:
		inputs = append(inputs, namedInput{name: "-", stdin: true})
	default:
		c.showUsage()
		return ExitErr
	}

	var cat helium.CatalogResolver
	if cfg.catalogs && !cfg.noCatalogs {
		cat = c.loadCatalogFromEnv(ctx)
	}

	var schema *xsd.Schema
	if cfg.schemaFile != "" {
		var t0 time.Time
		if cfg.timing {
			t0 = time.Now()
		}

		var err error
		schema, err = c.compileSchema(ctx, cfg)
		if cfg.timing {
			_, _ = fmt.Fprintf(c.stderr, "Compiling schema took %s\n", time.Since(t0))
		}
		if err != nil {
			_, _ = fmt.Fprintf(c.stderr, "%s\n", err)
			return ExitSchemaComp
		}
	}

	out := c.stdout
	var pending *pendingOutput
	if cfg.outputFile != "" {
		// Refuse to write output over any input or the schema. This is a
		// fast, friendly pre-flight rejection; the temp-file-then-rename
		// scheme below is what actually closes the truncate-before-read hole
		// for resources resolved LATER (e.g. a --path DTD).
		if code := c.checkOutputCollision(cfg, inputs); code != ExitOK {
			return code
		}

		// Write to a sibling temp file and rename onto the destination only
		// after processing succeeds. os.Create on the destination would
		// truncate it up front, destroying any input/DTD/entity that the same
		// path is read from later during parse or validation.
		p, err := newPendingOutput(cfg.outputFile)
		if err != nil {
			_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
			return ExitErr
		}
		pending = p
		out = p.File()
	}

	exitCode := ExitOK
	for _, input := range inputs {
		code := c.processInput(ctx, cfg, input, cat, schema, out)
		exitCode = mergeExitCode(exitCode, code)
	}

	if pending != nil {
		// Only publish the output on a fully successful run. On any error the
		// temp file is discarded and the destination is left untouched.
		if exitCode != ExitOK {
			pending.Cleanup()
			return exitCode
		}
		// Fold the commit (flush/close/rename) error into the exit status: a
		// failed commit means the output may be incomplete, which must not be
		// reported as success.
		if err := pending.Commit(); err != nil {
			_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
			exitCode = mergeExitCode(exitCode, ExitErr)
		}
	}

	return exitCode
}

// checkOutputCollision reports a non-OK exit code if the configured output
// file refers to the same file as any XML input or the schema.
func (c *command) checkOutputCollision(cfg *config, inputs []namedInput) int {
	for _, input := range inputs {
		if input.stdin {
			continue
		}
		if samePath(cfg.outputFile, input.name) {
			_, _ = fmt.Fprintf(c.stderr, "%s: --output %q would overwrite input %q\n", c.prog, cfg.outputFile, input.name)
			return ExitErr
		}
	}
	if cfg.schemaFile != "" && samePath(cfg.outputFile, cfg.schemaFile) {
		_, _ = fmt.Fprintf(c.stderr, "%s: --output %q would overwrite schema %q\n", c.prog, cfg.outputFile, cfg.schemaFile)
		return ExitErr
	}
	return ExitOK
}

func (c *command) showVersion() {
	_, _ = fmt.Fprintf(c.stderr, "%s: using helium (%s)\n", c.prog, commitID())
}

func (c *command) showUsage() {
	_, _ = fmt.Fprintf(c.stderr, `Usage : %s [options] XMLfiles ...
	Parse the XML files and output the result of the parsing
	--version : display the version of the XML library used
	--recover : output what was parsable on broken XML documents
	--noent : substitute entity references by their value
	--loaddtd : fetch external DTD
	--dtdattr : loaddtd + populate tree with inherited attributes
	--valid : validate the document with the DTD
	--nowarning : do not emit warnings from parser/validator
	--pedantic : enable pedantic error reporting
	--noblanks : drop (ignorable?) blanks spaces
	--nsclean : remove redundant namespace declarations
	--nocdata : replace CDATA section by equivalent text nodes
	--nonet : refuse to fetch DTDs or entities over network
	--huge : lift the tunable parser limits (element depth, name length, DTD content-model depth, entity-amplification ratio); the absolute 1 GB entity-expansion ceiling is always kept
	--noenc : ignore any encoding specified inside the document
	--noxincludenode : do not generate XInclude START/END nodes
	--nofixup-base-uris : do not fixup xml:base URIs in XInclude
	--noout : do not print the result tree
	--format : reformat/reindent the output
	--pretty LEVEL : pretty-print the output (0=none, any level >=1 enables --format)
	--encode ENCODING : output in the given encoding
	--output FILE : save to a given file
	--c14n : save in W3C canonical format v1.0 (with comments)
	--c14n11 : save in W3C canonical format v1.1 (with comments)
	--exc-c14n : save in W3C exclusive canonical format (with comments)
	--xinclude : do XInclude processing
	--schema FILE : do validation against the WXS schema
	--xpath EXPR : evaluate the XPath expression, imply --noout
	--catalogs : use catalogs from $XML_CATALOG_FILES
	--nocatalogs : do not use any catalogs
	--path DIRS : set search path for DTD/entities (colon-separated)
	--quiet : suppress non-error output
	--timing : print timing information to stderr
	--dropdtd : remove the DOCTYPE of the result
	--repeat N : parse N times for benchmarking
	--max-input-bytes N : cap bytes read per input (0 = unlimited)
	--max-depth N : cap element nesting depth (default 256, 0 = unlimited)
`, c.prog)
}

func (c *command) parseArgs(args []string) (*config, []string) {
	cfg := &config{
		parser:        helium.NewParser(),
		pretty:        -1,
		repeat:        1,
		maxInputBytes: DefaultMaxInputBytes,
		maxDepth:      -1,
	}
	var files []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && len(arg) > 1 {
			arg = "-" + arg
		}

		switch arg {
		case flagVersion:
			cfg.version = true
		case "--recover":
			cfg.parser = cfg.parser.RecoverOnError(true)
		case "--noent":
			cfg.parser = cfg.parser.SubstituteEntities(true)
			cfg.loadExternal = true
		case "--loaddtd":
			cfg.parser = cfg.parser.LoadExternalDTD(true)
			cfg.loadExternal = true
		case "--dtdattr":
			cfg.parser = cfg.parser.DefaultDTDAttributes(true)
			cfg.loadExternal = true
		case "--valid":
			cfg.parser = cfg.parser.ValidateDTD(true)
			cfg.dtdValid = true
			cfg.loadExternal = true
		case "--nowarning":
			cfg.parser = cfg.parser.SuppressWarnings(true)
		case "--pedantic":
			cfg.parser = cfg.parser.PedanticErrors(true)
		case "--noblanks":
			cfg.parser = cfg.parser.StripBlanks(true)
		case "--nsclean":
			cfg.parser = cfg.parser.CleanNamespaces(true)
		case "--nocdata":
			cfg.parser = cfg.parser.MergeCDATA(true)
		case "--nonet":
			cfg.parser = cfg.parser.AllowNetwork(false)
		case "--huge":
			// Recorded and applied post-loop (see parseArgs end) so flag order
			// does not matter relative to --max-depth.
			cfg.huge = true
		case "--noenc":
			cfg.parser = cfg.parser.IgnoreEncoding(true)
		case "--noxincludenode":
			cfg.noXIncNode = true
		case "--nofixup-base-uris":
			cfg.noBaseFixup = true
		case "--noout":
			cfg.noout = true
		case "--format":
			cfg.format = true
		case "--c14n":
			cfg.c14nMode = 1
		case "--c14n11":
			cfg.c14nMode = 2
		case "--exc-c14n":
			cfg.c14nMode = 3
		case "--xinclude":
			cfg.doXInclude = true
		case "--catalogs":
			cfg.catalogs = true
		case "--nocatalogs":
			cfg.noCatalogs = true
		case "--quiet":
			cfg.quiet = true
		case "--timing":
			cfg.timing = true
		case "--dropdtd":
			cfg.dropdtd = true
		case "--schema":
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintf(c.stderr, "%s: --schema requires an argument\n", c.prog)
				return nil, nil
			}
			cfg.schemaFile = args[i] //nolint:gosec // bounds checked above
		case "--xpath":
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintf(c.stderr, "%s: --xpath requires an argument\n", c.prog)
				return nil, nil
			}
			cfg.xpathExpr = args[i] //nolint:gosec // bounds checked above
			cfg.noout = true
		case "--output":
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintf(c.stderr, "%s: --output requires an argument\n", c.prog)
				return nil, nil
			}
			cfg.outputFile = args[i] //nolint:gosec // bounds checked above
		case "--encode":
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintf(c.stderr, "%s: --encode requires an argument\n", c.prog)
				return nil, nil
			}
			cfg.encode = args[i] //nolint:gosec // bounds checked above
			if henc.Load(cfg.encode) == nil {
				_, _ = fmt.Fprintf(c.stderr, "%s: --encode: unsupported encoding %q\n", c.prog, cfg.encode)
				return nil, nil
			}
			// US-ASCII (and its aliases) maps to the UTF-8 encoder, so the
			// serializer would emit raw UTF-8 bytes for any character outside
			// the ASCII range while still declaring US-ASCII. Reject it rather
			// than produce output that does not match its declared encoding.
			if henc.IsASCII(cfg.encode) {
				_, _ = fmt.Fprintf(c.stderr, "%s: --encode: unsupported encoding %q\n", c.prog, cfg.encode)
				return nil, nil
			}
		case "--pretty":
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintf(c.stderr, "%s: --pretty requires an argument\n", c.prog)
				return nil, nil
			}
			n, err := strconv.Atoi(args[i]) //nolint:gosec // bounds checked above
			if err != nil {
				_, _ = fmt.Fprintf(c.stderr, "%s: --pretty: invalid argument %q\n", c.prog, args[i]) //nolint:gosec // bounds checked above
				return nil, nil
			}
			cfg.pretty = n
		case "--path":
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintf(c.stderr, "%s: --path requires an argument\n", c.prog)
				return nil, nil
			}
			cfg.pathDirs = args[i] //nolint:gosec // bounds checked above
		case "--max-depth":
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintf(c.stderr, "%s: --max-depth requires an argument\n", c.prog)
				return nil, nil
			}
			n, err := strconv.Atoi(args[i]) //nolint:gosec // bounds checked above
			if err != nil || n < 0 {
				_, _ = fmt.Fprintf(c.stderr, "%s: --max-depth: invalid argument %q\n", c.prog, args[i]) //nolint:gosec // bounds checked above
				return nil, nil
			}
			cfg.maxDepth = n
		case "--repeat":
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintf(c.stderr, "%s: --repeat requires an argument\n", c.prog)
				return nil, nil
			}
			n, err := strconv.Atoi(args[i]) //nolint:gosec // bounds checked above
			if err != nil || n < 1 {
				_, _ = fmt.Fprintf(c.stderr, "%s: --repeat: invalid argument %q\n", c.prog, args[i]) //nolint:gosec // bounds checked above
				return nil, nil
			}
			cfg.repeat = n
		case flagMaxInputBytes:
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintf(c.stderr, "%s: --max-input-bytes requires an argument\n", c.prog)
				return nil, nil
			}
			n, err := strconv.ParseInt(args[i], 10, 64) //nolint:gosec // bounds checked above
			if err != nil || n < 0 {
				_, _ = fmt.Fprintf(c.stderr, "%s: --max-input-bytes: invalid argument %q\n", c.prog, args[i]) //nolint:gosec // bounds checked above
				return nil, nil
			}
			cfg.maxInputBytes = n
		default:
			if strings.HasPrefix(arg, "--") {
				_, _ = fmt.Fprintf(c.stderr, "%s: unrecognized option %s\n", c.prog, arg) //nolint:gosec // CLI diagnostic output
				return nil, nil
			}
			files = append(files, arg)
		}
	}

	// XPath result serialization prints node values, attributes, and namespace
	// nodes directly and never re-encodes them, so --encode cannot be honored
	// on that path. Reject the combination instead of silently ignoring it.
	if cfg.encode != "" && cfg.xpathExpr != "" {
		_, _ = fmt.Fprintf(c.stderr, "%s: --encode cannot be combined with --xpath\n", c.prog)
		return nil, nil
	}

	// --noout suppresses all writing, so opening (and thus truncating) the
	// output file would silently destroy its contents. Reject the combination
	// rather than clobber the file. --xpath also sets noout but still writes
	// its result to the output destination, so it is exempt.
	if cfg.outputFile != "" && cfg.noout && cfg.xpathExpr == "" {
		_, _ = fmt.Fprintf(c.stderr, "%s: --output cannot be combined with --noout\n", c.prog)
		return nil, nil
	}

	// A flag requesting external DTD/entity loading is the user's explicit
	// opt-in, so lift the parser's default XXE block. The permissive FS that
	// makes loading actually reach the filesystem is installed at parse time
	// (see run), either via --path's search FS or a plain permissive root.
	if cfg.loadExternal {
		cfg.parser = cfg.parser.BlockXXE(false)
	}

	// Apply --huge then --max-depth so the result is independent of flag order.
	// --huge lifts the tunable limits (including the depth cap); an explicit
	// --max-depth then re-imposes its cap and wins, being the more specific flag.
	if cfg.huge {
		cfg.parser = cfg.parser.
			MaxNameLength(-1).
			MaxEntityAmplification(-1).
			MaxContentModelDepth(-1).
			MaxNodeContentSize(-1).
			MaxDepth(0)
	}
	if cfg.maxDepth >= 0 {
		cfg.parser = cfg.parser.MaxDepth(cfg.maxDepth)
	}

	return cfg, files
}

// catalogChain resolves through an ordered list of catalogs, returning the
// first non-empty match. XML_CATALOG_FILES is a whitespace-separated list, so
// every listed catalog must participate in resolution, in listed order.
//
// A catalog break (the OASIS/libxml2 "cut" signal) from any catalog stops the
// chain: when a catalog reports a break it has decided "no match, do not keep
// searching", so the chain must not fall through to later catalogs.
type catalogChain []*catalog.Catalog

func (cc catalogChain) Resolve(ctx context.Context, pubID, sysID string) string {
	for _, cat := range cc {
		uri, broke := cat.ResolveResult(ctx, pubID, sysID)
		if uri != "" {
			return uri
		}
		if broke {
			return ""
		}
	}
	return ""
}

func (cc catalogChain) ResolveURI(ctx context.Context, uri string) string {
	for _, cat := range cc {
		resolved, broke := cat.ResolveURIResult(ctx, uri)
		if resolved != "" {
			return resolved
		}
		if broke {
			return ""
		}
	}
	return ""
}

func (c *command) loadCatalogFromEnv(ctx context.Context) helium.CatalogResolver {
	envFiles := os.Getenv("XML_CATALOG_FILES")
	if envFiles == "" {
		return nil
	}
	var chain catalogChain
	for f := range strings.FieldsSeq(envFiles) {
		cat, err := catalog.Load(ctx, f)
		if err != nil {
			_, _ = fmt.Fprintf(c.stderr, "%s: failed to load catalog %s: %s\n", c.prog, f, err)
			continue
		}
		chain = append(chain, cat)
	}
	if len(chain) == 0 {
		return nil
	}
	return chain
}

// pathDirs splits cfg.pathDirs (colon-separated, xmllint style) into a list of
// non-empty search directories for DTD/entity lookup.
func (c *command) pathDirs(cfg *config) []string {
	if cfg.pathDirs == "" {
		return nil
	}
	var dirs []string
	for _, d := range splitSearchPath(cfg.pathDirs) {
		if d != "" {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// splitSearchPath splits a colon-separated DTD/entity search path (xmllint
// style) while keeping a Windows drive-letter prefix ("D:\\dtd", "C:/x")
// attached to its directory. A naive strings.Split on ':' would shatter
// "D:\\dtd" into "D" and "\\dtd", corrupting the search path on Windows and
// making a DTD resolved via --path unfindable (validation then spuriously
// fails). A colon is treated as a drive separator — and NOT a list separator —
// only when it is the SECOND character of a segment, follows a single ASCII
// letter, and is itself followed by a path separator ('/' or '\\') or the end
// of the segment. This is GOOS-independent (string-shape based), so the
// behavior is exercised on POSIX too; a genuine POSIX list like
// "/a/b:/c/d" still splits normally because "/a/b" does not match the
// drive-prefix shape.
func splitSearchPath(s string) []string {
	var out []string
	start := 0
	for i := range len(s) {
		if s[i] != ':' {
			continue
		}
		// A "X:" at the very start of the current segment, where X is a single
		// ASCII letter and the next char is a separator (or segment end), is a
		// Windows drive prefix, not a list separator.
		if i == start+1 && uripath.IsWindowsDriveLetter(s[start]) {
			if i+1 >= len(s) || s[i+1] == '/' || s[i+1] == '\\' {
				continue
			}
		}
		out = append(out, s[start:i])
		start = i + 1
	}
	out = append(out, s[start:])
	return out
}

func (c *command) compileSchema(ctx context.Context, cfg *config) (*xsd.Schema, error) {
	// Compile with a Label and an ErrorHandler so fatal schema diagnostics
	// (file/line/detail) reach stderr. Without a handler the xsd compiler
	// discards them and the user sees only the terminal "schema compilation
	// failed" summary with no clue what went wrong.
	handler := &compileErrorHandler{w: c.stderr, suppressWarnings: cfg.quiet}
	// The xsd compiler now denies nested-schema FS access by default; the CLI is
	// a trusted local tool, so restore permissive host access for
	// xs:include/xs:import/xs:redefine (mirrors the parser FS lift below).
	schema, err := xsd.NewCompiler().
		Label(cfg.schemaFile).
		FS(iofsPermissiveRoot()).
		ErrorHandler(handler).
		CompileFile(ctx, cfg.schemaFile)
	if err == nil && handler.fatal {
		// The xsd compiler may still return a (schema, nil) for a schema with
		// fatal diagnostics; treat any drained fatal diagnostic as a failure so
		// the CLI never validates against a malformed schema.
		err = errSchemaCompilation
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to compile schema: %w", c.prog, err)
	}
	return schema, nil
}

func (c *command) processInput(ctx context.Context, cfg *config, input namedInput, cat helium.CatalogResolver, schema *xsd.Schema, out io.Writer) int {
	var buf []byte
	var err error
	if input.stdin {
		buf, err = readInput(c.stdin, "-", cfg.maxInputBytes)
	} else {
		buf, err = readInputFile(input.name, cfg.maxInputBytes)
	}
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
		return ExitReadFile
	}

	var doc *helium.Document
	for range cfg.repeat {
		var t0 time.Time
		if cfg.timing {
			t0 = time.Now()
		}

		p := cfg.parser
		if !input.stdin {
			p = p.BaseURI(input.name)
		}
		if cat != nil {
			p = p.Catalog(cat)
		}
		if dirs := c.pathDirs(cfg); len(dirs) > 0 {
			p = p.FS(pathSearchFS{base: iofsPermissiveRoot(), dirs: dirs})
		} else if cfg.loadExternal {
			// External loading opted in (see parseArgs) but no --path given:
			// NewParser now defaults to a deny-all FS, so install the permissive
			// root that the historical CLI used to open DTDs/entities.
			p = p.FS(iofsPermissiveRoot())
		}

		doc, err = p.Parse(ctx, buf)
		if cfg.timing {
			_, _ = fmt.Fprintf(c.stderr, "Parsing took %s\n", time.Since(t0))
		}
		if err != nil {
			_, _ = fmt.Fprintf(c.stderr, "%s\n", err)
			if doc == nil {
				return ExitErr
			}
		}
	}

	if doc == nil {
		return ExitErr
	}

	if cfg.doXInclude {
		var t0 time.Time
		if cfg.timing {
			t0 = time.Now()
		}
		// xinclude.NewProcessor() now denies all filesystem access by default;
		// the CLI processes user-supplied local files, so install the same
		// permissive (or --path-rooted) FS the parser uses above to preserve the
		// historical --xinclude behavior of reading includes off disk.
		xiFS := iofsPermissiveRoot()
		if dirs := c.pathDirs(cfg); len(dirs) > 0 {
			xiFS = pathSearchFS{base: iofsPermissiveRoot(), dirs: dirs}
		}
		xiProc := xinclude.NewProcessor().Resolver(xinclude.NewFSResolver(xiFS))
		if cfg.noXIncNode {
			xiProc = xiProc.NoXIncludeMarkers()
		}
		if cfg.noBaseFixup {
			xiProc = xiProc.NoBaseFixup()
		}
		if !input.stdin {
			xiProc = xiProc.BaseURI(input.name)
		}
		_, xiErr := xiProc.Process(ctx, doc)
		if cfg.timing {
			_, _ = fmt.Fprintf(c.stderr, "XInclude took %s\n", time.Since(t0))
		}
		if xiErr != nil {
			_, _ = fmt.Fprintf(c.stderr, "%s\n", xiErr)
			return ExitErr
		}
	}

	if schema != nil {
		var t0 time.Time
		if cfg.timing {
			t0 = time.Now()
		}
		err := xsd.NewValidator(schema).Validate(ctx, doc)
		if cfg.timing {
			_, _ = fmt.Fprintf(c.stderr, "Validating took %s\n", time.Since(t0))
		}
		if err != nil {
			_, _ = fmt.Fprint(c.stderr, err)
			return ExitValidation
		}
	}

	if cfg.dtdValid && err != nil {
		return ExitValidation
	}

	if cfg.xpathExpr != "" {
		return c.evalXPath(ctx, cfg, doc, out)
	}

	if cfg.noout {
		return ExitOK
	}

	// Honor the requested output encoding. Setting the document encoding
	// makes helium.Writer load the matching encoder and emit a matching
	// encoding declaration. C14N output is always UTF-8 per spec, so the
	// encoding only applies to the standard serialization path.
	if cfg.encode != "" && cfg.c14nMode == 0 {
		doc.SetEncoding(cfg.encode)
	}

	var t0 time.Time
	if cfg.timing {
		t0 = time.Now()
	}

	if cfg.c14nMode > 0 {
		var mode c14n.Mode
		switch cfg.c14nMode {
		case 1:
			mode = c14n.C14N10
		case 2:
			mode = c14n.C14N11
		case 3:
			mode = c14n.ExclusiveC14N10
		}
		cErr := c14n.NewCanonicalizer(mode).Comments().Canonicalize(doc, out)
		if cfg.timing {
			_, _ = fmt.Fprintf(c.stderr, "Saving took %s\n", time.Since(t0))
		}
		if cErr != nil {
			_, _ = fmt.Fprintf(c.stderr, "%s\n", cErr)
			return ExitErr
		}
		return ExitOK
	}

	writer := helium.NewWriter().IndentString("  ")
	if cfg.format {
		writer = writer.Format(true)
	}
	if cfg.dropdtd {
		writer = writer.IncludeDTD(false)
	}
	if dErr := writer.WriteTo(out, doc); dErr != nil {
		if cfg.timing {
			_, _ = fmt.Fprintf(c.stderr, "Saving took %s\n", time.Since(t0))
		}
		_, _ = fmt.Fprintf(c.stderr, "%s\n", dErr)
		return ExitErr
	}
	if cfg.timing {
		_, _ = fmt.Fprintf(c.stderr, "Saving took %s\n", time.Since(t0))
	}
	return ExitOK
}

func (c *command) evalXPath(ctx context.Context, cfg *config, doc *helium.Document, out io.Writer) int {
	expr, err := xpath1.Compile(cfg.xpathExpr)
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
		return ExitXPath
	}

	res, err := xpath1.NewEvaluator().Evaluate(ctx, expr, doc)
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
		return ExitXPath
	}

	switch res.Type {
	case xpath1.NodeSetResult:
		for _, n := range res.NodeSet {
			switch n.Type() {
			case helium.AttributeNode:
				attr, ok := helium.AsNode[*helium.Attribute](n)
				if !ok {
					_, _ = fmt.Fprintf(c.stderr, "%s: unexpected attribute node type %T\n", c.prog, n)
					return ExitErr
				}
				_, _ = fmt.Fprintf(out, " %s=%q\n", attr.Name(), attr.Value())
			case helium.NamespaceDeclNode, helium.NamespaceNode:
				ns, ok := n.(interface {
					Prefix() string
					URI() string
				})
				if !ok {
					_, _ = fmt.Fprintf(c.stderr, "%s: unexpected namespace node type %T\n", c.prog, n)
					return ExitErr
				}
				if ns.Prefix() == "" {
					_, _ = fmt.Fprintf(out, " xmlns=%q\n", ns.URI())
				} else {
					_, _ = fmt.Fprintf(out, " xmlns:%s=%q\n", ns.Prefix(), ns.URI())
				}
			default:
				writer := helium.NewWriter()
				if err := writer.WriteTo(out, n); err != nil {
					_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
					return ExitErr
				}
				_, _ = fmt.Fprintln(out)
			}
		}
	case xpath1.BooleanResult:
		if res.Bool {
			_, _ = fmt.Fprintln(out, "true")
		} else {
			_, _ = fmt.Fprintln(out, "false")
		}
	case xpath1.NumberResult:
		_, _ = fmt.Fprintf(out, "%g\n", res.Number)
	default:
		_, _ = fmt.Fprintln(out, res.String)
	}

	return ExitOK
}
