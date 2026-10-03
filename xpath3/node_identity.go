package xpath3

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// nodeIdentityKey is a map key for node identity (see ixpath.SameNode): the
// NamespaceNodeKey of a namespace node of an element, the pointer of any other
// node.
type nodeIdentityKey struct {
	node helium.Node
	ns   ixpath.NSNodeKey
}

func makeNodeIdentityKey(n helium.Node) nodeIdentityKey {
	if nsk, ok := ixpath.NamespaceNodeKey(n); ok {
		return nodeIdentityKey{ns: nsk}
	}
	return nodeIdentityKey{node: n}
}

// StableNodeID returns a unique string identifier for a node.  For a
// namespace node of an element (which the namespace axis recreates on each
// traversal) the ID is derived from the parent element pointer and the
// namespace prefix, so that the same logical namespace node always gets the
// same ID. Any other node, a parentless namespace node included, gets an ID
// derived from its pointer.
func StableNodeID(n helium.Node) string {
	if n == nil {
		return ""
	}
	if nsk, ok := ixpath.NamespaceNodeKey(n); ok {
		parentHex := strings.TrimPrefix(fmt.Sprintf("%p", nsk.Parent), "0x")
		prefixHex := hex.EncodeToString([]byte(nsk.Prefix))
		if prefixHex == "" {
			prefixHex = "00"
		}
		return "idns" + parentHex + prefixHex
	}
	return fmt.Sprintf("id%p", n)
}
