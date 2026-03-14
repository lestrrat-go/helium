<?xml version="1.0" encoding="UTF-8"?>
<xsl:package
  name="http://www.w3.org/xslt30-test/accumulator/accumulator-017"
  package-version="1.0"
  declared-modes="no"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="http://accum001/"
  exclude-result-prefixes="xs f" version="3.0">

  <!-- Accumulators are always applicable to a temporary tree -->
  
  <xsl:variable name="v" as="element(doc)">
    <xsl:copy-of select="doc('accumulator-001.xml')/doc"/>
  </xsl:variable>
  
  
  <xsl:accumulator name="figNr" as="xs:integer" initial-value="0">
    <xsl:accumulator-rule match="chap" select="0"/>
    <xsl:accumulator-rule match="fig" select="$value + 1"/>
  </xsl:accumulator>

  <xsl:template name="xsl:initial-template" visibility="public">
    <xsl:apply-templates select="$v"/>
  </xsl:template>
  
  <xsl:mode on-no-match="shallow-skip"/>
  
  <xsl:template match="fig">
    <xsl:apply-templates/>
    <p>Figure <xsl:value-of select="accumulator-before('figNr')"/>: <xsl:value-of select="@alt"/></p>
  </xsl:template>

  <xsl:template match="doc">
    <figures>
      <xsl:apply-templates/>
    </figures>
  </xsl:template>
</xsl:package>
