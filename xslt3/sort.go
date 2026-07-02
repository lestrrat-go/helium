package xslt3

import (
	"cmp"
	"context"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/internal/xmlchar"
	"github.com/lestrrat-go/helium/xpath3"
)

// sortKey is a compiled xsl:sort specification.
type sortKey struct {
	Select    *xpath3.Expression
	Body      []instruction // sequence constructor (when select is absent)
	Order     *avt          // "ascending" or "descending"
	DataType  *avt          // "text" or lexicon.TypeNumber
	CaseOrder *avt          // "upper-first" or "lower-first"
	Lang      *avt
	Collation *avt // collation URI
	Stable    *avt // "yes"/"no"/"true"/"false"/"1"/"0"
}

const (
	sortOrderAscending  = "ascending"
	sortOrderDescending = "descending"
)

// sortMode determines how a sort level compares keys.
type sortMode uint8

const (
	sortModeText sortMode = iota
	sortModeNumber
)

// dataTypeMode tracks whether a sort level's data type is explicit or auto-detected.
type dataTypeMode uint8

const (
	dataTypeAuto       dataTypeMode = iota // no explicit data-type attr; detect from first numeric result
	dataTypeText                           // explicit data-type="text"
	dataTypeNumber                         // explicit data-type=lexicon.TypeNumber
	dataTypeNumberAuto                     // auto-detected from first numeric/date value
)

// sortValueKind identifies the type of a pre-extracted sort key.
type sortValueKind uint8

const (
	sortValueText sortValueKind = iota
	sortValueNumber
	sortValueNaN
)

// sortValue is a pre-extracted, typed sort key. Numeric values are parsed
// once at extraction time; no string→float conversion happens during comparison.
type sortValue struct {
	kind     sortValueKind
	str      string
	num      float64
	typeName string // original XSD type name (for XTDE1030 checking)
	// atom carries the original atomized value (when hasAtom is true) so the
	// XTDE1030 type-consistency check can probe true XPath comparability via
	// xpath3.ValueCompare rather than comparing raw type names. Numeric/date
	// pre-conversion to num is independent of this — atom always reflects the
	// untouched source type for the consistency gate.
	atom    xpath3.AtomicValue
	hasAtom bool
}

// resolvedLevel holds the fully resolved configuration for one sort level.
type resolvedLevel struct {
	mode    sortMode
	desc    bool
	compare func(a, b string) int // collation comparator; nil = codepoint
}

// resolvedSort holds the fully resolved per-level sort configuration.
// Built once before sorting; no per-comparison avt evaluation needed.
type resolvedSort struct {
	levels []resolvedLevel
}

// keyed1 pairs a value with a single inline sort key and original index.
type keyed1[T any] struct {
	item  T
	key   sortValue
	index int
}

// keyedN pairs a value with pre-extracted typed sort keys and original index.
type keyedN[T any] struct {
	item  T
	keys  []sortValue
	index int
}

// keyedSlice is a named slice type for keyedN so that convertAutoNumeric
// can be defined once and passed to finalizeLevels.
type keyedSlice[T any] []keyedN[T]

func (ks keyedSlice[T]) convertAutoNumeric(level int) {
	for j := range ks {
		sv := &ks[j].keys[level]
		if sv.kind == sortValueText {
			*sv = parseToNumericSortValue(sv.str)
		}
	}
}

// --- Sort level resolution ---

// resolveSortOrder evaluates the order AVT (defaulting to ascending) and
// validates it, raising XTDE0030 for any value other than ascending/descending.
func resolveSortOrder(ctx context.Context, ec *execContext, sk *sortKey) (string, error) {
	order := sortOrderAscending
	if sk.Order != nil {
		var err error
		order, err = sk.Order.evaluate(ctx, ec.contextNode)
		if err != nil {
			return "", err
		}
		if order != sortOrderAscending && order != sortOrderDescending {
			return "", dynamicError(errCodeXTDE0030,
				"invalid order %q in xsl:sort; must be \"ascending\" or \"descending\"", order)
		}
	}
	return order, nil
}

// buildResolvedSort evaluates AVTs for order once and builds a resolvedLevel per sort level.
func buildResolvedSort(ctx context.Context, ec *execContext, sortKeys []*sortKey) (resolvedSort, error) {
	levels := make([]resolvedLevel, len(sortKeys))
	for i, sk := range sortKeys {
		order, err := resolveSortOrder(ctx, ec, sk)
		if err != nil {
			return resolvedSort{}, err
		}
		levels[i] = resolvedLevel{
			desc: order == sortOrderDescending,
		}
		if sk.Collation != nil {
			uri, err := sk.Collation.evaluate(ctx, ec.contextNode)
			if err != nil {
				return resolvedSort{}, err
			}
			cmpFn, err := xpath3.ResolveCollationCompareFunc(uri)
			if err != nil {
				return resolvedSort{}, dynamicError(errCodeXTDE1035,
					"unknown collation URI %q", uri)
			}
			levels[i].compare = cmpFn
		} else if sk.Lang != nil || sk.CaseOrder != nil {
			uri, err := buildImplicitCollationURI(ctx, ec, sk)
			if err != nil {
				return resolvedSort{}, err
			}
			if uri != "" {
				cmpFn, err := xpath3.ResolveCollationCompareFunc(uri)
				if err == nil {
					levels[i].compare = cmpFn
				}
			}
		}
	}
	return resolvedSort{levels: levels}, nil
}

