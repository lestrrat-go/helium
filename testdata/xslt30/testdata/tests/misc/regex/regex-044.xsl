<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test tokenize() with a pattern that matches an empty string. -->
  <?spec fo#func-tokenize?>
  <?error FORX0003?>
  
  <xsl:output encoding="iso-8859-1"/>
  
  <xsl:template match="/">
     <a><xsl:value-of select="tokenize('alphabet&#x20000;', '.?')" separator="+"/></a>
  </xsl:template>

</xsl:stylesheet>