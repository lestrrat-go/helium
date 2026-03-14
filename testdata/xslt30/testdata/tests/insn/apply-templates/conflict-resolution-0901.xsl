<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
<!-- Testing Match of parent/child pattern -->
<xsl:template match="doc">
   <out>
      <xsl:apply-templates select="//b"/>
   </out>
</xsl:template>
  
<xsl:template match="doc/a/b">
    <xsl:text>1</xsl:text>
</xsl:template>

<xsl:template match="doc/z/b">
    <xsl:text>2</xsl:text>
</xsl:template>



</xsl:stylesheet>
