package xslt3

import (
	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// Transform creates an Invocation that applies templates to the source
// document using the default mode.
func (ss *Stylesheet) Transform(source *helium.Document) Invocation {
	inv := newInvocation(ss, invocationTransform)
	inv.cfg.source = source
	return inv
}

// ApplyTemplates creates an Invocation that applies templates with
// explicit mode and selection control.
func (ss *Stylesheet) ApplyTemplates(source *helium.Document) Invocation {
	inv := newInvocation(ss, invocationApplyTemplates)
	inv.cfg.source = source
	return inv
}

// CallTemplate creates an Invocation that calls a named template directly.
func (ss *Stylesheet) CallTemplate(name string) Invocation {
	inv := newInvocation(ss, invocationCallTemplate)
	inv.cfg.initialTemplate = name
	return inv
}

// CallFunction creates an Invocation that calls a named function directly.
func (ss *Stylesheet) CallFunction(name string, args ...xpath3.Sequence) Invocation {
	inv := newInvocation(ss, invocationCallFunction)
	inv.cfg.initialFunction = name
	inv.cfg.initialArgs = args
	return inv
}
