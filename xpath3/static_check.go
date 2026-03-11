package xpath3

import (
	"fmt"
	"strings"
)

// knownXSTypeNames lists valid type local names in the xs: namespace
// for use in SequenceType (instance of, cast as, etc.).
var knownXSTypeNames = map[string]struct{}{
	"string": {}, "integer": {}, "decimal": {}, "double": {},
	"float": {}, "boolean": {}, "date": {}, "dateTime": {},
	"time": {}, "duration": {}, "dayTimeDuration": {},
	"yearMonthDuration": {}, "anyURI": {}, "QName": {},
	"base64Binary": {}, "hexBinary": {}, "untypedAtomic": {},
	"anyAtomicType": {}, "long": {}, "int": {}, "short": {},
	"byte": {}, "unsignedLong": {}, "unsignedInt": {},
	"unsignedShort": {}, "unsignedByte": {},
	"nonNegativeInteger": {}, "nonPositiveInteger": {},
	"positiveInteger": {}, "negativeInteger": {},
	"normalizedString": {}, "token": {}, "language": {},
	"Name": {}, "NCName": {}, "NMTOKEN": {}, "ENTITY": {},
	"ID": {}, "IDREF": {}, "gDay": {}, "gMonth": {},
	"gMonthDay": {}, "gYear": {}, "gYearMonth": {},
	"dateTimeStamp": {}, "error": {}, "numeric": {},
	"NOTATION": {},
	// List types (rejected separately as XPST0051):
	"NMTOKENS": {}, "IDREFS": {}, "ENTITIES": {},
}

