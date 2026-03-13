package main

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
	"github.com/lestrrat-go/helium/internal/cliutil"
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
	parseOptions helium.ParseOption

	doXInclude bool
	c14nMode   int
	schemaFile string
	xpathExpr  string
	catalogs   bool
	noCatalogs bool
	pathDirs   string

	noout      bool
	format     bool
	outputFile string
	encode     string
	pretty     int

	quiet   bool
	timing  bool
	dropdtd bool
	repeat  int

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

func Run(prog string, args []string) int {
	return newCommand(prog).run(args)
}

func newCommand(prog string) *command {
	return &command{
		prog:     prog,
		stdin:    os.Stdin,
		stdout:   os.Stdout,
		stderr:   os.Stderr,
		stdinTTY: cliutil.IsTty(os.Stdin.Fd()),
	}
}

func (c *command) run(args []string) int {
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

	if cfg.pretty >= 1 {
		cfg.format = true
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

	var cat *catalog.Catalog
	if cfg.catalogs && !cfg.noCatalogs {
		var err error
		cat, err = c.loadCatalogFromEnv(ctx)
		if err != nil {
			fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
		}
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
			fmt.Fprintf(c.stderr, "Compiling schema took %s\n", time.Since(t0))
		}
		if err != nil {
			fmt.Fprintf(c.stderr, "%s\n", err)
			return ExitSchemaComp
		}
	}

	var out io.Writer = c.stdout
	if cfg.outputFile != "" {
		f, err := os.Create(cfg.outputFile) //nolint:gosec // CLI output path is user supplied
		if err != nil {
			fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
			return ExitErr
		}
		defer func() { _ = f.Close() }()
		out = f
	}

	exitCode := ExitOK
	for _, input := range inputs {
		code := c.processInput(ctx, cfg, input, cat, schema, out)
		if code != ExitOK {
			exitCode = code
		}
	}
	return exitCode
}

func (c *command) showVersion() {
	fmt.Fprintf(c.stderr, "%s: using helium version %s\n", c.prog, helium.Version)
}

func (c *command) showUsage() {
	fmt.Fprintf(c.stderr, `Usage : %s [options] XMLfiles ...
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
	--huge : remove any internal arbitrary parser limits
	--noenc : ignore any encoding specified inside the document
	--noxincludenode : do not generate XInclude START/END nodes
	--nofixup-base-uris : do not fixup xml:base URIs in XInclude
	--noout : do not print the result tree
	--format : reformat/reindent the output
	--pretty LEVEL : pretty-print the output (0=none, 1=format, 2=format+attrs)
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
`, c.prog)
}

