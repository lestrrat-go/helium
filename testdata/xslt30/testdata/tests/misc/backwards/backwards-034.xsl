<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- PURPOSE: effect of BC mode on errors in logical expressions -->
  
  <xsl:param name="zero" select="0"/>

  <xsl:template name="main">
    <out><xsl:value-of select="2 + 2 = 5 and 30 idiv $zero = 3"/></out>
  </xsl:template>


</xsl:stylesheet>
