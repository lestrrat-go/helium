<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  expand-text="yes">

 <!-- Purpose: repeated copying, appending one new element on each cycle -->

  <xsl:mode name="phase1" on-no-match="shallow-copy"/>
  <xsl:mode name="phase2" on-no-match="shallow-copy"/>
  <xsl:mode name="phase3" on-no-match="shallow-copy"/>
  <xsl:mode name="phase4" on-no-match="shallow-copy"/>
  
  <xsl:variable name="after-phase-1">
    <xsl:apply-templates select="/" mode="phase1"/>
  </xsl:variable>
  
  <xsl:variable name="after-phase-2">
    <xsl:apply-templates select="$after-phase-1" mode="phase2"/>
  </xsl:variable>
  
  <xsl:variable name="after-phase-3">
    <xsl:apply-templates select="$after-phase-2" mode="phase3"/>
  </xsl:variable>
  
  <xsl:variable name="after-phase-4">
    <xsl:apply-templates select="$after-phase-3" mode="phase4"/>
  </xsl:variable>
  
  <xsl:template match="/">
    <xsl:copy-of select="$after-phase-4"/>
  </xsl:template>

  <xsl:template match="doc1 | doc2" mode="phase1">
    <xsl:copy>
      <xsl:copy-of select="node()"/>
      <count>{count(*)}</count>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="doc1 | doc2" mode="phase2">
    <xsl:copy>
      <xsl:copy-of select="node()"/>
      <sum>{sum(*/@level)}</sum>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="doc1 | doc2" mode="phase3">
    <xsl:copy>
      <xsl:copy-of select="node()"/>
      <concat>{string-join(*/@origin, ' ')}</concat>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="doc1 | doc2" mode="phase4">
    <xsl:copy>
      <xsl:copy-of select="node()"/>
      <last>{max(*/string(@origin))}</last>
    </xsl:copy>
  </xsl:template>

</xsl:stylesheet>