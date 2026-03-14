<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


<xsl:import href="imp26d.xsl"/>

<!-- No template for outer or inner -->

<xsl:template match="middle">
  <C>
    <xsl:value-of select="name(.)"/>
    <xsl:text> Switching to inner...
</xsl:text>
    <xsl:apply-templates select="inner"/>
  </C>
</xsl:template>

</xsl:stylesheet>