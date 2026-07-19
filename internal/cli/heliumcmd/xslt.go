package heliumcmd

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
)

// ExitXSLT is returned when stylesheet compilation or transformation fails.
const ExitXSLT = 11

type xsltConfig struct {
	stylesheetFile string
	outputFile     string
	timing         bool
	noout          bool
	version        bool
	params         []xsltParam
	maxInputBytes  int64
	maxDepth       int
	// substituteEntities records --noent; loadExternalDTD records --loaddtd.
	// loadExternal is set by either and is the explicit opt-in that lifts the
	// stylesheet parser's default XXE block and installs an FS confined to the
	// stylesheet's directory so its external DTDs/entities actually load.
	// Without it the stylesheet is parsed with the secure NewParser default and
	// no external resource reaches the filesystem.
	substituteEntities bool
	loadExternalDTD    bool
	loadExternal       bool
}

type xsltParam struct {
	name   string
	value  string
	isExpr bool // true for --param (XPath), false for --stringparam
}

type xsltInput struct {
	name  string
	stdin bool
}

type xsltCommand struct {
	prog     string
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
	stdinTTY bool
}

func newXSLTCommandWithIO(prog string, stdin io.Reader, stdout, stderr io.Writer, stdinTTY bool) *xsltCommand {
	return &xsltCommand{
		prog:     prog,
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
		stdinTTY: stdinTTY,
	}
}

func (c *xsltCommand) runContext(ctx context.Context, args []string) int {
	cfg, files := c.parseArgs(args)
	if cfg == nil {
		c.showUsage()
		return ExitErr
	}

	if cfg.version {
		c.showVersion()
		return ExitOK
	}

	var inputs []xsltInput
	switch {
	case len(files) > 0:
		for _, f := range files {
			inputs = append(inputs, xsltInput{name: f})
		}
	case !c.stdinTTY:
		inputs = append(inputs, xsltInput{name: "-", stdin: true})
	default:
		c.showUsage()
		return ExitErr
	}

	// Compile the stylesheet.
	ssBuf, err := readInputFile(cfg.stylesheetFile, cfg.maxInputBytes)
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: failed to read stylesheet: %s\n", c.prog, err)
		return ExitReadFile
	}

	var t0 time.Time
	if cfg.timing {
		t0 = time.Now()
	}

	// NewParser is secure by default: it blocks external DTD/entity loading and
	// uses a deny-all FS, so a hostile stylesheet cannot read local files via a
	// SYSTEM entity. The legacy permissive behavior is opt-in only, mirroring
	// the lint command's --noent/--loaddtd flags. When opted in, external loads
	// are confined to the stylesheet's own directory (not a raw permissive root)
	// so an attacker-controlled SYSTEM identifier still cannot exfiltrate
	// arbitrary local files.
	// Absolutize the stylesheet path ONCE and use the result for the parser base
	// URI, the confined FS root, and the compiler base URI. With a RELATIVE
	// stylesheet path (e.g. "sub/main.xsl") a relative base URI would make the
	// parser resolve "style.dtd" to "sub/style.dtd", which the confined FS
	// (rooted at the absolute "sub" directory) would then join under its root
	// AGAIN, producing a nonexistent "sub/sub/style.dtd". An absolute base makes
	// every system ID resolve to an absolute path that lands directly inside the
	// confined root.
	absSS, err := filepath.Abs(cfg.stylesheetFile)
	if err != nil {
		absSS = cfg.stylesheetFile
	}

	ssParser := helium.NewParser().BaseURI(absSS)
	// Expand the stylesheet's INTERNAL general entities by default. NewParser's
	// secure default leaves SubstituteEntities off, which not only blocks XXE but
	// also stops a perfectly safe internal-subset entity (e.g.
	// <!DOCTYPE xsl:stylesheet [<!ENTITY msg "ok">]> ... &msg;) from expanding:
	// xslt3 only compiles text/CDATA in sequence constructors, so an unexpanded
	// EntityRefNode silently drops the value. SubstituteEntities(true) on top of
	// the still-default BlockXXE(true)/LoadExternalDTD(false) expands internal
	// entities while external DTD/entity content stays blocked, mirroring xslt3's
	// own secure parser (xslt3/xslt3.go). The XXE block is what keeps a SYSTEM
	// entity's external replacement text from loading even with substitution on.
	//
	// --loaddtd loads the external DTD subset (its declarations / default
	// attributes) but, on its own, must NOT substitute external-DTD-defined
	// general entities; that remains an explicit --noent opt-in. So suppress the
	// internal-substitution default when --loaddtd is given without --noent.
	if cfg.substituteEntities || !cfg.loadExternalDTD {
		ssParser = ssParser.SubstituteEntities(true)
	}
	if cfg.loadExternalDTD {
		ssParser = ssParser.LoadExternalDTD(true)
	}
	if cfg.loadExternal {
		ssParser = ssParser.BlockXXE(false).FS(newConfinedDirFS(absSS))
	}
	ssParser = applyMaxDepth(ssParser, cfg.maxDepth)
	ssDoc, err := ssParser.Parse(ctx, ssBuf)
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: failed to parse stylesheet: %s\n", c.prog, err)
		return ExitXSLT
	}

	// Install a filesystem URIResolver rooted at the stylesheet so local
	// xsl:include/xsl:import modules load. Without one, the compiler
	// default-denies module loading and local stylesheets fail to compile.
	ss, err := xslt3.NewCompiler().
		BaseURI(absSS).
		URIResolver(fileResolver{maxInputBytes: cfg.maxInputBytes}).
		Compile(ctx, ssDoc)
	if cfg.timing {
		_, _ = fmt.Fprintf(c.stderr, "Compiling stylesheet took %s\n", time.Since(t0))
	}
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: failed to compile stylesheet: %s\n", c.prog, err)
		return ExitXSLT
	}

	// Build parameters.
	params, err := c.buildParams(ctx, cfg)
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
		return ExitErr
	}

	// Determine output destination.
	out := c.stdout
	var pending *pendingOutput
	if cfg.outputFile != "" {
		// Refuse to overwrite any input or the stylesheet. This is a fast,
		// friendly pre-flight rejection; the temp-file-then-rename scheme
		// below is what actually closes the truncate-before-read hole for
		// files resolved at transform time (e.g. fn:transform with a
		// stylesheet-location read through the retained URIResolver).
		if code := c.checkOutputCollision(cfg, inputs); code != ExitOK {
			return code
		}

		// Write to a sibling temp file and rename onto the destination only
		// after processing succeeds. os.Create on the destination would
		// truncate it up front, destroying any file the transform reads later.
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
		code := c.processInput(ctx, cfg, input, ss, params, out)
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
		// failed commit means the output may be incomplete.
		if err := pending.Commit(); err != nil {
			_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
			exitCode = mergeExitCode(exitCode, ExitErr)
		}
	}

	return exitCode
}

