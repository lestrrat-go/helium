package shim_test

// Test-fixture-only literals used across shim package tests. Constants
// duplicating values in internal/lexicon (PrefixXML, PrefixXMLNS, AttrValue)
// are referenced via lexicon.* directly at the call site.
const (
	testFoo                  = "foo"
	testBar                  = "bar"
	testBaz                  = "baz"
	testHello                = "hello"
	testWorld                = "world"
	testHelloWorld           = "hello world"
	testStr                  = "str"
	testTest                 = "test"
	testAttr                 = "attr"
	testSpace                = "space"
	testFirst                = "First"
	testPlainV42             = "<Plain><V>42</V></Plain>"
	testCannotUnmarshalIface = "cannot unmarshal into interface {}"
	testNs2                  = "ns2"
	testTag                  = "tag"
)
