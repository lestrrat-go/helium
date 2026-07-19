package heliumcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
)

// writerErrorHandler writes each error to an io.Writer.
type writerErrorHandler struct {
	w io.Writer
}

func (h *writerErrorHandler) Handle(_ context.Context, err error) {
	_, _ = fmt.Fprint(h.w, err)
}

// errSchemaCompilation is the CLI-side sentinel for a schema that produced
// fatal compilation diagnostics. The xsd compiler may still return a non-nil
// schema with a nil error in that case (the terminal failure contract lives in
// the xsd package); the CLI must not validate against such a schema.
var errSchemaCompilation = errors.New("schema compilation failed")

// compileErrorHandler streams schema compilation diagnostics to an io.Writer
// (as writerErrorHandler does) and additionally records whether any fatal
// diagnostic was seen, so the CLI can fail compilation even when the xsd
// compiler returns a (schema, nil) for a malformed schema.
//
// When suppressWarnings is set (e.g. under --quiet), non-fatal/non-error
// diagnostics (warning level, or errors that carry no severity, which are
// treated as warnings per helium.ErrorLeveler) are not printed. Fatal and
// error-level diagnostics are always printed, and the fatal sentinel is still
// recorded regardless of suppression.
type compileErrorHandler struct {
	w                io.Writer
	fatal            bool
	suppressWarnings bool
}

func (h *compileErrorHandler) Handle(_ context.Context, err error) {
	// Errors that do not implement ErrorLeveler are treated as warnings.
	level := helium.ErrorLevelWarning
	if l, ok := err.(helium.ErrorLeveler); ok {
		level = l.ErrorLevel()
	}
	if level == helium.ErrorLevelFatal {
		h.fatal = true
	}
	if h.suppressWarnings && level < helium.ErrorLevelError {
		return
	}
	_, _ = fmt.Fprint(h.w, err)
}

type xsdValidateConfig struct {
	schemaFile    string
	timing        bool
	version       bool
	maxInputBytes int64
	maxDepth      int
}

type xsdValidateInput struct {
	name  string
	stdin bool
}

type xsdValidateCommand struct {
	prog     string
	stdin    io.Reader
	stderr   io.Writer
	stdinTTY bool
}

func newXSDValidateCommandWithIO(prog string, stdin io.Reader, stderr io.Writer, stdinTTY bool) *xsdValidateCommand {
	return &xsdValidateCommand{
		prog:     prog,
		stdin:    stdin,
		stderr:   stderr,
		stdinTTY: stdinTTY,
	}
}

func (c *xsdValidateCommand) runContext(ctx context.Context, args []string) int {
	cfg, files := c.parseArgs(args)
	if cfg == nil {
		c.showUsage()
		return ExitErr
	}

	if cfg.version {
		c.showVersion()
		return ExitOK
	}

	var inputs []xsdValidateInput
	switch {
	case len(files) > 0:
		for _, f := range files {
			inputs = append(inputs, xsdValidateInput{name: f})
		}
	case !c.stdinTTY:
		inputs = append(inputs, xsdValidateInput{name: "-", stdin: true})
	default:
		c.showUsage()
		return ExitErr
	}

	var t0 time.Time
	if cfg.timing {
		t0 = time.Now()
	}
	// Compile with a Label and an ErrorHandler so fatal schema diagnostics
	// (file/line/detail) reach stderr before the summary error, rather than
	// being discarded.
	ceh := &compileErrorHandler{w: c.stderr}
	// The xsd compiler now denies nested-schema FS access by default; the CLI is
	// a trusted local tool, so restore permissive host access for
	// xs:include/xs:import/xs:redefine.
	schema, err := xsd.NewCompiler().
		Label(cfg.schemaFile).
		FS(iofsPermissiveRoot()).
		ErrorHandler(ceh).
		CompileFile(ctx, cfg.schemaFile)
	if cfg.timing {
		_, _ = fmt.Fprintf(c.stderr, "Compiling schema took %s\n", time.Since(t0))
	}
	if err == nil && ceh.fatal {
		err = errSchemaCompilation
	}
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: failed to compile schema: %s\n", c.prog, err)
		return ExitSchemaComp
	}

	exitCode := ExitOK
	for _, input := range inputs {
		code := c.processInput(ctx, cfg, input, schema)
		exitCode = mergeExitCode(exitCode, code)
	}
	return exitCode
}

func (c *xsdValidateCommand) showVersion() {
	_, _ = fmt.Fprintf(c.stderr, "%s: using helium (%s)\n", c.prog, commitID())
}

func (c *xsdValidateCommand) showUsage() {
	_, _ = fmt.Fprintf(c.stderr, `Usage : %s [options] SCHEMA [XMLfiles ...]
	Validate XML files against an XML Schema
	--timing : print timing information to stderr
	--max-input-bytes N : cap bytes read per input (0 = unlimited)
	--max-depth N : cap element nesting depth (default 256, 0 = unlimited)
	--version : display the version of the XML library used
`, c.prog)
}

func (c *xsdValidateCommand) parseArgs(args []string) (*xsdValidateConfig, []string) {
	cfg := &xsdValidateConfig{maxInputBytes: DefaultMaxInputBytes, maxDepth: -1}
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case flagVersion:
			cfg.version = true
		case "--timing":
			cfg.timing = true
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
		case flagMaxDepth:
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

	if len(positional) == 0 {
		_, _ = fmt.Fprintf(c.stderr, "%s: schema is required\n", c.prog)
		return nil, nil
	}

	cfg.schemaFile = positional[0]
	return cfg, positional[1:]
}

func (c *xsdValidateCommand) processInput(ctx context.Context, cfg *xsdValidateConfig, input xsdValidateInput, schema *xsd.Schema) int {
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
		_, _ = fmt.Fprintf(c.stderr, "Parsing took %s\n", time.Since(t0))
	}
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
		return ExitErr
	}

	if cfg.timing {
		t0 = time.Now()
	}
	handler := &writerErrorHandler{w: c.stderr}
	err = xsd.NewValidator(schema).ErrorHandler(handler).Validate(ctx, doc)
	if cfg.timing {
		_, _ = fmt.Fprintf(c.stderr, "Validating took %s\n", time.Since(t0))
	}
	if err != nil {
		return ExitValidation
	}
	return ExitOK
}
