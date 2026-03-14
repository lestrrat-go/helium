<?xml version="1.0"?> 
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:template match="doc">
   <out>
      <xsl:apply-templates select="p"/>
   </out>
</xsl:template>

<xsl:template match="p">
    <xsl:value-of select="position()=2"/><xsl:text>,</xsl:text>
</xsl:template>

</xsl:stylesheet>