func resolveLevel1(ctx context.Context, ec *execContext, sk *sortKey) (resolvedLevel, error) {
	order, err := resolveSortOrder(ctx, ec, sk)
	if err != nil {
		return resolvedLevel{}, err
	}
	// XTDE0030: validate lang attribute value
	if sk.Lang != nil {
		lang, err := sk.Lang.evaluate(ctx, ec.contextNode)
		if err != nil {
			return resolvedLevel{}, err
		}
		if !isValidLanguageTag(lang) {
			return resolvedLevel{}, dynamicError(errCodeXTDE0030,
				"invalid language tag %q in xsl:sort", lang)
		}
	}
	rl := resolvedLevel{desc: order == sortOrderDescending}
	if sk.Collation != nil {
		uri, err := sk.Collation.evaluate(ctx, ec.contextNode)
		if err != nil {
			return resolvedLevel{}, err
		}
		cmpFn, err := xpath3.ResolveCollationCompareFunc(uri)
		if err != nil {
			return resolvedLevel{}, dynamicError(errCodeXTDE1035,
				"unknown collation URI %q", uri)
		}
		rl.compare = cmpFn
	} else if sk.Lang != nil || sk.CaseOrder != nil {
		// Build a UCA collation from lang/case-order when no explicit collation.
		uri, err := buildImplicitCollationURI(ctx, ec, sk)
		if err != nil {
			return resolvedLevel{}, err
		}
		if uri != "" {
			cmpFn, err := xpath3.ResolveCollationCompareFunc(uri)
			if err == nil {
				rl.compare = cmpFn
			}
		}
	}
	return rl, nil
}

// buildImplicitCollationURI constructs a UCA collation URI from lang and
// case-order attributes when no explicit collation is specified. An evaluated
// case-order that is neither "upper-first" nor "lower-first" raises XTDE0030.
func buildImplicitCollationURI(ctx context.Context, ec *execContext, sk *sortKey) (string, error) {
	var params []string
	if sk.Lang != nil {
		lang, err := sk.Lang.evaluate(ctx, ec.contextNode)
		if err != nil {
			return "", err
		}
		if lang != "" {
			params = append(params, "lang="+lang)
		}
	}
	if sk.CaseOrder != nil {
		co, err := sk.CaseOrder.evaluate(ctx, ec.contextNode)
		if err != nil {
			return "", err
		}
		switch co {
		case "upper-first":
			params = append(params, "caseFirst=upper")
		case "lower-first":
			params = append(params, "caseFirst=lower")
		default:
			return "", dynamicError(errCodeXTDE0030,
				"invalid case-order %q in xsl:sort; must be \"upper-first\" or \"lower-first\"", co)
		}
	}
	if len(params) == 0 {
		return "", nil
	}
	return "http://www.w3.org/2013/collation/UCA?" + strings.Join(params, ";"), nil
}

// validateSortKeyAttrs validates sort key attribute values (order, data-type,
// case-order, lang, collation, stable) regardless of whether there are nodes
// to sort.
func validateSortKeyAttrs(ctx context.Context, ec *execContext, sk *sortKey) error {
	if _, err := resolveSortOrder(ctx, ec, sk); err != nil {
		return err
	}
	if sk.DataType != nil {
		dt, err := sk.DataType.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		if !isValidSortDataType(dt) {
			return dynamicError(errCodeXTDE0030,
				"invalid data-type %q in xsl:sort; must be \"text\", \"number\", or a QName", dt)
		}
	}
	if sk.CaseOrder != nil {
		co, err := sk.CaseOrder.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		if co != "upper-first" && co != "lower-first" {
			return dynamicError(errCodeXTDE0030,
				"invalid case-order %q in xsl:sort; must be \"upper-first\" or \"lower-first\"", co)
		}
	}
	if sk.Lang != nil {
		lang, err := sk.Lang.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		if !isValidLanguageTag(lang) {
			return dynamicError(errCodeXTDE0030,
				"invalid language tag %q in xsl:sort", lang)
		}
	}
	if sk.Stable != nil {
		stable, err := sk.Stable.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		// helium always performs a stable sort; the evaluated value is
		// validated for conformance but does not change sort behavior
		// (honoring stable="no" via an unstable sort is out of scope).
		if _, ok := parseXSDBool(stable); !ok {
			return dynamicError(errCodeXTDE0030,
				"invalid stable %q in xsl:sort; must be a valid xs:boolean", stable)
		}
	}
	if sk.Collation != nil {
		uri, err := sk.Collation.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		if _, err := xpath3.ResolveCollationCompareFunc(uri); err != nil {
			return dynamicError(errCodeXTDE1035,
				"unknown collation URI %q", uri)
		}
	}
	return nil
}

// sortKeyTypesComparable reports whether two atomic sort-key values are mutually
// ORDERABLE under XPath value-comparison semantics, reusing xpath3.ValueCompare
// with the less-than operator (`lt`) as the orderability oracle instead of
// comparing raw type names. Sorting needs ordering, not just equality, so the
// gate must probe `lt`: two values are consistent only if `lt` is DEFINED
// between them. This honors the repo's XSD 1.1 model automatically — the numeric
// family, xs:dateTimeStamp ⊂ xs:dateTime, xs:anyURI/xs:string, and
// untypedAtomic-as-string are all mutually orderable — while equality-only
// families (e.g. mixed xs:yearMonthDuration + xs:dayTimeDuration, which define
// `eq` but raise XPTY0004 on `lt`) and genuinely incomparable families (e.g.
// xs:date vs xs:integer) are correctly rejected with XTDE1030.
func sortKeyTypesComparable(a, b xpath3.AtomicValue) bool {
	_, err := xpath3.ValueCompare(xpath3.TokenLt, a, b)
	return err == nil
}

// validateSortLevelTypes is the SINGLE per-level XTDE1030 validation routine
// shared by both the single-key and multi-key sort paths. It raises XTDE1030
// when the values for one sort level have mutually incomparable atomic types.
//
// Levels with an explicit data-type="text" (every value is stringified) or
// data-type="number" (every value is cast to xs:double) make mixed original
// atomic types trivially comparable, so the check is skipped entirely for them.
// Only default-data-type levels (dataTypeAuto / dataTypeNumberAuto) compare
// values by their own atomic types and need the consistency check.
func validateSortLevelTypes(values []sortValue, mode dataTypeMode) error {
	if mode == dataTypeText || mode == dataTypeNumber {
		return nil
	}
	var ref *xpath3.AtomicValue
	for i := range values {
		v := &values[i]
		if !v.hasAtom {
			continue
		}
		// xs:duration is only partially ordered and cannot be used as a sort key.
		if v.atom.TypeName == xpath3.TypeDuration {
			return dynamicError(errCodeXTDE1030,
				"sort keys of type %s are only partially ordered", xpath3.TypeDuration)
		}
		if ref == nil {
			ref = &v.atom
			continue
		}
		if !sortKeyTypesComparable(*ref, v.atom) {
			return dynamicError(errCodeXTDE1030,
				"sort keys have incompatible types: %s and %s", ref.TypeName, v.atom.TypeName)
		}
	}
	return nil
}

