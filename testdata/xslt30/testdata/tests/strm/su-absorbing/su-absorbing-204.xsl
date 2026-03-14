<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:saxon="http://saxon.sf.net/"
  xmlns:mf="http://example.com/mf"
  exclude-result-prefixes="#all"
  version="3.0">
  

  
  <xsl:function name="mf:check" streamability="absorbing">
    <xsl:param name="input" as="node()"/>
    <in x="{$input!node-name()}" y="{$input!@name}"/>
    <xsl:text>&#10;</xsl:text>
    <xsl:sequence select="$input!*!mf:check(.)"/>
  </xsl:function>
  
  <xsl:template name="xsl:initial-template">
    <out>
    <xsl:source-document href="su-absorbing-204.xml" streamable="yes">
      <xsl:sequence select="mf:check(*)"/>
    </xsl:source-document>
    </out>
  </xsl:template>
  
</xsl:stylesheet>