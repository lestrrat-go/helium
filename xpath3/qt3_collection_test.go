package xpath3_test

import "testing"

func TestQT3Collections(t *testing.T) {
	t.Parallel()

	qt3RunTests(t, []qt3Test{
		{
			Name:  "source-backed collection",
			XPath: `exists(collection("urn:qt3:docs")/works) and exists(collection("urn:qt3:docs")/staff)`,
			Collections: []qt3Collection{
				{
					URI: "urn:qt3:docs",
					SourceDocs: []qt3SourceDoc{
						{DocPath: "docs/works.xml", URI: "urn:qt3:works"},
						{DocPath: "docs/staff.xml", URI: "urn:qt3:staff"},
					},
				},
			},
			Assertions: []qt3Assertion{qt3AssertTrue()},
		},
		{
			Name:  "query-backed collection",
			XPath: `sum(collection("urn:qt3:ints"))`,
			Collections: []qt3Collection{
				{URI: "urn:qt3:ints", Query: "1 to 10"},
			},
			Assertions: []qt3Assertion{qt3AssertEq("55")},
		},
		{
			Name:  "uri-collection returns anyURI",
			XPath: `count(uri-collection("urn:qt3:docs"))`,
			Collections: []qt3Collection{
				{
					URI: "urn:qt3:docs",
					SourceDocs: []qt3SourceDoc{
						{DocPath: "docs/works.xml", URI: "urn:qt3:works"},
						{DocPath: "docs/staff.xml", URI: "urn:qt3:staff"},
					},
				},
			},
			Assertions: []qt3Assertion{qt3AssertEq("2")},
		},
	})
}
