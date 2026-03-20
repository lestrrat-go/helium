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

type atomicTypeRequirement struct {
	prefix string
	name   string
}

type prefixValidationPlan struct {
	prefixes    []string
	atomicTypes []atomicTypeRequirement
}

func newPrefixPlanBuilder() prefixPlanBuilder {
	return prefixPlanBuilder{}
}

func (p prefixValidationPlan) Validate(namespaces map[string]string, strict bool, decls SchemaDeclarations) error {
	for _, prefix := range p.prefixes {
		if err := validatePrefix(prefix, namespaces, strict); err != nil {
			return err
		}
	}
	for _, req := range p.atomicTypes {
		if req.prefix == "" {
			// Unprefixed type names are allowed when schema declarations
			// are available (XSLT schema-aware mode) and the type exists.
			if decls != nil {
				// Schema declarations present — allow unprefixed names.
				// They may be resolved via default namespace or as
				// no-namespace schema types at evaluation time.
				continue
			}
			// No schema declarations: unprefixed non-builtin names are
			// an error (XPST0081) in pure XPath mode.
			if namespaces == nil || namespaces[""] == "" {
				return &XPathError{
					Code:    errCodeXPST0081,
					Message: fmt.Sprintf("unprefixed type name %q requires a default element namespace", req.name),
				}
			}
			continue
		}
		if req.prefix == "xs" || req.prefix == "xsd" {
			switch req.name {
			case "NMTOKENS", "IDREFS", "ENTITIES":
				return &XPathError{
					Code:    "XPST0051",
					Message: fmt.Sprintf("xs:%s is a list type and cannot be used as an atomic type", req.name),
				}
			}
			if _, ok := knownXSTypeNames[req.name]; !ok {
				return &XPathError{
					Code:    "XPST0051",
					Message: fmt.Sprintf("unknown type xs:%s", req.name),
				}
			}
		}
		if err := validatePrefix(req.prefix, namespaces, strict); err != nil {
			return err
		}
	}
	return nil
}


func (b prefixPlanBuilder) plan() prefixValidationPlan {
	return prefixValidationPlan{
		prefixes:    b.prefixes,
		atomicTypes: b.atomicTypes,
	}
}

type prefixPlanBuilder struct {
	prefixes        []string
	atomicTypes     []atomicTypeRequirement
	seenPrefixes    map[string]struct{}
	seenAtomicTypes map[atomicTypeRequirement]struct{}
}

