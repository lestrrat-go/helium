<?xml version='1.0' ?>
<xsl:stylesheet xmlns:xs="http://www.w3.org/2001/XMLSchema" 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test that promotion from double to float is not allowed -->
  
  <xsl:template name="main" match="/">    
    <xsl:variable name="num1" select="xs:double('1e4')" as="xs:double"/> 
    <xsl:variable name="num2" select="$num1" as="xs:float" /> 
    <a><xsl:value-of select="$num2" /></a>
  </xsl:template>
  
</xsl:stylesheet>