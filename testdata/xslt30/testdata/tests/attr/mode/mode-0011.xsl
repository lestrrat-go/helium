<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:math="http://www.w3.org/2005/xpath-functions/math" xmlns:xp="http://saxonica.com/xpathParse"
  exclude-result-prefixes="xs math" version="3.0">
  <xsl:mode on-no-match="shallow-copy"/>
  <xsl:function name="xp:is.AVT" as="xs:boolean">
    <xsl:param name="test" as="attribute()"/>
    <xsl:sequence select="contains($test,'{')"/>
  </xsl:function>

  <xsl:template match="@*[xp:is.AVT(.)]">
    <xsl:attribute name="{local-name(.)}" select="'IS-AVT'||."/>
  </xsl:template>
  
  <xsl:template name="main">
    <xsl:apply-templates select="document('')/*"/>
  </xsl:template>
</xsl:stylesheet>