func (c *command) parseArgs(args []string) (*config, []string) {
	cfg := &config{
		pretty: -1,
		repeat: 1,
	}
	var files []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && len(arg) > 1 {
			arg = "-" + arg
		}

		switch arg {
		case "--version":
			cfg.version = true
		case "--recover":
			cfg.parseOptions.Set(helium.ParseRecover)
		case "--noent":
			cfg.parseOptions.Set(helium.ParseNoEnt)
		case "--loaddtd":
			cfg.parseOptions.Set(helium.ParseDTDLoad)
		case "--dtdattr":
			cfg.parseOptions.Set(helium.ParseDTDAttr)
			cfg.parseOptions.Set(helium.ParseDTDLoad)
		case "--valid":
			cfg.parseOptions.Set(helium.ParseDTDValid)
			cfg.parseOptions.Set(helium.ParseDTDLoad)
		case "--nowarning":
			cfg.parseOptions.Set(helium.ParseNoWarning)
		case "--pedantic":
			cfg.parseOptions.Set(helium.ParsePedantic)
		case "--noblanks":
			cfg.parseOptions.Set(helium.ParseNoBlanks)
		case "--nsclean":
			cfg.parseOptions.Set(helium.ParseNsClean)
		case "--nocdata":
			cfg.parseOptions.Set(helium.ParseNoCDATA)
		case "--nonet":
			cfg.parseOptions.Set(helium.ParseNoNet)
		case "--huge":
			cfg.parseOptions.Set(helium.ParseHuge)
		case "--noenc":
			cfg.parseOptions.Set(helium.ParseIgnoreEnc)
		case "--noxincludenode":
			cfg.parseOptions.Set(helium.ParseNoXIncNode)
		case "--nofixup-base-uris":
			cfg.parseOptions.Set(helium.ParseNoBaseFix)
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
			cfg.parseOptions.Set(helium.ParseXInclude)
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
				fmt.Fprintf(c.stderr, "%s: --schema requires an argument\n", c.prog)
				return nil, nil
			}
			cfg.schemaFile = args[i] //nolint:gosec // bounds checked above
		case "--xpath":
			i++
			if i >= len(args) {
				fmt.Fprintf(c.stderr, "%s: --xpath requires an argument\n", c.prog)
				return nil, nil
			}
			cfg.xpathExpr = args[i] //nolint:gosec // bounds checked above
			cfg.noout = true
		case "--output":
			i++
			if i >= len(args) {
				fmt.Fprintf(c.stderr, "%s: --output requires an argument\n", c.prog)
				return nil, nil
			}
			cfg.outputFile = args[i] //nolint:gosec // bounds checked above
		case "--encode":
			i++
			if i >= len(args) {
				fmt.Fprintf(c.stderr, "%s: --encode requires an argument\n", c.prog)
				return nil, nil
			}
			cfg.encode = args[i] //nolint:gosec // bounds checked above
		case "--pretty":
			i++
			if i >= len(args) {
				fmt.Fprintf(c.stderr, "%s: --pretty requires an argument\n", c.prog)
				return nil, nil
			}
			n, err := strconv.Atoi(args[i]) //nolint:gosec // bounds checked above
			if err != nil {
				fmt.Fprintf(c.stderr, "%s: --pretty: invalid argument %q\n", c.prog, args[i]) //nolint:gosec // bounds checked above
				return nil, nil
			}
			cfg.pretty = n
		case "--path":
			i++
			if i >= len(args) {
				fmt.Fprintf(c.stderr, "%s: --path requires an argument\n", c.prog)
				return nil, nil
			}
			cfg.pathDirs = args[i] //nolint:gosec // bounds checked above
		case "--repeat":
			i++
			if i >= len(args) {
				fmt.Fprintf(c.stderr, "%s: --repeat requires an argument\n", c.prog)
				return nil, nil
			}
			n, err := strconv.Atoi(args[i]) //nolint:gosec // bounds checked above
			if err != nil || n < 1 {
				fmt.Fprintf(c.stderr, "%s: --repeat: invalid argument %q\n", c.prog, args[i]) //nolint:gosec // bounds checked above
				return nil, nil
			}
			cfg.repeat = n
		default:
			if strings.HasPrefix(arg, "--") {
				fmt.Fprintf(c.stderr, "%s: unrecognized option %s\n", c.prog, arg) //nolint:gosec // CLI diagnostic output
				return nil, nil
			}
			files = append(files, arg)
		}
	}

	return cfg, files
}

func (c *command) loadCatalogFromEnv(ctx context.Context) (*catalog.Catalog, error) {
	envFiles := os.Getenv("XML_CATALOG_FILES")
	if envFiles == "" {
		return nil, nil
	}
	for _, f := range strings.Split(envFiles, " ") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		cat, err := catalog.Load(ctx, f)
		if err != nil {
			fmt.Fprintf(c.stderr, "%s: failed to load catalog %s: %s\n", c.prog, f, err)
			continue
		}
		return cat, nil
	}
	return nil, nil
}

func (c *command) compileSchema(ctx context.Context, cfg *config) (*xsd.Schema, error) {
	schema, err := xsd.CompileFile(ctx, cfg.schemaFile)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to compile schema: %w", c.prog, err)
	}
	return schema, nil
}

