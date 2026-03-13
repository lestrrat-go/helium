package xpath3_test

import "testing"

func TestQT3VariableSourceDocs(t *testing.T) {
	t.Parallel()

	qt3RunTests(t, []qt3Test{
		{
			Name:  "bind source docs to variables",
			XPath: `number($works/works/employee[1]/hours[1]) + number($staff/staff/employee[2]/grade[1])`,
			SourceDocs: []qt3SourceDoc{
				{Name: "works", DocPath: "docs/works.xml"},
				{Name: "staff", DocPath: "docs/staff.xml"},
			},
			Assertions: []qt3Assertion{qt3AssertEq("50")},
		},
		{
			Name:  "evaluate params against source docs",
			XPath: `string($target)`,
			SourceDocs: []qt3SourceDoc{
				{Name: "staff", DocPath: "docs/staff.xml"},
			},
			Params: []qt3Param{
				{Name: "target", Select: "$staff/staff/employee[2]/empname[1]"},
			},
			Assertions: []qt3Assertion{qt3AssertStringValue("Betty")},
		},
	})
}
