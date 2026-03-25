package xslt3

import "fmt"

// effectiveStripSpace returns the strip-space rules for the current execution
// scope. When executing code from a used package, the package's own rules
// are used (package-scoped isolation).
func (ec *execContext) effectiveStripSpace() []nameTest {
	if ec.currentPackage != nil && ec.currentPackage != ec.stylesheet {
		return ec.currentPackage.stripSpace
	}
	return ec.stylesheet.stripSpace
}

// effectivePreserveSpace returns the preserve-space rules for the current
// execution scope.
func (ec *execContext) effectivePreserveSpace() []nameTest {
	if ec.currentPackage != nil && ec.currentPackage != ec.stylesheet {
		return ec.currentPackage.preserveSpace
	}
	return ec.stylesheet.preserveSpace
}

// effectiveStripNamespaces returns the namespace bindings for strip/preserve
// resolution in the current execution scope.
func (ec *execContext) effectiveStripNamespaces() map[string]string {
	if ec.currentPackage != nil && ec.currentPackage != ec.stylesheet {
		return ec.currentPackage.namespaces
	}
	return ec.stylesheet.namespaces
}

// docCacheKey returns a package-scoped key for document caching. When
// executing in a used package with different strip-space rules, the key
// includes the package pointer so each package gets its own stripped copy.
func (ec *execContext) docCacheKey(uri string) string {
	if ec.currentPackage != nil && ec.currentPackage != ec.stylesheet {
		return fmt.Sprintf("%s@%p", uri, ec.currentPackage)
	}
	return uri
}
