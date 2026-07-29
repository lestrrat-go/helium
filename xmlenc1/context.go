package xmlenc1

import (
	"context"
	"errors"

	helium "github.com/lestrrat-go/helium"
)

func contextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func isContextErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// abort returns the context's error when the context is done, and err
// otherwise, so a live cancellation wins over a stage's own failure.
//
// Every stage of the encrypt and decrypt pipelines runs to completion once
// entered — a block encryption, a key unwrap, a serialization — so a
// cancellation that lands inside one is observable only after it returns. Its
// own error would then be reported as the reason the operation failed, hiding
// the fact that the caller asked to stop. Routing every error return of a
// ctx-carrying function through abort makes that one decision for the whole
// package instead of a judgement call per branch.
//
// A live context returns err unchanged, so what a non-cancelled failure
// reports is exactly what it reported before: CBC's deliberate collapse of
// every failure to ErrDecryptionFailed still holds and no padding oracle
// appears.
func abort(ctx context.Context, err error) error {
	if cerr := contextErr(ctx); cerr != nil {
		return cerr
	}
	return err
}

// eachSibling walks first and its following siblings, observing ctx as it
// goes, and hands each node to fn. It stops at the first error, which is
// abort's verdict on fn's own error.
//
// Every sibling walk in this package runs over a caller- or attacker-chosen
// number of nodes: the children of the subtree an Encryptor serializes, the
// nodes a decrypted plaintext parses to, and every child list the parse reads
// on the way to a value. A walk that polled only at its ends would run to the
// end of that input after the caller cancelled, and the window scales with the
// node count.
//
// The poll runs per node rather than every Nth node. Nothing here measured a
// cost worth trading exactness for: a poll is one mutex-guarded read against
// per-node work that serializes a subtree or steps a DOM, and a stride would
// make the bound depend on a constant nobody can derive from the input.
func eachSibling(ctx context.Context, first helium.Node, fn func(helium.Node) error) error {
	for node := first; node != nil; node = node.NextSibling() {
		if err := contextErr(ctx); err != nil {
			return err
		}
		if err := fn(node); err != nil {
			return abort(ctx, err)
		}
	}
	return nil
}

// eachChildElement hands each ELEMENT child of elem to fn and skips every other
// child kind, walking through eachSibling so the context is still observed once
// per child.
//
// Every walk in the parse path is over one element's children, whose number the
// document chooses, and each of them begins by skipping the children that are
// not elements — a comment or a processing instruction may appear anywhere and
// says nothing about the value being read. Layering that skip on eachSibling
// keeps the poll where eachSibling puts it, ahead of looking at the child, so a
// document that is nothing but skipped children is observed at the same rate as
// one the parse reads in full and no walk decides for itself when to poll.
func eachChildElement(ctx context.Context, elem *helium.Element, fn func(*helium.Element) error) error {
	return eachSibling(ctx, elem.FirstChild(), func(child helium.Node) error {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			return nil
		}
		return fn(e)
	})
}

// textContent joins the Content() of elem's DIRECT children into one string,
// observing the context on every sibling list that join reads — elem's own
// children and, for a child that answers Content() by aggregating a subtree,
// every descendant list of that subtree.
//
// It collects exactly what [internal/domutil.TextContent] collects — every
// direct child kind, the content each node reports for itself, in document
// order — so a value it reads for a live context is byte-for-byte the value
// that shared helper reads. The parse keeps its own copy rather than calling
// the shared one because that one is used by packages with no context to
// observe, and widening its signature would reach well past this package.
//
// The parse reaches it for one value whose child count the document alone
// decides: an EncryptedKey's CarriedKeyName. It is charged against neither
// MaxEncryptedKeyBytes nor MaxCipherValueBytes, and a child that carries no
// characters at all — a comment is the cheapest of them — costs the document
// nothing to repeat at any depth, so those polls are the only thing bounding
// how long a cancelled caller waits for that walk.
func textContent(ctx context.Context, elem *helium.Element) (string, error) {
	var sb []byte
	if err := eachSibling(ctx, elem.FirstChild(), func(child helium.Node) error {
		next, err := appendContent(ctx, sb, child)
		if err != nil {
			return err
		}
		sb = next
		return nil
	}); err != nil {
		return "", err
	}
	return string(sb), nil
}