// checkOutputCollision reports a non-OK exit code if the configured output
// file refers to the same file as any XML input or the stylesheet.
func (c *xsltCommand) checkOutputCollision(cfg *xsltConfig, inputs []xsltInput) int {
	for _, input := range inputs {
		if input.stdin {
			continue
		}
		if samePath(cfg.outputFile, input.name) {
			_, _ = fmt.Fprintf(c.stderr, "%s: --output %q would overwrite input %q\n", c.prog, cfg.outputFile, input.name)
			return ExitErr
		}
	}
	if samePath(cfg.outputFile, cfg.stylesheetFile) {
		_, _ = fmt.Fprintf(c.stderr, "%s: --output %q would overwrite stylesheet %q\n", c.prog, cfg.outputFile, cfg.stylesheetFile)
		return ExitErr
	}
	return ExitOK
}

func (c *xsltCommand) buildParams(ctx context.Context, cfg *xsltConfig) (*xslt3.Parameters, error) {
	if len(cfg.params) == 0 {
		return nil, nil //nolint:nilnil
	}

	params := xslt3.NewParameters()
	for _, p := range cfg.params {
		if p.isExpr {
			// Evaluate XPath expression to get the value.
			expr, err := xpath3.NewCompiler().Compile(p.value)
			if err != nil {
				return nil, fmt.Errorf("invalid XPath in --param %s: %w", p.name, err)
			}
			eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
			result, err := eval.Evaluate(ctx, expr, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate --param %s: %w", p.name, err)
			}
			params.Set(p.name, result.Sequence())
		} else {
			params.SetString(p.name, p.value)
		}
	}
	return params, nil
}

func (c *xsltCommand) processInput(ctx context.Context, cfg *xsltConfig, input xsltInput, ss *xslt3.Stylesheet, params *xslt3.Parameters, out io.Writer) int {
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

	var t0 time.Time
	if cfg.timing {
		t0 = time.Now()
	}

	p := applyMaxDepth(helium.NewParser(), cfg.maxDepth)
	if !input.stdin {
		p = p.BaseURI(input.name)
	}
	doc, err := p.Parse(ctx, buf)
	if cfg.timing {
		_, _ = fmt.Fprintf(c.stderr, "Parsing %s took %s\n", input.name, time.Since(t0))
	}
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
		return ExitErr
	}

	if cfg.timing {
		t0 = time.Now()
	}

	inv := ss.Transform(doc)
	if params != nil {
		inv = inv.GlobalParameters(params)
	}

	if cfg.noout {
		_, err = inv.Do(ctx)
	} else {
		err = inv.WriteTo(ctx, out)
	}
	if cfg.timing {
		_, _ = fmt.Fprintf(c.stderr, "Transforming %s took %s\n", input.name, time.Since(t0))
	}
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
		return ExitXSLT
	}
	return ExitOK
}