// validateSortKeyTypes1 routes a single-key sort's values through the shared
// per-level validator.
func validateSortKeyTypes1[T any](entries []keyed1[T], mode dataTypeMode) error {
	values := make([]sortValue, len(entries))
	for i := range entries {
		values[i] = entries[i].key
	}
	return validateSortLevelTypes(values, mode)
}

// validateSortKeyTypesN routes each level of a multi-key sort through the shared
// per-level validator.
func validateSortKeyTypesN[T any](entries keyedSlice[T], dtModes []dataTypeMode) error {
	values := make([]sortValue, len(entries))
	for level, m := range dtModes {
		for i := range entries {
			values[i] = entries[i].keys[level]
		}
		if err := validateSortLevelTypes(values, m); err != nil {
			return err
		}
	}
	return nil
}

// isValidSortDataType reports whether s is a permitted xsl:sort data-type
// value: "text", "number", or a QName denoting a non-absent namespace
// (a prefixed lexical QName or an EQName Q{uri}local). The effect of a QName
// data-type is implementation-defined; helium treats it as "text".
func isValidSortDataType(s string) bool {
	switch s {
	case "text", lexicon.TypeNumber:
		return true
	}
	if isValidEQName(s) {
		return true
	}
	// A bare NCName has an absent namespace and is not permitted; require a
	// prefix so the QName denotes a non-absent namespace.
	if strings.IndexByte(s, ':') > 0 && xmlchar.IsValidQName(s) {
		return true
	}
	return false
}

// isValidLanguageTag checks if s is a plausible BCP47 language tag.
// Rejects obviously invalid values (empty, contains quotes, whitespace).
func isValidLanguageTag(s string) bool {
	if s == "" {
		return true // empty = implementation default
	}
	for _, r := range s {
		if r == '\'' || r == '"' || r == ' ' || r == '\t' || r == '\n' {
			return false
		}
	}
	return true
}

// --- Comparators ---

func compareSortValues(a, b sortValue, level resolvedLevel) int {
	var c int
	switch level.mode {
	case sortModeNumber:
		c = compareFloat64(a, b)
	default:
		if level.compare != nil {
			c = level.compare(a.str, b.str)
		} else {
			c = cmp.Compare(a.str, b.str)
		}
	}
	if level.desc {
		c = -c
	}
	return c
}

func compareFloat64(a, b sortValue) int {
	aNaN := a.kind == sortValueNaN
	bNaN := b.kind == sortValueNaN
	if aNaN && bNaN {
		return 0
	}
	if aNaN {
		return -1
	}
	if bNaN {
		return 1
	}
	if a.num < b.num {
		return -1
	}
	if a.num > b.num {
		return 1
	}
	return 0
}

func (rs *resolvedSort) compareKeys(aKeys, bKeys []sortValue, aIdx, bIdx int) int {
	for i, level := range rs.levels {
		if c := compareSortValues(aKeys[i], bKeys[i], level); c != 0 {
			return c
		}
	}
	return cmp.Compare(aIdx, bIdx)
}

// --- Key evaluation ---

// fillSingletonSortValue records the atomized value of a singleton sort-key
// item on sv (so the XTDE1030 orderability gate can probe it via
// xpath3.ValueCompare), applies auto data-type promotion, and rebuilds sv as a
// numeric value for orderable duration/date/time types. It returns the original
// atomized value and whether one was captured; callers MUST set sv.atom/hasAtom
// from the result LAST, because the duration/date rebuild replaces sv wholesale
// and would otherwise drop the atom. Shared by the select and body sort-key
// paths so both record the source atom identically.
func fillSingletonSortValue(sv *sortValue, item xpath3.Item, dtMode *dataTypeMode, implicitTZ *time.Location) (xpath3.AtomicValue, bool) {
	var av xpath3.AtomicValue
	switch v := item.(type) {
	case xpath3.AtomicValue:
		av = v
	case xpath3.NodeItem:
		// Atomize to get the typed value. A schema/type-annotated node MUST then
		// flow through the SAME auto numeric/date/duration promotion an atomic
		// singleton receives, so a typed date/duration node sorts by its real
		// value rather than as text. Reuse this single atomized value for both
		// the comparison value and the XTDE1030 gate (don't atomize twice).
		atom, err := xpath3.AtomizeItem(v)
		if err != nil {
			return xpath3.AtomicValue{}, false
		}
		av = atom
	default:
		return xpath3.AtomicValue{}, false
	}
	applyAutoSortPromotion(sv, av, dtMode, implicitTZ)
	return av, true
}

