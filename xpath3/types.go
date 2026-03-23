package xpath3

import (
	"context"
	"encoding/hex"
	"fmt"
	"maps"
	"math"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/sequence"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// Item is the interface implemented by all XPath 3.1 item types.
type Item interface {
	itemTag()
}

// Sequence is an ordered collection of XPath 3.1 items.
// Implementations include ItemSlice (slice-backed) and lazy sequences via sequence.Range.
type Sequence = sequence.Interface[Item]

// ItemSlice is a Sequence backed by a plain slice of Item values.
type ItemSlice = sequence.Slice[Item]

// Nil-safe helpers — delegate to generic sequence package.
var (
	seqLen         = sequence.Len[Item]
	seqItems       = sequence.Items[Item]
	seqMaterialize = sequence.Materialize[Item]
)

var cloneSequence = sequence.Clone[Item]

// CloneSequence returns a deep copy of the Sequence.
func CloneSequence(seq Sequence) Sequence {
	return cloneSequence(seq)
}

func cloneSequences(seqs []Sequence) []Sequence {
	if seqs == nil {
		return nil
	}
	cloned := make([]Sequence, len(seqs))
	for i, seq := range seqs {
		cloned[i] = cloneSequence(seq)
	}
	return cloned
}

// --- NodeItem ---

// NodeItem wraps a helium.Node as an XPath item.
type NodeItem struct {
	Node           helium.Node
	TypeAnnotation string // optional xs:... type annotation (schema-aware)
	AtomizedType   string // optional built-in base type used for typed atomization
	ListItemType   string // non-empty when the type is a list; the item type name
}

func (NodeItem) itemTag() {}

// --- AtomicValue ---

// Atomic type name constants matching XSD types.
const (
	TypeString            = "xs:string"
	TypeInteger           = "xs:integer"
	TypeDecimal           = "xs:decimal"
	TypeDouble            = "xs:double"
	TypeFloat             = "xs:float"
	TypeBoolean           = "xs:boolean"
	TypeDate              = "xs:date"
	TypeDateTime          = "xs:dateTime"
	TypeTime              = "xs:time"
	TypeDuration          = "xs:duration"
	TypeDayTimeDuration   = "xs:dayTimeDuration"
	TypeYearMonthDuration = "xs:yearMonthDuration"
	TypeAnyURI            = "xs:anyURI"
	TypeQName             = "xs:QName"
	TypeBase64Binary      = "xs:base64Binary"
	TypeHexBinary         = "xs:hexBinary"
	TypeUntypedAtomic     = "xs:untypedAtomic"
	TypeAnyAtomicType     = "xs:anyAtomicType"

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
	TypeENTITIES         = "xs:ENTITIES"

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

	// Non-atomic / structural types
	TypeAnyType       = "xs:anyType"
	TypeAnySimpleType = "xs:anySimpleType"
	TypeUntyped       = "xs:untyped"
	TypeNOTATION      = "xs:NOTATION"
)

// QAnnotation returns the Q{ns}local annotation format for a namespace URI and local name.
func QAnnotation(ns, local string) string {
	return "Q{" + ns + "}" + local
}

// isSubtypeOf returns true if actualType is the same as or a subtype of targetType
// per the XSD type hierarchy.
// subtypeCache caches isSubtypeOf results. The type hierarchy is static,
// so results are deterministic and safe to cache globally.
var subtypeCache sync.Map // key: [2]string{actual,target} → bool

func isSubtypeOf(actualType, targetType string) bool {
	if actualType == targetType {
		return true
	}
	key := [2]string{actualType, targetType}
	if v, ok := subtypeCache.Load(key); ok {
		return v.(bool)
	}
	result := computeIsSubtypeOf(actualType, targetType)
	subtypeCache.Store(key, result)
	return result
}

// BuiltinIsSubtypeOf reports whether actualType is the same as or a subtype of
// targetType using only the built-in XSD type hierarchy (no schema lookup).
// This is exported for use by schema-aware backends (e.g. xslt3) that need to
// check the final leg of a type ancestry chain against a built-in base type.
func BuiltinIsSubtypeOf(actualType, targetType string) bool {
	return isSubtypeOf(actualType, targetType)
}

func computeIsSubtypeOf(actualType, targetType string) bool {
	// xs:numeric is a union of xs:integer, xs:decimal, xs:float, xs:double
	if targetType == TypeNumeric {
		return isSubtypeOf(actualType, TypeDecimal) ||
			actualType == TypeFloat || actualType == TypeDouble
	}
	// Walk up the type hierarchy
	cur := actualType
	for {
		parent, ok := xsdTypeParent[cur]
		if !ok {
			return false
		}
		if parent == targetType {
			return true
		}
		cur = parent
	}
}

// xsdTypeParent maps each derived XSD type to its parent (base) type.
var xsdTypeParent = map[string]string{
	// Numeric hierarchy
	TypeByte:               TypeShort,
	TypeShort:              TypeInt,
	TypeInt:                TypeLong,
	TypeLong:               TypeInteger,
	TypeUnsignedByte:       TypeUnsignedShort,
	TypeUnsignedShort:      TypeUnsignedInt,
	TypeUnsignedInt:        TypeUnsignedLong,
	TypeUnsignedLong:       TypeNonNegativeInteger,
	TypePositiveInteger:    TypeNonNegativeInteger,
	TypeNonNegativeInteger: TypeInteger,
	TypeNegativeInteger:    TypeNonPositiveInteger,
	TypeNonPositiveInteger: TypeInteger,
	TypeInteger:            TypeDecimal,
	TypeDecimal:            TypeAnyAtomicType,
	TypeFloat:              TypeAnyAtomicType,
	TypeDouble:             TypeAnyAtomicType,
	// Duration hierarchy
	TypeDayTimeDuration:   TypeDuration,
	TypeYearMonthDuration: TypeDuration,
	TypeDuration:          TypeAnyAtomicType,
	// Date/time hierarchy
	TypeDateTimeStamp: TypeDateTime,
	TypeDateTime:      TypeAnyAtomicType,
	TypeDate:          TypeAnyAtomicType,
	TypeTime:          TypeAnyAtomicType,
	// String hierarchy
	TypeNormalizedString: TypeString,
	TypeToken:            TypeNormalizedString,
	TypeLanguage:         TypeToken,
	TypeNMTOKEN:          TypeToken,
	TypeName:             TypeToken,
	TypeNCName:           TypeName,
	TypeID:               TypeNCName,
	TypeIDREF:            TypeNCName,
	TypeENTITY:           TypeNCName,
	TypeString:           TypeAnyAtomicType,
	// Other types
	TypeBoolean:       TypeAnyAtomicType,
	TypeAnyURI:        TypeAnyAtomicType,
	TypeQName:         TypeAnyAtomicType,
	TypeBase64Binary:  TypeAnyAtomicType,
	TypeHexBinary:     TypeAnyAtomicType,
	TypeUntypedAtomic: TypeAnyAtomicType,
	// Gregorian types
	TypeGDay:       TypeAnyAtomicType,
	TypeGMonth:     TypeAnyAtomicType,
	TypeGMonthDay:  TypeAnyAtomicType,
	TypeGYear:      TypeAnyAtomicType,
	TypeGYearMonth: TypeAnyAtomicType,
	// Complex type hierarchy (for element/attribute type tests)
	TypeUntyped:       TypeAnyType,
	TypeAnySimpleType: TypeAnyType,
	TypeAnyAtomicType: TypeAnySimpleType,
}

// AtomicValue represents an XSD atomic value with its type name.
type AtomicValue struct {
	TypeName string // e.g. "xs:string", "xs:integer"
	Value    any    // Go native backing value (see type table in design doc)
	BaseType string // built-in base type when TypeName is a user-defined schema type
}

func (AtomicValue) itemTag() {}

// StringVal returns the backing string value. Panics if not a string-backed type.
func (a AtomicValue) StringVal() string {
	return a.Value.(string)
}

// IntegerVal returns the backing int64 value. Panics if the value exceeds int64 range.
func (a AtomicValue) IntegerVal() int64 {
	return a.Value.(*big.Int).Int64()
}

// BigInt returns the backing *big.Int value.
// If the value is stored as a string, it attempts to parse it.
func (a AtomicValue) BigInt() *big.Int {
	if n, ok := a.Value.(*big.Int); ok {
		return n
	}
	if s, ok := a.Value.(string); ok {
		n, ok := new(big.Int).SetString(s, 10)
		if ok {
			return n
		}
	}
	return new(big.Int)
}

// BigRat returns the backing *big.Rat value.
// If the value is stored as a string (e.g. from a type annotation mismatch),
// it attempts to parse it.
func (a AtomicValue) BigRat() *big.Rat {
	if r, ok := a.Value.(*big.Rat); ok {
		return r
	}
	if s, ok := a.Value.(string); ok {
		r, ok := new(big.Rat).SetString(s)
		if ok {
			return r
		}
	}
	// Last resort: return zero
	return new(big.Rat)
}

// DoubleVal returns the backing float64 value (extracts from *FloatValue).
func (a AtomicValue) DoubleVal() float64 {
	return a.Value.(*FloatValue).Float64()
}

// FloatVal returns the backing *FloatValue.
func (a AtomicValue) FloatVal() *FloatValue {
	return a.Value.(*FloatValue)
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

// IsNaN returns true if the atomic value is NaN (xs:double or xs:float).
func (a AtomicValue) IsNaN() bool {
	if a.TypeName != TypeDouble && a.TypeName != TypeFloat {
		return false
	}
	fv, ok := a.Value.(*FloatValue)
	return ok && fv.IsNaN()
}

// IsNumeric returns true if the type is xs:integer (or derived), xs:decimal, xs:double, or xs:float.
// Also returns true for user-defined types whose underlying value is numeric.
func (a AtomicValue) IsNumeric() bool {
	if isIntegerDerived(a.TypeName) {
		return true
	}
	switch a.TypeName {
	case TypeDecimal, TypeDouble, TypeFloat:
		return true
	}
	// User-defined schema types: check the underlying Go value.
	switch a.Value.(type) {
	case *big.Int, *big.Rat, float64, float32, *FloatValue:
		return true
	}
	return false
}

// ToFloat64 converts any numeric atomic value to float64.
func (a AtomicValue) ToFloat64() float64 {
	if isIntegerDerived(a.TypeName) {
		n, ok := a.Value.(*big.Int)
		if !ok {
			return 0
		}
		f, _ := new(big.Float).SetInt(n).Float64()
		return f
	}
	switch a.TypeName {
	case TypeDouble, TypeFloat:
		return a.Value.(*FloatValue).Float64()
	case TypeDecimal:
		r, ok := a.Value.(*big.Rat)
		if !ok {
			return 0
		}
		f, _ := r.Float64()
		return f
	}
	return 0
}

// String returns a human-readable representation.
func (a AtomicValue) String() string {
	return fmt.Sprintf("%s(%v)", a.TypeName, a.Value)
}

// DecimalToString returns the canonical XSD decimal string for a *big.Rat.
func DecimalToString(r *big.Rat) string {
	if r.IsInt() {
		return r.Num().String()
	}
	// Use enough precision for exact representation, then trim trailing zeros
	n, exact := r.FloatPrec()
	if !exact {
		// Not exactly representable in decimal; use high precision
		s := r.FloatString(20)
		s = strings.TrimRight(s, "0")
		if strings.HasSuffix(s, ".") {
			s += "0"
		}
		return s
	}
	s := r.FloatString(n)
	s = strings.TrimRight(s, "0")
	if strings.HasSuffix(s, ".") {
		s += "0"
	}
	return s
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
// noTZLocation is a sentinel timezone used for XSD date/time values parsed
// without an explicit timezone. Go's time.UTC cannot be used as a sentinel
// because time.Parse returns time.UTC for inputs without a timezone suffix,
// making it impossible to distinguish "12:00:00" (no TZ) from "12:00:00Z" (explicit UTC).
// This location has offset 0 but is a distinct pointer from time.UTC.
var noTZLocation = time.FixedZone("noTZ", 0)

// HasTimezone returns true if the time value has an explicit timezone
// (i.e., was NOT parsed from a timezone-less XSD lexical form).
func HasTimezone(t time.Time) bool {
	return t.Location() != noTZLocation
}

type Duration struct {
	Months   int      // total months (years*12 + months)
	Seconds  float64  // total seconds (days*86400 + hours*3600 + minutes*60 + seconds)
	FracSec  *big.Rat // exact fractional seconds component (the part after decimal in 'S'), nil if integer
	Negative bool
}

// --- FunctionItem ---

// FunctionItem represents a callable function value (inline, named ref, partial application).
type FunctionItem struct {
	Arity      int
	Name       string // empty for anonymous
	Namespace  string // namespace URI (empty for anonymous or default)
	Invoke     func(ctx context.Context, args []Sequence) (Sequence, error)
	ParamTypes []SequenceType // parameter type annotations (nil if untyped)
	ReturnType *SequenceType  // return type annotation (nil if untyped)
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
	// Per XPath 3.1 spec, map keys use value-based equality (eq operator semantics).
	// Numerics that compare equal are the same key: xs:integer(1), xs:double(1.0e0),
	// xs:float(1.0), and xs:decimal(1.0) all refer to the same map entry.
	// xs:string and xs:untypedAtomic are compared as strings.
	// xs:anyURI promotes to xs:string for comparison.
	tn := key.TypeName

	// Normalize string-like types: xs:untypedAtomic and xs:anyURI promote to xs:string
	if tn == TypeUntypedAtomic || tn == TypeAnyURI {
		return mapKey{typeName: TypeString, value: key.Value}
	}

	// Normalize all numeric types to a common representation using exact
	// rational arithmetic so that precision is not lost across types.
	if key.IsNumeric() {
		f := key.ToFloat64()
		if math.IsNaN(f) {
			return mapKey{typeName: "numeric", value: "NaN"}
		}
		if math.IsInf(f, 1) {
			return mapKey{typeName: "numeric", value: "+Inf"}
		}
		if math.IsInf(f, -1) {
			return mapKey{typeName: "numeric", value: "-Inf"}
		}
		// Convert to big.Rat for exact comparison across numeric types
		var r *big.Rat
		switch key.TypeName {
		case TypeDecimal:
			r = new(big.Rat).Set(key.BigRat())
		case TypeInteger:
			r = new(big.Rat).SetInt(key.BigInt())
		default: // float, double
			r = new(big.Rat).SetFloat64(f)
		}
		return mapKey{typeName: "numeric", value: r.RatString()}
	}

	// Normalize duration types: xs:duration, xs:yearMonthDuration, xs:dayTimeDuration
	// that have equal values should be the same key
	if tn == TypeDuration || tn == TypeYearMonthDuration || tn == TypeDayTimeDuration {
		tn = TypeDuration
	}

	switch v := key.Value.(type) {
	case time.Time:
		// Per XPath 3.1, dateTimes with and without timezone are not equal
		// (they are incomparable), so they must be different map keys.
		if HasTimezone(v) {
			return mapKey{typeName: tn, value: v.UTC().Format(time.RFC3339Nano)}
		}
		return mapKey{typeName: tn, value: "notz:" + v.Format("2006-01-02T15:04:05.999999999")}
	case []byte:
		return mapKey{typeName: tn, value: hex.EncodeToString(v)}
	case QNameValue:
		// QName equality is based on URI and local name, not prefix
		return mapKey{typeName: tn, value: v.URI + "\x00" + v.Local}
	default:
		return mapKey{typeName: tn, value: key.Value}
	}
}

// NewMap creates a MapItem from a slice of MapEntry.
func NewMap(entries []MapEntry) MapItem {
	m := MapItem{
		entries: make([]mapEntry, len(entries)),
		index:   make(map[mapKey]int, len(entries)),
	}
	for i, e := range entries {
		m.entries[i] = mapEntry{key: e.Key, value: cloneSequence(e.Value)}
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
	return cloneSequence(m.entries[idx].value), true
}

// Put returns a new map with the given key-value pair added or replaced.
func (m MapItem) Put(key AtomicValue, value Sequence) MapItem {
	nk := normalizeMapKey(key)
	newEntries := make([]mapEntry, len(m.entries))
	copy(newEntries, m.entries)
	newIndex := make(map[mapKey]int, len(m.index)+1)
	maps.Copy(newIndex, m.index)

	if idx, ok := newIndex[nk]; ok {
		newEntries[idx] = mapEntry{key: key, value: cloneSequence(value)}
	} else {
		newEntries = append(newEntries, mapEntry{key: key, value: cloneSequence(value)})
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
	MergeCombine
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
					allEntries[idx] = mapEntry{key: e.key, value: cloneSequence(e.value)}
					continue
				case MergeReject:
					return MapItem{}, &XPathError{
						Code:    errCodeFOJS0003,
						Message: fmt.Sprintf("duplicate key in map merge: %v", e.key.Value),
					}
				case MergeCombine:
					// Combine: concatenate values
					allEntries[idx] = mapEntry{
						key:   allEntries[idx].key,
						value: ItemSlice(append(cloneSequence(allEntries[idx].value).Materialize(), e.value.Materialize()...)),
					}
					continue
				}
			}
			seen[nk] = len(allEntries)
			allEntries = append(allEntries, mapEntry{key: e.key, value: cloneSequence(e.value)})
		}
	}

	newIndex := make(map[mapKey]int, len(allEntries))
	for i, e := range allEntries {
		newIndex[normalizeMapKey(e.key)] = i
	}
	return MapItem{entries: allEntries, index: newIndex}, nil
}

// MapBuilder accumulates map entries without defensive cloning,
// producing a MapItem when Build is called. This is significantly
// faster than MergeMaps for large merges because it avoids
// per-entry cloneSequence calls and builds the index incrementally.
type MapBuilder struct {
	entries []mapEntry
	index   map[mapKey]int
	policy  MergePolicy
}

// NewMapBuilder creates a builder for accumulating map entries.
// sizeHint is used to pre-allocate internal storage.
func NewMapBuilder(policy MergePolicy, sizeHint int) *MapBuilder {
	return &MapBuilder{
		entries: make([]mapEntry, 0, sizeHint),
		index:   make(map[mapKey]int, sizeHint),
		policy:  policy,
	}
}

// Add inserts a key-value pair into the builder, applying the merge policy
// for duplicate keys. The value is NOT cloned — the caller must ensure the
// value is not mutated after this call (which is the case for values produced
// by the evaluator, since XPath values are immutable).
func (b *MapBuilder) Add(key AtomicValue, value Sequence) error {
	nk := normalizeMapKey(key)
	if idx, ok := b.index[nk]; ok {
		switch b.policy {
		case MergeUseFirst:
			return nil
		case MergeUseLast:
			b.entries[idx] = mapEntry{key: key, value: value}
			return nil
		case MergeReject:
			return &XPathError{
				Code:    errCodeFOJS0003,
				Message: fmt.Sprintf("duplicate key in map merge: %v", key.Value),
			}
		case MergeCombine:
			existing := b.entries[idx].value
			b.entries[idx] = mapEntry{
				key:   b.entries[idx].key,
				value: ItemSlice(append(seqMaterialize(existing), seqMaterialize(value)...)),
			}
			return nil
		}
	}
	b.index[nk] = len(b.entries)
	b.entries = append(b.entries, mapEntry{key: key, value: value})
	return nil
}

// Build returns the accumulated MapItem. The builder should not be used
// after calling Build.
func (b *MapBuilder) Build() MapItem {
	return MapItem{entries: b.entries, index: b.index}
}

// --- ArrayItem ---

// ArrayItem is an immutable indexed sequence of members. Uses 1-based indexing.
type ArrayItem struct {
	members []Sequence
}

func (ArrayItem) itemTag() {}

// NewArray creates an ArrayItem from a slice of Sequence members.
func NewArray(members []Sequence) ArrayItem {
	return ArrayItem{members: cloneSequences(members)}
}

// Get returns the member at the given 1-based index.
func (a ArrayItem) Get(index int) (Sequence, error) {
	if index < 1 || index > len(a.members) {
		return nil, &XPathError{
			Code:    errCodeFOAY0001,
			Message: fmt.Sprintf("array index %d out of bounds (size %d)", index, len(a.members)),
		}
	}
	return cloneSequence(a.members[index-1]), nil
}

// Size returns the number of members.
func (a ArrayItem) Size() int {
	return len(a.members)
}

// Put returns a new array with the member at the given 1-based index replaced.
func (a ArrayItem) Put(index int, value Sequence) (ArrayItem, error) {
	if index < 1 || index > len(a.members) {
		return ArrayItem{}, &XPathError{
			Code:    errCodeFOAY0001,
			Message: fmt.Sprintf("array index %d out of bounds (size %d)", index, len(a.members)),
		}
	}
	newMembers := make([]Sequence, len(a.members))
	copy(newMembers, a.members)
	newMembers[index-1] = cloneSequence(value)
	return ArrayItem{members: newMembers}, nil
}

// Append returns a new array with the value appended.
func (a ArrayItem) Append(value Sequence) ArrayItem {
	newMembers := make([]Sequence, len(a.members)+1)
	copy(newMembers, a.members)
	newMembers[len(a.members)] = cloneSequence(value)
	return ArrayItem{members: newMembers}
}

// Members returns all members (defensively cloned).
func (a ArrayItem) Members() []Sequence {
	return cloneSequences(a.members)
}

// members0 returns the underlying members slice without cloning.
// Callers must not mutate the returned slices.
func (a ArrayItem) members0() []Sequence {
	return a.members
}

// get0 returns the member at the given 1-based index without cloning.
// Callers must not mutate the returned sequence.
func (a ArrayItem) get0(index int) (Sequence, error) {
	if index < 1 || index > len(a.members) {
		return nil, &XPathError{
			Code:    errCodeFOAY0001,
			Message: fmt.Sprintf("array index %d out of bounds (size %d)", index, len(a.members)),
		}
	}
	return a.members[index-1], nil
}

// SubArray returns a new array from start to end (1-based, inclusive).
func (a ArrayItem) SubArray(start, length int) (ArrayItem, error) {
	if start < 1 || length < 0 || start+length-1 > len(a.members) {
		return ArrayItem{}, &XPathError{
			Code:    errCodeFOAY0001,
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
	var result ItemSlice
	for _, m := range a.members {
		for item := range m.Items() {
			if nested, ok := item.(ArrayItem); ok {
				result = append(result, nested.Flatten().Materialize()...)
			} else {
				result = append(result, item)
			}
		}
	}
	return result
}

// --- Atomization ---

// IsKnownXSDType returns true if name is a recognized XSD type in the
// xpath3 type hierarchy (the xsdTypeParent map or a base type).
func IsKnownXSDType(name string) bool {
	if _, ok := xsdTypeParent[name]; ok {
		return true
	}
	switch name {
	case TypeAnyAtomicType, TypeNumeric,
		TypeAnyType, TypeAnySimpleType, TypeUntyped,
		TypeNOTATION, TypeError,
		TypeNMTOKENS, TypeENTITIES, TypeIDREFS:
		return true
	}
	return false
}

func schemaAnnotationParts(name string) (local, ns string, ok bool) {
	if strings.HasPrefix(name, "Q{") {
		end := strings.IndexByte(name, '}')
		if end <= 1 || end == len(name)-1 {
			return "", "", false
		}
		return name[end+1:], name[2:end], true
	}
	if strings.HasPrefix(name, "xs:") || strings.HasPrefix(name, "xsd:") {
		return "", "", false
	}
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		return name[idx+1:], "", true
	}
	if name == "" {
		return "", "", false
	}
	return name, "", true
}

// resolveQNameFromNode resolves a QName string (e.g., "my:brown-bear") using
// the in-scope namespaces of the given node.
func resolveQNameFromNode(s string, node helium.Node) (QNameValue, error) {
	s = strings.TrimSpace(s)
	prefix, local := "", s
	if idx := strings.IndexByte(s, ':'); idx >= 0 {
		prefix = s[:idx]
		local = s[idx+1:]
	}
	var uri string
	scope := node
	if _, ok := scope.(*helium.Element); !ok {
		scope = node.Parent()
	}
	if prefix != "" {
		for p := scope; p != nil; p = p.Parent() {
			pe, ok := p.(*helium.Element)
			if !ok {
				continue
			}
			for _, ns := range pe.Namespaces() {
				if ns.Prefix() == prefix {
					uri = ns.URI()
					break
				}
			}
			if uri != "" {
				break
			}
		}
		if uri == "" {
			return QNameValue{}, fmt.Errorf("undeclared namespace prefix: %s", prefix)
		}
	} else {
		for p := scope; p != nil; p = p.Parent() {
			pe, ok := p.(*helium.Element)
			if !ok {
				continue
			}
			for _, ns := range pe.Namespaces() {
				if ns.Prefix() == "" {
					uri = ns.URI()
					break
				}
			}
			if uri != "" {
				break
			}
		}
	}
	return QNameValue{Prefix: prefix, Local: local, URI: uri}, nil
}

func atomizedTypeForAnnotation(annotation string, decls SchemaDeclarations) string {
	switch annotation {
	case "", TypeUntypedAtomic, TypeUntyped:
		return ""
	}
	if IsKnownXSDType(annotation) {
		return annotation
	}
	if decls == nil {
		return ""
	}

	// Check if this is a union type — if so, return the annotation itself
	// so that AtomizeItem can try member types via the SchemaDeclarations.
	members := decls.UnionMemberTypes(annotation)
	if len(members) > 0 {
		// For union types, find the first member type that resolves to a
		// concrete built-in type — this will be used for atomization.
		for _, m := range members {
			mType := atomizedTypeForAnnotation(m, decls)
			if mType != "" && mType != TypeAnySimpleType && mType != TypeAnyAtomicType {
				return mType
			}
		}
		return ""
	}

	current := annotation
	for i := 0; i < 32; i++ {
		local, ns, ok := schemaAnnotationParts(current)
		if !ok {
			return ""
		}
		baseType, ok := decls.LookupSchemaType(local, ns)
		if !ok || baseType == "" || baseType == current {
			return ""
		}
		switch baseType {
		case TypeUntypedAtomic, TypeUntyped:
			return ""
		}
		// xs:anySimpleType is not a useful atomized type — it means we
		// reached the top of the simple type hierarchy without finding a
		// concrete type (e.g. for union types). Return empty.
		if baseType == TypeAnySimpleType || baseType == TypeAnyAtomicType {
			return ""
		}
		if IsKnownXSDType(baseType) {
			return baseType
		}
		current = baseType
	}
	return ""
}

// AtomizeItem converts a single item to an atomic value per XPath 3.1 Section 2.6.2.
func AtomizeItem(item Item) (AtomicValue, error) {
	switch v := item.(type) {
	case AtomicValue:
		return v, nil
	case NodeItem:
		s := ixpath.StringValue(v.Node)
		if v.TypeAnnotation != "" && v.TypeAnnotation != TypeUntypedAtomic {
			// QName-like types need namespace resolution from the node's scope.
			if v.TypeAnnotation == TypeQName || v.TypeAnnotation == TypeNOTATION ||
				v.AtomizedType == TypeQName || v.AtomizedType == TypeNOTATION {
				if qv, err := resolveQNameFromNode(s, v.Node); err == nil {
					typeName := v.TypeAnnotation
					if typeName == "" {
						typeName = v.AtomizedType
					}
					if typeName == "" {
						typeName = TypeQName
					}
					return AtomicValue{TypeName: typeName, Value: qv}, nil
				}
			}
			cast, err := CastFromString(s, v.TypeAnnotation)
			if err == nil {
				return cast, nil
			}
		}
		if v.AtomizedType != "" && v.AtomizedType != TypeUntypedAtomic && v.AtomizedType != v.TypeAnnotation {
			cast, err := CastFromString(s, v.AtomizedType)
			if err == nil {
				// Preserve the user-defined type annotation so that
				// "instance of" checks match the original schema type.
				if v.TypeAnnotation != "" && !IsKnownXSDType(v.TypeAnnotation) {
					cast.BaseType = cast.TypeName
					cast.TypeName = v.TypeAnnotation
				}
				return cast, nil
			}
		}
		// XPath 3.1 Section 2.6.2: typed value of PI, comment, and namespace nodes is xs:string
		if v.Node != nil {
			switch v.Node.Type() {
			case helium.ProcessingInstructionNode, helium.CommentNode, helium.NamespaceNode:
				return AtomicValue{
					TypeName: TypeString,
					Value:    s,
				}, nil
			}
		}
		return AtomicValue{
			TypeName: TypeUntypedAtomic,
			Value:    s,
		}, nil
	case ArrayItem:
		// XPath 3.1: atomizing an array atomizes each member and concatenates
		if v.Size() == 0 {
			return AtomicValue{}, &XPathError{
				Code:    "FOTY0013",
				Message: "cannot atomize empty array to single atomic value",
			}
		}
		if v.Size() == 1 {
			member, _ := v.Get(1)
			if seqLen(member) == 1 {
				return AtomizeItem(member.Get(0))
			}
		}
		return AtomicValue{}, &XPathError{
			Code:    "FOTY0013",
			Message: "cannot atomize array with multiple members to single atomic value",
		}
	default:
		return AtomicValue{}, &XPathError{
			Code:    "FOTY0013",
			Message: fmt.Sprintf("cannot atomize %T", item),
		}
	}
}
