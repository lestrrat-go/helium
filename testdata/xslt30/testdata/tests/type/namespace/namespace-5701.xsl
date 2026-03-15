<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" >

<?spec fo#func-namespace-uri?>
    <!-- Purpose: Test of namespace-uri() on namespaces. -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="*">
  <xsl:text>Namespaces for </xsl:text><xsl:value-of select="name(.)"/><xsl:text>:</xsl:text>
  <xsl:for-each select="namespace::*">
    <xsl:text>|</xsl:text><xsl:value-of select="namespace-uri(.)"/><xsl:text>;</xsl:text>
  </xsl:for-each>
  <xsl:apply-templates/>
</xsl:template>

</xsl:stylesheet>