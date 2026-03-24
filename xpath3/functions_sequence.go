package xpath3

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sort"
	"time"

	"github.com/lestrrat-go/helium"
)

func init() {
	registerFn("empty", 1, 1, fnEmpty)
	registerFn("exists", 1, 1, fnExists)
	registerFn("head", 1, 1, fnHead)
	registerFn("tail", 1, 1, fnTail)
	registerFn("insert-before", 3, 3, fnInsertBefore)
	registerFn("remove", 2, 2, fnRemove)
	registerFn("reverse", 1, 1, fnReverse)
	registerFn("subsequence", 2, 3, fnSubsequence)
	registerFn("unordered", 1, 1, fnUnordered)
	registerFn("zero-or-one", 1, 1, fnZeroOrOne)
	registerFn("one-or-more", 1, 1, fnOneOrMore)
	registerFn("exactly-one", 1, 1, fnExactlyOne)
	registerFn("deep-equal", 2, 3, fnDeepEqual)
	registerFn("index-of", 2, 3, fnIndexOf)
	registerFn("last", 0, 0, fnLast)
	registerFn("position", 0, 0, fnPosition)
	registerFn("sort", 1, 3, fnSort)
	registerFn("flatten", 1, 1, fnFlatten)
}

func fnEmpty(_ context.Context, args []Sequence) (Sequence, error) {
	return SingleBoolean(seqLen(args[0]) == 0), nil
}

func fnExists(_ context.Context, args []Sequence) (Sequence, error) {
	return SingleBoolean(seqLen(args[0]) > 0), nil
}

func fnHead(_ context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return nil, nil
	}
	return ItemSlice{args[0].Get(0)}, nil
}

func fnTail(_ context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) <= 1 {
		return nil, nil
	}
	return ItemSlice(seqMaterialize(args[0])[1:]), nil
}

func fnInsertBefore(_ context.Context, args []Sequence) (Sequence, error) {
	target := args[0]
	if seqLen(args[1]) == 0 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "fn:insert-before: position argument is an empty sequence"}
	}
	a, err := AtomizeItem(args[1].Get(0))
	if err != nil {
		return nil, err
	}
	pos := int(a.ToFloat64())
	inserts := args[2]

	if pos < 1 {
		pos = 1
	}
	tItems := seqMaterialize(target)
	if pos > len(tItems)+1 {
		pos = len(tItems) + 1
	}

	iItems := seqMaterialize(inserts)
	result := make(ItemSlice, 0, len(tItems)+len(iItems))
	result = append(result, tItems[:pos-1]...)
	result = append(result, iItems...)
	result = append(result, tItems[pos-1:]...)
	return result, nil
}

func fnRemove(_ context.Context, args []Sequence) (Sequence, error) {
	target := args[0]
	posVal, err := coerceArgToInteger(args[1])
	if err != nil {
		return nil, err
	}
	pos := int(posVal)

	tItems := seqMaterialize(target)
	if pos < 1 || pos > len(tItems) {
		return target, nil
	}

	result := make(ItemSlice, 0, len(tItems)-1)
	result = append(result, tItems[:pos-1]...)
	result = append(result, tItems[pos:]...)
	return result, nil
}

func fnReverse(_ context.Context, args []Sequence) (Sequence, error) {
	seq := args[0]
	items := seqMaterialize(seq)
	result := make(ItemSlice, len(items))
	for i, item := range items {
		result[len(items)-1-i] = item
	}
	return result, nil
}

