<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#apply-imports?>
    <!-- Purpose: Show apply-imports matching a template deeper into the import tree. -->

<xsl:import href="fragments/imp25b.xsl"/>

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="/doc">
  <out>
    <xsl:text>Match on /doc in top xsl</xsl:text>
    <xsl:apply-imports/>
  </out>
</xsl:template>

<xsl:template match="bar">
  <xsl:text> - match on bar in top xsl</xsl:text>
</xsl:template>

</xsl:stylesheet>