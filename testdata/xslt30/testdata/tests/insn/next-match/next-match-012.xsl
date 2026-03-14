<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <!-- Purpose: Test xsl:next-match within an attribute which is inside an attribute set.-->
  
  <xsl:import href="next-match-012a.xsl"/>
   
  <xsl:template match="doc">
    <a xsl:use-attribute-sets="myAttrib">hello</a>
  </xsl:template>
  
  <xsl:template match="*">
    <b>next matched</b>
  </xsl:template>
  
  <xsl:attribute-set name="myAttrib">
    <xsl:attribute name="a1">5</xsl:attribute>
    <xsl:attribute name="a2">
      <xsl:next-match />
    </xsl:attribute>
  </xsl:attribute-set>
  
  
</xsl:stylesheet>