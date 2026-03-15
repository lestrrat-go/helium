<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test that "." in a regex matches a surrogate pair -->
  <?spec fo#func-matches?>
  
  <xsl:output encoding="iso-8859-1"/>
  
  <xsl:template match="/">
     <a><xsl:value-of select="matches('@&#x20000;@', '@.@')"/></a>
  </xsl:template>

</xsl:stylesheet>