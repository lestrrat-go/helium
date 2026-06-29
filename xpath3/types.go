package xpath3

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// Type family constants used for aggregate, comparison, and collation grouping.
const (
	familyNumeric    = "numeric"
	familyDurationYM = "duration:YM"
	familyDurationDT = "duration:DT"
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

// validNilSequence is a typed nil Sequence returned when an empty XPath
// sequence with no error is the intentional result.
var validNilSequence Sequence

// cloneSequence returns a deep copy of the Sequence. It materializes the source
// (matching the timing of the former shallow sequence.Clone — bounded call
// sites pre-charge the sequence length against OpLimit BEFORE invoking this, so
// the materialize cost is already accounted for) and then deep-copies each item
// so that pointer-backed atomic values (*big.Int, *big.Rat, *FloatValue,
// []byte) and nested maps/arrays are not shared with the source. This preserves
// XPath 3.1 value semantics: mutating a value after it was inserted into a
// map/array must not change the stored map/array.
func cloneSequence(seq Sequence) Sequence {
	if seq == nil {
		return nil
	}
	src := seq.Materialize()
	if len(src) == 0 {
		return ItemSlice(append([]Item(nil), src...))
	}
	cloned := make([]Item, len(src))
	for i, item := range src {
		cloned[i] = deepCloneItem(item)
	}
	return ItemSlice(cloned)
}

// deepCloneItem returns a value-semantics copy of an Item.
//
// Only AtomicValue with a pointer- or slice-backed payload is actually copied:
// those payloads (*big.Int, *big.Rat, *FloatValue, []byte, Duration's *big.Rat
// fields) are the sole way a caller can hold a reference and later mutate a value
// after it was inserted into a map/array.
//
// MapItem and ArrayItem are returned AS-IS, NOT recursively cloned. They are
// immutable (Put/Append/Remove/SubArray are all copy-on-write — the backing
// entry/member slices are never mutated in place), and every value placed inside
// one passed through cloneSequence at ingress, so any pointer atomics they hold
// were already detached from caller storage. Recursively re-cloning a nested
// map/array on every insert would turn incremental construction of a
// depth-N structure into O(N^2)/unbounded work and defeat the OpLimit/maxNodes
// bounds, which charge per inserted value once — not per nesting level on every
// insert.
//
// NodeItem keeps its underlying DOM node identity shared (the node itself is
// never cloned), but copies its mutable metadata slice (UnionMemberTypes) so a
// caller cannot mutate it after insertion. FunctionItem keeps its closure and
// any DOM identity shared, but copies its mutable type-metadata (ParamTypes /
// ReturnType) for the same reason.
func deepCloneItem(item Item) Item {
	switch v := item.(type) {
	case AtomicValue:
		// Only re-box when the payload actually carries shared mutable state.
		// Returning the original boxed Item for immutable payloads (int64,
		// string, bool, float64, time.Time, QNameValue, by-value Duration
		// without rationals, …) avoids an interface re-allocation per item on
		// the hot clone path.
		if cloned, copied := deepCloneAtomicValue(v); copied {
			return cloned
		}
		return item
	case NodeItem:
		if v.UnionMemberTypes == nil && v.ListItemLeaves == nil {
			return item
		}
		v.UnionMemberTypes = append([]string(nil), v.UnionMemberTypes...)
		v.ListItemLeaves = append([]*NodeItemUnionMember(nil), v.ListItemLeaves...)
		// ActiveUnionMember / ListItemLeaves entries point at immutable
		// NodeItemUnionMember values (never mutated after construction), so the
		// pointers are safely shared across clones.
		return v
	case FunctionItem:
		if v.ParamTypes == nil && v.ReturnType == nil {
			return item
		}
		v.ParamTypes = cloneSequenceTypes(v.ParamTypes)
		v.ReturnType = cloneSequenceTypePtr(v.ReturnType)
		return v
	default:
		return item
	}
}

// cloneSequenceTypePtr deep-copies a *SequenceType, preserving nil.
func cloneSequenceTypePtr(st *SequenceType) *SequenceType {
	if st == nil {
		return nil
	}
	cloned := cloneSequenceType(*st)
	return &cloned
}

// cloneSequenceTypes deep-copies a slice of SequenceType, preserving nil.
func cloneSequenceTypes(sts []SequenceType) []SequenceType {
	if sts == nil {
		return nil
	}
	cloned := make([]SequenceType, len(sts))
	for i := range sts {
		cloned[i] = cloneSequenceType(sts[i])
	}
	return cloned
}

// cloneSequenceType deep-copies a SequenceType's mutable metadata. Three item
// tests carry mutable backing reachable from a SequenceType: FunctionTest
// (ParamTypes slice + ReturnType, each itself a SequenceType), MapTest
// (ValType) and ArrayTest (MemberType). Those nested SequenceTypes can in turn
// hold a FunctionTest, so cloning recurses through all three. Every other
// NodeTest is an immutable value with no shared mutable backing, so it is kept
// as-is.
func cloneSequenceType(st SequenceType) SequenceType {
	switch t := st.ItemTest.(type) {
	case FunctionTest:
		t.ParamTypes = cloneSequenceTypes(t.ParamTypes)
		t.ReturnType = cloneSequenceType(t.ReturnType)
		st.ItemTest = t
	case MapTest:
		t.ValType = cloneSequenceType(t.ValType)
		st.ItemTest = t
	case ArrayTest:
		t.MemberType = cloneSequenceType(t.MemberType)
		st.ItemTest = t
	}
	return st
}

// deepCloneAtomicValue copies an AtomicValue when its Go payload is pointer- or
// slice-backed, duplicating that payload so a later mutation of the original
// cannot reach the stored copy. It returns (copy, true) only when a duplication
// was needed; for immutable value payloads it returns (zero, false) so the
// caller can keep the original (already-boxed) Item and skip a re-allocation.
func deepCloneAtomicValue(a AtomicValue) (AtomicValue, bool) {
	switch v := a.Value.(type) {
	case *big.Int:
		a.Value = new(big.Int).Set(v)
	case *big.Rat:
		a.Value = new(big.Rat).Set(v)
	case *FloatValue:
		a.Value = v.clone()
	case []byte:
		a.Value = append([]byte(nil), v...)
	case Duration:
		a.Value = v.clone()
	case *Duration:
		d := v.clone()
		a.Value = &d
	default:
		return AtomicValue{}, false
	}
	return a, true
}

// cloneMapKey returns a value-semantics copy of a map key. Map keys are single
// AtomicValues, so this is O(1). A pointer-backed key (e.g. xs:integer backed by
// *big.Int) could otherwise be mutated by the caller after insertion and, because
// single-entry maps recompute the stored key in Get, the map's key would change
// after construction. Cloning at every map ingress detaches the stored key from
// caller-held storage. Immutable payloads keep their original (already-boxed)
// AtomicValue and skip the re-allocation.
func cloneMapKey(key AtomicValue) AtomicValue {
	if cloned, copied := deepCloneAtomicValue(key); copied {
		return cloned
	}
	return key
}

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
	Node             helium.Node
	TypeAnnotation   string   // optional xs:... type annotation (schema-aware)
	AtomizedType     string   // optional built-in base type used for typed atomization
	ListItemType     string   // non-empty when the type is a list; the item type name
	ListItemAtomized string   // built-in base of the list item type (e.g. xs:QName for a QName-derived item)
	UnionMemberTypes []string // member type names for union types (for atomization)
	// ListItemLeaves is non-nil when the list ITEM type is a UNION: one entry per
	// whitespace token of the node value (aligned with xsdListFields), the
	// value-dependent ACTIVE leaf member for THAT token, precomputed in nodeItemFor.
	// So a list of a union atomizes each token through its own active member, matching
	// $value, instead of one static ListItemAtomized base. A nil entry means the token
	// resolved no member (the atomizer falls back to the static path).
	ListItemLeaves []*NodeItemUnionMember
	// ActiveUnionMember is the value-dependent ACTIVE LEAF member of a union-typed
	// node, precomputed in nodeItemFor via full schema-aware validation (nil when the
	// node is not a union or no member resolved). It is the resolved LEAF — when the
	// first validating member is itself a union, resolution descends into it (mirror
	// of fixedUnionActiveMember) so a nested union reaches its atomic/list leaf. This
	// agrees with the $value path's active-member selection.
	ActiveUnionMember *NodeItemUnionMember
	// QNameNoDefaultNS, when true, atomizes an UNPREFIXED QName/NOTATION value to
	// NO namespace instead of resolving the node's in-scope default namespace —
	// XSD value-space semantics (a QName VALUE, unlike a name, does not pick up the
	// default namespace). Set from the evaluator's QNameValueNoDefaultNamespace
	// option (used by xsd assertions); off by default so general XPath/XQuery and
	// XSLT atomization keep the default-namespace behavior.
	QNameNoDefaultNS bool
}