// applyAutoSortPromotion records av's type on sv and, for an auto-detect
// data-type level, rewrites sv as a numeric sortValue for orderable
// duration/date/time types (flipping the level to dataTypeNumberAuto). Shared by
// atomic and atomized-node singleton keys so both promote identically.
//
// Only an auto-detect level rewrites orderable duration/date/time keys into a
// numeric sortValue. An explicit data-type="text" level must keep the string
// value already stored in sv.str — a numeric rewrite there would blank str and
// make every such key compare as "" under text comparison. The atom is recorded
// by the caller either way so the XTDE1030 orderability gate still sees the true
// source type.
//
// The orderable-type detection is BaseType-AWARE: a schema-derived
// date/time/duration is recognized and rewritten by value, mirroring what the
// XTDE1030 gate accepts. It delegates to atomicToNumericSortValue, which resolves
// the built-in primitive through PromoteSchemaType; a plain numeric reaches here
// only after the IsNumeric flip + early return, so this delegation never touches
// numerics (those stay text and are converted later by convertAutoNumeric).
func applyAutoSortPromotion(sv *sortValue, av xpath3.AtomicValue, dtMode *dataTypeMode, implicitTZ *time.Location) {
	sv.typeName = av.TypeName
	if *dtMode == dataTypeAuto && av.IsNumeric() {
		*dtMode = dataTypeNumberAuto
	}
	if *dtMode != dataTypeAuto {
		return
	}
	if nv, ok := atomicToNumericSortValue(av, implicitTZ); ok {
		*sv = nv
		*dtMode = dataTypeNumberAuto
	}
}

// evaluateSortKey evaluates a single sort key for one item and returns its typed value.
// Uses EvaluateReuse when evalState is non-nil to avoid per-item evalContext allocation.
func evaluateSortKey(ctx context.Context, ec *execContext, sk *sortKey, node helium.Node, dtMode *dataTypeMode, evalState *xpath3.EvalState) (sortValue, error) {
	savedInSort := ec.inSortKeyEval
	ec.inSortKeyEval = true
	defer func() { ec.inSortKeyEval = savedInSort }()
	if sk.Select != nil {
		var result xpath3.Result
		var err error
		if evalState != nil {
			if ec.contextItem != nil {
				evalState.SetContextItem(ec.contextItem)
			}
			result, err = sk.Select.EvaluateReuse(ctx, evalState, node)
		} else {
			r, e := ec.evalXPath(ctx, sk.Select, node)
			if e != nil {
				return sortValue{}, dynamicError(errCodeXTDE0700, "sort key evaluation error: %v", e)
			}
			result = *r
			err = nil
		}
		if err != nil {
			return sortValue{}, dynamicError(errCodeXTDE0700, "sort key evaluation error: %v", err)
		}

		seq := result.Sequence()

		// XTTE1020: sort key value must be a single atomic value after atomization
		seqLen := 0
		if seq != nil {
			seqLen = sequence.Len(seq)
		}
		strVal := result.StringValue()
		// XSLT 1.0 behavior (backwards-compatible processing): the sort key uses
		// the FIRST item; the rest are discarded rather than raising XTTE1020
		// (§13.1.2).
		if seqLen > 1 && ec.isCompatExpr(sk.Select) {
			seq = xpath3.ItemSlice{seq.Get(0)}
			strVal = stringifyItem(seq.Get(0))
			seqLen = 1
		}
		if seqLen > 1 {
			return sortValue{}, dynamicError(errCodeXTTE1020,
				"sort key value is a sequence of %d items; a single value is required", seqLen)
		}

		implicitTZ := ec.currentTime.Location()
		if *dtMode == dataTypeNumber {
			// Explicit data-type=lexicon.TypeNumber: use number() semantics.
			// Dates/times are not numeric → convert via string → NaN.
			return extractNumericSortValue(seq, strVal, true, implicitTZ), nil
		}
		if *dtMode == dataTypeNumberAuto {
			// Auto-detected numeric: preserve date/time ordering.
			sv := extractNumericSortValue(seq, strVal, false, implicitTZ)
			// Atomize ANY singleton item for the XTDE1030 gate — a NodeItem is
			// atomizable too. A schema-typed date/duration NODE must ALSO supply
			// its comparison value from the atomized typed value: extractNumeric-
			// SortValue only converts already-atomic items, so a typed node would
			// otherwise have fallen back to parsing its lexical string and sorted
			// as NaN. Recompute the numeric value from the atom when orderable
			// (for an atomic singleton this reproduces the same value).
			if seqLen == 1 {
				sv = applyNumberAutoSingleton(sv, seq.Get(0), implicitTZ)
			}
			return sv, nil
		}

		sv := sortValue{kind: sortValueText, str: strVal}
		var keyAtom xpath3.AtomicValue
		var haveAtom bool
		if seqLen == 1 {
			keyAtom, haveAtom = fillSingletonSortValue(&sv, seq.Get(0), dtMode, implicitTZ)
		}
		// Carry the original atom for the XTDE1030 consistency gate. The
		// duration/date branches above rebuild sv as a numeric value but the
		// gate must still see the untouched source type, so set it last.
		sv.atom = keyAtom
		sv.hasAtom = haveAtom
		return sv, nil
	}

	if len(sk.Body) == 0 {
		if *dtMode == dataTypeNumber || *dtMode == dataTypeNumberAuto {
			return sortValue{kind: sortValueNaN}, nil
		}
		return sortValue{}, nil
	}

	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	if node != nil {
		ec.currentNode = node
		ec.contextNode = node
	}
	ec.temporaryOutputDepth++
	val, err := ec.evaluateBody(ctx, sk.Body)
	ec.temporaryOutputDepth--
	ec.currentNode = savedCurrent
	ec.contextNode = savedContext
	if err != nil {
		return sortValue{}, dynamicError(errCodeXTDE0700, "sort key evaluation error: %v", err)
	}

	implicitTZBody := ec.currentTime.Location()
	if *dtMode == dataTypeNumber {
		// Explicit data-type="number" levels skip the XTDE1030 gate entirely
		// (every value is cast to xs:double), so no atom needs recording.
		return extractNumericSortValue(val, stringifySequence(val), true, implicitTZBody), nil
	}
	if *dtMode == dataTypeNumberAuto {
		// Auto-detected numeric (a prior item flipped the mode). The gate still
		// runs for this level, so preserve the original atom/type for every
		// singleton atomic result — mirroring the select path — instead of
		// dropping it and bypassing the XTDE1030 check on later items.
		sv := extractNumericSortValue(val, stringifySequence(val), false, implicitTZBody)
		if val != nil && sequence.Len(val) == 1 {
			// Mirror the select path: atomize the singleton for the XTDE1030 gate
			// AND recompute the comparison value from the atomized typed value so
			// a schema-typed date/duration NODE body result sorts by value rather
			// than as NaN (extractNumericSortValue only converts already-atomic
			// items).
			sv = applyNumberAutoSingleton(sv, val.Get(0), implicitTZBody)
		}
		return sv, nil
	}

	sv := sortValue{kind: sortValueText, str: stringifySequence(val)}
	var keyAtom xpath3.AtomicValue
	var haveAtom bool
	if val != nil && sequence.Len(val) == 1 {
		keyAtom, haveAtom = fillSingletonSortValue(&sv, val.Get(0), dtMode, implicitTZBody)
	}
	// Carry the original atom for the XTDE1030 consistency gate, mirroring the
	// select path: record it LAST so the duration/date rebuild inside the helper
	// does not drop it, and so validateSortLevelTypes can see every singleton
	// atomic/atomized-node body result instead of bypassing the type check.
	sv.atom = keyAtom
	sv.hasAtom = haveAtom
	return sv, nil
}