// checkPrefixes walks the AST and validates that all namespace prefixes
// are bound. This catches XPST0081 static errors even in unreachable branches
// (e.g., "if (true()) then 1 else $unbound:var").
func checkPrefixes(node Expr, namespaces map[string]string) error {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case VariableExpr:
		return validatePrefix(n.Prefix, namespaces)
	case FunctionCall:
		if err := validatePrefix(n.Prefix, namespaces); err != nil {
			return err
		}
		for _, arg := range n.Args {
			if err := checkPrefixes(arg, namespaces); err != nil {
				return err
			}
		}
	case NamedFunctionRef:
		return validatePrefix(n.Prefix, namespaces)
	case CastExpr:
		if err := validatePrefix(n.Type.Prefix, namespaces); err != nil {
			return err
		}
		return checkPrefixes(n.Expr, namespaces)
	case CastableExpr:
		if err := validatePrefix(n.Type.Prefix, namespaces); err != nil {
			return err
		}
		return checkPrefixes(n.Expr, namespaces)
	case InstanceOfExpr:
		if err := checkPrefixesInSequenceType(n.Type, namespaces); err != nil {
			return err
		}
		return checkPrefixes(n.Expr, namespaces)
	case TreatAsExpr:
		if err := checkPrefixesInSequenceType(n.Type, namespaces); err != nil {
			return err
		}
		return checkPrefixes(n.Expr, namespaces)
	case LocationPath:
		for i := range n.Steps {
			if err := checkPrefixesInStep(&n.Steps[i], namespaces); err != nil {
				return err
			}
		}
	case PathExpr:
		if err := checkPrefixes(n.Filter, namespaces); err != nil {
			return err
		}
		if n.Path != nil {
			for i := range n.Path.Steps {
				if err := checkPrefixesInStep(&n.Path.Steps[i], namespaces); err != nil {
					return err
				}
			}
		}
	case PathStepExpr:
		if err := checkPrefixes(n.Left, namespaces); err != nil {
			return err
		}
		return checkPrefixes(n.Right, namespaces)
	case FilterExpr:
		if err := checkPrefixes(n.Expr, namespaces); err != nil {
			return err
		}
		for _, pred := range n.Predicates {
			if err := checkPrefixes(pred, namespaces); err != nil {
				return err
			}
		}
	case BinaryExpr:
		if err := checkPrefixes(n.Left, namespaces); err != nil {
			return err
		}
		return checkPrefixes(n.Right, namespaces)
	case UnaryExpr:
		return checkPrefixes(n.Operand, namespaces)
	case ConcatExpr:
		if err := checkPrefixes(n.Left, namespaces); err != nil {
			return err
		}
		return checkPrefixes(n.Right, namespaces)
	case SimpleMapExpr:
		if err := checkPrefixes(n.Left, namespaces); err != nil {
			return err
		}
		return checkPrefixes(n.Right, namespaces)
	case RangeExpr:
		if err := checkPrefixes(n.Start, namespaces); err != nil {
			return err
		}
		return checkPrefixes(n.End, namespaces)
	case UnionExpr:
		if err := checkPrefixes(n.Left, namespaces); err != nil {
			return err
		}
		return checkPrefixes(n.Right, namespaces)
	case IntersectExceptExpr:
		if err := checkPrefixes(n.Left, namespaces); err != nil {
			return err
		}
		return checkPrefixes(n.Right, namespaces)
	case IfExpr:
		if err := checkPrefixes(n.Cond, namespaces); err != nil {
			return err
		}
		if err := checkPrefixes(n.Then, namespaces); err != nil {
			return err
		}
		return checkPrefixes(n.Else, namespaces)
	case FLWORExpr:
		for _, clause := range n.Clauses {
			if err := checkPrefixesInFLWORClause(clause, namespaces); err != nil {
				return err
			}
		}
		return checkPrefixes(n.Return, namespaces)
	case QuantifiedExpr:
		for _, b := range n.Bindings {
			if err := checkPrefixesInVarName(b.Var, namespaces); err != nil {
				return err
			}
			if err := checkPrefixes(b.Domain, namespaces); err != nil {
				return err
			}
		}
		return checkPrefixes(n.Satisfies, namespaces)
	case TryCatchExpr:
		if err := checkPrefixes(n.Try, namespaces); err != nil {
			return err
		}
		for _, c := range n.Catches {
			if err := checkPrefixes(c.Expr, namespaces); err != nil {
				return err
			}
		}
	case DynamicFunctionCall:
		if err := checkPrefixes(n.Func, namespaces); err != nil {
			return err
		}
		for _, arg := range n.Args {
			if err := checkPrefixes(arg, namespaces); err != nil {
				return err
			}
		}
	case InlineFunctionExpr:
		for _, param := range n.Params {
			if param.TypeHint != nil {
				if err := checkPrefixesInSequenceType(*param.TypeHint, namespaces); err != nil {
					return err
				}
			}
		}
		if n.ReturnType != nil {
			if err := checkPrefixesInSequenceType(*n.ReturnType, namespaces); err != nil {
				return err
			}
		}
		return checkPrefixes(n.Body, namespaces)
	case LookupExpr:
		if err := checkPrefixes(n.Expr, namespaces); err != nil {
			return err
		}
		return checkPrefixes(n.Key, namespaces)
	case UnaryLookupExpr:
		return checkPrefixes(n.Key, namespaces)
	case MapConstructorExpr:
		for _, p := range n.Pairs {
			if err := checkPrefixes(p.Key, namespaces); err != nil {
				return err
			}
			if err := checkPrefixes(p.Value, namespaces); err != nil {
				return err
			}
		}
	case ArrayConstructorExpr:
		for _, item := range n.Items {
			if err := checkPrefixes(item, namespaces); err != nil {
				return err
			}
		}
	case SequenceExpr:
		for _, item := range n.Items {
			if err := checkPrefixes(item, namespaces); err != nil {
				return err
			}
		}
	}
	return nil
}

// validatePrefix checks if a non-empty prefix is bound in user namespaces or defaultPrefixNS.
func validatePrefix(prefix string, namespaces map[string]string) error {
	if prefix == "" || prefix == "*" {
		return nil
	}
	if namespaces != nil {
		if _, ok := namespaces[prefix]; ok {
			return nil
		}
	}
	if _, ok := defaultPrefixNS[prefix]; ok {
		return nil
	}
	if prefix == "xml" || prefix == "xmlns" {
		return nil
	}
	return &XPathError{
		Code:    "XPST0081",
		Message: "undeclared namespace prefix: " + prefix,
	}
}