// NodeItemUnionMember carries per-member atomization metadata for a union-typed
// node, precomputed from SchemaDeclarations so value-dependent active-member
// resolution during atomization needs no schema access.
type NodeItemUnionMember struct {
	TypeName     string // member type annotation name
	Atomized     string // built-in base of an atomic member (for typing)
	ListItem     string // non-empty if the member is a list: its item type name
	ListItemAtom string // built-in base of the list item (e.g. xs:QName)
	// ListItemLeaves is non-nil when this member is a LIST whose item type is itself a
	// UNION: one entry per whitespace token of the union value (aligned with
	// xsdListFields), the value-dependent ACTIVE leaf member for THAT token, precomputed
	// in resolveActiveUnionLeafRec. So a union whose active member is a list-of-union
	// atomizes each token through its own active member (matching $value), not the static
	// ListItemAtom base. A nil entry means the token resolved no member.
	ListItemLeaves []*NodeItemUnionMember
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

	// TypeLong and the following are derived integer types.
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

	// TypeNormalizedString and the following are derived string types.
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

	// TypeGDay and the following are Gregorian date part types.
	TypeGDay       = "xs:gDay"
	TypeGMonth     = "xs:gMonth"
	TypeGMonthDay  = "xs:gMonthDay"
	TypeGYear      = "xs:gYear"
	TypeGYearMonth = "xs:gYearMonth"

	// TypeDateTimeStamp and the following are other derived types.
	TypeDateTimeStamp = "xs:dateTimeStamp"
	TypeError         = "xs:error"
	TypeNumeric       = "xs:numeric"

	// TypeAnyType and the following are non-atomic / structural types.
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
		return v.(bool) //nolint:forcetypeassert
	}
	result := computeIsSubtypeOf(actualType, targetType)
	subtypeCache.Store(key, result)
	return result
}

