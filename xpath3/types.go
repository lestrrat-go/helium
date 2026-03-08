package xpath3

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	helium "github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// Item is the interface implemented by all XPath 3.1 item types.
type Item interface {
	itemTag()
}

// Sequence is an ordered collection of items.
type Sequence []Item

// --- NodeItem ---

// NodeItem wraps a helium.Node as an XPath item.
type NodeItem struct {
	Node helium.Node
}

func (NodeItem) itemTag() {}

// --- AtomicValue ---

// Atomic type name constants matching XSD types.
const (
	TypeString             = "xs:string"
	TypeInteger            = "xs:integer"
	TypeDecimal            = "xs:decimal"
	TypeDouble             = "xs:double"
	TypeFloat              = "xs:float"
	TypeBoolean            = "xs:boolean"
	TypeDate               = "xs:date"
	TypeDateTime           = "xs:dateTime"
	TypeTime               = "xs:time"
	TypeDuration           = "xs:duration"
	TypeDayTimeDuration    = "xs:dayTimeDuration"
	TypeYearMonthDuration  = "xs:yearMonthDuration"
	TypeAnyURI             = "xs:anyURI"
	TypeQName              = "xs:QName"
	TypeBase64Binary       = "xs:base64Binary"
	TypeHexBinary          = "xs:hexBinary"
	TypeUntypedAtomic      = "xs:untypedAtomic"
	TypeAnyAtomicType      = "xs:anyAtomicType"

	// Derived integer types
	TypeLong               = "xs:long"
	TypeInt                = "xs:int"
	TypeShort              = "xs:short"
	TypeByte               = "xs:byte"
	TypeUnsignedLong       = "xs:unsignedLong"
	TypeUnsignedInt        = "xs:unsignedInt"
	TypeUnsignedShort      = "xs:unsignedShort"
	TypeUnsignedByte       = "xs:unsignedByte"
	TypeNonNegativeInteger = "xs:nonNegativeInteger"
	TypeNonPositiveInteger = "xs:nonPositiveInteger"
	TypePositiveInteger    = "xs:positiveInteger"
	TypeNegativeInteger    = "xs:negativeInteger"

	// Derived string types
	TypeNormalizedString = "xs:normalizedString"
	TypeToken            = "xs:token"
	TypeLanguage         = "xs:language"
	TypeName             = "xs:Name"
	TypeNCName           = "xs:NCName"
	TypeNMTOKEN          = "xs:NMTOKEN"
	TypeNMTOKENS         = "xs:NMTOKENS"
	TypeENTITY           = "xs:ENTITY"
	TypeID               = "xs:ID"
	TypeIDREF            = "xs:IDREF"
	TypeIDREFS           = "xs:IDREFS"

	// Gregorian date part types
	TypeGDay       = "xs:gDay"
	TypeGMonth     = "xs:gMonth"
	TypeGMonthDay  = "xs:gMonthDay"
	TypeGYear      = "xs:gYear"
	TypeGYearMonth = "xs:gYearMonth"

	// Other derived types
	TypeDateTimeStamp = "xs:dateTimeStamp"
	TypeError         = "xs:error"
	TypeNumeric       = "xs:numeric"
)

// AtomicValue represents an XSD atomic value with its type name.
type AtomicValue struct {
	TypeName string // e.g. "xs:string", "xs:integer"
	Value    any    // Go native backing value (see type table in design doc)
}

func (AtomicValue) itemTag() {}

// StringVal returns the backing string value. Panics if not a string-backed type.
func (a AtomicValue) StringVal() string {
	return a.Value.(string)
}

// IntegerVal returns the backing int64 value.
func (a AtomicValue) IntegerVal() int64 {
	return a.Value.(int64)
}

// DoubleVal returns the backing float64 value.
func (a AtomicValue) DoubleVal() float64 {
	return a.Value.(float64)
}

// BooleanVal returns the backing bool value.
func (a AtomicValue) BooleanVal() bool {
	return a.Value.(bool)
}

// TimeVal returns the backing time.Time value.
func (a AtomicValue) TimeVal() time.Time {
	return a.Value.(time.Time)
}

// DurationVal returns the backing Duration value.
func (a AtomicValue) DurationVal() Duration {
	return a.Value.(Duration)
}

// BytesVal returns the backing []byte value.
func (a AtomicValue) BytesVal() []byte {
	return a.Value.([]byte)
}

// QNameVal returns the backing QNameValue.
func (a AtomicValue) QNameVal() QNameValue {
	return a.Value.(QNameValue)
}

// IsNumeric returns true if the type is xs:integer (or derived), xs:decimal, xs:double, or xs:float.
func (a AtomicValue) IsNumeric() bool {
	if isIntegerDerived(a.TypeName) {
		return true
	}
	switch a.TypeName {
	case TypeDecimal, TypeDouble, TypeFloat:
		return true
	}
	return false
}

