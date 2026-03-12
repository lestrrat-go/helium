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

type prefixCheck func(map[string]string) error

type prefixValidationPlan []prefixCheck

func (p prefixValidationPlan) Validate(namespaces map[string]string) error {
	for _, check := range p {
		if err := check(namespaces); err != nil {
			return err
		}
	}
	return nil
}

func buildPrefixValidationPlan(node Expr) prefixValidationPlan {
	var plan prefixValidationPlan
	appendPrefixChecks(&plan, node)
	return plan
}

func appendPrefixChecks(plan *prefixValidationPlan, node Expr) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case VariableExpr:
		addPrefixCheck(plan, n.Prefix)
	case FunctionCall:
		addPrefixCheck(plan, n.Prefix)
		for _, arg := range n.Args {
			appendPrefixChecks(plan, arg)
		}
	case NamedFunctionRef:
		addPrefixCheck(plan, n.Prefix)
	case CastExpr:
		addPrefixCheck(plan, n.Type.Prefix)
		appendPrefixChecks(plan, n.Expr)
	case CastableExpr:
		addPrefixCheck(plan, n.Type.Prefix)
		appendPrefixChecks(plan, n.Expr)
	case InstanceOfExpr:
		appendSequenceTypePrefixChecks(plan, n.Type)
		appendPrefixChecks(plan, n.Expr)
	case TreatAsExpr:
		appendSequenceTypePrefixChecks(plan, n.Type)
		appendPrefixChecks(plan, n.Expr)
	case LocationPath:
		for i := range n.Steps {
			appendStepPrefixChecks(plan, &n.Steps[i])
		}
	case PathExpr:
		appendPrefixChecks(plan, n.Filter)
		if n.Path != nil {
			for i := range n.Path.Steps {
				appendStepPrefixChecks(plan, &n.Path.Steps[i])
			}
		}
	case PathStepExpr:
		appendPrefixChecks(plan, n.Left)
		appendPrefixChecks(plan, n.Right)
	case FilterExpr:
		appendPrefixChecks(plan, n.Expr)
		for _, pred := range n.Predicates {
			appendPrefixChecks(plan, pred)
		}
	case BinaryExpr:
		appendPrefixChecks(plan, n.Left)
		appendPrefixChecks(plan, n.Right)
	case UnaryExpr:
		appendPrefixChecks(plan, n.Operand)
	case ConcatExpr:
		appendPrefixChecks(plan, n.Left)
		appendPrefixChecks(plan, n.Right)
	case SimpleMapExpr:
		appendPrefixChecks(plan, n.Left)
		appendPrefixChecks(plan, n.Right)
	case RangeExpr:
		appendPrefixChecks(plan, n.Start)
		appendPrefixChecks(plan, n.End)
	case UnionExpr:
		appendPrefixChecks(plan, n.Left)
		appendPrefixChecks(plan, n.Right)
	case IntersectExceptExpr:
		appendPrefixChecks(plan, n.Left)
		appendPrefixChecks(plan, n.Right)
	case IfExpr:
		appendPrefixChecks(plan, n.Cond)
		appendPrefixChecks(plan, n.Then)
		appendPrefixChecks(plan, n.Else)
	case FLWORExpr:
		for _, clause := range n.Clauses {
			appendFLWORClausePrefixChecks(plan, clause)
		}
		appendPrefixChecks(plan, n.Return)
	case QuantifiedExpr:
		for _, b := range n.Bindings {
			addVarNamePrefixCheck(plan, b.Var)
			appendPrefixChecks(plan, b.Domain)
		}
		appendPrefixChecks(plan, n.Satisfies)
	case TryCatchExpr:
		appendPrefixChecks(plan, n.Try)
		for _, c := range n.Catches {
			appendPrefixChecks(plan, c.Expr)
		}
	case DynamicFunctionCall:
		appendPrefixChecks(plan, n.Func)
		for _, arg := range n.Args {
			appendPrefixChecks(plan, arg)
		}
	case InlineFunctionExpr:
		for _, param := range n.Params {
			if param.TypeHint != nil {
				appendSequenceTypePrefixChecks(plan, *param.TypeHint)
			}
		}
		if n.ReturnType != nil {
			appendSequenceTypePrefixChecks(plan, *n.ReturnType)
		}
		appendPrefixChecks(plan, n.Body)
	case LookupExpr:
		appendPrefixChecks(plan, n.Expr)
		appendPrefixChecks(plan, n.Key)
	case UnaryLookupExpr:
		appendPrefixChecks(plan, n.Key)
	case MapConstructorExpr:
		for _, p := range n.Pairs {
			appendPrefixChecks(plan, p.Key)
			appendPrefixChecks(plan, p.Value)
		}
	case ArrayConstructorExpr:
		for _, item := range n.Items {
			appendPrefixChecks(plan, item)
		}
	case SequenceExpr:
		for _, item := range n.Items {
			appendPrefixChecks(plan, item)
		}
	}
}