// isAtomicSubtypeOf reports whether the atomic value's type is targetType or a
// subtype of it, honoring both the built-in XSD hierarchy and the value's
// built-in BaseType (set when TypeName is a user-defined schema type derived by
// restriction). Unlike PromoteSchemaType it never falls back to the Go value's
// kind, so a sibling type (e.g. xs:dateTime relative to xs:date) is correctly
// rejected rather than reinterpreted from its time.Time payload.
func isAtomicSubtypeOf(av AtomicValue, targetType string) bool {
	if isSubtypeOf(av.TypeName, targetType) {
		return true
	}
	if av.BaseType != "" && IsKnownXSDType(av.BaseType) && isSubtypeOf(av.BaseType, targetType) {
		return true
	}
	return false
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
	return a.Value.(string) //nolint:forcetypeassert
}

// IntegerVal returns the backing int64 value. Panics if the value exceeds int64 range.
func (a AtomicValue) IntegerVal() int64 {
	if v, ok := a.Value.(int64); ok {
		return v
	}
	return a.Value.(*big.Int).Int64() //nolint:forcetypeassert
}

// Int64Val returns the int64 value if it fits, or (0, false) otherwise.
func (a AtomicValue) Int64Val() (int64, bool) {
	switch v := a.Value.(type) {
	case int64:
		return v, true
	case *big.Int:
		if v.IsInt64() {
			return v.Int64(), true
		}
		return 0, false
	}
	return 0, false
}

