package xpointer

import helium "github.com/lestrrat-go/helium"

// CopyNode creates a deep copy of src, owned by targetDoc.
// This is a thin wrapper around helium.CopyNode.
func CopyNode(src helium.Node, targetDoc *helium.Document) (helium.Node, error) {
	return helium.CopyNode(src, targetDoc)
}
