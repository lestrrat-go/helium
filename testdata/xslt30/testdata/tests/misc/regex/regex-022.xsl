<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test regex negation of a difference (Priscilla Walmsley bug) -->
  <?spec fo#func-matches?>
  
  <xsl:output encoding="iso-8859-1"/>
  
  <xsl:template match="/">
     <a><xsl:value-of select='matches("cz", "[^a-z-[c-e]]z")'/></a>
  </xsl:template>

</xsl:stylesheet>