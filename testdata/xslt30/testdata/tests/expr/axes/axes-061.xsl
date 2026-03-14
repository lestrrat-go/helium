<?xml version="1.0"?> 

  <!-- PURPOSE: Test optimization of path expressions under union, intersect, and difference . -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <xsl:template match="a">
  <out>
  <u><xsl:value-of select="/a/b/@id | /a/b/@dummy | /a/c/@id | /a/d/@id " separator=","/></u>
  <d><xsl:value-of select="/a/b/@* except /a/b/@dummy" separator=","/></d>  
  </out>
  </xsl:template>


</xsl:stylesheet>
