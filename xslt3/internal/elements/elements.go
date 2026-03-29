package elements

import "slices"

// ElementContext describes where an XSLT element is allowed.
type ElementContext int

const (
	CtxTopLevel    ElementContext = 1 << iota // allowed as xsl:stylesheet child
	CtxInstruction                            // allowed in sequence constructors
	CtxChildOnly                              // only valid as child of specific parent(s)
	CtxRoot                                   // root wrapper (stylesheet/transform/package)
)

// ElementInfo describes an XSLT element's metadata.
type ElementInfo struct {
	MinVersion   string              // "1.0", "2.0", "3.0"
	Context      ElementContext      // bitmask: where this element is allowed
	Implemented  bool                // false for recognized-but-unsupported elements
	AllowedAttrs map[string]struct{} // element-specific unprefixed attrs (nil = unchecked)
	Parents      []string            // for CtxChildOnly: valid parent element names
}

// Registry holds metadata for all recognized XSLT elements.
type Registry struct {
	defs         map[string]ElementInfo
	topLevel     map[string]struct{}
	instructions map[string]struct{}
	childParents map[string][]string
}

// NewRegistry creates a fully initialized element registry.
func NewRegistry() *Registry {
	r := &Registry{defs: elementDefs()}
	r.precompute()
	return r
}

func (r *Registry) precompute() {
	r.topLevel = make(map[string]struct{})
	r.instructions = make(map[string]struct{})
	r.childParents = make(map[string][]string)
	for name, info := range r.defs {
		if info.Context&CtxTopLevel != 0 {
			r.topLevel[name] = struct{}{}
		}
		if info.Context&CtxInstruction != 0 {
			r.instructions[name] = struct{}{}
		}
		if info.Context&CtxChildOnly != 0 {
			r.childParents[name] = info.Parents
		}
	}
}

// IsKnown returns true if name is a recognized XSLT element.
func (r *Registry) IsKnown(name string) bool {
	_, ok := r.defs[name]
	return ok
}

// IsTopLevel returns true if name is allowed as a top-level declaration.
func (r *Registry) IsTopLevel(name string) bool {
	_, ok := r.topLevel[name]
	return ok
}

// IsInstruction returns true if name is allowed in sequence constructors.
func (r *Registry) IsInstruction(name string) bool {
	_, ok := r.instructions[name]
	return ok
}

// IsImplemented returns true if the element is recognized and implemented.
func (r *Registry) IsImplemented(name string) bool {
	info, ok := r.defs[name]
	return ok && info.Implemented
}

// MinVersion returns the minimum XSLT version for the element, or "" if unknown.
func (r *Registry) MinVersion(name string) string {
	info, ok := r.defs[name]
	if !ok {
		return ""
	}
	return info.MinVersion
}

// AllowedAttrs returns the set of allowed unprefixed attributes for the element.
// The second return value is false if the element is unknown.
func (r *Registry) AllowedAttrs(name string) (map[string]struct{}, bool) {
	info, ok := r.defs[name]
	if !ok {
		return nil, false
	}
	return info.AllowedAttrs, true
}

// ValidParents returns the list of valid parent elements for a child-only element.
// Returns nil if the element is not child-only.
func (r *Registry) ValidParents(name string) []string {
	return r.childParents[name]
}

// IsValidChild returns true if child is allowed as a direct child of parent.
// For elements without CtxChildOnly, this always returns false (they are not
// restricted to specific parents).
func (r *Registry) IsValidChild(child, parent string) bool {
	parents, ok := r.childParents[child]
	if !ok {
		return false
	}
	return slices.Contains(parents, parent)
}
