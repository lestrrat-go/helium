<?xml version="1.0"?> 
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:template match="doc">
   <out>
      <xsl:apply-templates/>
   </out>
</xsl:template>
  
<xsl:template match="a[position()=4]">
    <xsl:value-of select="."/>
</xsl:template>

<xsl:template match="a[position()=3]">
    <xsl:value-of select="."/><xsl:text>,</xsl:text>
</xsl:template>

<xsl:template match="a[position()=2]">
    <xsl:value-of select="."/><xsl:text>,</xsl:text>
</xsl:template>

<xsl:template match="a[position()=1]">
    <xsl:value-of select="."/><xsl:text>,</xsl:text>
</xsl:template>
 
</xsl:stylesheet>
