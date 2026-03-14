<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema">
                
<!-- Test a global parameter with an "as" constraint. -->
<!-- (This test needs to be called specially: it's designed for use from the command line) --> 
  <?spec xslt#parameters?>

  <xsl:param name="x" as="xs:integer" select="17"/>           
  
  <xsl:template match="/">
    <xsl:apply-templates select="*"/>
  </xsl:template>

  <xsl:template match="*">
    <out val="{$x}" is-integer="{$x instance of xs:integer}"/>
  </xsl:template>
  
</xsl:stylesheet>
