package xpath3

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
)

type nodeIdentityKey struct {
	node     helium.Node
	parent   helium.Node
	prefix   string
	isNSNode bool
}

func sameNode(a, b helium.Node) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Type() != helium.NamespaceNode || b.Type() != helium.NamespaceNode {
		return false
	}
	return a.Parent() == b.Parent() && a.Name() == b.Name()
}

func makeNodeIdentityKey(n helium.Node) nodeIdentityKey {
	if n != nil && n.Type() == helium.NamespaceNode {
		return nodeIdentityKey{
			parent:   n.Parent(),
			prefix:   n.Name(),
			isNSNode: true,
		}
	}
	return nodeIdentityKey{node: n}
}

// StableNodeID returns a unique string identifier for a node.  For
// namespace nodes (which are recreated on each axis traversal) the ID
// is derived from the parent element pointer and the namespace prefix
// so that the same logical namespace node always gets the same ID.
func StableNodeID(n helium.Node) string {
	if n == nil {
		return ""
	}
	if n.Type() == helium.NamespaceNode {
		parentHex := strings.TrimPrefix(fmt.Sprintf("%p", n.Parent()), "0x")
		prefixHex := hex.EncodeToString([]byte(n.Name()))
		if prefixHex == "" {
			prefixHex = "00"
		}
		return "idns" + parentHex + prefixHex
	}
	return fmt.Sprintf("id%p", n)
}
