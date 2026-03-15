<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="4.0">
  
  <xsl:param name="p" select="1" static="true"/>

  <xsl:template name="main">
    <out>ok</out>
  </xsl:template>
  
  <xsl:decimal-format _inter-character-spacing="$p + 1"/>
   
</xsl:stylesheet>
