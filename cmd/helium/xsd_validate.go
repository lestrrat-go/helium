package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/cliutil"
	"github.com/lestrrat-go/helium/xsd"
)

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

func RunXSDValidate(prog string, args []string) int {
	return newXSDValidateCommand(prog).run(args)
}

func newXSDValidateCommand(prog string) *xsdValidateCommand {
	return &xsdValidateCommand{
		prog:     prog,
		stdin:    os.Stdin,
		stderr:   os.Stderr,
		stdinTTY: cliutil.IsTty(os.Stdin.Fd()),
	}
}

func (c *xsdValidateCommand) run(args []string) int {
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
	schema, err := xsd.CompileFile(ctx, cfg.schemaFile)
	if cfg.timing {
		fmt.Fprintf(c.stderr, "Compiling schema took %s\n", time.Since(t0))
	}
	if err != nil {
		fmt.Fprintf(c.stderr, "%s: failed to compile schema: %s\n", c.prog, err)
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

func (c *xsdValidateCommand) showVersion() {
	fmt.Fprintf(c.stderr, "%s: using helium version %s\n", c.prog, helium.Version)
}

func (c *xsdValidateCommand) showUsage() {
	fmt.Fprintf(c.stderr, `Usage : %s [options] XMLfiles ...
	Validate XML files against an XML Schema
	--schema FILE : validate against FILE
	--timing : print timing information to stderr
	--version : display the version of the XML library used
`, c.prog)
}

func (c *xsdValidateCommand) parseArgs(args []string) (*xsdValidateConfig, []string) {
	cfg := &xsdValidateConfig{}
	var files []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--version":
			cfg.version = true
		case "--timing":
			cfg.timing = true
		case "--schema":
			i++
			if i >= len(args) {
				fmt.Fprintf(c.stderr, "%s: --schema requires an argument\n", c.prog)
				return nil, nil
			}
			cfg.schemaFile = args[i] //nolint:gosec // bounds checked above
		default:
			if len(arg) > 0 && arg[0] == '-' {
				fmt.Fprintf(c.stderr, "%s: unrecognized option %s\n", c.prog, arg)
				return nil, nil
			}
			files = append(files, arg)
		}
	}

	if !cfg.version && cfg.schemaFile == "" {
		fmt.Fprintf(c.stderr, "%s: --schema is required\n", c.prog)
		return nil, nil
	}

	return cfg, files
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
		fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
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
		fmt.Fprintf(c.stderr, "Parsing took %s\n", time.Since(t0))
	}
	if err != nil {
		fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
		return ExitErr
	}

	if cfg.timing {
		t0 = time.Now()
	}
	err = xsd.Validate(ctx, doc, schema)
	if cfg.timing {
		fmt.Fprintf(c.stderr, "Validating took %s\n", time.Since(t0))
	}
	if err != nil {
		fmt.Fprint(c.stderr, err)
		return ExitValidation
	}
	return ExitOK
}
