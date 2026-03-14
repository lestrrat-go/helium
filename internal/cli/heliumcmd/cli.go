package heliumcmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/lestrrat-go/helium/internal/cliutil"
)

type ioContextKey struct{}

type ioContext struct {
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
	stdinTTY *bool
}

func WithIO(parent context.Context, stdin io.Reader, stdout, stderr io.Writer) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	cfg := getIOContext(parent)
	next := &ioContext{}
	if cfg != nil {
		*next = *cfg
	}
	next.stdin = stdin
	next.stdout = stdout
	next.stderr = stderr
	return context.WithValue(parent, ioContextKey{}, next)
}

func WithStdinTTY(parent context.Context, stdinTTY bool) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	cfg := getIOContext(parent)
	next := &ioContext{}
	if cfg != nil {
		*next = *cfg
	}
	next.stdinTTY = &stdinTTY
	return context.WithValue(parent, ioContextKey{}, next)
}

func getIOContext(ctx context.Context) *ioContext {
	if ctx == nil {
		return nil
	}
	cfg, _ := ctx.Value(ioContextKey{}).(*ioContext)
	return cfg
}

func resolveIO(ctx context.Context) (io.Reader, io.Writer, io.Writer, bool) {
	stdin := io.Reader(os.Stdin)
	stdout := io.Writer(os.Stdout)
	stderr := io.Writer(os.Stderr)
	stdinTTY := cliutil.IsTty(os.Stdin.Fd())

	cfg := getIOContext(ctx)
	if cfg == nil {
		return stdin, stdout, stderr, stdinTTY
	}
	if cfg.stdin != nil {
		stdin = cfg.stdin
	}
	if cfg.stdout != nil {
		stdout = cfg.stdout
	}
	if cfg.stderr != nil {
		stderr = cfg.stderr
	}
	if cfg.stdinTTY != nil {
		stdinTTY = *cfg.stdinTTY
	}
	return stdin, stdout, stderr, stdinTTY
}

func Execute(ctx context.Context, args []string) int {
	if ctx == nil {
		ctx = context.Background()
	}

	stdin, stdout, stderr, stdinTTY := resolveIO(ctx)
	if len(args) == 0 {
		showUsage(stderr)
		return ExitErr
	}

	switch args[0] {
	case "lint":
		return newCommandWithIO("helium lint", stdin, stdout, stderr, stdinTTY).runContext(ctx, args[1:])
	case "relaxng":
		return runRelaxNG(ctx, stderr, stdin, stdinTTY, args[1:])
	case "schematron":
		return runSchematron(ctx, stderr, stdin, stdinTTY, args[1:])
	case "xpath":
		return newXPathCommandWithIO("helium xpath", stdin, stdout, stderr, stdinTTY).runContext(ctx, args[1:])
	case "xsd":
		return runXSD(ctx, stderr, stdin, stdinTTY, args[1:])
	default:
		_, _ = fmt.Fprintf(stderr, "helium: unknown subcommand %q\n", args[0])
		showUsage(stderr)
		return ExitErr
	}
}

func runRelaxNG(ctx context.Context, stderr io.Writer, stdin io.Reader, stdinTTY bool, args []string) int {
	if len(args) == 0 {
		showRelaxNGUsage(stderr)
		return ExitErr
	}

	switch args[0] {
	case "validate":
		return newRelaxNGValidateCommandWithIO("helium relaxng validate", stdin, stderr, stdinTTY).runContext(ctx, args[1:])
	default:
		_, _ = fmt.Fprintf(stderr, "helium relaxng: unknown subcommand %q\n", args[0])
		showRelaxNGUsage(stderr)
		return ExitErr
	}
}

func runSchematron(ctx context.Context, stderr io.Writer, stdin io.Reader, stdinTTY bool, args []string) int {
	if len(args) == 0 {
		showSchematronUsage(stderr)
		return ExitErr
	}

	switch args[0] {
	case "validate":
		return newSchematronValidateCommandWithIO("helium schematron validate", stdin, stderr, stdinTTY).runContext(ctx, args[1:])
	default:
		_, _ = fmt.Fprintf(stderr, "helium schematron: unknown subcommand %q\n", args[0])
		showSchematronUsage(stderr)
		return ExitErr
	}
}

func runXSD(ctx context.Context, stderr io.Writer, stdin io.Reader, stdinTTY bool, args []string) int {
	if len(args) == 0 {
		showXSDUsage(stderr)
		return ExitErr
	}

	switch args[0] {
	case "validate":
		return newXSDValidateCommandWithIO("helium xsd validate", stdin, stderr, stdinTTY).runContext(ctx, args[1:])
	default:
		_, _ = fmt.Fprintf(stderr, "helium xsd: unknown subcommand %q\n", args[0])
		showXSDUsage(stderr)
		return ExitErr
	}
}

func showUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, `Usage: helium <command> [options]

Available commands:
  lint    Parse and lint XML documents
  relaxng RELAX NG operations
  schematron Schematron operations
  xpath   Evaluate XPath expressions
  xsd     XML Schema operations

Planned commands:
  xslt`)
}

func showXSDUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, `Usage: helium xsd <command> [options]

Available commands:
  validate    Validate XML documents against an XML Schema`)
}

func showRelaxNGUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, `Usage: helium relaxng <command> [options]

Available commands:
  validate    Validate XML documents against a RELAX NG schema`)
}

func showSchematronUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, `Usage: helium schematron <command> [options]

Available commands:
  validate    Validate XML documents against a Schematron schema`)
}
