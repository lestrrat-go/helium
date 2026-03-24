<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-logical-expressions?>
  <!-- PURPOSE:  Test of boolean "and" operator with one value false and one value true -->
<xsl:template match="/">
  <out>
    <xsl:value-of select="false() and true()"/><xsl:text>,</xsl:text>
  </out>
  </xsl:template>
 
</xsl:stylesheet>