func (c *xsltCommand) showVersion() {
	_, _ = fmt.Fprintf(c.stderr, "%s: using helium (%s)\n", c.prog, commitID())
}

func (c *xsltCommand) showUsage() {
	_, _ = fmt.Fprintf(c.stderr, `Usage: %s [options] STYLESHEET [XMLfiles ...]
	Transform XML files using an XSLT 3.0 stylesheet

Options:
	--output FILE    : write output to FILE
	-o FILE          : write output to FILE
	--param NAME VAL : pass XPath parameter
	--stringparam NAME VAL : pass string parameter
	--noout          : suppress output
	--noent          : substitute entities, loading the stylesheet's external entities (opt-in; off by default)
	--loaddtd        : load the stylesheet's external DTD subset (opt-in; off by default)
	--timing         : print timing information to stderr
	--max-input-bytes N : cap bytes read per input (0 = unlimited)
	--max-depth N : cap element nesting depth (default 256, 0 = unlimited)
	--version        : display the version of the XML library used
`, c.prog)
}

func (c *xsltCommand) parseArgs(args []string) (*xsltConfig, []string) {
	cfg := &xsltConfig{maxInputBytes: DefaultMaxInputBytes, maxDepth: -1}
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case flagVersion:
			cfg.version = true
		case flagTiming:
			cfg.timing = true
		case "--noout":
			cfg.noout = true
		case "--noent":
			cfg.substituteEntities = true
			cfg.loadExternal = true
		case "--loaddtd":
			cfg.loadExternalDTD = true
			cfg.loadExternal = true
		case "--output", "-o":
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintf(c.stderr, "%s: %s requires a filename\n", c.prog, arg)
				return nil, nil
			}
			cfg.outputFile = args[i]
		case "--param":
			if i+2 >= len(args) {
				_, _ = fmt.Fprintf(c.stderr, "%s: --param requires NAME and VALUE\n", c.prog)
				return nil, nil
			}
			cfg.params = append(cfg.params, xsltParam{name: args[i+1], value: args[i+2], isExpr: true})
			i += 2
		case "--stringparam":
			if i+2 >= len(args) {
				_, _ = fmt.Fprintf(c.stderr, "%s: --stringparam requires NAME and VALUE\n", c.prog)
				return nil, nil
			}
			cfg.params = append(cfg.params, xsltParam{name: args[i+1], value: args[i+2], isExpr: false})
			i += 2
		case flagMaxInputBytes:
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintf(c.stderr, "%s: --max-input-bytes requires an argument\n", c.prog)
				return nil, nil
			}
			n, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil || n < 0 {
				_, _ = fmt.Fprintf(c.stderr, "%s: --max-input-bytes: invalid argument %q\n", c.prog, args[i])
				return nil, nil
			}
			cfg.maxInputBytes = n
		case flagMaxDepth:
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintf(c.stderr, "%s: --max-depth requires an argument\n", c.prog)
				return nil, nil
			}
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 0 {
				_, _ = fmt.Fprintf(c.stderr, "%s: --max-depth: invalid argument %q\n", c.prog, args[i])
				return nil, nil
			}
			cfg.maxDepth = n
		default:
			if len(arg) > 0 && arg[0] == '-' {
				_, _ = fmt.Fprintf(c.stderr, "%s: unrecognized option %s\n", c.prog, arg)
				return nil, nil
			}
			positional = append(positional, arg)
		}
	}

	if cfg.version {
		return cfg, positional
	}

	// --noout suppresses writing, so opening (truncating) the output file would
	// silently destroy its contents. Reject the combination.
	if cfg.outputFile != "" && cfg.noout {
		_, _ = fmt.Fprintf(c.stderr, "%s: --output cannot be combined with --noout\n", c.prog)
		return nil, nil
	}

	if len(positional) == 0 {
		_, _ = fmt.Fprintf(c.stderr, "%s: stylesheet is required\n", c.prog)
		return nil, nil
	}

	cfg.stylesheetFile = positional[0]
	return cfg, positional[1:]
}
