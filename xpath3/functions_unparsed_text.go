package xpath3

import (
	"context"
	"errors"

	"github.com/lestrrat-go/helium/internal/unparsedtext"
)

func init() {
	registerFn("unparsed-text", 1, 2, fnUnparsedText)
	registerFn("unparsed-text-available", 1, 2, fnUnparsedTextAvailable)
	registerFn("unparsed-text-lines", 1, 2, fnUnparsedTextLines)
}

func unparsedTextConfig(ctx context.Context) *unparsedtext.Config {
	ec := getFnContext(ctx)
	if ec == nil {
		return &unparsedtext.Config{
			BaseURI: baseURIFromContext(ctx),
		}
	}
	cfg := &unparsedtext.Config{
		BaseURI:    ec.baseURI,
		HTTPClient: ec.httpClient,
		MaxBytes:   ec.maxResourceBytes,
	}
	if ec.uriResolver != nil {
		cfg.URIResolver = ec.uriResolver
	}
	return cfg
}

func wrapUnparsedTextError(err error) error {
	if err == nil {
		return nil
	}
	if ue, ok := errors.AsType[*unparsedtext.Error](err); ok {
		return &XPathError{Code: ue.Code, Message: ue.Message}
	}
	return err
}

func fnUnparsedText(ctx context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return validNilSequence, nil
	}
	href, err := coerceArgToString(ctx, args[0])
	if err != nil {
		return nil, err
	}
	encoding := ""
	if len(args) > 1 {
		encoding, err = coerceArgToString(ctx, args[1])
		if err != nil {
			return nil, err
		}
	}

	text, err := unparsedtext.LoadText(ctx, unparsedTextConfig(ctx), href, encoding)
	if err != nil {
		return nil, wrapUnparsedTextError(err)
	}
	return SingleString(text), nil
}

func fnUnparsedTextAvailable(ctx context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return SingleBoolean(false), nil
	}
	href, err := coerceArgToString(ctx, args[0])
	if err != nil {
		return nil, err
	}
	encoding := ""
	if len(args) > 1 {
		encoding, err = coerceArgToString(ctx, args[1])
		if err != nil {
			return nil, err
		}
	}

	available := unparsedtext.IsAvailable(ctx, unparsedTextConfig(ctx), href, encoding)
	return SingleBoolean(available), nil
}

func fnUnparsedTextLines(ctx context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return validNilSequence, nil
	}
	href, err := coerceArgToString(ctx, args[0])
	if err != nil {
		return nil, err
	}
	encoding := ""
	if len(args) > 1 {
		encoding, err = coerceArgToString(ctx, args[1])
		if err != nil {
			return nil, err
		}
	}

	// Bound line production by the EFFECTIVE budget actually in force for this
	// call — the smaller of the node-set cap (maxNodes) and the remaining op
	// budget — so an OpLimit far below maxNodes stops splitting after ~OpLimit
	// lines instead of allocating a []string proportional to the resource's full
	// line count. LoadTextLinesBounded stops once limit+1 lines would be produced;
	// the +1 lets the op charge below detect the overflow precisely.
	ec := getFnContext(ctx)
	maxNodes := fnMaxNodes(ec)
	limit := maxNodes
	opBounded := false
	if rem, bounded := fnRemainingOps(ec); bounded && rem < limit {
		limit = rem
		opBounded = true
	}
	// SplitLinesBounded treats limit <= 0 as "no bound"; a fully-exhausted op
	// budget (rem == 0) must still cap allocation, so floor the splitter bound at
	// 1. The op charge below then rejects the single produced line via ErrOpLimit.
	splitLimit := limit
	if opBounded && splitLimit < 1 {
		splitLimit = 1
	}

	lines, truncated, err := unparsedtext.LoadTextLinesBounded(ctx, unparsedTextConfig(ctx), href, encoding, splitLimit)
	if err != nil {
		return nil, wrapUnparsedTextError(err)
	}
	// Charge the produced lines against the op-counter (honoring cancellation)
	// BEFORE materializing one Item per line, matching how functions_array /
	// functions_map guard one-shot materialization. When the op budget is the
	// binding constraint this surfaces ErrOpLimit; truncation past the node-set
	// cap surfaces ErrNodeSetLimit.
	if err := fnCountOps(ctx, ec, len(lines)); err != nil {
		return nil, err
	}
	if truncated {
		return nil, ErrNodeSetLimit
	}
	result := make(ItemSlice, len(lines))
	for i, line := range lines {
		result[i] = AtomicValue{TypeName: TypeString, Value: line}
	}
	return result, nil
}
