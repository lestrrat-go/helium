<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


<xsl:import href="imp21c.xsl"/>

<xsl:template match="foo">
  <B><xsl:apply-imports/></B>
</xsl:template>

<xsl:template match="goo">
  <B><xsl:apply-imports/></B>
</xsl:template>

<xsl:template match="bar">
  <xsl:text> - match on bar in imp21b.xsl</xsl:text>
</xsl:template>

</xsl:stylesheet>