func appendPrefixChecks(plan *prefixPlanBuilder, node Expr) {
	if node == nil {
		return
	}

	appendExprLocalPrefixChecks(plan, node)

	switch n := node.(type) {
	case FunctionCall:
		for _, arg := range n.Args {
			appendPrefixChecks(plan, arg)
		}
	case CastExpr:
		appendPrefixChecks(plan, n.Expr)
	case CastableExpr:
		appendPrefixChecks(plan, n.Expr)
	case InstanceOfExpr:
		appendPrefixChecks(plan, n.Expr)
	case TreatAsExpr:
		appendPrefixChecks(plan, n.Expr)
	case PathExpr:
		appendPrefixChecks(plan, n.Filter)
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

func appendExprLocalPrefixChecks(plan *prefixPlanBuilder, node Expr) {
	switch n := node.(type) {
	case VariableExpr:
		addPrefixCheck(plan, n.Prefix)
	case FunctionCall:
		addPrefixCheck(plan, n.Prefix)
	case NamedFunctionRef:
		addPrefixCheck(plan, n.Prefix)
	case CastExpr:
		addPrefixCheck(plan, n.Type.Prefix)
	case CastableExpr:
		addPrefixCheck(plan, n.Type.Prefix)
	case InstanceOfExpr:
		appendSequenceTypePrefixChecks(plan, n.Type)
	case TreatAsExpr:
		appendSequenceTypePrefixChecks(plan, n.Type)
	case vmLocationPathExpr:
		for i := range n.Steps {
			appendNodeTestPrefixChecks(plan, n.Steps[i].NodeTest)
			for _, pred := range n.Steps[i].Predicates {
				appendVMPredicatePrefixChecks(plan, pred)
			}
		}
	case *LocationPath:
		for i := range n.Steps {
			appendStepLocalPrefixChecks(plan, &n.Steps[i])
		}
	case vmPathExpr:
		if n.Path != nil {
			for i := range n.Path.Steps {
				appendNodeTestPrefixChecks(plan, n.Path.Steps[i].NodeTest)
				for _, pred := range n.Path.Steps[i].Predicates {
					appendVMPredicatePrefixChecks(plan, pred)
				}
			}
		}
	case PathExpr:
		if n.Path != nil {
			for i := range n.Path.Steps {
				appendStepLocalPrefixChecks(plan, &n.Path.Steps[i])
			}
		}
	case FLWORExpr:
		for _, clause := range n.Clauses {
			appendFLWORClauseLocalPrefixChecks(plan, clause)
		}
	case QuantifiedExpr:
		for _, b := range n.Bindings {
			addVarNamePrefixCheck(plan, b.Var)
		}
	case TryCatchExpr:
		for _, c := range n.Catches {
			for _, code := range c.Codes {
				addCatchCodePrefixCheck(plan, code)
			}
		}
	case InlineFunctionExpr:
		for _, param := range n.Params {
			addVarNamePrefixCheck(plan, param.Name)
			if param.TypeHint != nil {
				appendSequenceTypePrefixChecks(plan, *param.TypeHint)
			}
		}
		if n.ReturnType != nil {
			appendSequenceTypePrefixChecks(plan, *n.ReturnType)
		}
	}
}

func appendStepLocalPrefixChecks(plan *prefixPlanBuilder, s *Step) {
	appendNodeTestPrefixChecks(plan, s.NodeTest)
}


func appendVMPredicatePrefixChecks(plan *prefixPlanBuilder, pred Expr) {
	switch p := pred.(type) {
	case vmPositionPredicateExpr:
		return
	case vmAttributeExistsPredicateExpr:
		appendNodeTestPrefixChecks(plan, p.NodeTest)
	case vmAttributeEqualsStringPredicateExpr:
		appendNodeTestPrefixChecks(plan, p.NodeTest)
		appendPrefixChecks(plan, p.Fallback)
	default:
		appendPrefixChecks(plan, pred)
	}
}

func appendVMPredicatePrefixChecks(plan *prefixPlanBuilder, pred Expr) {
	switch p := pred.(type) {
	case vmPositionPredicateExpr:
		return
	case vmAttributeExistsPredicateExpr:
		appendNodeTestPrefixChecks(plan, p.NodeTest)
	case vmAttributeEqualsStringPredicateExpr:
		appendNodeTestPrefixChecks(plan, p.NodeTest)
		appendPrefixChecks(plan, p.Fallback)
	default:
		appendPrefixChecks(plan, pred)
	}
}

func appendNodeTestPrefixChecks(plan *prefixPlanBuilder, nt NodeTest) {
	if nt == nil {
		return
	}
	switch t := nt.(type) {
	case NameTest:
		addPrefixCheck(plan, t.Prefix)
	case AtomicOrUnionType:
		addAtomicOrUnionTypeCheck(plan, t)
	case ElementTest:
		addQNameStringPrefixCheck(plan, t.Name)
		addQNameStringPrefixCheck(plan, t.TypeName)
	case AttributeTest:
		addQNameStringPrefixCheck(plan, t.Name)
		addQNameStringPrefixCheck(plan, t.TypeName)
	case SchemaElementTest:
		addQNameStringPrefixCheck(plan, t.Name)
	case SchemaAttributeTest:
		addQNameStringPrefixCheck(plan, t.Name)
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

func appendSequenceTypePrefixChecks(plan *prefixPlanBuilder, st SequenceType) {
	appendNodeTestPrefixChecks(plan, st.ItemTest)
}

func appendFLWORClausePrefixChecks(plan *prefixPlanBuilder, clause FLWORClause) {
	appendFLWORClauseLocalPrefixChecks(plan, clause)
	switch c := clause.(type) {
	case ForClause:
		appendPrefixChecks(plan, c.Expr)
	case LetClause:
		appendPrefixChecks(plan, c.Expr)
	}
}

func appendFLWORClauseLocalPrefixChecks(plan *prefixPlanBuilder, clause FLWORClause) {
	switch c := clause.(type) {
	case ForClause:
		addVarNamePrefixCheck(plan, c.Var)
		if c.PosVar != "" {
			addVarNamePrefixCheck(plan, c.PosVar)
		}
	case LetClause:
		addVarNamePrefixCheck(plan, c.Var)
	}
}

func addPrefixCheck(plan *prefixPlanBuilder, prefix string) {
	if prefix == "" || prefix == "*" {
		return
	}
	if plan.seenPrefixes == nil {
		plan.seenPrefixes = make(map[string]struct{}, 4)
	}
	if _, ok := plan.seenPrefixes[prefix]; ok {
		return
	}
	plan.seenPrefixes[prefix] = struct{}{}
	plan.prefixes = append(plan.prefixes, prefix)
}

func addVarNamePrefixCheck(plan *prefixPlanBuilder, varName string) {
	if strings.HasPrefix(varName, "Q{") {
		return
	}
	if idx := strings.IndexByte(varName, ':'); idx >= 0 {
		addPrefixCheck(plan, varName[:idx])
	}
}

// addQNameStringPrefixCheck extracts the prefix from a "prefix:local" QName string
// and adds it to the prefix validation plan. Handles Q{uri}local and empty strings.
func addQNameStringPrefixCheck(plan *prefixPlanBuilder, qname string) {
	if qname == "" || qname == "*" {
		return
	}
	addVarNamePrefixCheck(plan, qname)
}

// addCatchCodePrefixCheck extracts the prefix from a catch error code
// and adds it to the prefix validation plan. Handles wildcards and Q{uri} forms.
func addCatchCodePrefixCheck(plan *prefixPlanBuilder, code string) {
	if code == "" || code == "*" {
		return
	}
	if strings.HasPrefix(code, "Q{") {
		return // URIQualifiedName, no prefix to validate
	}
	if idx := strings.IndexByte(code, ':'); idx >= 0 {
		prefix := code[:idx]
		if prefix != "*" {
			addPrefixCheck(plan, prefix)
		}
	}
}

func addAtomicOrUnionTypeCheck(plan *prefixPlanBuilder, t AtomicOrUnionType) {
	req := atomicTypeRequirement{prefix: t.Prefix, name: t.Name}
	if plan.seenAtomicTypes == nil {
		plan.seenAtomicTypes = make(map[atomicTypeRequirement]struct{}, 4)
	}
	if _, ok := plan.seenAtomicTypes[req]; ok {
		return
	}
	plan.seenAtomicTypes[req] = struct{}{}
	plan.atomicTypes = append(plan.atomicTypes, req)
}

// validatePrefix checks if a non-empty prefix is bound in user namespaces or defaultPrefixNS.
// When strict is true, the defaultPrefixNS fallback is skipped (XSLT mode).
func validatePrefix(prefix string, namespaces map[string]string, strict bool) error {
	if prefix == "" || prefix == "*" {
		return nil
	}
	if namespaces != nil {
		if _, ok := namespaces[prefix]; ok {
			return nil
		}
	}
	if !strict {
		if _, ok := defaultPrefixNS[prefix]; ok {
			return nil
		}
	}
	if prefix == "xml" || prefix == "xmlns" {
		return nil
	}
	return &XPathError{
		Code:    errCodeXPST0081,
		Message: "undeclared namespace prefix: " + prefix,
	}
}
