package xsd_test

import (
	"path"
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
)

// xstsCase is a single W3C XML Schema Test Suite (XSTS) test group, restricted
// to the XSD 1.1 subset. All referenced schema and instance file contents are
// embedded in Files (keyed by suite-root-relative slash path) so the tests run
// with no external clone present.
//
// The per-contributor slices (xstsIbmCases, xstsSaxonCases, ...) are generated
// by tools/xstsgen into w3c_xsts_<contributor>_gen_test.go.
type xstsCase struct {
	ID          string
	SchemaRel   string
	SchemaValid bool
	Files       map[string]string
	Instances   []xstsInstance
}

type xstsInstance struct {
	Name  string
	Rel   string
	Valid bool
}

// xstsSkip (defined in w3c_xsts_skip_test.go) maps a case ID to a skip reason
// for the known XSD 1.1 conformance gaps in the suite.

// xstsAllCases concatenates every generated per-contributor slice.
var xstsAllCases = func() []xstsCase {
	var all []xstsCase
	all = append(all, xstsIbmCases...)
	all = append(all, xstsOracleCases...)
	all = append(all, xstsSaxonCases...)
	all = append(all, xstsWgCases...)
	return all
}()

func TestW3CXSTS(t *testing.T) {
	for _, c := range xstsAllCases {
		t.Run(c.ID, func(t *testing.T) {
			if reason := xstsSkip[c.ID]; reason != "" {
				t.Skip(reason)
			}
			runXSTSCase(t, c)
		})
	}
}

func runXSTSCase(t *testing.T, c xstsCase) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("%s: panic: %v", c.ID, r)
		}
	}()

	mapfs := fstest.MapFS{}
	for rel, content := range c.Files {
		mapfs[rel] = &fstest.MapFile{Data: []byte(content)}
	}

	schemaSrc, ok := c.Files[c.SchemaRel]
	if !ok {
		t.Errorf("%s: primary schema %q not embedded", c.ID, c.SchemaRel)
		return
	}

	doc, perr := helium.NewParser().Parse(t.Context(), []byte(schemaSrc))

	var schema *xsd.Schema
	var cerr error
	if perr != nil {
		cerr = perr
	} else {
		schema, cerr = xsd.NewCompiler().
			Version(xsd.Version11).
			FS(mapfs).
			BaseDir(path.Dir(c.SchemaRel)).
			Compile(t.Context(), doc)
	}

	gotSchemaValid := cerr == nil
	if gotSchemaValid != c.SchemaValid {
		t.Errorf("%s: schema validity: expected %t, got %t (err=%v)",
			c.ID, c.SchemaValid, gotSchemaValid, cerr)
	}
	if !gotSchemaValid || schema == nil {
		return
	}

	for _, inst := range c.Instances {
		src, ok := c.Files[inst.Rel]
		if !ok {
			t.Errorf("%s/%s: instance %q not embedded", c.ID, inst.Name, inst.Rel)
			continue
		}
		idoc, ierr := helium.NewParser().Parse(t.Context(), []byte(src))
		var verr error
		if ierr != nil {
			verr = ierr
		} else {
			verr = xsd.NewValidator(schema).Validate(t.Context(), idoc)
		}
		gotValid := verr == nil
		if gotValid != inst.Valid {
			t.Errorf("%s/%s: instance validity: expected %t, got %t (err=%v)",
				c.ID, inst.Name, inst.Valid, gotValid, verr)
		}
	}
}