// ToFloat64 converts any numeric atomic value to float64.
func (a AtomicValue) ToFloat64() float64 {
	if isIntegerDerived(a.TypeName) {
		return float64(a.Value.(int64))
	}
	switch a.TypeName {
	case TypeDouble, TypeFloat:
		return a.Value.(float64)
	case TypeDecimal:
		// v1: decimal stored as string, parse to float64 for arithmetic
		s := a.Value.(string)
		var f float64
		if _, err := fmt.Sscanf(s, "%f", &f); err != nil {
			return math.NaN()
		}
		return f
	}
	return math.NaN()
}

// String returns a human-readable representation.
func (a AtomicValue) String() string {
	return fmt.Sprintf("%s(%v)", a.TypeName, a.Value)
}

// --- Supporting Types ---

// QNameValue holds the components of an xs:QName value.
type QNameValue struct {
	Prefix string
	Local  string
	URI    string
}

// Duration represents an XSD duration (used for xs:duration, xs:dayTimeDuration,
// xs:yearMonthDuration).
type Duration struct {
	Months  int // total months (years*12 + months)
	Seconds float64 // total seconds (days*86400 + hours*3600 + minutes*60 + seconds)
	Negative bool
}

// --- FunctionItem ---

// FunctionItem represents a callable function value (inline, named ref, partial application).
type FunctionItem struct {
	Arity  int
	Name   string // empty for anonymous
	Invoke func(ctx context.Context, args []Sequence) (Sequence, error)
}

func (FunctionItem) itemTag() {}

// --- MapItem ---

// MapItem is an immutable key-value map. Put returns a new map (copy-on-write).
type MapItem struct {
	entries []mapEntry
	index   map[mapKey]int // index into entries
}

func (MapItem) itemTag() {}

// MapEntry is a public key-value pair for constructing maps.
type MapEntry struct {
	Key   AtomicValue
	Value Sequence
}

type mapEntry struct {
	key   AtomicValue
	value Sequence
}

// mapKey is the normalized key used for O(1) lookup in the index map.
// time.Time is normalized to UTC RFC3339Nano string, []byte to hex string.
type mapKey struct {
	typeName string
	value    any // only types with Go map equality: string, int64, float64, bool
}

func normalizeMapKey(key AtomicValue) mapKey {
	switch v := key.Value.(type) {
	case time.Time:
		return mapKey{typeName: key.TypeName, value: v.UTC().Format(time.RFC3339Nano)}
	case []byte:
		return mapKey{typeName: key.TypeName, value: hex.EncodeToString(v)}
	default:
		return mapKey{typeName: key.TypeName, value: key.Value}
	}
}

// NewMap creates a MapItem from a slice of MapEntry.
func NewMap(entries []MapEntry) MapItem {
	m := MapItem{
		entries: make([]mapEntry, len(entries)),
		index:   make(map[mapKey]int, len(entries)),
	}
	for i, e := range entries {
		m.entries[i] = mapEntry{key: e.Key, value: e.Value}
		m.index[normalizeMapKey(e.Key)] = i
	}
	return m
}

// Get returns the value associated with key, or (nil, false) if not found.
func (m MapItem) Get(key AtomicValue) (Sequence, bool) {
	idx, ok := m.index[normalizeMapKey(key)]
	if !ok {
		return nil, false
	}
	return m.entries[idx].value, true
}

// Put returns a new map with the given key-value pair added or replaced.
func (m MapItem) Put(key AtomicValue, value Sequence) MapItem {
	nk := normalizeMapKey(key)
	newEntries := make([]mapEntry, len(m.entries))
	copy(newEntries, m.entries)
	newIndex := make(map[mapKey]int, len(m.index)+1)
	for k, v := range m.index {
		newIndex[k] = v
	}

	if idx, ok := newIndex[nk]; ok {
		newEntries[idx] = mapEntry{key: key, value: value}
	} else {
		newEntries = append(newEntries, mapEntry{key: key, value: value})
		newIndex[nk] = len(newEntries) - 1
	}
	return MapItem{entries: newEntries, index: newIndex}
}

// Contains returns true if the map contains the given key.
func (m MapItem) Contains(key AtomicValue) bool {
	_, ok := m.index[normalizeMapKey(key)]
	return ok
}

// Keys returns all keys in insertion order.
func (m MapItem) Keys() []AtomicValue {
	keys := make([]AtomicValue, len(m.entries))
	for i, e := range m.entries {
		keys[i] = e.key
	}
	return keys
}

// Size returns the number of entries.
func (m MapItem) Size() int {
	return len(m.entries)
}

// ForEach calls fn for each entry in insertion order. Stops on first error.
func (m MapItem) ForEach(fn func(AtomicValue, Sequence) error) error {
	for _, e := range m.entries {
		if err := fn(e.key, e.value); err != nil {
			return err
		}
	}
	return nil
}

