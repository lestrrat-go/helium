package xpath3

import "fmt"

func evalInstanceOfExpr(ec *evalContext, e InstanceOfExpr) (Sequence, error) {
	seq, err := eval(ec, e.Expr)
	if err != nil {
		return nil, err
	}
	return SingleBoolean(matchesSequenceType(seq, e.Type, ec)), nil
}

func evalCastExpr(ec *evalContext, e CastExpr) (Sequence, error) {
	seq, err := eval(ec, e.Expr)
	if err != nil {
		return nil, err
	}
	if len(seq) == 0 {
		if e.AllowEmpty {
			return nil, nil
		}
		return nil, &XPathError{Code: "XPTY0004", Message: "cast requires non-empty sequence"}
	}
	if len(seq) > 1 {
		return nil, &XPathError{Code: "XPTY0004", Message: "cast requires singleton"}
	}
	av, err := AtomizeItem(seq[0])
	if err != nil {
		return nil, err
	}
	targetType := resolveAtomicTypeName(e.Type, ec)
	result, err := CastAtomic(av, targetType)
	if err != nil {
		return nil, err
	}
	return SingleAtomic(result), nil
}

func evalCastableExpr(ec *evalContext, e CastableExpr) (Sequence, error) {
	targetType := resolveAtomicTypeName(e.Type, ec)
	// Abstract types raise a static error even for castable (XPST0080)
	if isAbstractCastTarget(targetType) {
		return nil, &XPathError{
			Code:    "XPST0080",
			Message: fmt.Sprintf("cannot use abstract type %s as castable target", targetType),
		}
	}
	seq, err := eval(ec, e.Expr)
	if err != nil {
		return nil, err
	}
	if len(seq) == 0 {
		return SingleBoolean(e.AllowEmpty), nil
	}
	if len(seq) > 1 {
		return SingleBoolean(false), nil
	}
	av, err := AtomizeItem(seq[0])
	if err != nil {
		return SingleBoolean(false), nil
	}
	_, castErr := CastAtomic(av, targetType)
	return SingleBoolean(castErr == nil), nil
}

func evalTreatAsExpr(ec *evalContext, e TreatAsExpr) (Sequence, error) {
	seq, err := eval(ec, e.Expr)
	if err != nil {
		return nil, err
	}
	if !matchesSequenceType(seq, e.Type, ec) {
		return nil, &XPathError{Code: "XPDY0050", Message: "treat as type mismatch"}
	}
	return seq, nil
}

func resolveAtomicTypeName(tn AtomicTypeName, ec *evalContext) string {
	if tn.Prefix == "" {
		return "xs:" + tn.Name
	}
	if tn.Prefix == "xs" || tn.Prefix == "xsd" {
		return "xs:" + tn.Name
	}
	// Resolve via namespace context
	if ec.namespaces != nil {
		if uri, ok := ec.namespaces[tn.Prefix]; ok {
			if uri == "http://www.w3.org/2001/XMLSchema" {
				return "xs:" + tn.Name
			}
		}
	}
	return tn.Prefix + ":" + tn.Name
}

func matchesSequenceType(seq Sequence, st SequenceType, ec *evalContext) bool {
	if st.Void {
		return len(seq) == 0
	}
	switch st.Occurrence {
	case OccurrenceExactlyOne:
		if len(seq) != 1 {
			return false
		}
	case OccurrenceZeroOrOne:
		if len(seq) > 1 {
			return false
		}
	case OccurrenceOneOrMore:
		if len(seq) == 0 {
			return false
		}
	case OccurrenceZeroOrMore:
		// any length ok
	}
	for _, item := range seq {
		if !matchesItemType(item, st.ItemTest, ec) {
			return false
		}
	}
	return true
}

func matchesItemType(item Item, test NodeTest, ec *evalContext) bool {
	if test == nil {
		return true
	}
	switch t := test.(type) {
	case AnyItemTest:
		return true
	case TypeTest:
		ni, ok := item.(NodeItem)
		if !ok {
			return false
		}
		return matchTypeTest(t, ni.Node)
	case NameTest:
		ni, ok := item.(NodeItem)
		if !ok {
			return false
		}
		return matchNameTest(t, ni.Node, AxisChild, ec)
	case ElementTest:
		ni, ok := item.(NodeItem)
		if !ok {
			return false
		}
		return matchNodeTest(t, ni.Node, AxisChild, ec)
	case AttributeTest:
		ni, ok := item.(NodeItem)
		if !ok {
			return false
		}
		return matchNodeTest(t, ni.Node, AxisAttribute, ec)
	case AtomicOrUnionType:
		av, ok := item.(AtomicValue)
		if !ok {
			return false
		}
		targetType := "xs:" + t.Name
		if t.Prefix != "" && t.Prefix != "xs" && t.Prefix != "xsd" {
			targetType = t.Prefix + ":" + t.Name
		}
		if targetType == TypeAnyAtomicType {
			return true
		}
		return isSubtypeOf(av.TypeName, targetType)
	case FunctionTest:
		_, ok := item.(FunctionItem)
		return ok
	case MapTest:
		_, ok := item.(MapItem)
		return ok
	case ArrayTest:
		_, ok := item.(ArrayItem)
		return ok
	}
	return false
}
