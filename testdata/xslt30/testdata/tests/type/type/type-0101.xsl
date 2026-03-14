<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test validation of LANGUAGE type. -->
  <?spec xpath#id-predefined-types?>

  <xsl:template match="/">
    <out>;
        <a><xsl:value-of select="xs:language('es')"/></a>
        <a><xsl:value-of select="xs:language('EN')"/></a>        
        <a><xsl:value-of select="xs:language('en-us')"/></a>
        <a><xsl:value-of select="xs:language('EN-US')"/></a>        
        <a><xsl:value-of select="xs:language('en-us-texan')"/></a> 
        <a><xsl:value-of select="xs:language('i-jamaican')"/></a> 
        <a><xsl:value-of select="xs:language('x-navajo')"/></a>                               
    </out>
  </xsl:template>

</xsl:stylesheet>