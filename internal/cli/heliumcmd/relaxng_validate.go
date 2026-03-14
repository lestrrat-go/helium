package heliumcmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
)

type relaxNGValidateConfig struct {
	schemaFile string
	timing     bool
	version    bool
}

type relaxNGValidateInput struct {
	name  string
	stdin bool
}

type relaxNGValidateCommand struct {
	prog     string
	stdin    io.Reader
	stderr   io.Writer
	stdinTTY bool
}

func newRelaxNGValidateCommandWithIO(prog string, stdin io.Reader, stderr io.Writer, stdinTTY bool) *relaxNGValidateCommand {
	return &relaxNGValidateCommand{
		prog:     prog,
		stdin:    stdin,
		stderr:   stderr,
		stdinTTY: stdinTTY,
	}
}

func (c *relaxNGValidateCommand) runContext(ctx context.Context, args []string) int {
	cfg, files := c.parseArgs(args)
	if cfg == nil {
		c.showUsage()
		return ExitErr
	}

	if cfg.version {
		c.showVersion()
		return ExitOK
	}

	var inputs []relaxNGValidateInput
	switch {
	case len(files) > 0:
		for _, f := range files {
			inputs = append(inputs, relaxNGValidateInput{name: f})
		}
	case !c.stdinTTY:
		inputs = append(inputs, relaxNGValidateInput{name: "-", stdin: true})
	default:
		c.showUsage()
		return ExitErr
	}

	var t0 time.Time
	if cfg.timing {
		t0 = time.Now()
	}
	grammar, err := relaxng.CompileFile(ctx, cfg.schemaFile)
	if cfg.timing {
		_, _ = fmt.Fprintf(c.stderr, "Compiling schema took %s\n", time.Since(t0))
	}
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: failed to compile schema: %s\n", c.prog, err)
		return ExitSchemaComp
	}

	exitCode := ExitOK
	for _, input := range inputs {
		code := c.processInput(ctx, cfg, input, grammar)
		exitCode = mergeExitCode(exitCode, code)
	}
	return exitCode
}

func (c *relaxNGValidateCommand) showVersion() {
	_, _ = fmt.Fprintf(c.stderr, "%s: using helium version %s\n", c.prog, helium.Version)
}

func (c *relaxNGValidateCommand) showUsage() {
	_, _ = fmt.Fprintf(c.stderr, `Usage : %s [options] SCHEMA [XMLfiles ...]
	Validate XML files against a RELAX NG schema
	--timing : print timing information to stderr
	--version : display the version of the XML library used
`, c.prog)
}

func (c *relaxNGValidateCommand) parseArgs(args []string) (*relaxNGValidateConfig, []string) {
	cfg := &relaxNGValidateConfig{}
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

func (c *relaxNGValidateCommand) processInput(ctx context.Context, cfg *relaxNGValidateConfig, input relaxNGValidateInput, grammar *relaxng.Grammar) int {
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
	err = relaxng.Validate(doc, grammar)
	if cfg.timing {
		_, _ = fmt.Fprintf(c.stderr, "Validating took %s\n", time.Since(t0))
	}
	if err != nil {
		_, _ = fmt.Fprint(c.stderr, err)
		return ExitValidation
	}
	return ExitOK
}
