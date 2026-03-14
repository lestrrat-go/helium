<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://my.com/"
exclude-result-prefixes="xs my">

  <!-- PURPOSE: Test cast as xs:boolean. -->
  <?spec xpath#id-cast?>
  <?spec fo#casting-boolean?>
  
  <xsl:param name="p"/> <!-- force run-time evaluation -->
  
  <xsl:template match="/">
  <out>
    <a><xsl:value-of select="(concat('0', $p)) cast as xs:boolean"/></a>
    <a><xsl:value-of select="('1') cast as xs:boolean"/></a>  
    <a><xsl:value-of select="xs:boolean('false')"/></a>  
    <a><xsl:value-of select="xs:boolean('true')"/></a> 
  </out>  
  </xsl:template>                             


</xsl:stylesheet>