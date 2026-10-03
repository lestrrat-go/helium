package helium

import (
	"bytes"
	"context"
)

// UnsafeSetParentForTesting exposes the package-private raw parent-pointer
// setter to the external helium_test package. It compiles into the test binary
// only, so raw parent linkage stays unreachable from outside the package in an
// ordinary build. Tests use it to build a deliberately corrupt tree and
// exercise the traversal cycle guards.
func UnsafeSetParentForTesting(n Node, parent Node) {
	unsafeSetParent(n, parent)
}

// UnsafeAppendChildForTesting exposes the package-private no-preflight child
// append to the external helium_test package. It compiles into the test binary
// only, so linking a child without the cycle-guard and duplicate-attribute
// preflight stays unreachable from outside the package in an ordinary build.
func UnsafeAppendChildForTesting(parent MutableNode, child Node) error {
	return appendFastChild(parent, child)
}

// UnsafeSetNextSiblingForTesting exposes the package-private raw next-sibling
// pointer setter to the external helium_test package. It compiles into the test
// binary only, so raw sibling linkage stays unreachable from outside the
// package in an ordinary build. Tests use it to build a deliberately corrupt
// tree and exercise the traversal cycle guards.
func UnsafeSetNextSiblingForTesting(n Node, next Node) {
	unsafeSetNextSibling(n, next)
}

// ErrContentCursorForTesting exposes the internal error element content
// returns when it is reached without a UTF-8 input cursor. Every entry point
// installs one before content starts, so tests and fuzz targets assert that
// no parse ever returns it.
var ErrContentCursorForTesting = errContentCursor

// ParseStateForTesting is the parser bookkeeping a finished parse leaves
// behind. Tests read it to check a resource property of one parse directly,
// without reading process-wide memory statistics.
type ParseStateForTesting struct {
	// EntityExpansionBytes is the entity-expansion byte count the
	// amplification guard charged during the parse.
	EntityExpansionBytes int64
	// AttributeDefaultSetAllocated reports whether the parse allocated the
	// <!ATTLIST> default dedup set.
	AttributeDefaultSetAllocated bool
}

// ParseStateOfParseForTesting runs the parse phase of [Parser.Parse] on b and
// returns the parse error together with the bookkeeping the parse left behind.
// The post-parse steps Parse runs on a built tree (XInclude, DTD validation)
// are skipped.
func ParseStateOfParseForTesting(ctx context.Context, p Parser, b []byte) (ParseStateForTesting, error) {
	p = p.normalized()
	pctx := &parserCtx{rawInput: b, baseURI: p.cfg.baseURI}
	if err := pctx.init(p.cfg, bytes.NewReader(b), len(b)); err != nil {
		return ParseStateForTesting{}, err
	}
	err := pctx.parseDocument(ctx)
	state := ParseStateForTesting{
		EntityExpansionBytes:         pctx.sizeentcopy,
		AttributeDefaultSetAllocated: pctx.attsDefaultSeen != nil,
	}
	_ = pctx.release()
	return state, err
}
