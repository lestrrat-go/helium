<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Test error() function with a string arguments -->
  <?spec fo#func-error?><?error XPTY0004?>
  
 <xsl:param name="p" select="'2'" as="xs:string" xmlns:xs="http://www.w3.org/2001/XMLSchema"/> 
<xsl:template match="/">
  <z><xsl:if test="$p='2'"><xsl:value-of select="error('application error - p=2')"/></xsl:if></z>
</xsl:template>
 
</xsl:stylesheet>