func appendStepPrefixChecks(plan *prefixValidationPlan, s *Step) {
	appendNodeTestPrefixChecks(plan, s.NodeTest)
	for _, pred := range s.Predicates {
		appendPrefixChecks(plan, pred)
	}
}

func appendNodeTestPrefixChecks(plan *prefixValidationPlan, nt NodeTest) {
	if nt == nil {
		return
	}
	switch t := nt.(type) {
	case NameTest:
		addPrefixCheck(plan, t.Prefix)
	case AtomicOrUnionType:
		addAtomicOrUnionTypeCheck(plan, t)
	case DocumentTest:
		appendNodeTestPrefixChecks(plan, t.Inner)
	case FunctionTest:
		for _, pt := range t.ParamTypes {
			appendSequenceTypePrefixChecks(plan, pt)
		}
		appendSequenceTypePrefixChecks(plan, t.ReturnType)
	case MapTest:
		appendNodeTestPrefixChecks(plan, t.KeyType)
		appendSequenceTypePrefixChecks(plan, t.ValType)
	case ArrayTest:
		appendSequenceTypePrefixChecks(plan, t.MemberType)
	}
}

func appendSequenceTypePrefixChecks(plan *prefixValidationPlan, st SequenceType) {
	appendNodeTestPrefixChecks(plan, st.ItemTest)
}

func appendFLWORClausePrefixChecks(plan *prefixValidationPlan, clause FLWORClause) {
	switch c := clause.(type) {
	case ForClause:
		addVarNamePrefixCheck(plan, c.Var)
		if c.PosVar != "" {
			addVarNamePrefixCheck(plan, c.PosVar)
		}
		appendPrefixChecks(plan, c.Expr)
	case LetClause:
		addVarNamePrefixCheck(plan, c.Var)
		appendPrefixChecks(plan, c.Expr)
	}
}

func addPrefixCheck(plan *prefixValidationPlan, prefix string) {
	if prefix == "" || prefix == "*" {
		return
	}
	p := prefix
	*plan = append(*plan, func(namespaces map[string]string) error {
		return validatePrefix(p, namespaces)
	})
}

func addVarNamePrefixCheck(plan *prefixValidationPlan, varName string) {
	if strings.HasPrefix(varName, "Q{") {
		return
	}
	if idx := strings.IndexByte(varName, ':'); idx >= 0 {
		addPrefixCheck(plan, varName[:idx])
	}
}

func addAtomicOrUnionTypeCheck(plan *prefixValidationPlan, t AtomicOrUnionType) {
	prefix := t.Prefix
	name := t.Name
	*plan = append(*plan, func(namespaces map[string]string) error {
		if prefix == "" {
			if namespaces == nil || namespaces[""] == "" {
				return &XPathError{
					Code:    errCodeXPST0081,
					Message: fmt.Sprintf("unprefixed type name %q requires a default element namespace", name),
				}
			}
		}
		if prefix == "xs" || prefix == "xsd" {
			switch name {
			case "NMTOKENS", "IDREFS", "ENTITIES":
				return &XPathError{
					Code:    "XPST0051",
					Message: fmt.Sprintf("xs:%s is a list type and cannot be used as an atomic type", name),
				}
			}
			if _, ok := knownXSTypeNames[name]; !ok {
				return &XPathError{
					Code:    "XPST0051",
					Message: fmt.Sprintf("unknown type xs:%s", name),
				}
			}
		}
		return validatePrefix(prefix, namespaces)
	})
}

// checkPrefixes walks the AST and validates that all namespace prefixes
// are bound. This catches XPST0081 static errors even in unreachable branches
// (e.g., "if (true()) then 1 else $unbound:var").
func checkPrefixes(node Expr, namespaces map[string]string) error {
	return buildPrefixValidationPlan(node).Validate(namespaces)
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
		Code:    errCodeXPST0081,
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
					Code:    errCodeXPST0081,
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
