<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-logical-expressions?>
  <!-- PURPOSE:  Test of boolean "and" operator with one value true and one value false -->
<xsl:template match="/">
  <out>
    <xsl:value-of select="true() and false()"/><xsl:text>,</xsl:text>
  </out>
  </xsl:template>
 
</xsl:stylesheet>
