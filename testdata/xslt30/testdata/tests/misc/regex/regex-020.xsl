<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test tokenize with a regex that's not known statically. -->
  <?spec fo#func-tokenize?>
  
  <xsl:output encoding="iso-8859-1"/>
  
  <xsl:template match="/">
     <xsl:param name="t">a,b,c,d,e,f,g</xsl:param>
     <xsl:param name="r" select="','"/>
     <a><xsl:value-of select="tokenize($t, $r)"/></a>
  </xsl:template>

</xsl:stylesheet>