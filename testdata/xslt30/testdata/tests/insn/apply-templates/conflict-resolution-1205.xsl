<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#apply-imports?>
<!-- Testing xsl:next-match with parameters -->

<xsl:template match="doc" priority="10">
   <out>
      <xsl:apply-templates select="foo">
         <xsl:with-param name="p" select="17"/>
      </xsl:apply-templates>
   </out>
</xsl:template>
  
<xsl:template match="foo" priority="5">
    <xsl:param name="p"/>
    <xsl:text>(5)</xsl:text>
    <xsl:next-match><xsl:with-param name="p" select="$p"/></xsl:next-match>
</xsl:template>

<xsl:template match="foo" priority="2">
    <xsl:param name="p"/>
    <xsl:text>(2)</xsl:text>
    <e p="{$p}"/>
    <xsl:next-match><xsl:with-param name="p" select="$p"/></xsl:next-match>
</xsl:template>

<xsl:template match="*" priority="3">
    <xsl:param name="p"/>
    <xsl:text>(3)</xsl:text>
    <xsl:next-match><xsl:with-param name="p" select="$p"/></xsl:next-match>
</xsl:template>

<xsl:template match="node()" priority="4">
    <xsl:param name="p"/>
    <xsl:text>(4)</xsl:text>
    <xsl:next-match><xsl:with-param name="p" select="$p"/></xsl:next-match>
</xsl:template>

<xsl:template match="text()" priority="10">
  
</xsl:template>

</xsl:stylesheet>