func checkPrefixesInStep(s *Step, namespaces map[string]string) error {
	if err := checkPrefixesInNodeTest(s.NodeTest, namespaces); err != nil {
		return err
	}
	for _, pred := range s.Predicates {
		if err := checkPrefixes(pred, namespaces); err != nil {
			return err
		}
	}
	return nil
}

func checkPrefixesInNodeTest(nt NodeTest, namespaces map[string]string) error {
	if nt == nil {
		return nil
	}
	switch t := nt.(type) {
	case NameTest:
		return validatePrefix(t.Prefix, namespaces)
	case AtomicOrUnionType:
		// Unprefixed atomic type names require a default element namespace binding.
		// Without one, the name cannot be resolved (XPST0081).
		if t.Prefix == "" {
			if namespaces == nil || namespaces[""] == "" {
				return &XPathError{
					Code:    "XPST0081",
					Message: fmt.Sprintf("unprefixed type name %q requires a default element namespace", t.Name),
				}
			}
		}
		if t.Prefix == "xs" || t.Prefix == "xsd" {
			// XSD list types (NMTOKENS, IDREFS, ENTITIES) are not valid atomic/union
			// types in SequenceType — they are list types (XPST0051).
			switch t.Name {
			case "NMTOKENS", "IDREFS", "ENTITIES":
				return &XPathError{
					Code:    "XPST0051",
					Message: fmt.Sprintf("xs:%s is a list type and cannot be used as an atomic type", t.Name),
				}
			}
			// Unknown type name in xs: namespace
			if _, ok := knownXSTypeNames[t.Name]; !ok {
				return &XPathError{
					Code:    "XPST0051",
					Message: fmt.Sprintf("unknown type xs:%s", t.Name),
				}
			}
		}
		return validatePrefix(t.Prefix, namespaces)
	case DocumentTest:
		return checkPrefixesInNodeTest(t.Inner, namespaces)
	case FunctionTest:
		for _, pt := range t.ParamTypes {
			if err := checkPrefixesInSequenceType(pt, namespaces); err != nil {
				return err
			}
		}
		return checkPrefixesInSequenceType(t.ReturnType, namespaces)
	case MapTest:
		if err := checkPrefixesInNodeTest(t.KeyType, namespaces); err != nil {
			return err
		}
		return checkPrefixesInSequenceType(t.ValType, namespaces)
	case ArrayTest:
		return checkPrefixesInSequenceType(t.MemberType, namespaces)
	}
	return nil
}

func checkPrefixesInSequenceType(st SequenceType, namespaces map[string]string) error {
	return checkPrefixesInNodeTest(st.ItemTest, namespaces)
}

func checkPrefixesInFLWORClause(clause FLWORClause, namespaces map[string]string) error {
	switch c := clause.(type) {
	case ForClause:
		if err := checkPrefixesInVarName(c.Var, namespaces); err != nil {
			return err
		}
		if c.PosVar != "" {
			if err := checkPrefixesInVarName(c.PosVar, namespaces); err != nil {
				return err
			}
		}
		return checkPrefixes(c.Expr, namespaces)
	case LetClause:
		if err := checkPrefixesInVarName(c.Var, namespaces); err != nil {
			return err
		}
		return checkPrefixes(c.Expr, namespaces)
	}
	return nil
}

// checkPrefixesInVarName checks a FLWOR variable name like "prefix:local" for undeclared prefix.
func checkPrefixesInVarName(varName string, namespaces map[string]string) error {
	// URIQualifiedName (Q{uri}local) — prefix is already resolved
	if strings.HasPrefix(varName, "Q{") {
		return nil
	}
	if idx := strings.IndexByte(varName, ':'); idx >= 0 {
		return validatePrefix(varName[:idx], namespaces)
	}
	return nil
}