// BigInt returns the backing *big.Int value.
// If the value is stored as int64, it promotes to *big.Int.
// If the value is stored as a string, it attempts to parse it.
func (a AtomicValue) BigInt() *big.Int {
	switch v := a.Value.(type) {
	case *big.Int:
		return v
	case int64:
		return big.NewInt(v)
	case string:
		n, ok := new(big.Int).SetString(v, 10)
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

// DoubleVal returns the backing float64 value. The backing value is normally a
// *FloatValue, but a schema-derived or directly constructed double/float can
// carry a plain float64/float32; route through ToFloat64 so no caller panics.
func (a AtomicValue) DoubleVal() float64 {
	return a.ToFloat64()
}

// FloatVal returns the backing value as a *FloatValue. A schema-derived or
// directly constructed double/float can carry a plain float64/float32; wrap it
// at the precision implied by the type so callers never panic on a non-pointer
// backing value.
func (a AtomicValue) FloatVal() *FloatValue {
	// A schema-derived xs:float can be backed by a *FloatValue at double
	// precision (e.g. NewDouble(16777217)). Honor the effective numeric type so
	// such values are narrowed to single precision rather than returned as-is.
	singlePrec := a.effectiveNumericType() == TypeFloat
	if fv, ok := a.Value.(*FloatValue); ok {
		if singlePrec {
			return NewFloat(fv.Float64())
		}
		return fv
	}
	f := a.ToFloat64()
	if singlePrec {
		return NewFloat(f)
	}
	return NewDouble(f)
}

// BooleanVal returns the backing bool value.
func (a AtomicValue) BooleanVal() bool {
	return a.Value.(bool) //nolint:forcetypeassert
}

// TimeVal returns the backing time.Time value.
func (a AtomicValue) TimeVal() time.Time {
	return a.Value.(time.Time) //nolint:forcetypeassert
}

// DurationVal returns the backing Duration value.
func (a AtomicValue) DurationVal() Duration {
	return a.Value.(Duration) //nolint:forcetypeassert
}

// BytesVal returns the backing []byte value.
func (a AtomicValue) BytesVal() []byte {
	return a.Value.([]byte) //nolint:forcetypeassert
}

// QNameVal returns the backing QNameValue.
func (a AtomicValue) QNameVal() QNameValue {
	return a.Value.(QNameValue) //nolint:forcetypeassert
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
	case int64, *big.Int, *big.Rat, float64, float32, *FloatValue:
		return true
	}
	return false
}

// effectiveNumericType resolves the built-in numeric type that governs a value's
// conversion: the TypeName itself when it is a recognized numeric built-in, else
// the BaseType of a schema-derived value (custom TypeName whose BaseType names a
// built-in numeric ancestor). This mirrors PromoteSchemaType's BaseType fallback
// so schema-derived float/double/decimal/integer values convert correctly.
func (a AtomicValue) effectiveNumericType() string {
	if isIntegerDerived(a.TypeName) {
		return a.TypeName
	}
	switch a.TypeName {
	case TypeDecimal, TypeDouble, TypeFloat:
		return a.TypeName
	}
	if a.BaseType != "" && IsKnownXSDType(a.BaseType) {
		return a.BaseType
	}
	return a.TypeName
}

// ToFloat64 converts any numeric atomic value to float64.
func (a AtomicValue) ToFloat64() float64 {
	et := a.effectiveNumericType()
	if isIntegerDerived(et) {
		switch v := a.Value.(type) {
		case int64:
			return float64(v)
		case *big.Int:
			f, _ := new(big.Float).SetInt(v).Float64()
			return f
		}
		return 0
	}
	switch et {
	case TypeDouble, TypeFloat:
		// xs:double/xs:float atomics are normally backed by *FloatValue, but a
		// schema-derived double/float (or a value constructed directly) can carry
		// a plain float64/float32 backing value. Accept all forms so callers on
		// the map-key path never panic.
		var f float64
		switch v := a.Value.(type) {
		case *FloatValue:
			f = v.Float64()
		case float64:
			f = v
		case float32:
			f = float64(v)
		default:
			return 0
		}
		if et == TypeFloat {
			// An effective xs:float (including a schema-derived float backed by a
			// double-precision *FloatValue or a plain float64) must be narrowed to
			// single precision so ToFloat64/DoubleVal stay consistent with FloatVal.
			return float64(float32(f))
		}
		return f
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
	SecRat   *big.Rat // exact total dayTime seconds magnitude (>=0); authoritative when non-nil, sign carried by Negative
	Negative bool
}

// clone returns a deep copy of the Duration, duplicating its *big.Rat fields so
// a mutation of the original cannot reach the copy. Used by value-semantics deep
// cloning of atomic values stored in maps/arrays.
func (d Duration) clone() Duration {
	if d.FracSec != nil {
		d.FracSec = new(big.Rat).Set(d.FracSec)
	}
	if d.SecRat != nil {
		d.SecRat = new(big.Rat).Set(d.SecRat)
	}
	return d
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

	// Normalize string-like types: xs:untypedAtomic and xs:anyURI promote to
	// xs:string, as do types derived from xs:string (e.g. xs:NCName, xs:token),
	// so they share a key with an equal xs:string value. Schema-derived atomics
	// can carry a custom TypeName whose BaseType is the string-like ancestor, so
	// consult BaseType as well to fold those against an equal xs:string key.
	if tn == TypeUntypedAtomic || tn == TypeAnyURI || isStringDerived(tn) ||
		key.BaseType == TypeAnyURI || isStringDerived(key.BaseType) {
		return mapKey{typeName: TypeString, value: key.Value}
	}

	// Normalize all numeric types to a common representation using exact
	// rational arithmetic so that precision is not lost across types.
	if key.IsNumeric() {
		// Fast path: integer keys stored as int64 — use directly as map key
		// to avoid big.Rat/RatString overhead. This is safe because int64
		// has Go map equality and any numeric type that equals this integer
		// will also be detected as an exact integer below.
		if v, ok := key.Value.(int64); ok {
			return mapKey{typeName: familyNumeric, value: v}
		}
		// Exact numeric types must normalize with exact arithmetic, never float64:
		// ToFloat64 overflows values > MaxFloat64 to +Inf, collapsing distinct keys.
		// Also consult BaseType so a user-defined schema type derived from
		// xs:integer/xs:decimal is normalized exactly rather than via float64.
		if isIntegerDerived(key.TypeName) || isIntegerDerived(key.BaseType) {
			bi := key.BigInt()
			if bi.IsInt64() {
				return mapKey{typeName: familyNumeric, value: bi.Int64()}
			}
			return mapKey{typeName: familyNumeric, value: new(big.Rat).SetInt(bi).RatString()}
		}
		if key.TypeName == TypeDecimal || key.BaseType == TypeDecimal {
			r := key.BigRat()
			if r.IsInt() && r.Num().IsInt64() {
				return mapKey{typeName: familyNumeric, value: r.Num().Int64()}
			}
			return mapKey{typeName: familyNumeric, value: new(big.Rat).Set(r).RatString()}
		}
		// Remaining numerics are xs:float / xs:double (and user-defined types whose
		// effective base is a float/double). ToFloat64 keys off TypeName only, so a
		// schema-derived double/float (custom TypeName, BaseType=xs:double/xs:float)
		// would collapse to 0. Promote to the built-in base type first so ToFloat64
		// reads the underlying FloatValue exactly.
		f := PromoteSchemaType(key).ToFloat64()
		// xs:float keys are compared in single precision: a plain float64/float32
		// backing (schema-derived or directly constructed) may not yet be rounded,
		// so collapse to single precision before keying. Otherwise 16777217 and
		// 16777216 — equal as xs:float — would land in distinct slots.
		if key.effectiveNumericType() == TypeFloat {
			f = float64(float32(f))
		}
		if math.IsNaN(f) {
			return mapKey{typeName: familyNumeric, value: "NaN"}
		}
		if math.IsInf(f, 1) {
			return mapKey{typeName: familyNumeric, value: "+Inf"}
		}
		if math.IsInf(f, -1) {
			return mapKey{typeName: familyNumeric, value: "-Inf"}
		}
		// Remaining numerics are xs:float / xs:double (and user-defined types whose
		// underlying value is a float). Build an exact rational and use the int64
		// fast path ONLY when the value is an exact integer that fits in int64.
		// A naive "f <= math.MaxInt64" guard is unsafe: math.MaxInt64 (2^63-1)
		// rounds UP to 2^63 as a float64, so xs:double(2^63) would pass the check
		// and int64(f) would overflow-wrap to -2^63, colliding with xs:integer(-2^63).
		// Routing through the rational avoids that: r.Num().IsInt64() is exact.
		r := new(big.Rat).SetFloat64(f)
		if r == nil {
			// SetFloat64 returns nil for NaN/Inf; those are handled above, but
			// guard defensively so a stray non-finite value still produces a key.
			return mapKey{typeName: familyNumeric, value: strconv.FormatFloat(f, 'g', -1, 64)}
		}
		if r.IsInt() && r.Num().IsInt64() {
			return mapKey{typeName: familyNumeric, value: r.Num().Int64()}
		}
		return mapKey{typeName: familyNumeric, value: r.RatString()}
	}

	// Normalize duration types: xs:duration, xs:yearMonthDuration, xs:dayTimeDuration
	// that have equal values should be the same key. Schema-derived durations carry
	// a custom TypeName whose BaseType is a built-in duration ancestor, so consult
	// BaseType as well to fold those against an equal built-in duration key.
	if tn == TypeDuration || tn == TypeYearMonthDuration || tn == TypeDayTimeDuration ||
		key.BaseType == TypeDuration || key.BaseType == TypeYearMonthDuration ||
		key.BaseType == TypeDayTimeDuration {
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
	case Duration:
		// Duration holds a *big.Rat (FracSec) whose pointer breaks Go map
		// equality, and Seconds (a float64) can carry an exact fractional part
		// that int64() would truncate. Canonicalize to value content using the
		// exact-rational helper: signed total months and signed EXACT total
		// seconds (whole seconds plus the rational fraction). durationToRat
		// folds the sign into each rational, and big.Rat normalizes -0 to 0,
		// so -PT0S and PT0S produce the same key.
		months := durationToRat(v, true)
		secs := durationToRat(v, false)
		canon := months.RatString() + "|" + secs.RatString()
		// The Go value is unambiguously a duration, so fold every duration-derived
		// key (built-in or schema-derived through any number of levels) to
		// TypeDuration. This catches schema types whose BaseType is itself another
		// custom duration type and never resolves to a built-in duration above.
		return mapKey{typeName: TypeDuration, value: canon}
	default:
		return mapKey{typeName: tn, value: key.Value}
	}
}

// newSingleEntryMap creates a MapItem with exactly one entry without cloning the value.
// Used by map:entry where the value is already owned by the caller.
func newSingleEntryMap(key AtomicValue, value Sequence) MapItem {
	return MapItem{
		entries: []mapEntry{{key: cloneMapKey(key), value: value}},
	}
}

// NewMap creates a MapItem from a slice of MapEntry.
func NewMap(entries []MapEntry) MapItem {
	// Fast path for single-entry maps (e.g., map:entry results):
	// skip the index map allocation entirely.
	if len(entries) == 1 {
		return MapItem{
			entries: []mapEntry{{key: cloneMapKey(entries[0].Key), value: cloneSequence(entries[0].Value)}},
		}
	}
	m := MapItem{
		entries: make([]mapEntry, len(entries)),
		index:   make(map[mapKey]int, len(entries)),
	}
	for i, e := range entries {
		m.entries[i] = mapEntry{key: cloneMapKey(e.Key), value: cloneSequence(e.Value)}
		m.index[normalizeMapKey(e.Key)] = i
	}
	return m
}

// Get returns the value associated with key, or (nil, false) if not found.
func (m MapItem) Get(key AtomicValue) (Sequence, bool) {
	if m.index == nil {
		// Single-entry fast path: compare directly
		if len(m.entries) == 1 && normalizeMapKey(m.entries[0].key) == normalizeMapKey(key) {
			return cloneSequence(m.entries[0].value), true
		}
		return nil, false
	}
	idx, ok := m.index[normalizeMapKey(key)]
	if !ok {
		return nil, false
	}
	return cloneSequence(m.entries[idx].value), true
}

// get0 returns the value associated with key without cloning it, or
// (nil, false) if not found. The returned Sequence is the one stored in the
// map (possibly a borrowed lazy sequence); callers must not mutate it and must
// clone before exposing it outward. This lets bounded walkers check the stored
// value's length before deciding whether to clone/materialize it.
func (m MapItem) get0(key AtomicValue) (Sequence, bool) {
	if m.index == nil {
		if len(m.entries) == 1 && normalizeMapKey(m.entries[0].key) == normalizeMapKey(key) {
			return m.entries[0].value, true
		}
		return nil, false
	}
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
	newIndex := make(map[mapKey]int, len(m.entries)+1)
	// Rebuild index from entries when it was nil (single-entry maps)
	for i, e := range newEntries {
		newIndex[normalizeMapKey(e.key)] = i
	}

	if idx, ok := newIndex[nk]; ok {
		newEntries[idx] = mapEntry{key: cloneMapKey(key), value: cloneSequence(value)}
	} else {
		newEntries = append(newEntries, mapEntry{key: cloneMapKey(key), value: cloneSequence(value)})
		newIndex[nk] = len(newEntries) - 1
	}
	return MapItem{entries: newEntries, index: newIndex}
}

// Contains returns true if the map contains the given key.
func (m MapItem) Contains(key AtomicValue) bool {
	if m.index == nil {
		return len(m.entries) == 1 && normalizeMapKey(m.entries[0].key) == normalizeMapKey(key)
	}
	_, ok := m.index[normalizeMapKey(key)]
	return ok
}

// Keys returns all keys in insertion order. Keys are cloned so a caller cannot
// mutate a pointer-backed key and change the stored map key.
func (m MapItem) Keys() []AtomicValue {
	keys := make([]AtomicValue, len(m.entries))
	for i, e := range m.entries {
		keys[i] = cloneMapKey(e.key)
	}
	return keys
}

// keys0 returns the map's keys in insertion order without cloning. The returned
// AtomicValues are the map's own stored keys; callers must not mutate them.
// Used by trusted internal paths that only read keys.
func (m MapItem) keys0() []AtomicValue {
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
// The key and value passed to fn are clones of the map's stored entry, so a
// caller that mutates a pointer-backed atomic through fn cannot change the
// supposedly-immutable map. Trusted internal callers that need to avoid the
// clone (e.g. bounded/lazy iteration) use forEach0 instead.
func (m MapItem) ForEach(fn func(AtomicValue, Sequence) error) error {
	for _, e := range m.entries {
		if err := fn(cloneMapKey(e.key), cloneSequence(e.value)); err != nil {
			return err
		}
	}
	return nil
}

// forEach0 calls fn for each entry in insertion order WITHOUT cloning the key or
// value. The key and value passed to fn are the map's own backing storage;
// callers must not mutate either and must clone before exposing them outward.
// Used by trusted internal/lazy/bounded paths that preserve laziness and bounds.
func (m MapItem) forEach0(fn func(AtomicValue, Sequence) error) error {
	for _, e := range m.entries {
		if err := fn(e.key, e.value); err != nil {
			return err
		}
	}
	return nil
}

// entries0 returns the map's entries in insertion order without cloning. The
// returned slice and its value sequences are the map's own backing storage;
// callers must not mutate either. Used by bounded walkers that iterate value
// sequences in place rather than allocating a []Sequence copy.
func (m MapItem) entries0() []mapEntry {
	return m.entries
}

// Remove returns a new map with the given key removed.
func (m MapItem) Remove(key AtomicValue) MapItem {
	nk := normalizeMapKey(key)
	// Handle single-entry maps without index
	if m.index == nil {
		if len(m.entries) == 1 && normalizeMapKey(m.entries[0].key) == nk {
			return MapItem{}
		}
		return m
	}
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
					allEntries[idx] = mapEntry{key: cloneMapKey(e.key), value: cloneSequence(e.value)}
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
			allEntries = append(allEntries, mapEntry{key: cloneMapKey(e.key), value: cloneSequence(e.value)})
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
// by the evaluator, since XPath values are immutable). The key IS cloned
// (an O(1) single-atomic copy) so a pointer-backed key cannot be mutated
// after insertion and silently change the stored map key.
func (b *MapBuilder) Add(key AtomicValue, value Sequence) error {
	nk := normalizeMapKey(key)
	if idx, ok := b.index[nk]; ok {
		switch b.policy {
		case MergeUseFirst:
			return nil
		case MergeUseLast:
			b.entries[idx] = mapEntry{key: cloneMapKey(key), value: value}
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
	b.entries = append(b.entries, mapEntry{key: cloneMapKey(key), value: value})
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
	size := len(a.members)
	// Bounds are checked without forming start+length, which can overflow for
	// huge positions and bypass the guard, leading to a make([]Sequence, length)
	// panic. With start>=1 established first, size-start+1 cannot overflow.
	if start < 1 || start > size+1 || length < 0 || length > size-start+1 {
		return ArrayItem{}, &XPathError{
			Code:    errCodeFOAY0001,
			Message: fmt.Sprintf("array subarray(%d, %d) out of bounds (size %d)", start, length, size),
		}
	}
	newMembers := make([]Sequence, length)
	copy(newMembers, a.members[start-1:start-1+length])
	return ArrayItem{members: newMembers}, nil
}

// Flatten returns all members concatenated into a single sequence.
// Nested arrays are recursively flattened per XPath 3.1 spec.
//
// Leaf items are deep-cloned via deepCloneItem so that pointer-backed atomics
// (*big.Int, *big.Rat, *FloatValue, []byte) held in the array's internal
// storage are not exposed to the caller; mutating the returned sequence must
// not mutate the original array. Nested arrays/maps are flattened recursively
// (their own leaves are cloned at that level) and any non-flattened maps are
// kept shared by value — they are immutable and were detached at ingress — to
// avoid O(N^2) deep recursion and stay within the resource bounds.
func (a ArrayItem) Flatten() Sequence {
	var result ItemSlice
	for _, m := range a.members {
		for item := range m.Items() {
			if nested, ok := item.(ArrayItem); ok {
				result = append(result, nested.Flatten().Materialize()...)
			} else {
				result = append(result, deepCloneItem(item))
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
	if _, local, ok := strings.Cut(name, ":"); ok {
		return local, "", true
	}
	if name == "" {
		return "", "", false
	}
	return name, "", true
}

// resolveQNameFromNode resolves a QName string (e.g., "my:brown-bear") using
// the in-scope namespaces of the given node.
func resolveQNameFromNode(s string, node helium.Node, noDefaultNS bool) (QNameValue, error) {
	// The split kernel trims and splits at the first colon; this site does not
	// NCName-validate, so the validNC result is ignored.
	prefix, local, _, _ := domutil.SplitLexicalQName(s)
	var uri string
	scope := node
	if _, ok := scope.(*helium.Element); !ok {
		scope = node.Parent()
	}
	if prefix == "xml" {
		// The "xml" prefix is predeclared (XML Namespaces) and need not be bound on
		// any node — matches xsd.resolveLexicalQName, so e.g. xml:lang atomizes as a
		// valid xs:QName in an assertion instead of failing with an undeclared prefix.
		uri = lexicon.NamespaceXML
	} else if prefix != "" {
		// prefix is non-empty here so an empty-URI binding cannot occur and the
		// first-match-wins helper matches the original skip-empty walk exactly.
		var found bool
		uri, found = domutil.LookupNSPrefixURI(scope, prefix)
		if !found {
			return QNameValue{}, fmt.Errorf("undeclared namespace prefix: %s", prefix)
		}
	} else if !noDefaultNS {
		// Default-namespace resolution keeps the skip-empty walk inline: an
		// ancestor xmlns="" must not stop the search for an outer default.
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
	for range 32 {
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
				if qv, err := resolveQNameFromNode(s, v.Node, v.QNameNoDefaultNS); err == nil {
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
		// Union types: try each member type until one succeeds.
		if len(v.UnionMemberTypes) > 0 {
			for _, memberType := range v.UnionMemberTypes {
				cast, err := CastFromString(s, memberType)
				if err == nil {
					cast.BaseType = cast.TypeName
					cast.TypeName = v.TypeAnnotation
					return cast, nil
				}
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
