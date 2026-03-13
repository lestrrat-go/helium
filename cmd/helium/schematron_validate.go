package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/cliutil"
	"github.com/lestrrat-go/helium/schematron"
)

type schematronValidateConfig struct {
	schemaFile string
	timing     bool
	version    bool
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

func RunSchematronValidate(prog string, args []string) int {
	return newSchematronValidateCommand(prog).run(args)
}

func newSchematronValidateCommand(prog string) *schematronValidateCommand {
	return &schematronValidateCommand{
		prog:     prog,
		stdin:    os.Stdin,
		stderr:   os.Stderr,
		stdinTTY: cliutil.IsTty(os.Stdin.Fd()),
	}
}

func (c *schematronValidateCommand) run(args []string) int {
	ctx := context.Background()

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
	schema, err := schematron.CompileFile(ctx, cfg.schemaFile, schematron.WithSchemaFilename(cfg.schemaFile))
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
		if code != ExitOK {
			exitCode = code
		}
	}
	return exitCode
}

func (c *schematronValidateCommand) showVersion() {
	_, _ = fmt.Fprintf(c.stderr, "%s: using helium version %s\n", c.prog, helium.Version)
}

func (c *schematronValidateCommand) showUsage() {
	_, _ = fmt.Fprintf(c.stderr, `Usage : %s [options] SCHEMA [XMLfiles ...]
	Validate XML files against a Schematron schema
	--timing : print timing information to stderr
	--version : display the version of the XML library used
`, c.prog)
}

func (c *schematronValidateCommand) parseArgs(args []string) (*schematronValidateConfig, []string) {
	cfg := &schematronValidateConfig{}
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--version":
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

func (c *schematronValidateCommand) processInput(ctx context.Context, cfg *schematronValidateConfig, input schematronValidateInput, schema *schematron.Schema) int {
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
		p.SetBaseURI(input.name)
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
	err = schematron.Validate(ctx, doc, schema, schematron.WithFilename(input.name))
	if cfg.timing {
		_, _ = fmt.Fprintf(c.stderr, "Validating took %s\n", time.Since(t0))
	}
	if err != nil {
		_, _ = fmt.Fprint(c.stderr, err)
		return ExitValidation
	}
	return ExitOK
}