// Remove returns a new map with the given key removed.
func (m MapItem) Remove(key AtomicValue) MapItem {
	nk := normalizeMapKey(key)
	idx, ok := m.index[nk]
	if !ok {
		return m
	}
	newEntries := make([]mapEntry, 0, len(m.entries)-1)
	newIndex := make(map[mapKey]int, len(m.index)-1)
	for i, e := range m.entries {
		if i == idx {
			continue
		}
		newIndex[normalizeMapKey(e.key)] = len(newEntries)
		newEntries = append(newEntries, e)
	}
	return MapItem{entries: newEntries, index: newIndex}
}

// MergePolicy controls how duplicate keys are handled in MergeMaps.
type MergePolicy int

const (
	MergeUseFirst MergePolicy = iota
	MergeUseLast
	MergeReject
)

// MergeMaps merges multiple maps according to the given policy.
func MergeMaps(maps []MapItem, policy MergePolicy) (MapItem, error) {
	var allEntries []mapEntry
	seen := map[mapKey]int{}

	for _, m := range maps {
		for _, e := range m.entries {
			nk := normalizeMapKey(e.key)
			if idx, ok := seen[nk]; ok {
				switch policy {
				case MergeUseFirst:
					continue
				case MergeUseLast:
					allEntries[idx] = e
					continue
				case MergeReject:
					return MapItem{}, &XPathError{
						Code:    "FOJS0003",
						Message: fmt.Sprintf("duplicate key in map merge: %v", e.key.Value),
					}
				}
			}
			seen[nk] = len(allEntries)
			allEntries = append(allEntries, e)
		}
	}

	newIndex := make(map[mapKey]int, len(allEntries))
	for i, e := range allEntries {
		newIndex[normalizeMapKey(e.key)] = i
	}
	return MapItem{entries: allEntries, index: newIndex}, nil
}

// --- ArrayItem ---

// ArrayItem is an immutable indexed sequence of members. Uses 1-based indexing.
type ArrayItem struct {
	members []Sequence
}

func (ArrayItem) itemTag() {}

// NewArray creates an ArrayItem from a slice of Sequence members.
func NewArray(members []Sequence) ArrayItem {
	return ArrayItem{members: members}
}

// Get returns the member at the given 1-based index.
func (a ArrayItem) Get(index int) (Sequence, error) {
	if index < 1 || index > len(a.members) {
		return nil, &XPathError{
			Code:    "FOAY0001",
			Message: fmt.Sprintf("array index %d out of bounds (size %d)", index, len(a.members)),
		}
	}
	return a.members[index-1], nil
}

// Size returns the number of members.
func (a ArrayItem) Size() int {
	return len(a.members)
}

// Put returns a new array with the member at the given 1-based index replaced.
func (a ArrayItem) Put(index int, value Sequence) (ArrayItem, error) {
	if index < 1 || index > len(a.members) {
		return ArrayItem{}, &XPathError{
			Code:    "FOAY0001",
			Message: fmt.Sprintf("array index %d out of bounds (size %d)", index, len(a.members)),
		}
	}
	newMembers := make([]Sequence, len(a.members))
	copy(newMembers, a.members)
	newMembers[index-1] = value
	return ArrayItem{members: newMembers}, nil
}

// Append returns a new array with the value appended.
func (a ArrayItem) Append(value Sequence) ArrayItem {
	newMembers := make([]Sequence, len(a.members)+1)
	copy(newMembers, a.members)
	newMembers[len(a.members)] = value
	return ArrayItem{members: newMembers}
}

// Members returns all members.
func (a ArrayItem) Members() []Sequence {
	return a.members
}

// SubArray returns a new array from start to end (1-based, inclusive).
func (a ArrayItem) SubArray(start, length int) (ArrayItem, error) {
	if start < 1 || length < 0 || start+length-1 > len(a.members) {
		return ArrayItem{}, &XPathError{
			Code:    "FOAY0001",
			Message: fmt.Sprintf("array subarray(%d, %d) out of bounds (size %d)", start, length, len(a.members)),
		}
	}
	newMembers := make([]Sequence, length)
	copy(newMembers, a.members[start-1:start-1+length])
	return ArrayItem{members: newMembers}, nil
}

// Flatten returns all members concatenated into a single sequence.
// Nested arrays are recursively flattened per XPath 3.1 spec.
func (a ArrayItem) Flatten() Sequence {
	var result Sequence
	for _, m := range a.members {
		for _, item := range m {
			if nested, ok := item.(ArrayItem); ok {
				result = append(result, nested.Flatten()...)
			} else {
				result = append(result, item)
			}
		}
	}
	return result
}

// --- Atomization ---

// AtomizeItem converts a single item to an atomic value per XPath 3.1 Section 2.6.2.
func AtomizeItem(item Item) (AtomicValue, error) {
	switch v := item.(type) {
	case AtomicValue:
		return v, nil
	case NodeItem:
		return AtomicValue{
			TypeName: TypeUntypedAtomic,
			Value:    ixpath.StringValue(v.Node),
		}, nil
	default:
		return AtomicValue{}, &XPathError{
			Code:    "FOTY0013",
			Message: fmt.Sprintf("cannot atomize %T", item),
		}
	}
}
