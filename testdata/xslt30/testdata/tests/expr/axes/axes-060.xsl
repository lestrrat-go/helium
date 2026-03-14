<?xml version="1.0"?> 

  <!-- PURPOSE: Test optimization of axes expressions under union, intersect, and difference . -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <xsl:template match="a">
  <out>
  <u><xsl:value-of select="(b | c | d)/@id" separator=","/></u>
  <i><xsl:value-of select="(b intersect *)/@id" separator=","/></i>  
  <d><xsl:value-of select="(* except b)/@id" separator=","/></d>  
  </out>
  </xsl:template>


</xsl:stylesheet>
