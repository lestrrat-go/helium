package xslt3_test

import (
	"errors"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

var errHandlerAborted = errors.New("handler aborted transform")

type messageAbortHandler struct {
	called bool
}

func (r *messageAbortHandler) HandleMessage(msg string, terminate bool) error {
	r.called = true
	return errHandlerAborted
}

type resultDocumentAbortHandler struct {
	called bool
}

func (r *resultDocumentAbortHandler) HandleResultDocument(href string, doc *helium.Document, outDef *xslt3.OutputDef) error {
	r.called = true
	return errHandlerAborted
}

type rawResultAbortHandler struct {
	called bool
}

func (r *rawResultAbortHandler) HandleRawResult(seq xpath3.Sequence) error {
	r.called = true
	return errHandlerAborted
}

type primaryItemsAbortHandler struct {
	called bool
}

func (r *primaryItemsAbortHandler) HandlePrimaryItems(seq xpath3.Sequence) error {
	r.called = true
	return errHandlerAborted
}

type annotationAbortHandler struct {
	called bool
}

func (r *annotationAbortHandler) HandleAnnotations(annotations map[helium.Node]string, declarations xpath3.SchemaDeclarations) error {
	r.called = true
	return errHandlerAborted
}

func TestMessageHandlerErrorAbortsTransform(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:message terminate="no">boom</xsl:message>
    <out/>
  </xsl:template>
</xsl:stylesheet>`)

	handler := &messageAbortHandler{}
	_, err := ss.Transform(parseTransformSource(t)).MessageHandler(handler).Do(t.Context())

	require.True(t, handler.called)
	require.ErrorIs(t, err, errHandlerAborted)
}

func TestResultDocumentHandlerErrorAbortsTransform(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document href="secondary.xml"><secondary/></xsl:result-document>
    <out/>
  </xsl:template>
</xsl:stylesheet>`)

	handler := &resultDocumentAbortHandler{}
	_, err := ss.Transform(parseTransformSource(t)).ResultDocumentHandler(handler).Do(t.Context())

	require.True(t, handler.called)
	require.ErrorIs(t, err, errHandlerAborted)
}

func TestRawResultHandlerErrorAbortsTransform(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/" as="xs:string">
    <xsl:sequence select="'hello'"/>
  </xsl:template>
</xsl:stylesheet>`)

	handler := &rawResultAbortHandler{}
	_, err := ss.Transform(parseTransformSource(t)).RawResultHandler(handler).Do(t.Context())

	require.True(t, handler.called)
	require.ErrorIs(t, err, errHandlerAborted)
}

func TestPrimaryItemsHandlerErrorAbortsTransform(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="adaptive"/>
  <xsl:template match="/">
    <xsl:sequence select="1"/>
  </xsl:template>
</xsl:stylesheet>`)

	handler := &primaryItemsAbortHandler{}
	_, err := ss.Transform(parseTransformSource(t)).PrimaryItemsHandler(handler).Do(t.Context())

	require.True(t, handler.called)
	require.ErrorIs(t, err, errHandlerAborted)
}

func TestAnnotationHandlerErrorAbortsTransform(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:import-schema namespace="">
    <xs:schema>
      <xs:element name="out" type="xs:integer"/>
    </xs:schema>
  </xsl:import-schema>
  <xsl:template match="/">
    <xsl:element name="out" type="xs:integer">42</xsl:element>
  </xsl:template>
</xsl:stylesheet>`)

	handler := &annotationAbortHandler{}
	_, err := ss.Transform(parseTransformSource(t)).AnnotationHandler(handler).Do(t.Context())

	require.True(t, handler.called)
	require.ErrorIs(t, err, errHandlerAborted)
}
