package xslt3

import (
	"github.com/lestrrat-go/helium/xpath3"
)

// Parameters holds named XSLT parameter bindings keyed by expanded name
// string (Q{uri}local form). It is a mutable collection: callers build it
// up, then pass it to a Compiler or Invocation. The receiver clones it at
// terminal method time.
type Parameters struct {
	values map[string]xpath3.Sequence
}

// NewParameters creates an empty Parameters collection.
func NewParameters() *Parameters {
	return &Parameters{}
}

// Set binds a parameter name to a Sequence value.
func (p *Parameters) Set(name string, value xpath3.Sequence) {
	if p.values == nil {
		p.values = make(map[string]xpath3.Sequence)
	}
	p.values[name] = value
}

// SetString binds a parameter to a single xs:string value.
func (p *Parameters) SetString(name, value string) {
	p.Set(name, xpath3.SingleString(value))
}

// SetAtomic binds a parameter to a single atomic value.
func (p *Parameters) SetAtomic(name string, value xpath3.AtomicValue) {
	p.Set(name, xpath3.SingleAtomic(value))
}

// Get returns the Sequence bound to name, or false if unbound.
func (p *Parameters) Get(name string) (xpath3.Sequence, bool) {
	if p.values == nil {
		return nil, false
	}
	seq, ok := p.values[name]
	return seq, ok
}

// Delete removes a parameter binding.
func (p *Parameters) Delete(name string) {
	delete(p.values, name)
}

// Clear removes all parameter bindings.
func (p *Parameters) Clear() {
	clear(p.values)
}

// Len returns the number of parameter bindings.
func (p *Parameters) Len() int {
	return len(p.values)
}

// Clone returns a deep copy of the Parameters.
func (p *Parameters) Clone() *Parameters {
	if p == nil {
		return nil
	}
	cloned := &Parameters{}
	if p.values != nil {
		cloned.values = make(map[string]xpath3.Sequence, len(p.values))
		for name, seq := range p.values {
			cloned.values[name] = xpath3.CloneSequence(seq)
		}
	}
	return cloned
}

// toMap returns the underlying map for internal use.
func (p *Parameters) toMap() map[string]xpath3.Sequence {
	if p == nil {
		return nil
	}
	return p.values
}
