package xslt3_test

import (
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// These tests pin the xslt3->xsd BOUNDARY translation (schemaResolverFS.Open):
// a NESTED xs:include/xs:import/xs:redefine reached while an xsl:import-schema
// compiles is loaded through the xslt3 resolver, and the resolver's error must be
// re-expressed in XSD's fs.FS vocabulary so xsd's readNestedSchema classifies it
// the SAME as the xslt3 side. A CONFIRMED resolution miss (a resolver reporting
// not-found, even one that does NOT itself satisfy fs.ErrNotExist) is demoted so
// an OPTIONAL nested include is skipped; a permission denial or a post-open read
// failure stays FATAL.

// boundaryImportSheet imports a schema by schema-location. The imported main.xsd
// carries a nested xs:include whose target the resolver controls.
const boundaryImportSheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s">
  <xsl:import-schema namespace="http://example.com/s" schema-location="main.xsd"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

// boundaryMainXSD is self-sufficient once its nested include is skipped: root is
// typed by a builtin, so a demoted (skipped) include leaves a valid schema.
const boundaryMainXSD = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/s"
           xmlns:s="http://example.com/s"
           elementFormDefault="qualified">
  <xs:include schemaLocation="missing.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

// nestedFailResolver is a compile-time URIResolver (method Resolve) that serves
// main.xsd from a map but fails the nested include (basename "missing.xsd") in a
// chosen way: a Resolve error (resolution phase) or a reader that fails on Read
// (post-open). It models a resolver whose not-found error does NOT satisfy
// fs.ErrNotExist — the over-rejection the boundary translation fixes.
type nestedFailResolver struct {
	main       string // content served for main.xsd
	failBase   string // basename that fails
	resolveErr error  // if non-nil, Resolve returns this for failBase
	readErr    error  // else Resolve returns a reader that fails Read with this
}

func (r nestedFailResolver) Resolve(uri string) (io.ReadCloser, error) {
	if baseName(uri) == r.failBase {
		if r.resolveErr != nil {
			return nil, r.resolveErr
		}
		return badReadCloser{r.readErr}, nil
	}
	if baseName(uri) == "main.xsd" {
		return io.NopCloser(strings.NewReader(r.main)), nil
	}
	return nil, &resolverNotFoundError{uri: uri}
}

func compileBoundaryImport(t *testing.T, resolver nestedFailResolver) error {
	t.Helper()
	ctx := t.Context()
	doc, err := helium.NewParser().Parse(ctx, []byte(boundaryImportSheet))
	require.NoError(t, err)
	_, err = xslt3.NewCompiler().
		BaseURI("mem://stylesheets/main.xsl").
		URIResolver(resolver).
		Compile(ctx, doc)
	return err
}

// TestNestedIncludeResolverNotFoundDemoted is the repro: the resolver serves
// main.xsd but reports a bare (non-fs.ErrNotExist) "not found" for the nested
// include. The optional include must be skipped and the stylesheet must compile,
// not be over-rejected as "schema content invalid".
func TestNestedIncludeResolverNotFoundDemoted(t *testing.T) {
	err := compileBoundaryImport(t, nestedFailResolver{
		main:       boundaryMainXSD,
		failBase:   "missing.xsd",
		resolveErr: &resolverNotFoundError{uri: "missing.xsd"},
	})
	require.NoError(t, err, "a resolver not-found for an optional nested include must warn/skip, not fail compile")
}

// TestNestedIncludeResolverPermissionFatal verifies a PERMISSION denial on the
// nested include stays FATAL through the boundary (not demoted as a miss).
func TestNestedIncludeResolverPermissionFatal(t *testing.T) {
	err := compileBoundaryImport(t, nestedFailResolver{
		main:       boundaryMainXSD,
		failBase:   "missing.xsd",
		resolveErr: fs.ErrPermission,
	})
	require.Error(t, err, "a permission denial on a nested include must be fatal, not demoted")
}

// TestNestedIncludePostOpenReadFatal verifies a POST-OPEN read failure on the
// nested include (reader obtained then Read fails) stays FATAL through the
// boundary — it is not a resolution miss.
func TestNestedIncludePostOpenReadFatal(t *testing.T) {
	err := compileBoundaryImport(t, nestedFailResolver{
		main:     boundaryMainXSD,
		failBase: "missing.xsd",
		readErr:  io.ErrUnexpectedEOF,
	})
	require.Error(t, err, "a post-open read failure on a nested include must be fatal, not demoted")
}
