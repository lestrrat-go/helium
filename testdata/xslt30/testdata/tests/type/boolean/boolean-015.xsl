<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
  <!-- PURPOSE: Test of '=' operator with one number with leading zero, one not. -->
<xsl:template match="/">
  <out>
    <!-- originally written as <xsl:value-of select="1 = '001'"/>. At 2.0, you have to be
         explicit whether you want a string or a numeric comparison -->
    <xsl:value-of select="1 = number('001')"/>
  </out>
</xsl:template>
 
</xsl:stylesheet>
