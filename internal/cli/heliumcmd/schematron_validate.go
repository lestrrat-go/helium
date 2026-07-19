package heliumcmd

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/schematron"
)

type schematronValidateConfig struct {
	schemaFile    string
	timing        bool
	version       bool
	maxInputBytes int64
	maxDepth      int
}

type schematronValidateInput struct {
	name  string
	stdin bool
}

type schematronValidateCommand struct {
	prog     string
	stdin    io.Reader
	stderr   io.Writer
	stdinTTY bool
}

func newSchematronValidateCommandWithIO(prog string, stdin io.Reader, stderr io.Writer, stdinTTY bool) *schematronValidateCommand {
	return &schematronValidateCommand{
		prog:     prog,
		stdin:    stdin,
		stderr:   stderr,
		stdinTTY: stdinTTY,
	}
}

func (c *schematronValidateCommand) runContext(ctx context.Context, args []string) int {
	cfg, files := c.parseArgs(args)
	if cfg == nil {
		c.showUsage()
		return ExitErr
	}

	if cfg.version {
		c.showVersion()
		return ExitOK
	}

	var inputs []schematronValidateInput
	switch {
	case len(files) > 0:
		for _, f := range files {
			inputs = append(inputs, schematronValidateInput{name: f})
		}
	case !c.stdinTTY:
		inputs = append(inputs, schematronValidateInput{name: "-", stdin: true})
	default:
		c.showUsage()
		return ExitErr
	}

	var t0 time.Time
	if cfg.timing {
		t0 = time.Now()
	}
	// Attach an ErrorHandler so fatal schema diagnostics (file/line/detail)
	// reach stderr before the summary error, rather than being discarded.
	ceh := &writerErrorHandler{w: c.stderr}
	schema, err := schematron.NewCompiler().Label(cfg.schemaFile).ErrorHandler(ceh).CompileFile(ctx, cfg.schemaFile)
	if cfg.timing {
		_, _ = fmt.Fprintf(c.stderr, "Compiling schema took %s\n", time.Since(t0))
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

func (c *schematronValidateCommand) showVersion() {
	_, _ = fmt.Fprintf(c.stderr, "%s: using helium (%s)\n", c.prog, commitID())
}

func (c *schematronValidateCommand) showUsage() {
	_, _ = fmt.Fprintf(c.stderr, `Usage : %s [options] SCHEMA [XMLfiles ...]
	Validate XML files against a Schematron schema
	--timing : print timing information to stderr
	--max-input-bytes N : cap bytes read per input (0 = unlimited)
	--max-depth N : cap element nesting depth (default 256, 0 = unlimited)
	--version : display the version of the XML library used
`, c.prog)
}

func (c *schematronValidateCommand) parseArgs(args []string) (*schematronValidateConfig, []string) {
	cfg := &schematronValidateConfig{maxInputBytes: DefaultMaxInputBytes, maxDepth: -1}
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

func (c *schematronValidateCommand) processInput(ctx context.Context, cfg *schematronValidateConfig, input schematronValidateInput, schema *schematron.Schema) int {
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
	err = schematron.NewValidator(schema).Label(input.name).ErrorHandler(handler).Validate(ctx, doc)
	if cfg.timing {
		_, _ = fmt.Fprintf(c.stderr, "Validating took %s\n", time.Since(t0))
	}
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s fails to validate\n", input.name)
		return ExitValidation
	}
	return ExitOK
}