func fnSubsequence(_ context.Context, args []Sequence) (Sequence, error) {
	seq := args[0]
	if seqLen(args[1]) == 0 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "subsequence: starting position is required"}
	}
	a, err := AtomizeItem(args[1].Get(0))
	if err != nil {
		return nil, err
	}
	// Cast untypedAtomic to double (per XPath function calling convention)
	if a.TypeName == TypeUntypedAtomic {
		a, err = CastAtomic(a, TypeDouble)
		if err != nil {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "subsequence: starting position must be numeric"}
		}
	}
	if !isSubtypeOf(a.TypeName, TypeNumeric) {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "subsequence: starting position must be numeric"}
	}
	startF := math.Round(a.ToFloat64())

	hasLength := len(args) > 2
	if hasLength && seqLen(args[2]) == 0 {
		hasLength = false // $length is xs:double? — empty means no bound
	}
	var lengthF float64
	if hasLength {
		la, err := AtomizeItem(args[2].Get(0))
		if err != nil {
			return nil, err
		}
		if la.TypeName == TypeUntypedAtomic {
			la, err = CastAtomic(la, TypeDouble)
			if err != nil {
				return nil, &XPathError{Code: errCodeXPTY0004, Message: "subsequence: length must be numeric"}
			}
		}
		if !isSubtypeOf(la.TypeName, TypeNumeric) {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "subsequence: length must be numeric"}
		}
		lengthF = math.Round(la.ToFloat64())
	}

	// Handle NaN start or NaN length
	if math.IsNaN(startF) {
		return nil, nil
	}
	if hasLength && math.IsNaN(lengthF) {
		return nil, nil
	}

	// Compute end position: startF + lengthF (only when length is given)
	var endF float64
	if hasLength {
		endF = startF + lengthF
		// -INF + INF = NaN → empty result
		if math.IsNaN(endF) {
			return nil, nil
		}
	}

	// Items at position p where startF <= p (and p < endF if length given)
	var result ItemSlice
	i := 0
	for item := range seqItems(seq) {
		p := float64(i + 1)
		if p < startF {
			i++
			continue
		}
		if hasLength && p >= endF {
			break
		}
		result = append(result, item)
		i++
	}
	return result, nil
}

func fnUnordered(_ context.Context, args []Sequence) (Sequence, error) {
	return args[0], nil
}

func fnZeroOrOne(_ context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) > 1 {
		return nil, &XPathError{Code: "FORG0003", Message: fmt.Sprintf("zero-or-one() called with sequence of length %d", seqLen(args[0]))}
	}
	return args[0], nil
}

func fnOneOrMore(_ context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return nil, &XPathError{Code: "FORG0004", Message: "one-or-more() called with empty sequence"}
	}
	return args[0], nil
}

func fnExactlyOne(_ context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) != 1 {
		return nil, &XPathError{Code: "FORG0005", Message: fmt.Sprintf("exactly-one() called with sequence of length %d", seqLen(args[0]))}
	}
	return args[0], nil
}

type deepEqualOptions struct {
	coll       *collationImpl
	implicitTZ *time.Location
}

func fnDeepEqual(ctx context.Context, args []Sequence) (Sequence, error) {
	coll, err := getCollation(ctx, args, 2)
	if err != nil {
		return nil, err
	}
	opts := deepEqualOptions{coll: coll}
	if ec := getFnContext(ctx); ec != nil {
		opts.implicitTZ = ec.getImplicitTimezone()
	}
	eq, err := deepEqualSequence(args[0], args[1], opts)
	if err != nil {
		return nil, err
	}
	return SingleBoolean(eq), nil
}

func deepEqualSequence(a, b Sequence, opts deepEqualOptions) (bool, error) {
	if seqLen(a) != seqLen(b) {
		return false, nil
	}
	for i := range seqLen(a) {
		eq, err := deepEqualItem(a.Get(i), b.Get(i), opts)
		if err != nil {
			return false, err
		}
		if !eq {
			return false, nil
		}
	}
	return true, nil
}

