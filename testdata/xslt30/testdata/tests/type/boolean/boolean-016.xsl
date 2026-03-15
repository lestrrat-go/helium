<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
  <!-- PURPOSE: Test of '=' operator one true value compared against a non-empty string. -->
<xsl:template match="/">
  <out>
    <!-- original test was <xsl:value-of select="true()='xxx'"/>. This isn't allowed in XPath 2.0 -->
    <xsl:value-of select="true()=boolean('xxx')"/>
  </out>
</xsl:template>
 
</xsl:stylesheet>
