package heliumcmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
	"github.com/lestrrat-go/helium/xpath3"
)

type xpathConfig struct {
	engine  string
	expr    string
	version bool
}

type xpathCommand struct {
	prog     string
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
	stdinTTY bool
}

func newXPathCommandWithIO(prog string, stdin io.Reader, stdout, stderr io.Writer, stdinTTY bool) *xpathCommand {
	return &xpathCommand{
		prog:     prog,
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
		stdinTTY: stdinTTY,
	}
}

func (c *xpathCommand) runContext(ctx context.Context, args []string) int {
	cfg, files := c.parseArgs(args)
	if cfg == nil {
		c.showUsage()
		return ExitErr
	}

	if cfg.version {
		c.showVersion()
		return ExitOK
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

	exitCode := ExitOK
	for _, input := range inputs {
		code := c.processInput(ctx, cfg, input)
		exitCode = mergeExitCode(exitCode, code)
	}
	return exitCode
}

func (c *xpathCommand) showVersion() {
	_, _ = fmt.Fprintf(c.stderr, "%s: using helium version %s\n", c.prog, helium.Version)
}

func (c *xpathCommand) showUsage() {
	_, _ = fmt.Fprintf(c.stderr, `Usage : %s [options] EXPR [XMLfiles ...]
	Evaluate an XPath expression against XML input
	--engine N : XPath engine version (1 or 3, default 3)
	--version : display the version of the XML library used
`, c.prog)
}

func (c *xpathCommand) parseArgs(args []string) (*xpathConfig, []string) {
	cfg := &xpathConfig{engine: "3"}
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--version":
			cfg.version = true
		case "--engine":
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintf(c.stderr, "%s: --engine requires an argument\n", c.prog)
				return nil, nil
			}
			cfg.engine = args[i] //nolint:gosec // bounds checked above
		default:
			if strings.HasPrefix(arg, "-") {
				_, _ = fmt.Fprintf(c.stderr, "%s: unrecognized option %s\n", c.prog, arg)
				return nil, nil
			}
			positional = append(positional, arg)
		}
	}

	if cfg.version {
		return cfg, positional
	}

	if cfg.engine != "1" && cfg.engine != "3" {
		_, _ = fmt.Fprintf(c.stderr, "%s: unsupported engine %q\n", c.prog, cfg.engine)
		return nil, nil
	}

	if len(positional) == 0 {
		_, _ = fmt.Fprintf(c.stderr, "%s: expression is required\n", c.prog)
		return nil, nil
	}
	if positional[0] == "" {
		_, _ = fmt.Fprintf(c.stderr, "%s: expression must not be empty\n", c.prog)
		return nil, nil
	}

	cfg.expr = positional[0]
	return cfg, positional[1:]
}

func (c *xpathCommand) processInput(ctx context.Context, cfg *xpathConfig, input namedInput) int {
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

	p := helium.NewParser()
	if !input.stdin {
		p.SetBaseURI(input.name)
	}

	doc, err := p.Parse(ctx, buf)
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
		return ExitErr
	}

	if cfg.engine == "1" {
		return c.evalXPath1(ctx, cfg, doc)
	}
	return c.evalXPath3(ctx, cfg, doc)
}

func (c *xpathCommand) evalXPath1(ctx context.Context, cfg *xpathConfig, doc *helium.Document) int {
	expr, err := xpath1.Compile(cfg.expr)
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
		return ExitXPath
	}

	res, err := expr.Evaluate(ctx, doc)
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
		return ExitXPath
	}

	return c.printXPath1Result(res)
}

func (c *xpathCommand) printXPath1Result(res *xpath1.Result) int {
	switch res.Type {
	case xpath1.NodeSetResult:
		for _, n := range res.NodeSet {
			if code := c.printXPathNode(n); code != ExitOK {
				return code
			}
		}
	case xpath1.BooleanResult:
		if res.Bool {
			_, _ = fmt.Fprintln(c.stdout, "true")
		} else {
			_, _ = fmt.Fprintln(c.stdout, "false")
		}
	case xpath1.NumberResult:
		_, _ = fmt.Fprintf(c.stdout, "%g\n", res.Number)
	default:
		_, _ = fmt.Fprintln(c.stdout, res.String)
	}
	return ExitOK
}

func (c *xpathCommand) evalXPath3(ctx context.Context, cfg *xpathConfig, doc *helium.Document) int {
	expr, err := xpath3.NewCompiler().Compile(cfg.expr)
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
		return ExitXPath
	}

	res, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, expr, doc)
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
		return ExitXPath
	}

	if nodes, err := res.Nodes(); err == nil {
		for _, n := range nodes {
			if code := c.printXPathNode(n); code != ExitOK {
				return code
			}
		}
		return ExitOK
	}

	if b, ok := res.IsBoolean(); ok {
		if b {
			_, _ = fmt.Fprintln(c.stdout, "true")
		} else {
			_, _ = fmt.Fprintln(c.stdout, "false")
		}
		return ExitOK
	}

	if n, ok := res.IsNumber(); ok {
		_, _ = fmt.Fprintf(c.stdout, "%g\n", n)
		return ExitOK
	}

	if s, ok := res.IsString(); ok {
		_, _ = fmt.Fprintln(c.stdout, s)
		return ExitOK
	}

	for item := range res.Sequence().Items() {
		switch v := item.(type) {
		case xpath3.NodeItem:
			if code := c.printXPathNode(v.Node); code != ExitOK {
				return code
			}
		case xpath3.AtomicValue:
			_, _ = fmt.Fprintln(c.stdout, formatXPath3Atomic(v))
		default:
			_, _ = fmt.Fprintln(c.stdout, item)
		}
	}

	return ExitOK
}

func (c *xpathCommand) printXPathNode(n helium.Node) int {
	switch n.Type() {
	case helium.AttributeNode:
		attr := n.(*helium.Attribute) //nolint:forcetypeassert // node type checked above
		_, _ = fmt.Fprintf(c.stdout, " %s=%q\n", attr.Name(), attr.Value())
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
			_, _ = fmt.Fprintf(c.stdout, " xmlns=%q\n", ns.URI())
		} else {
			_, _ = fmt.Fprintf(c.stdout, " xmlns:%s=%q\n", ns.Prefix(), ns.URI())
		}
	default:
		d := helium.NewWriter()
		if err := d.WriteNode(c.stdout, n); err != nil {
			_, _ = fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
			return ExitErr
		}
		_, _ = fmt.Fprintln(c.stdout)
	}
	return ExitOK
}

func formatXPath3Atomic(v xpath3.AtomicValue) string {
	switch v.TypeName {
	case xpath3.TypeBoolean:
		if v.BooleanVal() {
			return "true"
		}
		return "false"
	case xpath3.TypeString, xpath3.TypeAnyURI, xpath3.TypeUntypedAtomic,
		xpath3.TypeNormalizedString, xpath3.TypeToken, xpath3.TypeLanguage,
		xpath3.TypeName, xpath3.TypeNCName, xpath3.TypeNMTOKEN,
		xpath3.TypeENTITY, xpath3.TypeID, xpath3.TypeIDREF:
		return v.StringVal()
	case xpath3.TypeDecimal:
		return xpath3.DecimalToString(v.BigRat())
	case xpath3.TypeDouble, xpath3.TypeFloat:
		return fmt.Sprintf("%g", v.ToFloat64())
	}

	if v.IsNumeric() {
		return v.BigInt().String()
	}
	return v.String()
}
