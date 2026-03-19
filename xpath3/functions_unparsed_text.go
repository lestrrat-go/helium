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
	if len(args[0]) == 0 {
		return nil, nil
	}
	href, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	encoding := ""
	if len(args) > 1 {
		encoding, err = coerceArgToString(args[1])
		if err != nil {
			return nil, err
		}
	}

	text, err := unparsedtext.LoadText(unparsedTextConfig(ctx), href, encoding)
	if err != nil {
		return nil, wrapUnparsedTextError(err)
	}
	return SingleString(text), nil
}

func fnUnparsedTextAvailable(ctx context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return SingleBoolean(false), nil
	}
	href, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	encoding := ""
	if len(args) > 1 {
		encoding, err = coerceArgToString(args[1])
		if err != nil {
			return nil, err
		}
	}

	available := unparsedtext.IsAvailable(unparsedTextConfig(ctx), href, encoding)
	return SingleBoolean(available), nil
}

func fnUnparsedTextLines(ctx context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	href, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	encoding := ""
	if len(args) > 1 {
		encoding, err = coerceArgToString(args[1])
		if err != nil {
			return nil, err
		}
	}

	lines, err := unparsedtext.LoadTextLines(unparsedTextConfig(ctx), href, encoding)
	if err != nil {
		return nil, wrapUnparsedTextError(err)
	}
	result := make(Sequence, len(lines))
	for i, line := range lines {
		result[i] = AtomicValue{TypeName: TypeString, Value: line}
	}
	return result, nil
}
