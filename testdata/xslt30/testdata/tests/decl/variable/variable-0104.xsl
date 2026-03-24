<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema">
                
<!-- Test use of range variables within the select expression of a global variable. -->
  <?spec xslt#global-variables?>

  <xsl:param name="p" select="10"/>
  <xsl:variable name="x" as="xs:integer" select="sum(for $i in 1 to $p return $i)"/>           
  
  <xsl:template match="/">
    <xsl:apply-templates select="*"/>
  </xsl:template>

  <xsl:template match="*">
    <xsl:variable name="z" select="22"/>
    <out x="{$x}" z="{$z}" is-integer="{$x instance of xs:integer}"/>
  </xsl:template>
  
</xsl:stylesheet>
