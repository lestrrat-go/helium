package schematron

import (
	"errors"
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

// QueryBinding identifies the query language binding a Schematron schema is
// written in. ISO/IEC 19757-3 names the binding in the queryBinding attribute
// of the <schema> element, and clause 6.4 requires an implementation that
// does not support the named binding to fail with an error.
//
// Only the bindings the standard defines normatively are recognized, and
// only those this package implements are accepted. The names the standard
// reserves without defining ("xpath", "xquery", "exslt", "stx") carry no
// agreed meaning, so a schema naming one is refused rather than guessed at.
type QueryBinding int

const (
	// QueryBindingUnspecified is the zero value. It names no binding, and
	// leaves the choice to the schema's queryBinding attribute or to the
	// configured default.
	QueryBindingUnspecified QueryBinding = iota

	// QueryBindingXPath1 evaluates expressions with XPath 1.0 as extended by
	// XSLT 1.0. It is the default query language binding of ISO/IEC 19757-3
	// Annex C, which covers a schema with no queryBinding attribute and one
	// whose value is "xslt".
	QueryBindingXPath1

	// QueryBindingXPath3 evaluates expressions with XPath 3.1. It covers the
	// XSLT 3.0 binding of ISO/IEC 19757-3:2020 Annex J ("xslt3"), whose
	// query language is XPath 3.1, and the XPath 3.0 binding of Annex K
	// ("xpath3"), which XPath 3.1 is a superset of.
	QueryBindingXPath3
)

// ErrUnsupportedQueryBinding is returned by [ParseQueryBinding], and by
// [Compiler.Compile]/[Compiler.CompileFile], when a schema names a query
// language binding this package does not implement. The XSLT 2.0 and XPath
// 2.0 bindings (ISO/IEC 19757-3 Annexes H and I) are deliberately not
// implemented, and neither are the reserved names the standard leaves
// undefined.
var ErrUnsupportedQueryBinding = errors.New("schematron: unsupported query language binding")

// String returns the canonical name of the binding.
func (b QueryBinding) String() string {
	switch b {
	case QueryBindingUnspecified:
		return "unspecified"
	case QueryBindingXPath1:
		return "xpath1"
	case QueryBindingXPath3:
		return "xpath3"
	}
	return fmt.Sprintf("QueryBinding(%d)", int(b))
}

// supportedQueryBindings lists the accepted queryBinding values, in the order
// they appear in the "unsupported binding" error message.
var supportedQueryBindings = []string{"xslt", "xslt3", "xpath3"}

// ParseQueryBinding maps a queryBinding attribute value to a [QueryBinding].
//
// The value is compared without regard to case, which ISO/IEC 19757-3 states
// for each binding it defines ("in any mix of upper and lower case
// letters"), and surrounding whitespace is trimmed, which the attribute's
// declared xsd:token type collapses anyway. An empty value names no binding
// and yields [QueryBindingUnspecified]; any value outside the accepted set
// yields an error wrapping [ErrUnsupportedQueryBinding].
func ParseQueryBinding(s string) (QueryBinding, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return QueryBindingUnspecified, nil
	case "xslt":
		return QueryBindingXPath1, nil
	case "xslt3", "xpath3":
		return QueryBindingXPath3, nil
	}
	return QueryBindingUnspecified, unsupportedQueryBinding(s)
}

// newEngine returns the engine that evaluates expressions for this binding.
func (b QueryBinding) newEngine() engine {
	switch b {
	case QueryBindingXPath1:
		return xpath1Engine{}
	case QueryBindingXPath3:
		return xpath3Engine{}
	}
	return nil
}

// resolveQueryBinding decides which query language binding a schema compiles
// with. A binding forced through [Compiler.QueryBinding] always wins, then
// the schema's own queryBinding attribute, then the fallback set through
// [Compiler.DefaultQueryBinding], and finally XPath 1.0, the default binding
// of ISO/IEC 19757-3 Annex C.
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
