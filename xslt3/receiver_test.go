package xslt3_test

import (
	"errors"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

var errReceiverAborted = errors.New("receiver aborted transform")

type messageAbortReceiver struct {
	called bool
}

func (r *messageAbortReceiver) HandleMessage(msg string, terminate bool) error {
	r.called = true
	return errReceiverAborted
}

type resultDocumentAbortReceiver struct {
	called bool
}

func (r *resultDocumentAbortReceiver) HandleResultDocument(href string, doc *helium.Document) error {
	r.called = true
	return errReceiverAborted
}

type resultDocumentOutputAbortReceiver struct {
	called bool
}

func (r *resultDocumentOutputAbortReceiver) HandleResultDocumentOutput(href string, outDef *xslt3.OutputDef) error {
	r.called = true
	return errReceiverAborted
}

type rawResultAbortReceiver struct {
	called bool
}

func (r *rawResultAbortReceiver) HandleRawResult(seq xpath3.Sequence) error {
	r.called = true
	return errReceiverAborted
}

type primaryItemsAbortReceiver struct {
	called bool
}

func (r *primaryItemsAbortReceiver) HandlePrimaryItems(seq xpath3.Sequence) error {
	r.called = true
	return errReceiverAborted
}

type annotationAbortReceiver struct {
	called bool
}

func (r *annotationAbortReceiver) HandleAnnotations(annotations map[helium.Node]string, declarations xpath3.SchemaDeclarations) error {
	r.called = true
	return errReceiverAborted
}

func TestMessageReceiverErrorAbortsTransform(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:message terminate="no">boom</xsl:message>
    <out/>
  </xsl:template>
</xsl:stylesheet>`)

	receiver := &messageAbortReceiver{}
	_, err := ss.Transform(parseTransformSource(t)).Receiver(receiver).Do(t.Context())

	require.True(t, receiver.called)
	require.ErrorIs(t, err, errReceiverAborted)
}

func TestResultDocumentReceiverErrorAbortsTransform(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document href="secondary.xml"><secondary/></xsl:result-document>
    <out/>
  </xsl:template>
</xsl:stylesheet>`)

	receiver := &resultDocumentAbortReceiver{}
	_, err := ss.Transform(parseTransformSource(t)).Receiver(receiver).Do(t.Context())

	require.True(t, receiver.called)
	require.ErrorIs(t, err, errReceiverAborted)
}

func TestResultDocumentOutputReceiverErrorAbortsTransform(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document href="secondary.txt" method="text">secondary</xsl:result-document>
    <out/>
  </xsl:template>
</xsl:stylesheet>`)

	receiver := &resultDocumentOutputAbortReceiver{}
	_, err := ss.Transform(parseTransformSource(t)).Receiver(receiver).Do(t.Context())

	require.True(t, receiver.called)
	require.ErrorIs(t, err, errReceiverAborted)
}

func TestRawResultReceiverErrorAbortsTransform(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/" as="xs:string">
    <xsl:sequence select="'hello'"/>
  </xsl:template>
</xsl:stylesheet>`)

	receiver := &rawResultAbortReceiver{}
	_, err := ss.Transform(parseTransformSource(t)).Receiver(receiver).Do(t.Context())

	require.True(t, receiver.called)
	require.ErrorIs(t, err, errReceiverAborted)
}

func TestPrimaryItemsReceiverErrorAbortsTransform(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="adaptive"/>
  <xsl:template match="/">
    <xsl:sequence select="1"/>
  </xsl:template>
</xsl:stylesheet>`)

	receiver := &primaryItemsAbortReceiver{}
	_, err := ss.Transform(parseTransformSource(t)).Receiver(receiver).Do(t.Context())

	require.True(t, receiver.called)
	require.ErrorIs(t, err, errReceiverAborted)
}

func TestAnnotationReceiverErrorAbortsTransform(t *testing.T) {
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

	receiver := &annotationAbortReceiver{}
	_, err := ss.Transform(parseTransformSource(t)).Receiver(receiver).Do(t.Context())

	require.True(t, receiver.called)
	require.ErrorIs(t, err, errReceiverAborted)
}
