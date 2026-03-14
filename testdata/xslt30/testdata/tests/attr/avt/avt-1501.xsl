<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#attribute-value-templates?>
  <!-- PURPOSE: Test context within an AVT. -->
  <xsl:template match="doc">
  <doc>:
    <xsl:for-each select="a">
       <out test="position = {position()}, last = {last()}"/>;
    </xsl:for-each>
  </doc>  
  </xsl:template>
 
</xsl:stylesheet>
