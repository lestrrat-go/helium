<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-logical-expressions?>
  <!-- PURPOSE:  Test of boolean "and" operator with two strings -->
<xsl:template match="/">
  <out>
    <xsl:value-of select="'11' and '00'"/>
  </out>
  </xsl:template>
 
</xsl:stylesheet>
