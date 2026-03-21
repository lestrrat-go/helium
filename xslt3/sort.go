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
	"github.com/lestrrat-go/helium/xpath3"
)

// SortKey is a compiled xsl:sort specification.
type SortKey struct {
	Select    *xpath3.Expression
	Body      []Instruction // sequence constructor (when select is absent)
	Order     *AVT          // "ascending" or "descending"
	DataType  *AVT          // "text" or "number"
	CaseOrder *AVT          // "upper-first" or "lower-first"
	Lang      *AVT
	Collation *AVT // collation URI
}

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
	dataTypeNumber                         // explicit data-type="number"
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
}

// resolvedLevel holds the fully resolved configuration for one sort level.
type resolvedLevel struct {
	mode    sortMode
	desc    bool
	compare func(a, b string) int // collation comparator; nil = codepoint
}

// resolvedSort holds the fully resolved per-level sort configuration.
// Built once before sorting; no per-comparison AVT evaluation needed.
type resolvedSort struct {
	levels []resolvedLevel
}

// --- Single-key entry types ---

// keyedNode1 pairs a node with a single inline sort key and original index.
type keyedNode1 struct {
	item  helium.Node
	key   sortValue
	index int
}

func (k keyedNode1) keyType() string { return k.key.typeName }

// keyedItem1 pairs an item with a single inline sort key and original index.
type keyedItem1 struct {
	item  xpath3.Item
	key   sortValue
	index int
}

func (k keyedItem1) keyType() string { return k.key.typeName }

// --- Multi-key entry types ---

// keyedNode pairs a node with its pre-extracted typed sort keys and original index.
type keyedNode struct {
	item  helium.Node
	keys  []sortValue
	index int
}

// keyedItem pairs an item with its pre-extracted typed sort keys and original index.
type keyedItem struct {
	item  xpath3.Item
	keys  []sortValue
	index int
}

// keyedGroup1 pairs a for-each-group group with a single inline sort key.
type keyedGroup1 struct {
	group fegGroup
	key   sortValue
	index int
}

func (k keyedGroup1) keyType() string { return k.key.typeName }

// keyedGroup pairs a for-each-group group with its pre-extracted sort keys.
type keyedGroup struct {
	group fegGroup
	keys  []sortValue
	index int
}

type keyedGroups []keyedGroup

func (kg keyedGroups) convertAutoNumeric(level int) {
	for j := range kg {
		sv := &kg[j].keys[level]
		if sv.kind == sortValueText {
			*sv = parseToNumericSortValue(sv.str)
		}
	}
}

// --- Sort level resolution ---

