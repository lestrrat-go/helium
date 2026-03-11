package xpath3

import (
	"fmt"
	"math/big"

	"github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

func evalLiteral(e LiteralExpr) (Sequence, error) {
	switch v := e.Value.(type) {
	case string:
		return SingleString(v), nil
	case *big.Int:
		return SingleIntegerBig(v), nil
	case *big.Rat:
		return SingleDecimal(v), nil
	case float64:
		return SingleDouble(v), nil
	}
	return nil, fmt.Errorf("%w: literal %T", ErrUnsupportedExpr, e.Value)
}

func evalVariable(ec *evalContext, e VariableExpr) (Sequence, error) {
	if ec.vars != nil {
		if v, ok := ec.vars[e.Name]; ok {
			return v, nil
		}
	}
	return nil, fmt.Errorf("%w: $%s", ErrUndefinedVariable, e.Name)
}

func evalSequenceExpr(ec *evalContext, e SequenceExpr) (Sequence, error) {
	if len(e.Items) == 0 {
		return nil, nil
	}
	var result Sequence
	for _, item := range e.Items {
		seq, err := eval(ec, item)
		if err != nil {
			return nil, err
		}
		result = append(result, seq...)
	}
	return result, nil
}

func evalLocationPath(ec *evalContext, lp *LocationPath) (Sequence, error) {
	var nodes []helium.Node

	if lp.Absolute {
		if ec.node == nil {
			return nil, &XPathError{Code: "XPDY0002", Message: "context item is absent"}
		}
		root := ixpath.DocumentRoot(ec.node)
		nodes = []helium.Node{root}
	} else {
		if ec.node == nil {
			return nil, &XPathError{Code: "XPDY0002", Message: "context item is absent"}
		}
		nodes = []helium.Node{ec.node}
	}

	var err error
	for _, step := range lp.Steps {
		if len(step.Predicates) > 0 {
			nodes, err = evalStepWithPredicates(ec, nodes, step)
		} else {
			nodes, err = evalStepNoPredicates(ec, nodes, step)
		}
		if err != nil {
			return nil, err
		}
	}

	result := make(Sequence, len(nodes))
	for i, n := range nodes {
		result[i] = NodeItem{Node: n}
	}
	return result, nil
}

func evalStepWithPredicates(ec *evalContext, nodes []helium.Node, step Step) ([]helium.Node, error) {
	var allFiltered []helium.Node
	for _, n := range nodes {
		candidates, err := ixpath.TraverseAxis(step.Axis, n, ec.maxNodes)
		if err != nil {
			return nil, err
		}
		if err := ec.countOps(len(candidates)); err != nil {
			return nil, err
		}
		matched := filterByNodeTest(candidates, step.NodeTest, step.Axis, ec)
		for _, pred := range step.Predicates {
			matched, err = applyPredicate(ec, matched, pred)
			if err != nil {
				return nil, err
			}
		}
		allFiltered = append(allFiltered, matched...)
	}
	return ixpath.DeduplicateNodes(allFiltered, ec.docOrder, ec.maxNodes)
}

func evalStepNoPredicates(ec *evalContext, nodes []helium.Node, step Step) ([]helium.Node, error) {
	var next []helium.Node
	for _, n := range nodes {
		candidates, err := ixpath.TraverseAxis(step.Axis, n, ec.maxNodes)
		if err != nil {
			return nil, err
		}
		if err := ec.countOps(len(candidates)); err != nil {
			return nil, err
		}
		next = append(next, filterByNodeTest(candidates, step.NodeTest, step.Axis, ec)...)
	}
	return ixpath.DeduplicateNodes(next, ec.docOrder, ec.maxNodes)
}

func filterByNodeTest(candidates []helium.Node, nt NodeTest, axis AxisType, ec *evalContext) []helium.Node {
	var matched []helium.Node
	for _, c := range candidates {
		if matchNodeTest(nt, c, axis, ec) {
			matched = append(matched, c)
		}
	}
	return matched
}

func matchNodeTest(nt NodeTest, n helium.Node, axis AxisType, ec *evalContext) bool {
	switch test := nt.(type) {
	case NameTest:
		return matchNameTest(test, n, axis, ec)
	case TypeTest:
		return matchTypeTest(test, n)
	case PITest:
		if n.Type() != helium.ProcessingInstructionNode {
			return false
		}
		if test.Target == "" {
			return true
		}
		return n.Name() == test.Target
	case ElementTest:
		if n.Type() != helium.ElementNode {
			return false
		}
		if test.Name != "" && test.Name != "*" {
			if ixpath.LocalNameOf(n) != test.Name {
				return false
			}
		}
		return true
	case AttributeTest:
		if _, ok := n.(*helium.Attribute); !ok {
			return false
		}
		if test.Name != "" && test.Name != "*" {
			if ixpath.LocalNameOf(n) != test.Name {
				return false
			}
		}
		return true
	case DocumentTest:
		if n.Type() != helium.DocumentNode {
			return false
		}
		if test.Inner != nil {
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if matchNodeTest(test.Inner, c, AxisChild, ec) {
					return true
				}
			}
			return false
		}
		return true
	case NamespaceNodeTest:
		return n.Type() == helium.NamespaceNode
	case AnyItemTest:
		return true
	}
	return false
}

