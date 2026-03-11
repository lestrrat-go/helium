package xpath3

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// collationImpl provides string comparison operations for a specific collation.
type collationImpl struct {
	compare   func(a, b string) int
	indexOf   func(s, substr string) (pos, matchLen int)
	hasPrefix func(s, prefix string) (bool, int)
	hasSuffix func(s, suffix string) (bool, int)
}

// codepointCollation is the default XPath collation using byte-level comparison.
var codepointCollation = &collationImpl{
	compare: strings.Compare,
	indexOf: func(s, substr string) (int, int) {
		idx := strings.Index(s, substr)
		if idx < 0 {
			return -1, 0
		}
		return idx, len(substr)
	},
	hasPrefix: func(s, prefix string) (bool, int) {
		if strings.HasPrefix(s, prefix) {
			return true, len(prefix)
		}
		return false, 0
	},
	hasSuffix: func(s, suffix string) (bool, int) {
		if strings.HasSuffix(s, suffix) {
			return true, len(suffix)
		}
		return false, 0
	},
}

// htmlASCIICaseInsensitiveCollation compares ASCII letters case-insensitively,
// all other characters by codepoint.
var htmlASCIICaseInsensitiveCollation = &collationImpl{
	compare: func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	},
	indexOf: func(s, substr string) (int, int) {
		ls := strings.ToLower(s)
		lsub := strings.ToLower(substr)
		idx := strings.Index(ls, lsub)
		if idx < 0 {
			return -1, 0
		}
		return idx, len(substr)
	},
	hasPrefix: func(s, prefix string) (bool, int) {
		if len(s) < len(prefix) {
			return false, 0
		}
		if strings.EqualFold(s[:len(prefix)], prefix) {
			return true, len(prefix)
		}
		return false, 0
	},
	hasSuffix: func(s, suffix string) (bool, int) {
		if len(s) < len(suffix) {
			return false, 0
		}
		if strings.EqualFold(s[len(s)-len(suffix):], suffix) {
			return true, len(suffix)
		}
		return false, 0
	},
}

// makeUCACollation creates a collation based on the Unicode Collation Algorithm
// with optional parameters parsed from the URI query string.
func makeUCACollation(params string) *collationImpl {
	opts := parseUCAParams(params)

	cl := collate.New(opts.tag, opts.collateOpts...)

	return &collationImpl{
		compare: func(a, b string) int {
			return cl.CompareString(a, b)
		},
		indexOf: func(s, substr string) (int, int) {
			return ucaIndexOf(cl, s, substr)
		},
		hasPrefix: func(s, prefix string) (bool, int) {
			return ucaHasPrefix(cl, s, prefix)
		},
		hasSuffix: func(s, suffix string) (bool, int) {
			return ucaHasSuffix(cl, s, suffix)
		},
	}
}

type ucaParams struct {
	tag         language.Tag
	collateOpts []collate.Option
}

func parseUCAParams(query string) ucaParams {
	p := ucaParams{tag: language.Und}

	if query == "" {
		return p
	}

	params := strings.Split(query, ";")
	for _, param := range params {
		kv := strings.SplitN(param, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key, val := kv[0], kv[1]
		switch key {
		case "lang":
			p.tag = language.Make(val)
		case "strength":
			switch val {
			case "primary":
				p.collateOpts = append(p.collateOpts, collate.IgnoreCase, collate.IgnoreDiacritics)
			case "secondary":
				p.collateOpts = append(p.collateOpts, collate.IgnoreCase)
			case "tertiary":
				// default strength, no options needed
			}
		case "numeric":
			if val == "yes" {
				p.collateOpts = append(p.collateOpts, collate.Numeric)
			}
		}
	}
	return p
}

// ucaIndexOf finds substr in s using UCA collation, scanning rune by rune.
// Tries varying match lengths to handle numeric collation (e.g., "001" == "1").
func ucaIndexOf(cl *collate.Collator, s, substr string) (int, int) {
	if substr == "" {
		return 0, 0
	}
	subRuneLen := len([]rune(substr))
	sRunes := []rune(s)

	for i := 0; i < len(sRunes); i++ {
		// Try match lengths from subRuneLen to end of string
		minLen := subRuneLen
		if minLen > len(sRunes)-i {
			minLen = len(sRunes) - i
		}
		maxLen := len(sRunes) - i
		for matchLen := minLen; matchLen <= maxLen; matchLen++ {
			candidate := string(sRunes[i : i+matchLen])
			if cl.CompareString(candidate, substr) == 0 {
				bytePos := len(string(sRunes[:i]))
				byteLen := len(candidate)
				return bytePos, byteLen
			}
		}
		// Also try shorter matches (for cases where collation compresses)
		for matchLen := minLen - 1; matchLen > 0; matchLen-- {
			candidate := string(sRunes[i : i+matchLen])
			if cl.CompareString(candidate, substr) == 0 {
				bytePos := len(string(sRunes[:i]))
				byteLen := len(candidate)
				return bytePos, byteLen
			}
		}
	}
	return -1, 0
}

// ucaHasPrefix checks if s starts with prefix under UCA collation.
func ucaHasPrefix(cl *collate.Collator, s, prefix string) (bool, int) {
	prefixRunes := []rune(prefix)
	sRunes := []rune(s)
	if len(sRunes) < len(prefixRunes) {
		return false, 0
	}
	candidate := string(sRunes[:len(prefixRunes)])
	if cl.CompareString(candidate, prefix) == 0 {
		return true, len(candidate)
	}
	return false, 0
}

// ucaHasSuffix checks if s ends with suffix under UCA collation.
func ucaHasSuffix(cl *collate.Collator, s, suffix string) (bool, int) {
	suffixRunes := []rune(suffix)
	sRunes := []rune(s)
	if len(sRunes) < len(suffixRunes) {
		return false, 0
	}
	candidate := string(sRunes[len(sRunes)-len(suffixRunes):])
	if cl.CompareString(candidate, suffix) == 0 {
		return true, len(candidate)
	}
	return false, 0
}

// codepointCollationURI is the default XPath collation URI.
const codepointCollationURI = "http://www.w3.org/2005/xpath-functions/collation/codepoint"

// ucaCollationURI is the Unicode Collation Algorithm base URI.
const ucaCollationURI = "http://www.w3.org/2013/collation/UCA"

// htmlASCIICaseInsensitiveURI is the HTML ASCII case-insensitive collation URI.
const htmlASCIICaseInsensitiveURI = "http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"

// resolveCollation resolves a collation URI string to a collation implementation.
// If baseURI is non-empty, relative URIs are resolved against it.
func resolveCollation(uri, baseURI string) (*collationImpl, error) {
	// Resolve relative URI if baseURI is provided
	if baseURI != "" && !strings.Contains(uri, "://") {
		base, err := url.Parse(baseURI)
		if err == nil {
			ref, err := url.Parse(uri)
			if err == nil {
				uri = base.ResolveReference(ref).String()
			}
		}
	}

	switch {
	case uri == codepointCollationURI:
		return codepointCollation, nil
	case uri == htmlASCIICaseInsensitiveURI:
		return htmlASCIICaseInsensitiveCollation, nil
	case strings.HasPrefix(uri, ucaCollationURI):
		params := ""
		if idx := strings.Index(uri, "?"); idx >= 0 {
			params = uri[idx+1:]
		}
		return makeUCACollation(params), nil
	default:
		return nil, &XPathError{Code: "FOCH0002", Message: fmt.Sprintf("unsupported collation: %s", uri)}
	}
}