// extractSortValues evaluates all sort keys for one item, returning a slice.
func extractSortValues(ctx context.Context, ec *execContext, sortKeys []*sortKey, node helium.Node, dtModes []dataTypeMode, evalState *xpath3.EvalState) ([]sortValue, error) {
	keys := make([]sortValue, len(sortKeys))
	for i, sk := range sortKeys {
		sv, err := evaluateSortKey(ctx, ec, sk, node, &dtModes[i], evalState)
		if err != nil {
			return nil, err
		}
		keys[i] = sv
	}
	return keys, nil
}

// --- Numeric extraction helpers ---

// extractNumericSortValue converts a sort key sequence to a numeric sort value.
// A single atomic item is converted directly when orderable; when numericOnly
// is true (explicit data-type=lexicon.TypeNumber), only actual numeric types
// use direct conversion per number() semantics, so dates/durations fall through
// to the fallback string → double (producing NaN). Otherwise fallback is parsed.
func extractNumericSortValue(seq xpath3.Sequence, fallback string, numericOnly bool, implicitTZ *time.Location) sortValue {
	if seq != nil && sequence.Len(seq) == 1 {
		if av, ok := seq.Get(0).(xpath3.AtomicValue); ok && (!numericOnly || av.IsNumeric()) {
			if sv, ok := atomicToNumericSortValue(av, implicitTZ); ok {
				return sv
			}
		}
	}
	return parseToNumericSortValue(fallback)
}

// applyNumberAutoSingleton finalizes a dataTypeNumberAuto singleton sort value.
// It atomizes the singleton item (recording the atom for the XTDE1030
// orderability gate) and, when the atomized typed value is orderable, replaces
// the comparison value with the numeric value derived from that typed value.
// This makes a schema-typed date/duration NODE sort by value instead of the
// NaN that extractNumericSortValue's string fallback would otherwise yield; for
// an atomic singleton it reproduces the same value extractNumericSortValue
// already computed. If atomization fails, sv is returned unchanged.
func applyNumberAutoSingleton(sv sortValue, item xpath3.Item, implicitTZ *time.Location) sortValue {
	av, err := xpath3.AtomizeItem(item)
	if err != nil {
		return sv
	}
	if nv, ok := atomicToNumericSortValue(av, implicitTZ); ok {
		sv = nv
	} else {
		sv.typeName = av.TypeName
	}
	sv.atom = av
	sv.hasAtom = true
	return sv
}

func parseToNumericSortValue(s string) sortValue {
	f := parseNumber(s)
	if math.IsNaN(f) {
		return sortValue{kind: sortValueNaN}
	}
	return sortValue{kind: sortValueNumber, num: f}
}

// atomicToNumericSortValue converts an atomic value to a numeric sort value
// for types that support ordering: numeric, duration, date/time.
// implicitTZ is used for xs:time values that lack an explicit timezone;
// pass nil to fall back to the system local timezone.
//
// It is BaseType-AWARE: a schema-derived atomic (a user-defined TypeName whose
// built-in BaseType is a numeric/date/time/duration primitive) is converted by
// its typed value, not skipped. This keeps the comparison-value conversion
// consistent with the XTDE1030 validation gate, which accepts derived types via
// xpath3.ValueCompare (itself promoting through PromoteSchemaType). The original
// av.TypeName is preserved on the returned sortValue so the gate still sees the
// untouched source type. Numerics are already BaseType-aware via IsNumeric/
// ToFloat64; the date/time/duration branches resolve the primitive type through
// PromoteSchemaType before switching on it.
func atomicToNumericSortValue(av xpath3.AtomicValue, implicitTZ *time.Location) (sortValue, bool) {
	if av.IsNumeric() {
		f := av.ToFloat64()
		if math.IsNaN(f) {
			return sortValue{kind: sortValueNaN, typeName: av.TypeName}, true
		}
		return sortValue{kind: sortValueNumber, num: f, typeName: av.TypeName}, true
	}
	// Resolve a schema-derived date/time/duration to its built-in primitive so the
	// switch matches a derived type, then preserve the original TypeName below.
	prim := xpath3.PromoteSchemaType(av)
	switch prim.TypeName {
	case xpath3.TypeYearMonthDuration:
		d := prim.DurationVal()
		f := float64(d.Months)
		if d.Negative {
			f = -f
		}
		return sortValue{kind: sortValueNumber, num: f, typeName: av.TypeName}, true
	case xpath3.TypeDayTimeDuration:
		d := prim.DurationVal()
		f := d.Seconds
		if d.Negative {
			f = -f
		}
		return sortValue{kind: sortValueNumber, num: f, typeName: av.TypeName}, true
	case xpath3.TypeDateTime, xpath3.TypeDateTimeStamp, xpath3.TypeDate:
		t := prim.TimeVal()
		t = xpath3.ApplyImplicitTZ(t, implicitTZ)
		// Use Unix seconds + fractional nanoseconds to avoid int64 overflow for large years
		f := float64(t.Unix()) + float64(t.Nanosecond())/1e9
		return sortValue{kind: sortValueNumber, num: f, typeName: av.TypeName}, true
	case xpath3.TypeTime:
		t := prim.TimeVal()
		// xs:time comparison uses reference date 1972-12-31 per F&O §10.4.4
		t = xpath3.TimeToReferenceDateTime(xpath3.ApplyImplicitTZ(t, implicitTZ))
		f := float64(t.Unix()) + float64(t.Nanosecond())/1e9
		return sortValue{kind: sortValueNumber, num: f, typeName: av.TypeName}, true
	}
	return sortValue{}, false
}

