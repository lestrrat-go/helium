<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


<xsl:template match="foo">
  <D>
    <xsl:value-of select="name(.)"/>
    <xsl:apply-templates select="bar"/>
  </D>
</xsl:template>

<xsl:template match="bar">
  <xsl:text> - match on bar in imp21d.xsl</xsl:text>
</xsl:template>

</xsl:stylesheet>
