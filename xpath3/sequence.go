package xpath3

import (
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/sequence"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// NewRangeSequence creates a lazy integer range sequence from start to end (inclusive).
// If start > end, returns nil (empty sequence per XPath spec).
func NewRangeSequence(start, end int64) Sequence {
	if start > end {
		return nil
	}
	n := int(end - start + 1)
	return sequence.NewRange(n, func(i int) Item {
		return AtomicValue{TypeName: TypeInteger, Value: start + int64(i)}
	})
}

// SingleNode creates a Sequence containing a single NodeItem.
func SingleNode(n helium.Node) Sequence {
	return ItemSlice{NodeItem{Node: n}}
}

// SingleAtomic creates a Sequence containing a single AtomicValue.
func SingleAtomic(v AtomicValue) Sequence {
	return ItemSlice{v}
}

// Pre-allocated singleton sequences for common boolean values.
var (
	seqTrue  Sequence = ItemSlice{AtomicValue{TypeName: TypeBoolean, Value: true}}
	seqFalse Sequence = ItemSlice{AtomicValue{TypeName: TypeBoolean, Value: false}}
)

// SingleBoolean creates a Sequence containing a single xs:boolean.
func SingleBoolean(b bool) Sequence {
	if b {
		return seqTrue
	}
	return seqFalse
}

// SingleInteger creates a Sequence containing a single xs:integer from int64.
func SingleInteger(n int64) Sequence {
	return ItemSlice{AtomicValue{TypeName: TypeInteger, Value: n}}
}

// SingleIntegerBig creates a Sequence containing a single xs:integer from *big.Int.
func SingleIntegerBig(n *big.Int) Sequence {
	return ItemSlice{AtomicValue{TypeName: TypeInteger, Value: n}}
}

// SingleDecimal creates a Sequence containing a single xs:decimal from *big.Rat.
func SingleDecimal(r *big.Rat) Sequence {
	return ItemSlice{AtomicValue{TypeName: TypeDecimal, Value: r}}
}

// SingleDouble creates a Sequence containing a single xs:double.
func SingleDouble(f float64) Sequence {
	return ItemSlice{AtomicValue{TypeName: TypeDouble, Value: NewDouble(f)}}
}

// SingleFloat creates a Sequence containing a single xs:float.
func SingleFloat(f float64) Sequence {
	return ItemSlice{AtomicValue{TypeName: TypeFloat, Value: NewFloat(f)}}
}

// SingleString creates a Sequence containing a single xs:string.
func SingleString(s string) Sequence {
	return ItemSlice{AtomicValue{TypeName: TypeString, Value: s}}
}

// EmptySequence returns an empty Sequence.
func EmptySequence() Sequence {
	return nil
}

// NodesFrom extracts all helium.Node values from a sequence.
// Returns (nodes, true) if all items are NodeItems, or (nil, false) if any are not.
func NodesFrom(seq Sequence) ([]helium.Node, bool) {
	if seqLen(seq) == 0 {
		return nil, true
	}
	nodes := make([]helium.Node, 0, seq.Len())
	for item := range seqItems(seq) {
		ni, ok := item.(NodeItem)
		if !ok {
			return nil, false
		}
		nodes = append(nodes, ni.Node)
	}
	return nodes, true
}

// AtomizeSequence atomizes all items in a sequence per XPath 3.1 Section 2.6.2.
func AtomizeSequence(seq Sequence) ([]AtomicValue, error) {
	if seq == nil {
		return nil, nil
	}
	result := make([]AtomicValue, 0, seq.Len())
	for item := range seqItems(seq) {
		// XPath 3.1: atomizing an array flattens its members
		if arr, ok := item.(ArrayItem); ok {
			for _, member := range arr.members0() {
				atoms, err := AtomizeSequence(member)
				if err != nil {
					return nil, err
				}
				result = append(result, atoms...)
			}
			continue
		}
		// List types: split whitespace-separated tokens and atomize each.
		if ni, ok := item.(NodeItem); ok {
			listItem := ni.ListItemType
			if listItem == "" {
				listItem = builtinListItemType(ni.TypeAnnotation)
			}
			if listItem != "" {
				s := ixpath.StringValue(ni.Node)
				tokens := strings.Fields(s)
				for _, tok := range tokens {
					cast, err := CastFromString(tok, listItem)
					if err != nil {
						// For user-defined schema types (Q{ns}local),
						// the value was already validated during
						// construction; store as string with the type name.
						if strings.HasPrefix(listItem, "Q{") {
							cast = AtomicValue{TypeName: listItem, Value: tok}
						} else {
							return nil, err
						}
					}
					result = append(result, cast)
				}
				continue
			}
		}
		av, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		result = append(result, av)
	}
	return result, nil
}

// builtinListItemType returns the item type for built-in XSD list types.
func builtinListItemType(typeName string) string {
	switch typeName {
	case TypeNMTOKENS:
		return TypeNMTOKEN
	case TypeIDREFS:
		return TypeIDREF
	case TypeENTITIES:
		return TypeENTITY
	}
	return ""
}

// EBV computes the Effective Boolean Value of a sequence per XPath 3.1 Section 2.4.3.
func EBV(seq Sequence) (bool, error) {
	n := seqLen(seq)
	if n == 0 {
		return false, nil
	}

	first := seq.Get(0)
	// Sequence starting with a node → true
	if _, ok := first.(NodeItem); ok {
		return true, nil
	}

	if n == 1 {
		return ebvSingle(first)
	}

	return false, &XPathError{
		Code:    errCodeFORG0006,
		Message: "effective boolean value not defined for sequence of length > 1 starting with non-node",
	}
}

func ebvSingle(item Item) (bool, error) {
	switch v := item.(type) {
	case AtomicValue:
		return ebvAtomic(v)
	case NodeItem:
		return true, nil
	default:
		return false, &XPathError{
			Code:    errCodeFORG0006,
			Message: fmt.Sprintf("effective boolean value not defined for %T", item),
		}
	}
}

func ebvAtomic(v AtomicValue) (bool, error) {
	// Per XPath 3.1 §2.4.3, EBV is defined for: boolean, string, anyURI,
	// untypedAtomic, string-derived types, and numeric types only.
	switch v.TypeName {
	case TypeBoolean:
		return v.Value.(bool), nil //nolint:forcetypeassert
	case TypeString, TypeAnyURI, TypeUntypedAtomic:
		s, _ := v.Value.(string)
		return s != "", nil
	}
	if isStringDerived(v.TypeName) {
		s, _ := v.Value.(string)
		return s != "", nil
	}
	if v.IsNumeric() {
		switch val := v.Value.(type) {
		case int64:
			return val != 0, nil
		case *big.Int:
			return val.Sign() != 0, nil
		case *big.Rat:
			return val.Sign() != 0, nil
		case *FloatValue:
			f := val.Float64()
			return f != 0 && !math.IsNaN(f), nil
		}
	}
	return false, &XPathError{
		Code:    errCodeFORG0006,
		Message: fmt.Sprintf("effective boolean value not defined for %s", v.TypeName),
	}
}
