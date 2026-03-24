<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="#all"
  xmlns:mf="http://example.com/mf"
  version="3.0">

  
  <xsl:mode on-no-match="shallow-copy" streamable="yes"/>
  
  <xsl:function name="mf:nest" as="element()?" streamability="absorbing">
    <xsl:param name="elements" as="element()*"/>
    <xsl:copy select="head($elements)">
      <xsl:attribute name="type">data</xsl:attribute>
      <xsl:sequence select="mf:nest(tail($elements))"/>
    </xsl:copy>
  </xsl:function>
  
  <xsl:template match="person">
    <xsl:copy>
      <xsl:sequence select="mf:nest(*)"/>
    </xsl:copy>
  </xsl:template>
  
</xsl:stylesheet>