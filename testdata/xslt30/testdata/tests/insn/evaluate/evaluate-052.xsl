<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="#all"
  expand-text="yes">
  
  <!-- Saxon bug 6004 -->
  
  <xsl:mode on-no-match="shallow-copy"/>
  
  <xsl:template match="value">
    <xsl:next-match/>
    <result>
      <xsl:evaluate xpath=". || '()'">
        <xsl:with-param name="p1" select="map{ 'foo': 'foo 1' }"/>
      </xsl:evaluate>
    </result>
  </xsl:template>
  
  <xsl:template match="/" name="xsl:initial-template">
    <xsl:copy>
      <xsl:apply-templates/>
      <xsl:comment>Run with {system-property('xsl:product-name')} {system-property('xsl:product-version')} {system-property('Q{http://saxon.sf.net/}platform')} at {current-dateTime()}</xsl:comment>
    </xsl:copy>
  </xsl:template>
  
</xsl:stylesheet>