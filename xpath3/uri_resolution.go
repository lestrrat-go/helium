package xpath3

import (
	"context"
	"net/url"

	"github.com/lestrrat-go/helium/internal/unparsedtext"
)

func baseURIFromContext(ctx context.Context) string {
	if ec := getFnContext(ctx); ec != nil {
		return ec.baseURI
	}
	return ""
}

func parseURIReference(raw string) (*url.URL, error) {
	if err := unparsedtext.ValidatePercentEncoding(raw); err != nil {
		return nil, err
	}
	return url.Parse(raw)
}

func isSupportedResourceScheme(scheme string) bool {
	switch scheme {
	case "file", "http", "https":
		return true
	default:
		return false
	}
}

func resolveURIReference(base, ref string) (string, error) {
	encodedBase := iriToURI(base)
	encodedRef := iriToURI(ref)
	needDecode := encodedBase != base || encodedRef != ref

	origScheme := ""
	if idx := indexScheme(encodedBase); idx > 0 {
		origScheme = encodedBase[:idx]
	}

	baseURL, err := url.Parse(encodedBase)
	if err != nil {
		return "", err
	}
	refURL, err := url.Parse(encodedRef)
	if err != nil {
		return "", err
	}

	resolved := baseURL.ResolveReference(refURL)
	result := resolved.String()
	if origScheme != "" && resolved.Scheme != origScheme {
		result = origScheme + result[len(resolved.Scheme):]
	}
	if needDecode {
		result = uriToIRI(result)
	}
	return result, nil
}

func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func indexScheme(raw string) int {
	for i := range len(raw) {
		switch raw[i] {
		case ':':
			return i
		case '/', '?', '#':
			return -1
		}
	}
	return -1
}
