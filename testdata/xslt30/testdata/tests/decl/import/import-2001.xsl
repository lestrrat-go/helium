<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">



<xsl:import href="fragments/imp27b.xsl"/>

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="doc" />
  </out>
</xsl:template>

</xsl:stylesheet>