// --- Data type mode resolution ---

func initDataTypeModes(ctx context.Context, ec *execContext, sortKeys []*sortKey) ([]dataTypeMode, error) {
	modes := make([]dataTypeMode, len(sortKeys))
	for i, sk := range sortKeys {
		if sk.DataType == nil {
			continue
		}
		dt, err := sk.DataType.evaluate(ctx, ec.contextNode)
		if err != nil {
			return nil, err
		}
		if dt == lexicon.TypeNumber {
			modes[i] = dataTypeNumber
		} else {
			modes[i] = dataTypeText
		}
	}
	return modes, nil
}

func initDataTypeMode1(ctx context.Context, ec *execContext, sk *sortKey) (dataTypeMode, error) {
	if sk.DataType == nil {
		return dataTypeAuto, nil
	}
	dt, err := sk.DataType.evaluate(ctx, ec.contextNode)
	if err != nil {
		return dataTypeAuto, err
	}
	if dt == lexicon.TypeNumber {
		return dataTypeNumber, nil
	}
	return dataTypeText, nil
}

// --- Finalization (auto-detect conversion) ---

func finalizeLevels(rs *resolvedSort, dtModes []dataTypeMode, entries interface{ convertAutoNumeric(level int) }) {
	for i, m := range dtModes {
		switch m {
		case dataTypeNumber, dataTypeNumberAuto:
			rs.levels[i].mode = sortModeNumber
			entries.convertAutoNumeric(i)
		default:
			rs.levels[i].mode = sortModeText
		}
	}
}

func finalizeLevel1[T any](level *resolvedLevel, dtMode dataTypeMode, entries []keyed1[T]) {
	switch dtMode {
	case dataTypeNumber, dataTypeNumberAuto:
		level.mode = sortModeNumber
		for i := range entries {
			k := &entries[i].key
			if k.kind == sortValueText {
				*k = parseToNumericSortValue(k.str)
			}
		}
	default:
		level.mode = sortModeText
	}
}

// --- Public dispatch ---

func sortNodes(ctx context.Context, ec *execContext, nodes []helium.Node, sortKeys []*sortKey) ([]helium.Node, error) {
	// Validate sort key attributes even if there are no nodes (XTDE0030).
	for _, sk := range sortKeys {
		if err := validateSortKeyAttrs(ctx, ec, sk); err != nil {
			return nil, err
		}
	}
	if len(sortKeys) == 0 || len(nodes) == 0 {
		return nodes, nil
	}
	if len(sortKeys) == 1 {
		return sortNodes1(ctx, ec, nodes, sortKeys[0])
	}
	return sortNodesN(ctx, ec, nodes, sortKeys)
}

func sortItems(ctx context.Context, ec *execContext, items xpath3.Sequence, sortKeys []*sortKey) (xpath3.Sequence, error) {
	if len(sortKeys) == 0 || items == nil || sequence.Len(items) == 0 {
		return items, nil
	}
	if len(sortKeys) == 1 {
		return sortItems1(ctx, ec, items, sortKeys[0])
	}
	return sortItemsN(ctx, ec, items, sortKeys)
}

func sortGroups(ctx context.Context, ec *execContext, groups []fegGroup, sortKeys []*sortKey, hasKey bool) ([]fegGroup, error) {
	for _, sk := range sortKeys {
		if err := validateSortKeyAttrs(ctx, ec, sk); err != nil {
			return nil, err
		}
	}
	if len(sortKeys) == 0 || len(groups) == 0 {
		return groups, nil
	}
	if len(sortKeys) == 1 {
		return sortGroups1(ctx, ec, groups, sortKeys[0], hasKey)
	}
	return sortGroupsN(ctx, ec, groups, sortKeys, hasKey)
}

// --- Single-key sort paths ---

func sortNodes1(ctx context.Context, ec *execContext, nodes []helium.Node, sk *sortKey) ([]helium.Node, error) {
	dtMode, err := initDataTypeMode1(ctx, ec, sk)
	if err != nil {
		return nil, err
	}

	evalState := ec.sortXPathEvalState(ctx)
	evalState.SetSize(len(nodes))

	// Save and restore currentNode so current() works correctly
	// within sort key expressions (XSLT spec 13.1.4).
	savedCurrent := ec.currentNode
	defer func() { ec.currentNode = savedCurrent }()

	entries := make([]keyed1[helium.Node], len(nodes))
	for i, node := range nodes {
		evalState.SetPosition(i + 1)
		ec.currentNode = node
		sv, err := evaluateSortKey(ctx, ec, sk, node, &dtMode, evalState)
		if err != nil {
			return nil, err
		}
		entries[i] = keyed1[helium.Node]{item: node, key: sv, index: i}
	}

	// XTDE1030: check for incompatible or non-orderable sort key types.
	if err := validateSortKeyTypes1(entries, dtMode); err != nil {
		return nil, err
	}

	level, err := resolveLevel1(ctx, ec, sk)
	if err != nil {
		return nil, err
	}
	finalizeLevel1(&level, dtMode, entries)

	slices.SortFunc(entries, func(a, b keyed1[helium.Node]) int {
		if c := compareSortValues(a.key, b.key, level); c != 0 {
			return c
		}
		return cmp.Compare(a.index, b.index)
	})

	for i, e := range entries {
		nodes[i] = e.item
	}
	return nodes, nil
}

