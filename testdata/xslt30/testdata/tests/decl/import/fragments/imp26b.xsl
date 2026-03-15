<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


<xsl:import href="imp26c.xsl"/>

<!-- No template for middle or inner -->

<xsl:template match="outer">
  <B>
    <xsl:value-of select="name(.)"/>
    <xsl:apply-imports/>
  </B>
</xsl:template>

</xsl:stylesheet>