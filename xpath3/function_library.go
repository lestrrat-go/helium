package xpath3

import "maps"

// FunctionLibrary holds user-defined functions for XPath evaluation.
// Functions can be registered by local name (for unqualified calls) or
// by qualified name (namespace URI + local name).
type FunctionLibrary struct {
	byLocal map[string]Function
	byQName map[QualifiedName]Function
}

// NewFunctionLibrary creates an empty FunctionLibrary.
func NewFunctionLibrary() *FunctionLibrary {
	return &FunctionLibrary{}
}

// FunctionLibraryFromMaps wraps raw function maps without cloning.
// The caller retains ownership of the maps. Typically used with
// NewEvaluator(EvalBorrowing) by internal callers that already own
// the data.
func FunctionLibraryFromMaps(byLocal map[string]Function, byQName map[QualifiedName]Function) *FunctionLibrary {
	return &FunctionLibrary{byLocal: byLocal, byQName: byQName}
}

// Set registers a function by local name (for unqualified function calls).
func (f *FunctionLibrary) Set(name string, fn Function) {
	if f.byLocal == nil {
		f.byLocal = make(map[string]Function)
	}
	f.byLocal[name] = fn
}

// SetNS registers a function by namespace URI and local name.
func (f *FunctionLibrary) SetNS(uri, name string, fn Function) {
	if f.byQName == nil {
		f.byQName = make(map[QualifiedName]Function)
	}
	f.byQName[QualifiedName{URI: uri, Name: name}] = fn
}

// Get looks up a function by local name.
func (f *FunctionLibrary) Get(name string) (Function, bool) {
	if f.byLocal == nil {
		return nil, false
	}
	fn, ok := f.byLocal[name]
	return fn, ok
}

// GetNS looks up a function by namespace URI and local name.
func (f *FunctionLibrary) GetNS(uri, name string) (Function, bool) {
	if f.byQName == nil {
		return nil, false
	}
	fn, ok := f.byQName[QualifiedName{URI: uri, Name: name}]
	return fn, ok
}

// Delete removes a function registered by local name.
func (f *FunctionLibrary) Delete(name string) {
	delete(f.byLocal, name)
}

// DeleteNS removes a function registered by qualified name.
func (f *FunctionLibrary) DeleteNS(uri, name string) {
	delete(f.byQName, QualifiedName{URI: uri, Name: name})
}

// Clear removes all registered functions.
func (f *FunctionLibrary) Clear() {
	clear(f.byLocal)
	clear(f.byQName)
}

// Clone returns a deep copy of the FunctionLibrary.
func (f *FunctionLibrary) Clone() *FunctionLibrary {
	if f == nil {
		return nil
	}
	return &FunctionLibrary{
		byLocal: maps.Clone(f.byLocal),
		byQName: maps.Clone(f.byQName),
	}
}

// localMap returns the local-name function map. For internal use.
func (f *FunctionLibrary) localMap() map[string]Function {
	if f == nil {
		return nil
	}
	return f.byLocal
}

// qnameMap returns the qualified-name function map. For internal use.
func (f *FunctionLibrary) qnameMap() map[QualifiedName]Function {
	if f == nil {
		return nil
	}
	return f.byQName
}
