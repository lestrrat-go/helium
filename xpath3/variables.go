package xpath3

import "maps"

// Variables holds named XPath variable bindings keyed by expanded name string.
// It is a mutable collection: callers build it up, then pass it to an Evaluator.
// The Evaluator clones it at terminal method time.
type Variables struct {
	values map[string]Sequence
}

// NewVariables creates an empty Variables collection.
func NewVariables() *Variables {
	return &Variables{}
}

// Set binds a variable name to a Sequence value.
func (v *Variables) Set(name string, value Sequence) {
	if v.values == nil {
		v.values = make(map[string]Sequence)
	}
	v.values[name] = value
}

// Get returns the Sequence bound to name, or false if unbound.
func (v *Variables) Get(name string) (Sequence, bool) {
	if v.values == nil {
		return nil, false
	}
	seq, ok := v.values[name]
	return seq, ok
}

// Delete removes a variable binding.
func (v *Variables) Delete(name string) {
	delete(v.values, name)
}

// Clear removes all variable bindings.
func (v *Variables) Clear() {
	clear(v.values)
}

// Len returns the number of variable bindings.
func (v *Variables) Len() int {
	return len(v.values)
}

// Clone returns a deep copy of the Variables.
func (v *Variables) Clone() *Variables {
	if v == nil {
		return nil
	}
	cloned := &Variables{}
	if v.values != nil {
		cloned.values = make(map[string]Sequence, len(v.values))
		for name, seq := range v.values {
			cloned.values[name] = cloneSequence(seq)
		}
	}
	return cloned
}

// toMap returns the underlying map. For internal use only.
func (v *Variables) toMap() map[string]Sequence {
	if v == nil {
		return nil
	}
	return v.values
}

// cloneFlatMap returns a shallow clone of the underlying map suitable for
// building a variableScope. For internal use.
func (v *Variables) cloneFlatMap() map[string]Sequence {
	if v == nil {
		return nil
	}
	return maps.Clone(v.values)
}
