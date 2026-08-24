package schematron

import (
	"errors"
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

// QueryBinding identifies the query language binding a Schematron schema is
// written in. ISO/IEC 19757-3 names the binding in the queryBinding
// attribute of the <schema> element and requires a processor that does not
// implement the named binding to report an error.
type QueryBinding int

const (
	// QueryBindingUnspecified is the zero value. It names no binding, and
	// leaves the choice to the schema's queryBinding attribute or to the
	// configured default.
	QueryBindingUnspecified QueryBinding = iota

	// QueryBindingXPath1 evaluates expressions with XPath 1.0. It covers the
	// queryBinding values "xslt", "xslt1", "xpath" and "xpath1", and is the
	// binding a schema with no queryBinding attribute gets.
	QueryBindingXPath1
)

// ErrUnsupportedQueryBinding is returned by [ParseQueryBinding], and by
// [Compiler.Compile]/[Compiler.CompileFile], when a schema names a query
// language binding this package does not implement. XSLT 2.0 and XPath 2.0
// bindings ("xslt2", "xpath2") are deliberately not implemented, and neither
// are the non-XPath bindings the standard mentions ("xquery", "stx",
// "exslt").
var ErrUnsupportedQueryBinding = errors.New("schematron: unsupported query language binding")

// String returns the canonical name of the binding.
func (b QueryBinding) String() string {
	switch b {
	case QueryBindingUnspecified:
		return "unspecified"
	case QueryBindingXPath1:
		return "xpath1"
	}
	return fmt.Sprintf("QueryBinding(%d)", int(b))
}

// supportedQueryBindings lists the accepted queryBinding values, in the order
// they appear in the "unsupported binding" error message.
var supportedQueryBindings = []string{"xslt", "xslt1", "xpath", "xpath1"}

// ParseQueryBinding maps a queryBinding attribute value to a [QueryBinding].
// The value is trimmed and matched without regard to case. An empty value
// names no binding and yields [QueryBindingUnspecified]; any other
// unrecognized value yields an error wrapping [ErrUnsupportedQueryBinding].
func ParseQueryBinding(s string) (QueryBinding, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return QueryBindingUnspecified, nil
	case "xslt", "xslt1", "xpath", "xpath1":
		return QueryBindingXPath1, nil
	}
	return QueryBindingUnspecified, unsupportedQueryBinding(s)
}

// newEngine returns the engine that evaluates expressions for this binding.
func (b QueryBinding) newEngine() engine {
	if b == QueryBindingXPath1 {
		return xpath1Engine{}
	}
	return nil
}

// resolveQueryBinding decides which query language binding a schema compiles
// with. A binding forced through [Compiler.QueryBinding] always wins, then
// the schema's own queryBinding attribute, then the fallback set through
// [Compiler.DefaultQueryBinding], and finally XPath 1.0.
func resolveQueryBinding(cfg *compileConfig, root *helium.Element) (QueryBinding, error) {
	b, err := pickQueryBinding(cfg, root)
	if err != nil {
		return QueryBindingUnspecified, err
	}
	if b.newEngine() == nil {
		return QueryBindingUnspecified, unsupportedQueryBinding(b.String())
	}
	return b, nil
}

func pickQueryBinding(cfg *compileConfig, root *helium.Element) (QueryBinding, error) {
	if cfg != nil && cfg.queryBinding != QueryBindingUnspecified {
		return cfg.queryBinding, nil
	}

	// The queryBinding attribute is unqualified, like every other Schematron
	// structural attribute.
	if raw := getStructuralAttr(root, "queryBinding"); strings.TrimSpace(raw) != "" {
		return ParseQueryBinding(raw)
	}

	if cfg != nil && cfg.defaultQueryBinding != QueryBindingUnspecified {
		return cfg.defaultQueryBinding, nil
	}
	return QueryBindingXPath1, nil
}

func unsupportedQueryBinding(name string) error {
	return fmt.Errorf("%w %q (supported: %s)", ErrUnsupportedQueryBinding, name, strings.Join(supportedQueryBindings, ", "))
}