func deepEqualItem(a, b Item, opts deepEqualOptions) (bool, error) {
	switch av := a.(type) {
	case AtomicValue:
		bv, ok := b.(AtomicValue)
		if !ok {
			return false, nil
		}
		return deepEqualAtomic(av, bv, opts)
	case NodeItem:
		bv, ok := b.(NodeItem)
		if !ok {
			return false, nil
		}
		return deepEqualNode(av.Node, bv.Node, opts), nil
	case MapItem:
		bv, ok := b.(MapItem)
		if !ok {
			return false, nil
		}
		return deepEqualMap(av, bv, opts)
	case ArrayItem:
		bv, ok := b.(ArrayItem)
		if !ok {
			return false, nil
		}
		return deepEqualArray(av, bv, opts)
	case FunctionItem:
		if _, ok := b.(FunctionItem); ok {
			return false, &XPathError{Code: "FOTY0015", Message: "deep-equal: cannot compare function items"}
		}
		return false, nil
	default:
		return false, nil
	}
}

func deepEqualAtomic(a, b AtomicValue, opts deepEqualOptions) (bool, error) {
	// NaN equals NaN for deep-equal purposes
	if a.TypeName == TypeDouble || a.TypeName == TypeFloat {
		af := a.ToFloat64()
		if b.TypeName == TypeDouble || b.TypeName == TypeFloat {
			bf := b.ToFloat64()
			if af != af && bf != bf { // both NaN
				return true, nil
			}
		}
	}
	aStr := isStringDerived(a.TypeName) || a.TypeName == TypeAnyURI
	bStr := isStringDerived(b.TypeName) || b.TypeName == TypeAnyURI
	if opts.coll != nil && aStr && bStr {
		return opts.coll.compare(stringFromAtomic(a), stringFromAtomic(b)) == 0, nil
	}
	eq, err := ValueCompareWithImplicitTimezone(TokenEq, a, b, opts.implicitTZ)
	if err != nil {
		// Incomparable types are not deep-equal
		return false, nil
	}
	return eq, nil
}

