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
// Every sibling walk in a ctx-carrying function runs over a caller- or
// attacker-chosen number of nodes: the children of the subtree an Encryptor
// serializes, and the nodes a decrypted plaintext parses to. A walk that
// polled only at its ends would run to the end of that input after the caller
// cancelled, and the window scales with the node count.
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
