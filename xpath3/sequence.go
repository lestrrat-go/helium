package xpath3

import (
	"fmt"
	"math"
	"math/big"

	"github.com/lestrrat-go/helium"
)

// SingleNode creates a Sequence containing a single NodeItem.
func SingleNode(n helium.Node) Sequence {
	return Sequence{NodeItem{Node: n}}
}

// SingleAtomic creates a Sequence containing a single AtomicValue.
func SingleAtomic(v AtomicValue) Sequence {
	return Sequence{v}
}

// SingleBoolean creates a Sequence containing a single xs:boolean.
func SingleBoolean(b bool) Sequence {
	return Sequence{AtomicValue{TypeName: TypeBoolean, Value: b}}
}

// SingleInteger creates a Sequence containing a single xs:integer from int64.
func SingleInteger(n int64) Sequence {
	return Sequence{AtomicValue{TypeName: TypeInteger, Value: big.NewInt(n)}}
}

// SingleIntegerBig creates a Sequence containing a single xs:integer from *big.Int.
func SingleIntegerBig(n *big.Int) Sequence {
	return Sequence{AtomicValue{TypeName: TypeInteger, Value: n}}
}

// SingleDecimal creates a Sequence containing a single xs:decimal from *big.Rat.
func SingleDecimal(r *big.Rat) Sequence {
	return Sequence{AtomicValue{TypeName: TypeDecimal, Value: r}}
}

// SingleDouble creates a Sequence containing a single xs:double.
func SingleDouble(f float64) Sequence {
	return Sequence{AtomicValue{TypeName: TypeDouble, Value: f}}
}

// SingleString creates a Sequence containing a single xs:string.
func SingleString(s string) Sequence {
	return Sequence{AtomicValue{TypeName: TypeString, Value: s}}
}

// EmptySequence returns an empty Sequence.
func EmptySequence() Sequence {
	return nil
}

// NodesFrom extracts all helium.Node values from a sequence.
// Returns (nodes, true) if all items are NodeItems, or (nil, false) if any are not.
func NodesFrom(seq Sequence) ([]helium.Node, bool) {
	if len(seq) == 0 {
		return nil, true
	}
	nodes := make([]helium.Node, 0, len(seq))
	for _, item := range seq {
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
	result := make([]AtomicValue, 0, len(seq))
	for _, item := range seq {
		// XPath 3.1: atomizing an array flattens its members
		if arr, ok := item.(ArrayItem); ok {
			for i := 1; i <= arr.Size(); i++ {
				member, _ := arr.Get(i)
				atoms, err := AtomizeSequence(member)
				if err != nil {
					return nil, err
				}
				result = append(result, atoms...)
			}
			continue
		}
		av, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		result = append(result, av)
	}
	return result, nil
}

// EBV computes the Effective Boolean Value of a sequence per XPath 3.1 Section 2.4.3.
func EBV(seq Sequence) (bool, error) {
	if len(seq) == 0 {
		return false, nil
	}

	// Sequence starting with a node → true
	if _, ok := seq[0].(NodeItem); ok {
		return true, nil
	}

	if len(seq) == 1 {
		return ebvSingle(seq[0])
	}

	return false, &XPathError{
		Code:    "FORG0006",
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
			Code:    "FORG0006",
			Message: fmt.Sprintf("effective boolean value not defined for %T", item),
		}
	}
}

func ebvAtomic(v AtomicValue) (bool, error) {
	// Check backing type first to avoid panics from mismatched TypeName/Value
	switch val := v.Value.(type) {
	case *big.Int:
		return val.Sign() != 0, nil
	case *big.Rat:
		return val.Sign() != 0, nil
	case float64:
		return val != 0 && !math.IsNaN(val), nil
	case bool:
		return val, nil
	case string:
		return val != "", nil
	}
	return false, &XPathError{
		Code:    "FORG0006",
		Message: fmt.Sprintf("effective boolean value not defined for %s", v.TypeName),
	}
}