func matchNameTest(test NameTest, n helium.Node, axis AxisType, ec *evalContext) bool {
	switch axis {
	case AxisAttribute:
		if _, ok := n.(*helium.Attribute); !ok {
			return false
		}
	case AxisNamespace:
		if n.Type() != helium.NamespaceNode {
			return false
		}
		if test.Local == "*" {
			return true
		}
		return n.Name() == test.Local
	default:
		if n.Type() != helium.ElementNode {
			return false
		}
	}

	if test.Local == "*" {
		if test.URI != "" {
			return ixpath.NodeNamespaceURI(n) == test.URI
		}
		if test.Prefix == "" {
			return true
		}
		return matchPrefix(test.Prefix, n, ec)
	}

	if ixpath.LocalNameOf(n) != test.Local {
		return false
	}

	if test.URI != "" {
		return ixpath.NodeNamespaceURI(n) == test.URI
	}
	if test.Prefix == "*" {
		// *:local matches any namespace
		return true
	}
	if test.Prefix != "" {
		return matchPrefix(test.Prefix, n, ec)
	}
	return true
}

func matchPrefix(prefix string, n helium.Node, ec *evalContext) bool {
	if ec.namespaces != nil {
		if uri, ok := ec.namespaces[prefix]; ok {
			return ixpath.NodeNamespaceURI(n) == uri
		}
	}
	return ixpath.NodePrefix(n) == prefix
}

func matchTypeTest(test TypeTest, n helium.Node) bool {
	switch test.Kind {
	case NodeKindNode:
		return true
	case NodeKindText:
		return n.Type() == helium.TextNode || n.Type() == helium.CDATASectionNode
	case NodeKindComment:
		return n.Type() == helium.CommentNode
	case NodeKindProcessingInstruction:
		return n.Type() == helium.ProcessingInstructionNode
	}
	return false
}

func applyPredicate(ec *evalContext, nodes []helium.Node, pred Expr) ([]helium.Node, error) {
	if err := ec.countOps(len(nodes)); err != nil {
		return nil, err
	}
	size := len(nodes)
	var result []helium.Node
	for i, n := range nodes {
		pctx := ec.withNode(n, i+1, size)
		r, err := eval(pctx, pred)
		if err != nil {
			return nil, err
		}
		match, err := predicateTrue(r, i+1)
		if err != nil {
			return nil, err
		}
		if match {
			result = append(result, n)
		}
	}
	return result, nil
}

// predicateTrue evaluates a predicate result per XPath spec:
// numeric → compare to position, otherwise → EBV.
func predicateTrue(r Sequence, position int) (bool, error) {
	if len(r) == 1 {
		if av, ok := r[0].(AtomicValue); ok && av.IsNumeric() {
			return av.ToFloat64() == float64(position), nil
		}
	}
	return EBV(r)
}