func sortItems1(ctx context.Context, ec *execContext, items xpath3.Sequence, sk *sortKey) (xpath3.Sequence, error) {
	dtMode, err := initDataTypeMode1(ctx, ec, sk)
	if err != nil {
		return nil, err
	}

	evalState := ec.sortXPathEvalState(ctx)
	// Set size/position/current() context for the FULL mixed sequence so
	// sort keys referencing position()/last()/current() compute correctly
	// (XSLT spec 13.1.4). Mirror the node-only path (sortNodes1).
	evalState.SetSize(sequence.Len(items))
	entries := make([]keyed1[xpath3.Item], sequence.Len(items))

	savedItem := ec.contextItem
	savedCurrent := ec.currentNode
	defer func() {
		ec.contextItem = savedItem
		ec.currentNode = savedCurrent
	}()

	for i := range sequence.Len(items) {
		evalState.SetPosition(i + 1)
		item := items.Get(i)
		ec.contextItem = item
		var node helium.Node
		if ni, ok := item.(xpath3.NodeItem); ok {
			node = ni.Node
			ec.contextItem = nil
			// current() resolves via ec.currentNode for node items.
			ec.currentNode = node
		} else {
			// current() resolves via ec.contextItem for non-node items.
			ec.currentNode = nil
		}
		sv, err := evaluateSortKey(ctx, ec, sk, node, &dtMode, evalState)
		if err != nil {
			return nil, err
		}
		entries[i] = keyed1[xpath3.Item]{item: item, key: sv, index: i}
	}

	// XTDE1030: check for heterogeneous sort key types that can't be compared.
	if err := validateSortKeyTypes1(entries, dtMode); err != nil {
		return nil, err
	}

	level, err := resolveLevel1(ctx, ec, sk)
	if err != nil {
		return nil, err
	}
	finalizeLevel1(&level, dtMode, entries)

	slices.SortFunc(entries, func(a, b keyed1[xpath3.Item]) int {
		if c := compareSortValues(a.key, b.key, level); c != 0 {
			return c
		}
		return cmp.Compare(a.index, b.index)
	})

	result := make(xpath3.ItemSlice, len(entries))
	for i, e := range entries {
		result[i] = e.item
	}
	return result, nil
}

// --- Multi-key sort paths ---

func sortNodesN(ctx context.Context, ec *execContext, nodes []helium.Node, sortKeys []*sortKey) ([]helium.Node, error) {
	dtModes, err := initDataTypeModes(ctx, ec, sortKeys)
	if err != nil {
		return nil, err
	}

	evalState := ec.sortXPathEvalState(ctx)
	evalState.SetSize(len(nodes))

	savedCurrent := ec.currentNode
	defer func() { ec.currentNode = savedCurrent }()

	entries := make(keyedSlice[helium.Node], len(nodes))
	for i, node := range nodes {
		evalState.SetPosition(i + 1)
		ec.currentNode = node
		keys, err := extractSortValues(ctx, ec, sortKeys, node, dtModes, evalState)
		if err != nil {
			return nil, err
		}
		entries[i] = keyedN[helium.Node]{item: node, keys: keys, index: i}
	}

	// XTDE1030: validate each sort key's type consistency across the sequence.
	if err := validateSortKeyTypesN(entries, dtModes); err != nil {
		return nil, err
	}

	rs, err := buildResolvedSort(ctx, ec, sortKeys)
	if err != nil {
		return nil, err
	}
	finalizeLevels(&rs, dtModes, entries)

	slices.SortFunc(entries, func(a, b keyedN[helium.Node]) int {
		return rs.compareKeys(a.keys, b.keys, a.index, b.index)
	})

	for i, e := range entries {
		nodes[i] = e.item
	}
	return nodes, nil
}

func sortItemsN(ctx context.Context, ec *execContext, items xpath3.Sequence, sortKeys []*sortKey) (xpath3.Sequence, error) {
	dtModes, err := initDataTypeModes(ctx, ec, sortKeys)
	if err != nil {
		return nil, err
	}

	evalState := ec.sortXPathEvalState(ctx)
	// Set size/position/current() context for the FULL mixed sequence so
	// sort keys referencing position()/last()/current() compute correctly
	// (XSLT spec 13.1.4). Mirror the node-only path (sortNodesN).
	evalState.SetSize(sequence.Len(items))
	entries := make(keyedSlice[xpath3.Item], sequence.Len(items))

	savedItem := ec.contextItem
	savedCurrent := ec.currentNode
	defer func() {
		ec.contextItem = savedItem
		ec.currentNode = savedCurrent
	}()

	for i := range sequence.Len(items) {
		evalState.SetPosition(i + 1)
		item := items.Get(i)
		ec.contextItem = item
		var node helium.Node
		if ni, ok := item.(xpath3.NodeItem); ok {
			node = ni.Node
			ec.contextItem = nil
			// current() resolves via ec.currentNode for node items.
			ec.currentNode = node
		} else {
			// current() resolves via ec.contextItem for non-node items.
			ec.currentNode = nil
		}
		keys, err := extractSortValues(ctx, ec, sortKeys, node, dtModes, evalState)
		if err != nil {
			return nil, err
		}
		entries[i] = keyedN[xpath3.Item]{item: item, keys: keys, index: i}
	}

	// XTDE1030: validate each sort key's type consistency across the sequence.
	if err := validateSortKeyTypesN(entries, dtModes); err != nil {
		return nil, err
	}

	rs, err := buildResolvedSort(ctx, ec, sortKeys)
	if err != nil {
		return nil, err
	}
	finalizeLevels(&rs, dtModes, entries)

	slices.SortFunc(entries, func(a, b keyedN[xpath3.Item]) int {
		return rs.compareKeys(a.keys, b.keys, a.index, b.index)
	})

	result := make(xpath3.ItemSlice, len(entries))
	for i, e := range entries {
		result[i] = e.item
	}
	return result, nil
}

