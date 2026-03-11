package xpath

import "errors"

const (
	// DefaultMaxRecursionDepth is the maximum expression evaluation depth,
	// matching libxml2's XPATH_MAX_RECURSION_DEPTH.
	DefaultMaxRecursionDepth = 5000

	// DefaultMaxNodeSetLength is the maximum number of nodes in a node-set,
	// matching libxml2's XPATH_MAX_NODESET_LENGTH.
	DefaultMaxNodeSetLength = 10_000_000
)

// ErrNodeSetLimit is returned when a node-set exceeds DefaultMaxNodeSetLength.
var ErrNodeSetLimit = errors.New("node-set length limit exceeded")