func (c *command) processInput(ctx context.Context, cfg *config, input namedInput, cat *catalog.Catalog, schema *xsd.Schema, out io.Writer) int {
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

	var doc *helium.Document
	for rep := 0; rep < cfg.repeat; rep++ {
		var t0 time.Time
		if cfg.timing {
			t0 = time.Now()
		}

		p := helium.NewParser()
		p.SetOption(cfg.parseOptions)
		if !input.stdin {
			p.SetBaseURI(input.name)
		}
		if cat != nil {
			p.SetCatalog(cat)
		}

		doc, err = p.Parse(ctx, buf)
		if cfg.timing {
			fmt.Fprintf(c.stderr, "Parsing took %s\n", time.Since(t0))
		}
		if err != nil {
			fmt.Fprintf(c.stderr, "%s\n", err)
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
		var xiOpts []xinclude.Option
		xiOpts = append(xiOpts, xinclude.WithParseFlags(cfg.parseOptions))
		if !input.stdin {
			xiOpts = append(xiOpts, xinclude.WithBaseURI(input.name))
		}
		_, xiErr := xinclude.Process(ctx, doc, xiOpts...)
		if cfg.timing {
			fmt.Fprintf(c.stderr, "XInclude took %s\n", time.Since(t0))
		}
		if xiErr != nil {
			fmt.Fprintf(c.stderr, "%s\n", xiErr)
		}
	}

	if schema != nil {
		var t0 time.Time
		if cfg.timing {
			t0 = time.Now()
		}
		err := xsd.Validate(ctx, doc, schema)
		if cfg.timing {
			fmt.Fprintf(c.stderr, "Validating took %s\n", time.Since(t0))
		}
		if err != nil {
			fmt.Fprint(c.stderr, err)
			return ExitValidation
		}
	}

	if cfg.parseOptions.IsSet(helium.ParseDTDValid) && err != nil {
		return ExitValidation
	}

	if cfg.xpathExpr != "" {
		return c.evalXPath(ctx, cfg, doc, out)
	}

	if cfg.noout {
		return ExitOK
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
		cErr := c14n.Canonicalize(out, doc, mode, c14n.WithComments())
		if cfg.timing {
			fmt.Fprintf(c.stderr, "Saving took %s\n", time.Since(t0))
		}
		if cErr != nil {
			fmt.Fprintf(c.stderr, "%s\n", cErr)
			return ExitErr
		}
		return ExitOK
	}

	var opts []helium.WriteOption
	if cfg.format {
		opts = append(opts, helium.WithFormat())
	}
	opts = append(opts, helium.WithIndentString("  "))
	if cfg.dropdtd {
		opts = append(opts, helium.WithSkipDTD())
	}
	d := helium.NewWriter(opts...)
	if dErr := d.WriteDoc(out, doc); dErr != nil {
		if cfg.timing {
			fmt.Fprintf(c.stderr, "Saving took %s\n", time.Since(t0))
		}
		fmt.Fprintf(c.stderr, "%s\n", dErr)
		return ExitErr
	}
	if cfg.timing {
		fmt.Fprintf(c.stderr, "Saving took %s\n", time.Since(t0))
	}
	return ExitOK
}

func (c *command) evalXPath(ctx context.Context, cfg *config, doc *helium.Document, out io.Writer) int {
	expr, err := xpath1.Compile(cfg.xpathExpr)
	if err != nil {
		fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
		return ExitXPath
	}

	res, err := expr.Evaluate(ctx, doc)
	if err != nil {
		fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
		return ExitXPath
	}

	switch res.Type {
	case xpath1.NodeSetResult:
		for _, n := range res.NodeSet {
			switch n.Type() {
			case helium.AttributeNode:
				attr := n.(*helium.Attribute) //nolint:forcetypeassert // node type checked above
				fmt.Fprintf(out, " %s=%q\n", attr.Name(), attr.Value())
			case helium.NamespaceDeclNode, helium.NamespaceNode:
				ns, ok := n.(interface {
					Prefix() string
					URI() string
				})
				if !ok {
					fmt.Fprintf(c.stderr, "%s: unexpected namespace node type %T\n", c.prog, n)
					return ExitErr
				}
				if ns.Prefix() == "" {
					fmt.Fprintf(out, " xmlns=%q\n", ns.URI())
				} else {
					fmt.Fprintf(out, " xmlns:%s=%q\n", ns.Prefix(), ns.URI())
				}
			default:
				d := helium.NewWriter()
				if err := d.WriteNode(out, n); err != nil {
					fmt.Fprintf(c.stderr, "%s: %s\n", c.prog, err)
					return ExitErr
				}
				fmt.Fprintln(out)
			}
		}
	case xpath1.BooleanResult:
		if res.Bool {
			fmt.Fprintln(out, "true")
		} else {
			fmt.Fprintln(out, "false")
		}
	case xpath1.NumberResult:
		fmt.Fprintf(out, "%g\n", res.Number)
	default:
		fmt.Fprintln(out, res.String)
	}

	return ExitOK
}