func deepEqualNode(a, b helium.Node, opts deepEqualOptions) bool {
	if a.Type() != b.Type() {
		return false
	}
	strEq := func(sa, sb string) bool {
		if opts.coll != nil {
			return opts.coll.compare(sa, sb) == 0
		}
		return sa == sb
	}
	switch a.Type() {
	case helium.DocumentNode:
		return deepEqualChildren(a, b, opts)
	case helium.ElementNode:
		ae, aOK := a.(*helium.Element)
		be, bOK := b.(*helium.Element)
		if !aOK || !bOK {
			return false
		}
		// Compare expanded name (local name + namespace URI)
		if ae.LocalName() != be.LocalName() || ae.URI() != be.URI() {
			return false
		}
		// Compare attributes (order-independent)
		aAttrs := ae.Attributes()
		bAttrs := be.Attributes()
		if len(aAttrs) != len(bAttrs) {
			return false
		}
		for _, aa := range aAttrs {
			found := false
			for _, ba := range bAttrs {
				if aa.LocalName() == ba.LocalName() && aa.URI() == ba.URI() {
					if !strEq(aa.Value(), ba.Value()) {
						return false
					}
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		// Compare children
		return deepEqualChildren(a, b, opts)
	case helium.AttributeNode:
		aa, aOK := a.(*helium.Attribute)
		ba, bOK := b.(*helium.Attribute)
		if !aOK || !bOK {
			return false
		}
		return aa.LocalName() == ba.LocalName() && aa.URI() == ba.URI() && strEq(aa.Value(), ba.Value())
	case helium.NamespaceNode:
		return a.Name() == b.Name() && string(a.Content()) == string(b.Content())
	case helium.TextNode, helium.CDATASectionNode:
		return strEq(string(a.Content()), string(b.Content()))
	case helium.CommentNode:
		return string(a.Content()) == string(b.Content())
	case helium.ProcessingInstructionNode:
		return a.Name() == b.Name() && string(a.Content()) == string(b.Content())
	default:
		return false
	}
}

func deepEqualChildren(a, b helium.Node, opts deepEqualOptions) bool {
	ac := a.FirstChild()
	bc := b.FirstChild()
	for ac != nil && bc != nil {
		if !deepEqualNode(ac, bc, opts) {
			return false
		}
		ac = ac.NextSibling()
		bc = bc.NextSibling()
	}
	return ac == nil && bc == nil
}

func deepEqualMap(a, b MapItem, opts deepEqualOptions) (bool, error) {
	if a.Size() != b.Size() {
		return false, nil
	}
	// For each key in a, find a matching key in b using deep-equal comparison
	// (handles cross-type numeric keys like xs:integer(1) == xs:double(1.0))
	aKeys := a.Keys()
	bKeys := b.Keys()
	bUsed := make([]bool, len(bKeys))
	for _, ak := range aKeys {
		found := false
		// Try exact lookup first (fast path)
		if bVal, ok := b.Get(ak); ok {
			aVal, _ := a.Get(ak)
			eq, err := deepEqualSequence(aVal, bVal, opts)
			if err != nil {
				return false, err
			}
			if eq {
				// Mark the matching b key as used
				for j, bk := range bKeys {
					if !bUsed[j] {
						if sameMapKey(ak, bk) {
							bUsed[j] = true
							break
						}
					}
				}
				found = true
			}
		}
		if !found {
			// Slow path: compare keys by value (cross-type)
			aVal, _ := a.Get(ak)
			for j, bk := range bKeys {
				if bUsed[j] {
					continue
				}
				if !sameMapKey(ak, bk) {
					continue
				}
				bVal, _ := b.Get(bk)
				eq, err := deepEqualSequence(aVal, bVal, opts)
				if err != nil {
					return false, err
				}
				if eq {
					bUsed[j] = true
					found = true
					break
				}
			}
		}
		if !found {
			return false, nil
		}
	}
	return true, nil
}

func sameMapKey(a, b AtomicValue) bool {
	return normalizeMapKey(a) == normalizeMapKey(b)
}

func deepEqualArray(a, b ArrayItem, opts deepEqualOptions) (bool, error) {
	am0, bm0 := a.members0(), b.members0()
	if len(am0) != len(bm0) {
		return false, nil
	}
	for i := range am0 {
		am, bm := am0[i], bm0[i]
		eq, err := deepEqualSequence(am, bm, opts)
		if err != nil {
			return false, err
		}
		if !eq {
			return false, nil
		}
	}
	return true, nil
}

func fnIndexOf(ctx context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[1]) == 0 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "index-of: search value must not be empty sequence"}
	}
	search, err := AtomizeItem(args[1].Get(0))
	if err != nil {
		return nil, err
	}
	// Explicit 3rd arg with empty sequence is a type error (xs:string, not xs:string?)
	if len(args) > 2 && seqLen(args[2]) == 0 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "collation argument must not be empty"}
	}
	// Resolve collation: explicit 3rd arg, or default collation
	coll, err := getCollation(ctx, args, 2)
	if err != nil {
		return nil, err
	}
	// codepointCollation is the default — treat as nil for fast path
	if coll == codepointCollation {
		coll = nil
	}
	// Per spec: untypedAtomic values are cast to xs:string for comparison
	if search.TypeName == TypeUntypedAtomic {
		search = AtomicValue{TypeName: TypeString, Value: search.StringVal()}
	}

	// Atomize sequence to handle arrays (XPath 3.1: atomizing flattens arrays)
	atoms, err := AtomizeSequence(args[0])
	if err != nil {
		return nil, err
	}

	var result ItemSlice
	for i, a := range atoms {
		if a.TypeName == TypeUntypedAtomic {
			a = AtomicValue{TypeName: TypeString, Value: a.StringVal()}
		}
		var eq bool
		if coll != nil && isStringDerived(a.TypeName) && isStringDerived(search.TypeName) {
			eq = coll.compare(a.StringVal(), search.StringVal()) == 0
		} else {
			matched, cmpErr := compareAtomic(TokenEq, a, search)
			if cmpErr != nil {
				continue // incomparable types are silently skipped
			}
			eq = matched
		}
		if eq {
			result = append(result, AtomicValue{TypeName: TypeInteger, Value: big.NewInt(int64(i + 1))})
		}
	}
	return result, nil
}