// errStopContent ends one sibling list of the content aggregation without
// ending the aggregation itself. helium's own aggregation stops a child list at
// a repeated node (a cyclic sibling pointer) and after a foreign-owned child (an
// entity reference's Entity child, whose siblings belong to the DTD declaration
// list); eachSibling ends a walk only on an error, so those two stops are
// spelled as one. appendOwnedContent swallows it, so no caller ever sees it, and
// eachSibling still routes it through abort, so a context that went down during
// the same step is reported as the cancellation instead of being lost to it.
var errStopContent = errors.New("stop content walk")

// appendContent appends to dst exactly the bytes n.Content() returns and polls
// ctx on every sibling list it reads on the way.
//
// A leaf answers Content() out of its own stored text, which is one bounded
// step; every other node answers it by aggregating its children, and that
// aggregation recurses over the whole descendant subtree. So a walk that polled
// once per DIRECT child would buy an attacker-chosen amount of traversal per
// poll, and one nesting level would defeat the poll entirely. The recursion
// below is helium's own (node.go, docnode.Content and aggregateOwnedContent)
// with the poll eachSibling puts ahead of each node: the same owned-child
// boundary, the same per-list repeat guard, the same active-path cycle guard,
// the same skips, and the same append order, so what it collects for a live
// context is byte-identical to what n.Content() returns.
func appendContent(ctx context.Context, dst []byte, n helium.Node) ([]byte, error) {
	if !aggregatesOwnContent(n) {
		return append(dst, n.Content()...), nil
	}
	return appendOwnedContent(ctx, dst, n, map[helium.Node]struct{}{n: {}})
}

// appendOwnedContent appends the content of owner's OWN children to dst. onPath
// is the set of nodes whose aggregation is currently in progress (owner
// included): a child already on it is a back-edge in the child pointers and is
// skipped, which is what terminates a cyclic tree. It is an active-path set and
// not a visited set, so a node reached twice by different paths is still
// collected twice, exactly as helium collects it.
//
// The per-list state (the walked list's owner, its repeat guard, and the buffer)
// is captured rather than held on a receiver because eachSibling takes a
// func(helium.Node) error and this package may not park a context on a struct
// field.
func appendOwnedContent(ctx context.Context, dst []byte, owner helium.Node, onPath map[helium.Node]struct{}) ([]byte, error) {
	seen := make(map[helium.Node]struct{})
	err := eachSibling(ctx, owner.FirstChild(), func(child helium.Node) error {
		if _, dup := seen[child]; dup {
			return errStopContent
		}
		seen[child] = struct{}{}
		next, err := appendChildContent(ctx, dst, child, onPath)
		if err != nil {
			return err
		}
		dst = next
		// A foreign-owned child's NextSibling() belongs to another node's list,
		// so the walk of owner's children ends with it.
		if child.Parent() != owner {
			return errStopContent
		}
		return nil
	})
	if err != nil && !errors.Is(err, errStopContent) {
		return nil, err
	}
	return dst, nil
}

// appendChildContent appends one child's contribution to dst: nothing when the
// child is already being aggregated further up the path, its own stored text
// when it is a leaf, and its aggregated subtree otherwise.
func appendChildContent(ctx context.Context, dst []byte, child helium.Node, onPath map[helium.Node]struct{}) ([]byte, error) {
	if _, active := onPath[child]; active {
		return dst, nil
	}
	if !aggregatesOwnContent(child) {
		return append(dst, child.Content()...), nil
	}
	onPath[child] = struct{}{}
	dst, err := appendOwnedContent(ctx, dst, child, onPath)
	delete(onPath, child)
	if err != nil {
		return nil, err
	}
	return dst, nil
}

// aggregatesOwnContent reports whether n answers Content() by aggregating its
// children rather than out of text it stores itself. It mirrors the node-kind
// split helium's own aggregation makes (node.go, aggregatesOwnContent): the
// kinds listed here are the leaves whose Content() cannot recurse, and every
// other kind — including any node kind added later — aggregates and is walked
// under the context.
func aggregatesOwnContent(n helium.Node) bool {
	switch n.(type) {
	case *helium.Text, *helium.Comment, *helium.CDATASection, *helium.ProcessingInstruction, *helium.Entity, *helium.NamespaceNodeWrapper:
		return false
	default:
		return true
	}
}
