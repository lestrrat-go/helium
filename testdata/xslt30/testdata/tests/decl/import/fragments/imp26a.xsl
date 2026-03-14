<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


<xsl:import href="imp26b.xsl"/>

<!-- No template for outer or inner -->

<xsl:template match="middle">
  <A>
    <xsl:value-of select="name(.)"/>
    <xsl:apply-imports/>
  </A>
</xsl:template>

</xsl:stylesheet>