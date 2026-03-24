<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
  <!-- PURPOSE: Test of '=' operator one false value compared against an empty string. -->
<xsl:template match="/">
  <out>
    <!-- original test was <xsl:value-of select="false()=''"/>. This isn't allowed in 2.0 -->
    <xsl:value-of select="false()=boolean('')"/>
  </out>
</xsl:template>
 
</xsl:stylesheet>
