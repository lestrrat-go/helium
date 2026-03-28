package heliumcmd

import (
	"context"
	"fmt"
	"io"
	"os"
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

type xsdValidateConfig struct {
	schemaFile string
	timing     bool
	version    bool
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
	schema, err := xsd.NewCompiler().CompileFile(ctx, cfg.schemaFile)
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

func (c *xsdValidateCommand) showVersion() {
	_, _ = fmt.Fprintf(c.stderr, "%s: using helium (%s)\n", c.prog, commitID())
}

func (c *xsdValidateCommand) showUsage() {
	_, _ = fmt.Fprintf(c.stderr, `Usage : %s [options] SCHEMA [XMLfiles ...]
	Validate XML files against an XML Schema
	--timing : print timing information to stderr
	--version : display the version of the XML library used
`, c.prog)
}

func (c *xsdValidateCommand) parseArgs(args []string) (*xsdValidateConfig, []string) {
	cfg := &xsdValidateConfig{}
	var positional []string

	for i := range args {
		arg := args[i]
		switch arg {
		case flagVersion:
			cfg.version = true
		case "--timing":
			cfg.timing = true
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
		buf, err = io.ReadAll(c.stdin)
	} else {
		buf, err = os.ReadFile(input.name)
	}
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
		return ExitReadFile
	}

	var t0 time.Time
	if cfg.timing {
		t0 = time.Now()
	}

	p := helium.NewParser()
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
