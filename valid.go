package helium

import (
	"errors"
	"strings"
)

func newElementContent(name string, ctype ElementContentType) (*ElementContent, error) {
	var prefix string
	var local string
	switch ctype {
	case ElementContentElement:
		if name == "" {
			return nil, errors.New("ElementContent (element) must have name")
		}
		if i := strings.IndexByte(name, ':'); i > -1 {
			prefix = name[:i]
			local = name[i+1:]
		}
	case ElementContentPCDATA, ElementContentSeq, ElementContentOr:
		if name != "" {
			return nil, errors.New("ElementContent (element) must NOT have name")
		}
	default:
		return nil, errors.New("invalid element content type")
	}

	ret := ElementContent{
		ctype:  ctype,
		coccur: ElementContentOnce,
		name:   local,
		prefix: prefix,
	}

	return &ret, nil
}
