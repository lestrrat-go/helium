package helium

import (
	"errors"
	"strings"

	"github.com/lestrrat-go/pdebug"
)

func newElementContent(name string, ctype ElementContentType) (*ElementContent, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START newElementContent '%s' (type = %d)", name, ctype)
		defer g.IRelease("END newElementContent")
	}
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
		} else {
			local = name
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

func (elem *ElementContent) copyElementContent() *ElementContent {
	if elem == nil {
		return nil
	}
	ret := &ElementContent{}
	ret.ctype = elem.ctype
	ret.coccur = elem.coccur
	ret.name = elem.name
	ret.prefix = elem.prefix

	if elem.c1 != nil {
		ret.c1 = elem.c1.copyElementContent()
	}
	if ret.c1 != nil {
		ret.c1.parent = ret
	}

	if elem.c2 != nil {
		prev := ret
		for cur := elem.c2; cur != nil; {
			tmp := &ElementContent{}
			tmp.name = cur.name
			tmp.ctype = cur.ctype
			tmp.coccur = cur.coccur
			tmp.prefix = cur.prefix
			prev.c2 = tmp
			if cur.c1 != nil {
				tmp.c1 = cur.c1.copyElementContent()
			}
			if tmp.c1 != nil {
				tmp.c1.parent = ret
			}

			prev = tmp
			cur = cur.c2
		}
	}
	return ret
}