func fnLast(ctx context.Context, _ []Sequence) (Sequence, error) {
	fc := getFnContext(ctx)
	if fc == nil || (fc.node == nil && fc.contextItem == nil) {
		return nil, &XPathError{Code: errCodeXPDY0002, Message: "last() requires a focus (context item)"}
	}
	return SingleInteger(int64(fc.size)), nil
}

func fnPosition(ctx context.Context, _ []Sequence) (Sequence, error) {
	fc := getFnContext(ctx)
	if fc == nil || (fc.node == nil && fc.contextItem == nil) {
		return nil, &XPathError{Code: errCodeXPDY0002, Message: "position() requires a focus (context item)"}
	}
	return SingleInteger(int64(fc.position)), nil
}

// fnSort implements fn:sort($input [, $collation [, $key]])
func fnSort(ctx context.Context, args []Sequence) (Sequence, error) {
	input := args[0]
	if seqLen(input) <= 1 {
		return input, nil
	}

	// Optional collation (2nd argument)
	var coll *collationImpl
	if len(args) >= 2 && seqLen(args[1]) > 0 {
		uri, err := coerceArgToString(args[1])
		if err != nil {
			return nil, err
		}
		if uri != "" {
			coll, err = resolveCollation(uri, "")
			if err != nil {
				return nil, err
			}
		}
	}

	// Optional key function (3rd argument)
	var keyFn *FunctionItem
	if len(args) >= 3 && seqLen(args[2]) > 0 {
		fi, err := extractFunctionItem(args[2])
		if err != nil {
			return nil, err
		}
		keyFn = &fi
	}

	// Compute sort keys
	type sortPair struct {
		item Item
		key  Sequence
	}
	inputItems := seqMaterialize(input)
	pairs := make([]sortPair, len(inputItems))
	for i, item := range inputItems {
		if keyFn != nil {
			k, err := keyFn.Invoke(ctx, []Sequence{ItemSlice{item}})
			if err != nil {
				return nil, err
			}
			pairs[i] = sortPair{item: item, key: k}
		} else {
			// Default key: fn:data() semantics (atomize, flattening arrays)
			atoms, err := AtomizeSequence(ItemSlice{item})
			if err != nil {
				pairs[i] = sortPair{item: item, key: nil}
			} else {
				key := make(ItemSlice, len(atoms))
				for k, a := range atoms {
					key[k] = a
				}
				pairs[i] = sortPair{item: item, key: key}
			}
		}
	}

	// Stable sort using comparison — compare key sequences lexicographically
	var sortErr error
	sort.SliceStable(pairs, func(i, j int) bool {
		if sortErr != nil {
			return false
		}
		ki := pairs[i].key
		kj := pairs[j].key
		minLen := seqLen(ki)
		if seqLen(kj) < minLen {
			minLen = seqLen(kj)
		}
		for idx := 0; idx < minLen; idx++ {
			ai, err1 := AtomizeItem(ki.Get(idx))
			aj, err2 := AtomizeItem(kj.Get(idx))
			if err1 != nil || err2 != nil {
				return false
			}
			cmp, err := valueCompareThreeWay(ai, aj, coll)
			if err != nil {
				sortErr = err
				return false
			}
			if cmp < 0 {
				return true
			}
			if cmp > 0 {
				return false
			}
			// Equal — continue to next key component
		}
		// All compared items equal; shorter key comes first
		return seqLen(ki) < seqLen(kj)
	})
	if sortErr != nil {
		return nil, sortErr
	}

	result := make(ItemSlice, len(pairs))
	for i, p := range pairs {
		result[i] = p.item
	}
	return result, nil
}

func fnFlatten(_ context.Context, args []Sequence) (Sequence, error) {
	var result ItemSlice
	flattenItems(args[0], &result)
	return result, nil
}

func flattenItems(seq Sequence, result *ItemSlice) {
	for item := range seqItems(seq) {
		if arr, ok := item.(ArrayItem); ok {
			for _, m := range arr.members0() {
				flattenItems(m, result)
			}
		} else {
			*result = append(*result, item)
		}
	}
}