// buildResolvedSort evaluates AVTs for order once and builds a resolvedLevel per sort level.
func buildResolvedSort(ctx context.Context, ec *execContext, sortKeys []*SortKey) (resolvedSort, error) {
	levels := make([]resolvedLevel, len(sortKeys))
	for i, sk := range sortKeys {
		order := "ascending"
		if sk.Order != nil {
			var err error
			order, err = sk.Order.evaluate(ctx, ec.contextNode)
			if err != nil {
				return resolvedSort{}, err
			}
		}
		levels[i] = resolvedLevel{
			desc: order == "descending",
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
			uri := buildImplicitCollationURI(ctx, ec, sk)
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

func resolveLevel1(ctx context.Context, ec *execContext, sk *SortKey) (resolvedLevel, error) {
	order := "ascending"
	if sk.Order != nil {
		var err error
		order, err = sk.Order.evaluate(ctx, ec.contextNode)
		if err != nil {
			return resolvedLevel{}, err
		}
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
	rl := resolvedLevel{desc: order == "descending"}
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
		uri := buildImplicitCollationURI(ctx, ec, sk)
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
// case-order attributes when no explicit collation is specified.
func buildImplicitCollationURI(ctx context.Context, ec *execContext, sk *SortKey) string {
	var params []string
	if sk.Lang != nil {
		lang, err := sk.Lang.evaluate(ctx, ec.contextNode)
		if err == nil && lang != "" {
			params = append(params, "lang="+lang)
		}
	}
	if sk.CaseOrder != nil {
		co, err := sk.CaseOrder.evaluate(ctx, ec.contextNode)
		if err == nil && co != "" {
			switch co {
			case "upper-first":
				params = append(params, "caseFirst=upper")
			case "lower-first":
				params = append(params, "caseFirst=lower")
			}
		}
	}
	if len(params) == 0 {
		return ""
	}
	return "http://www.w3.org/2013/collation/UCA?" + strings.Join(params, ";")
}

// validateSortKeyAttrs validates sort key attribute values (lang, collation, etc.)
// regardless of whether there are nodes to sort.
func validateSortKeyAttrs(ctx context.Context, ec *execContext, sk *SortKey) error {
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

// sortKeyFamily returns a family name for type compatibility checking.
// All numeric types belong to the "numeric" family and are mutually comparable.
func sortKeyFamily(tn string) string {
	switch tn {
	case xpath3.TypeInteger, xpath3.TypeDecimal, xpath3.TypeFloat, xpath3.TypeDouble,
		xpath3.TypeLong, xpath3.TypeInt, xpath3.TypeShort, xpath3.TypeByte,
		xpath3.TypeUnsignedLong, xpath3.TypeUnsignedInt, xpath3.TypeUnsignedShort, xpath3.TypeUnsignedByte,
		xpath3.TypeNonNegativeInteger, xpath3.TypeNonPositiveInteger,
		xpath3.TypePositiveInteger, xpath3.TypeNegativeInteger:
		return "numeric"
	}
	return tn
}

// checkSortKeyTypeConsistency raises XTDE1030 if sort keys have incompatible types.
// For example, mixing xs:untypedAtomic with xs:date is invalid because they
// can't be compared using the lt operator without explicit casting.
// All numeric types (integer, decimal, float, double, etc.) are mutually compatible.
func checkSortKeyTypeConsistency[T interface{ keyType() string }](entries []T) error {
	var firstNonString string
	var firstFamily string
	for _, e := range entries {
		tn := e.keyType()
		if tn == "" || tn == xpath3.TypeString || tn == xpath3.TypeUntypedAtomic {
			continue
		}
		// xs:duration is only partially ordered and cannot be used as a sort key
		if tn == xpath3.TypeDuration {
			return dynamicError(errCodeXTDE1030,
				"sort keys of type %s are only partially ordered", tn)
		}
		fam := sortKeyFamily(tn)
		if firstNonString == "" {
			firstNonString = tn
			firstFamily = fam
		} else if firstFamily != fam {
			return dynamicError(errCodeXTDE1030,
				"sort keys have incompatible types: %s and %s", firstNonString, tn)
		}
	}
	// If we have a non-string type AND string/untypedAtomic types, they're incompatible
	if firstNonString != "" {
		for _, e := range entries {
			tn := e.keyType()
			if tn == xpath3.TypeUntypedAtomic || tn == xpath3.TypeString {
				return dynamicError(errCodeXTDE1030,
					"sort keys have incompatible types: %s and %s", firstNonString, tn)
			}
		}
	}
	return nil
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

// evaluateSortKey evaluates a single sort key for one item and returns its typed value.
// Uses EvaluateReuse when evalState is non-nil to avoid per-item evalContext allocation.
func evaluateSortKey(ctx context.Context, ec *execContext, sk *SortKey, node helium.Node, dtMode *dataTypeMode, evalState *xpath3.EvalState) (sortValue, error) {
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
			result, err = sk.Select.EvaluateReuse(evalState, node)
		} else {
			xpathCtx := ec.newXPathContext(node)
			r, e := sk.Select.Evaluate(xpathCtx, node)
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
		if len(seq) > 1 {
			return sortValue{}, dynamicError(errCodeXTTE1020,
				"sort key value is a sequence of %d items; a single value is required", len(seq))
		}

		implicitTZ := ec.currentTime.Location()
		if *dtMode == dataTypeNumber {
			// Explicit data-type="number": use number() semantics.
			// Dates/times are not numeric → convert via string → NaN.
			return extractNumericValueExplicit(seq, result, implicitTZ), nil
		}
		if *dtMode == dataTypeNumberAuto {
			// Auto-detected numeric: preserve date/time ordering.
			sv := extractNumericValueFromResult(seq, result, implicitTZ)
			// Preserve original type name for XTDE1030 checking
			if len(seq) == 1 {
				if av, ok := seq[0].(xpath3.AtomicValue); ok {
					sv.typeName = av.TypeName
				}
			}
			return sv, nil
		}

		sv := sortValue{kind: sortValueText, str: result.StringValue()}
		if len(seq) == 1 {
			switch v := seq[0].(type) {
			case xpath3.AtomicValue:
				sv.typeName = v.TypeName
				if *dtMode == dataTypeAuto && v.IsNumeric() {
					*dtMode = dataTypeNumberAuto
				}
				// Duration types: use numeric comparison based on total months or seconds
				if v.TypeName == xpath3.TypeYearMonthDuration || v.TypeName == xpath3.TypeDayTimeDuration {
					d := v.DurationVal()
					var f float64
					if v.TypeName == xpath3.TypeYearMonthDuration {
						f = float64(d.Months)
					} else {
						f = d.Seconds
					}
					if d.Negative {
						f = -f
					}
					sv = sortValue{kind: sortValueNumber, num: f, typeName: v.TypeName}
					if *dtMode == dataTypeAuto {
						*dtMode = dataTypeNumberAuto
					}
				}
				// Date/time types: use Unix seconds for numeric comparison
				switch v.TypeName {
				case xpath3.TypeDateTime, xpath3.TypeDateTimeStamp, xpath3.TypeDate:
					t := xpath3.ApplyImplicitTZ(v.TimeVal(), implicitTZ)
					f := float64(t.Unix()) + float64(t.Nanosecond())/1e9
					sv = sortValue{kind: sortValueNumber, num: f, typeName: v.TypeName}
					if *dtMode == dataTypeAuto {
						*dtMode = dataTypeNumberAuto
					}
				case xpath3.TypeTime:
					// xs:time comparison uses reference date 1972-12-31 per F&O §10.4.4
					t := xpath3.TimeToReferenceDateTime(xpath3.ApplyImplicitTZ(v.TimeVal(), implicitTZ))
					f := float64(t.Unix()) + float64(t.Nanosecond())/1e9
					sv = sortValue{kind: sortValueNumber, num: f, typeName: v.TypeName}
					if *dtMode == dataTypeAuto {
						*dtMode = dataTypeNumberAuto
					}
				}
			case xpath3.NodeItem:
				// Atomize to get the typed value for type consistency checks
				if av, err := xpath3.AtomizeItem(v); err == nil {
					sv.typeName = av.TypeName
				}
			}
		}
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
		return extractNumericValueExplicitSeq(val, implicitTZBody), nil
	}
	if *dtMode == dataTypeNumberAuto {
		return extractNumericValueFromSeq(val, implicitTZBody), nil
	}

	sv := sortValue{kind: sortValueText, str: stringifySequence(val)}
	if *dtMode == dataTypeAuto && len(val) == 1 {
		if av, ok := val[0].(xpath3.AtomicValue); ok {
			if av.IsNumeric() {
				*dtMode = dataTypeNumberAuto
			} else if av.TypeName == xpath3.TypeYearMonthDuration || av.TypeName == xpath3.TypeDayTimeDuration {
				d := av.DurationVal()
				var f float64
				if av.TypeName == xpath3.TypeYearMonthDuration {
					f = float64(d.Months)
				} else {
					f = d.Seconds
				}
				if d.Negative {
					f = -f
				}
				sv = sortValue{kind: sortValueNumber, num: f, typeName: av.TypeName}
				*dtMode = dataTypeNumberAuto
			}
		}
	}
	return sv, nil
}

// extractSortValues evaluates all sort keys for one item, returning a slice.
func extractSortValues(ctx context.Context, ec *execContext, sortKeys []*SortKey, node helium.Node, dtModes []dataTypeMode, evalState *xpath3.EvalState) ([]sortValue, error) {
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

// extractNumericValueExplicit handles explicit data-type="number".
// Per XSLT spec, sort key values are converted using number() semantics.
// Only actual numeric types use direct conversion; others (date, duration)
// fall through to string → double (producing NaN for dates).
func extractNumericValueExplicit(seq xpath3.Sequence, result xpath3.Result, implicitTZ *time.Location) sortValue {
	if len(seq) == 1 {
		if av, ok := seq[0].(xpath3.AtomicValue); ok && av.IsNumeric() {
			if sv, ok := atomicToNumericSortValue(av, implicitTZ); ok {
				return sv
			}
		}
	}
	return parseToNumericSortValue(result.StringValue())
}

func extractNumericValueExplicitSeq(seq xpath3.Sequence, implicitTZ *time.Location) sortValue {
	if len(seq) == 1 {
		if av, ok := seq[0].(xpath3.AtomicValue); ok && av.IsNumeric() {
			if sv, ok := atomicToNumericSortValue(av, implicitTZ); ok {
				return sv
			}
		}
	}
	return parseToNumericSortValue(stringifySequence(seq))
}

func extractNumericValueFromResult(seq xpath3.Sequence, result xpath3.Result, implicitTZ *time.Location) sortValue {
	if len(seq) == 1 {
		if av, ok := seq[0].(xpath3.AtomicValue); ok {
			if sv, ok := atomicToNumericSortValue(av, implicitTZ); ok {
				return sv
			}
		}
	}
	return parseToNumericSortValue(result.StringValue())
}

func extractNumericValueFromSeq(seq xpath3.Sequence, implicitTZ *time.Location) sortValue {
	if len(seq) == 1 {
		if av, ok := seq[0].(xpath3.AtomicValue); ok {
			if sv, ok := atomicToNumericSortValue(av, implicitTZ); ok {
				return sv
			}
		}
	}
	return parseToNumericSortValue(stringifySequence(seq))
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
func atomicToNumericSortValue(av xpath3.AtomicValue, implicitTZ *time.Location) (sortValue, bool) {
	if av.IsNumeric() {
		f := av.ToFloat64()
		if math.IsNaN(f) {
			return sortValue{kind: sortValueNaN, typeName: av.TypeName}, true
		}
		return sortValue{kind: sortValueNumber, num: f, typeName: av.TypeName}, true
	}
	switch av.TypeName {
	case xpath3.TypeYearMonthDuration:
		d := av.DurationVal()
		f := float64(d.Months)
		if d.Negative {
			f = -f
		}
		return sortValue{kind: sortValueNumber, num: f, typeName: av.TypeName}, true
	case xpath3.TypeDayTimeDuration:
		d := av.DurationVal()
		f := d.Seconds
		if d.Negative {
			f = -f
		}
		return sortValue{kind: sortValueNumber, num: f, typeName: av.TypeName}, true
	case xpath3.TypeDateTime, xpath3.TypeDateTimeStamp, xpath3.TypeDate:
		t := av.TimeVal()
		t = xpath3.ApplyImplicitTZ(t, implicitTZ)
		// Use Unix seconds + fractional nanoseconds to avoid int64 overflow for large years
		f := float64(t.Unix()) + float64(t.Nanosecond())/1e9
		return sortValue{kind: sortValueNumber, num: f, typeName: av.TypeName}, true
	case xpath3.TypeTime:
		t := av.TimeVal()
		// xs:time comparison uses reference date 1972-12-31 per F&O §10.4.4
		t = xpath3.TimeToReferenceDateTime(xpath3.ApplyImplicitTZ(t, implicitTZ))
		f := float64(t.Unix()) + float64(t.Nanosecond())/1e9
		return sortValue{kind: sortValueNumber, num: f, typeName: av.TypeName}, true
	}
	return sortValue{}, false
}

// --- Data type mode resolution ---

func initDataTypeModes(ctx context.Context, ec *execContext, sortKeys []*SortKey) ([]dataTypeMode, error) {
	modes := make([]dataTypeMode, len(sortKeys))
	for i, sk := range sortKeys {
		if sk.DataType == nil {
			continue
		}
		dt, err := sk.DataType.evaluate(ctx, ec.contextNode)
		if err != nil {
			return nil, err
		}
		if dt == "number" {
			modes[i] = dataTypeNumber
		} else {
			modes[i] = dataTypeText
		}
	}
	return modes, nil
}

func initDataTypeMode1(ctx context.Context, ec *execContext, sk *SortKey) (dataTypeMode, error) {
	if sk.DataType == nil {
		return dataTypeAuto, nil
	}
	dt, err := sk.DataType.evaluate(ctx, ec.contextNode)
	if err != nil {
		return dataTypeAuto, err
	}
	if dt == "number" {
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

type keyedNodes []keyedNode

func (kn keyedNodes) convertAutoNumeric(level int) {
	for j := range kn {
		sv := &kn[j].keys[level]
		if sv.kind == sortValueText {
			*sv = parseToNumericSortValue(sv.str)
		}
	}
}

type keyedItems []keyedItem

func (ki keyedItems) convertAutoNumeric(level int) {
	for j := range ki {
		sv := &ki[j].keys[level]
		if sv.kind == sortValueText {
			*sv = parseToNumericSortValue(sv.str)
		}
	}
}

func finalizeLevel1Nodes(level *resolvedLevel, dtMode dataTypeMode, entries []keyedNode1) {
	switch dtMode {
	case dataTypeNumber, dataTypeNumberAuto:
		level.mode = sortModeNumber
		for i := range entries {
			if entries[i].key.kind == sortValueText {
				entries[i].key = parseToNumericSortValue(entries[i].key.str)
			}
		}
	default:
		level.mode = sortModeText
	}
}

func finalizeLevel1Items(level *resolvedLevel, dtMode dataTypeMode, entries []keyedItem1) {
	switch dtMode {
	case dataTypeNumber, dataTypeNumberAuto:
		level.mode = sortModeNumber
		for i := range entries {
			if entries[i].key.kind == sortValueText {
				entries[i].key = parseToNumericSortValue(entries[i].key.str)
			}
		}
	default:
		level.mode = sortModeText
	}
}

func finalizeLevel1Groups(level *resolvedLevel, dtMode dataTypeMode, entries []keyedGroup1) {
	switch dtMode {
	case dataTypeNumber, dataTypeNumberAuto:
		level.mode = sortModeNumber
		for i := range entries {
			if entries[i].key.kind == sortValueText {
				entries[i].key = parseToNumericSortValue(entries[i].key.str)
			}
		}
	default:
		level.mode = sortModeText
	}
}

// --- Public dispatch ---

func sortNodes(ctx context.Context, ec *execContext, nodes []helium.Node, sortKeys []*SortKey) ([]helium.Node, error) {
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

func sortItems(ctx context.Context, ec *execContext, items xpath3.Sequence, sortKeys []*SortKey) (xpath3.Sequence, error) {
	if len(sortKeys) == 0 || len(items) == 0 {
		return items, nil
	}
	if len(sortKeys) == 1 {
		return sortItems1(ctx, ec, items, sortKeys[0])
	}
	return sortItemsN(ctx, ec, items, sortKeys)
}

func sortGroups(ctx context.Context, ec *execContext, groups []fegGroup, sortKeys []*SortKey, hasKey bool) ([]fegGroup, error) {
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

func sortNodes1(ctx context.Context, ec *execContext, nodes []helium.Node, sk *SortKey) ([]helium.Node, error) {
	dtMode, err := initDataTypeMode1(ctx, ec, sk)
	if err != nil {
		return nil, err
	}

	evalState := ec.sortXPathEvalState()
	evalState.SetSize(len(nodes))

	// Save and restore currentNode so current() works correctly
	// within sort key expressions (XSLT spec 13.1.4).
	savedCurrent := ec.currentNode
	defer func() { ec.currentNode = savedCurrent }()

	entries := make([]keyedNode1, len(nodes))
	for i, node := range nodes {
		evalState.SetPosition(i + 1)
		ec.currentNode = node
		sv, err := evaluateSortKey(ctx, ec, sk, node, &dtMode, evalState)
		if err != nil {
			return nil, err
		}
		entries[i] = keyedNode1{item: node, key: sv, index: i}
	}

	// XTDE1030: check for incompatible or non-orderable sort key types.
	if err := checkSortKeyTypeConsistency(entries); err != nil {
		return nil, err
	}

	level, err := resolveLevel1(ctx, ec, sk)
	if err != nil {
		return nil, err
	}
	finalizeLevel1Nodes(&level, dtMode, entries)

	slices.SortFunc(entries, func(a, b keyedNode1) int {
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

func sortItems1(ctx context.Context, ec *execContext, items xpath3.Sequence, sk *SortKey) (xpath3.Sequence, error) {
	dtMode, err := initDataTypeMode1(ctx, ec, sk)
	if err != nil {
		return nil, err
	}

	evalState := ec.sortXPathEvalState()
	entries := make([]keyedItem1, len(items))

	savedItem := ec.contextItem
	defer func() { ec.contextItem = savedItem }()

	for i, item := range items {
		ec.contextItem = item
		var node helium.Node
		if ni, ok := item.(xpath3.NodeItem); ok {
			node = ni.Node
			ec.contextItem = nil
		}
		sv, err := evaluateSortKey(ctx, ec, sk, node, &dtMode, evalState)
		if err != nil {
			return nil, err
		}
		entries[i] = keyedItem1{item: item, key: sv, index: i}
	}

	// XTDE1030: check for heterogeneous sort key types that can't be compared.
	if err := checkSortKeyTypeConsistency(entries); err != nil {
		return nil, err
	}

	level, err := resolveLevel1(ctx, ec, sk)
	if err != nil {
		return nil, err
	}
	finalizeLevel1Items(&level, dtMode, entries)

	slices.SortFunc(entries, func(a, b keyedItem1) int {
		if c := compareSortValues(a.key, b.key, level); c != 0 {
			return c
		}
		return cmp.Compare(a.index, b.index)
	})

	for i, e := range entries {
		items[i] = e.item
	}
	return items, nil
}

// --- Multi-key sort paths ---

func sortNodesN(ctx context.Context, ec *execContext, nodes []helium.Node, sortKeys []*SortKey) ([]helium.Node, error) {
	dtModes, err := initDataTypeModes(ctx, ec, sortKeys)
	if err != nil {
		return nil, err
	}

	evalState := ec.sortXPathEvalState()
	evalState.SetSize(len(nodes))

	savedCurrent := ec.currentNode
	defer func() { ec.currentNode = savedCurrent }()

	entries := make(keyedNodes, len(nodes))
	for i, node := range nodes {
		evalState.SetPosition(i + 1)
		ec.currentNode = node
		keys, err := extractSortValues(ctx, ec, sortKeys, node, dtModes, evalState)
		if err != nil {
			return nil, err
		}
		entries[i] = keyedNode{item: node, keys: keys, index: i}
	}

	rs, err := buildResolvedSort(ctx, ec, sortKeys)
	if err != nil {
		return nil, err
	}
	finalizeLevels(&rs, dtModes, entries)

	slices.SortFunc(entries, func(a, b keyedNode) int {
		return rs.compareKeys(a.keys, b.keys, a.index, b.index)
	})

	for i, e := range entries {
		nodes[i] = e.item
	}
	return nodes, nil
}

func sortItemsN(ctx context.Context, ec *execContext, items xpath3.Sequence, sortKeys []*SortKey) (xpath3.Sequence, error) {
	dtModes, err := initDataTypeModes(ctx, ec, sortKeys)
	if err != nil {
		return nil, err
	}

	evalState := ec.sortXPathEvalState()
	entries := make(keyedItems, len(items))

	savedItem := ec.contextItem
	defer func() { ec.contextItem = savedItem }()

	for i, item := range items {
		ec.contextItem = item
		var node helium.Node
		if ni, ok := item.(xpath3.NodeItem); ok {
			node = ni.Node
			ec.contextItem = nil
		}
		keys, err := extractSortValues(ctx, ec, sortKeys, node, dtModes, evalState)
		if err != nil {
			return nil, err
		}
		entries[i] = keyedItem{item: item, keys: keys, index: i}
	}

	rs, err := buildResolvedSort(ctx, ec, sortKeys)
	if err != nil {
		return nil, err
	}
	finalizeLevels(&rs, dtModes, entries)

	slices.SortFunc(entries, func(a, b keyedItem) int {
		return rs.compareKeys(a.keys, b.keys, a.index, b.index)
	})

	for i, e := range entries {
		items[i] = e.item
	}
	return items, nil
}

func sortGroups1(ctx context.Context, ec *execContext, groups []fegGroup, sk *SortKey, hasKey bool) ([]fegGroup, error) {
	dtMode, err := initDataTypeMode1(ctx, ec, sk)
	if err != nil {
		return nil, err
	}

	entries := make([]keyedGroup1, len(groups))
	if err := ec.withSortGroupContext(groups, hasKey, func(i int, node helium.Node) error {
		sv, err := evaluateSortKey(ctx, ec, sk, node, &dtMode, nil)
		if err != nil {
			return err
		}
		entries[i] = keyedGroup1{group: groups[i], key: sv, index: i}
		return nil
	}); err != nil {
		return nil, err
	}

	if err := checkSortKeyTypeConsistency(entries); err != nil {
		return nil, err
	}

	level, err := resolveLevel1(ctx, ec, sk)
	if err != nil {
		return nil, err
	}
	finalizeLevel1Groups(&level, dtMode, entries)

	slices.SortFunc(entries, func(a, b keyedGroup1) int {
		if c := compareSortValues(a.key, b.key, level); c != 0 {
			return c
		}
		return cmp.Compare(a.index, b.index)
	})

	for i, e := range entries {
		groups[i] = e.group
	}
	return groups, nil
}

func sortGroupsN(ctx context.Context, ec *execContext, groups []fegGroup, sortKeys []*SortKey, hasKey bool) ([]fegGroup, error) {
	dtModes, err := initDataTypeModes(ctx, ec, sortKeys)
	if err != nil {
		return nil, err
	}

	entries := make(keyedGroups, len(groups))
	if err := ec.withSortGroupContext(groups, hasKey, func(i int, node helium.Node) error {
		keys, err := extractSortValues(ctx, ec, sortKeys, node, dtModes, nil)
		if err != nil {
			return err
		}
		entries[i] = keyedGroup{group: groups[i], keys: keys, index: i}
		return nil
	}); err != nil {
		return nil, err
	}

	rs, err := buildResolvedSort(ctx, ec, sortKeys)
	if err != nil {
		return nil, err
	}
	finalizeLevels(&rs, dtModes, entries)

	slices.SortFunc(entries, func(a, b keyedGroup) int {
		return rs.compareKeys(a.keys, b.keys, a.index, b.index)
	})

	for i, e := range entries {
		groups[i] = e.group
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
