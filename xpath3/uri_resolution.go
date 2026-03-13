package xpath3

import (
	"context"
	"fmt"
	"net/url"
)

func baseURIFromContext(ctx context.Context) string {
	if ec := getFnContext(ctx); ec != nil {
		return ec.baseURI
	}
	return ""
}

func parseURIReference(raw string) (*url.URL, error) {
	if err := validatePercentEncoding(raw); err != nil {
		return nil, err
	}
	return url.Parse(raw)
}

func isWindowsDriveScheme(parsed *url.URL) bool {
	if len(parsed.Scheme) != 1 {
		return false
	}
	b := parsed.Scheme[0]
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
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
	hadNonASCII := encodedBase != base || encodedRef != ref

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
	if hadNonASCII {
		result = uriToIRI(result)
	}
	return result, nil
}

func validatePercentEncoding(uri string) error {
	for i := 0; i < len(uri); i++ {
		if uri[i] == '%' {
			if i+2 >= len(uri) {
				return fmt.Errorf("incomplete percent-encoding at position %d", i)
			}
			if !isHexDigit(uri[i+1]) || !isHexDigit(uri[i+2]) {
				return fmt.Errorf("invalid percent-encoding %%%c%c", uri[i+1], uri[i+2])
			}
			i += 2
		}
	}
	return nil
}

func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func indexScheme(raw string) int {
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case ':':
			return i
		case '/', '?', '#':
			return -1
		}
	}
	return -1
}