func sortGroups1(ctx context.Context, ec *execContext, groups []fegGroup, sk *sortKey, hasKey bool) ([]fegGroup, error) {
	dtMode, err := initDataTypeMode1(ctx, ec, sk)
	if err != nil {
		return nil, err
	}

	entries := make([]keyed1[fegGroup], len(groups))
	if err := ec.withSortGroupContext(groups, hasKey, func(i int, node helium.Node) error {
		sv, err := evaluateSortKey(ctx, ec, sk, node, &dtMode, nil)
		if err != nil {
			return err
		}
		entries[i] = keyed1[fegGroup]{item: groups[i], key: sv, index: i}
		return nil
	}); err != nil {
		return nil, err
	}

	// XTDE1030: check for incompatible or non-orderable sort key types.
	if err := validateSortKeyTypes1(entries, dtMode); err != nil {
		return nil, err
	}

	level, err := resolveLevel1(ctx, ec, sk)
	if err != nil {
		return nil, err
	}
	finalizeLevel1(&level, dtMode, entries)

	slices.SortFunc(entries, func(a, b keyed1[fegGroup]) int {
		if c := compareSortValues(a.key, b.key, level); c != 0 {
			return c
		}
		return cmp.Compare(a.index, b.index)
	})

	for i, e := range entries {
		groups[i] = e.item
	}
	return groups, nil
}

func sortGroupsN(ctx context.Context, ec *execContext, groups []fegGroup, sortKeys []*sortKey, hasKey bool) ([]fegGroup, error) {
	dtModes, err := initDataTypeModes(ctx, ec, sortKeys)
	if err != nil {
		return nil, err
	}

	entries := make(keyedSlice[fegGroup], len(groups))
	if err := ec.withSortGroupContext(groups, hasKey, func(i int, node helium.Node) error {
		keys, err := extractSortValues(ctx, ec, sortKeys, node, dtModes, nil)
		if err != nil {
			return err
		}
		entries[i] = keyedN[fegGroup]{item: groups[i], keys: keys, index: i}
		return nil
	}); err != nil {
		return nil, err
	}

	// XTDE1030: validate each sort key's type consistency across the sequence.
	if err := validateSortKeyTypesN(entries, dtModes); err != nil {
		return nil, err
	}

	rs, err := buildResolvedSort(ctx, ec, sortKeys)
	if err != nil {
		return nil, err
	}
	finalizeLevels(&rs, dtModes, entries)

	slices.SortFunc(entries, func(a, b keyedN[fegGroup]) int {
		return rs.compareKeys(a.keys, b.keys, a.index, b.index)
	})

	for i, e := range entries {
		groups[i] = e.item
	}
	return groups, nil
}

func parseNumber(s string) float64 {
	s = strings.TrimSpace(s)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return math.NaN()
	}
	return f
}

func (ec *execContext) execPerformSort(ctx context.Context, inst *performSortInst) error {
	var seq xpath3.Sequence

	if inst.Select != nil {
		result, err := ec.evalXPath(ctx, inst.Select, ec.contextNode)
		if err != nil {
			return err
		}
		seq = result.Sequence()
	} else if len(inst.Body) > 0 {
		// Body acts as sequence constructor: evaluate items individually
		// so that each text item remains a separate sortable unit.
		var err error
		seq, err = ec.evaluateBodyAsSequence(ctx, inst.Body)
		if err != nil {
			return err
		}
	}
	// Validate sort key attributes even when the selected sequence is empty
	// (e.g. a bad collation must still raise XTDE1035). sortNodes/sortGroups
	// validate regardless of input; mirror that for the empty-input case here.
	for _, sk := range inst.Sort {
		if err := validateSortKeyAttrs(ctx, ec, sk); err != nil {
			return err
		}
	}

	if seq == nil || sequence.Len(seq) == 0 {
		return nil
	}

	// Try to extract nodes for node-based sorting
	nodes, allNodes := xpath3.NodesFrom(seq)
	if allNodes && len(nodes) > 0 {
		if len(inst.Sort) > 0 {
			var err error
			nodes, err = sortNodes(ctx, ec, nodes, inst.Sort)
			if err != nil {
				return err
			}
		}

		savedCurrent := ec.currentNode
		savedContext := ec.contextNode
		savedPos := ec.position
		savedSize := ec.size
		ec.size = len(nodes)
		defer func() {
			ec.currentNode = savedCurrent
			ec.contextNode = savedContext
			ec.position = savedPos
			ec.size = savedSize
		}()

		// Output sorted nodes
		for _, node := range nodes {
			if err := ec.copyNodeToOutput(node); err != nil {
				return err
			}
		}
		return nil
	}

	// Atomic sequence: sort by string value and output as text items
	if len(inst.Sort) > 0 {
		var err error
		seq, err = sortItems(ctx, ec, seq, inst.Sort)
		if err != nil {
			return err
		}
	}

	// In capture mode (e.g. inside xsl:function), push sorted items
	// directly so the caller receives a proper sequence of atomic values
	// rather than merged text nodes.
	out := ec.currentOutput()
	if out.captureItems && out.doc != nil && out.current == out.doc.DocumentElement() {
		out.pendingItems = append(out.pendingItems, sequence.Materialize(seq)...)
		if seq != nil && sequence.Len(seq) > 0 {
			out.noteOutput()
		}
		return nil
	}

	// Output atomic items separated by spaces
	idx := 0
	for item := range sequence.Items(seq) {
		if idx > 0 {
			sep := ec.resultDoc.CreateText([]byte(" "))
			if err := ec.addNode(sep); err != nil {
				return err
			}
		}
		av, ok := item.(xpath3.AtomicValue)
		if !ok {
			continue
		}
		s, err := xpath3.AtomicToString(av)
		if err != nil {
			continue
		}
		text := ec.resultDoc.CreateText([]byte(s))
		if err := ec.addNode(text); err != nil {
			return err
		}
		idx++
	}
	return nil
}
