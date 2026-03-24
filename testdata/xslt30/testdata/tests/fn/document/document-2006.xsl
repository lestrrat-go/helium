<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

   <?spec fo#func-doc-available?>
<!-- test use of doc-available function -->

<xsl:template match="/">
  <out>
  <a><xsl:value-of select="doc-available('doc07.xml')"/></a>
  <a><xsl:value-of select="doc-available('')"/></a>  
  <a><xsl:value-of select="doc-available('./xgespr%C3%A4ch.xml')"/></a> 
  <a><xsl:value-of select="doc-available('rubbish.xml')"/></a>  
  <a><xsl:value-of select="doc-available('xmdocs06.txt')"/></a>   
  </out>
</xsl:template>

</xsl:stylesheet>
