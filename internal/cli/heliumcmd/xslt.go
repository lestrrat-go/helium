package heliumcmd

import (
	"context"
	"fmt"
	"io"
	"os"
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
	ssBuf, err := os.ReadFile(cfg.stylesheetFile)
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: failed to read stylesheet: %s\n", c.prog, err)
		return ExitReadFile
	}

	var t0 time.Time
	if cfg.timing {
		t0 = time.Now()
	}

	ssDoc, err := helium.NewParser().
		LoadExternalDTD(true).
		SubstituteEntities(true).
		BaseURI(cfg.stylesheetFile).
		Parse(ctx, ssBuf)
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: failed to parse stylesheet: %s\n", c.prog, err)
		return ExitXSLT
	}

	ss, err := xslt3.NewCompiler().
		BaseURI(cfg.stylesheetFile).
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
	if cfg.outputFile != "" {
		f, err := os.Create(cfg.outputFile)
		if err != nil {
			_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
			return ExitErr
		}
		defer f.Close() //nolint:errcheck
		out = f
	}

	exitCode := ExitOK
	for _, input := range inputs {
		code := c.processInput(ctx, cfg, input, ss, params, out)
		exitCode = mergeExitCode(exitCode, code)
	}
	return exitCode
}

func (c *xsltCommand) buildParams(ctx context.Context, cfg *xsltConfig) (*xslt3.Parameters, error) {
	if len(cfg.params) == 0 {
		return nil, nil
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
	--timing         : print timing information to stderr
	--version        : display the version of the XML library used
`, c.prog)
}

func (c *xsltCommand) parseArgs(args []string) (*xsltConfig, []string) {
	cfg := &xsltConfig{}
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--version":
			cfg.version = true
		case "--timing":
			cfg.timing = true
		case "--noout":
			cfg.noout = true
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
		_, _ = fmt.Fprintf(c.stderr, "%s: stylesheet is required\n", c.prog)
		return nil, nil
	}

	cfg.stylesheetFile = positional[0]
	return cfg, positional[1:]
}
