<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#apply-imports?>
    <!-- Purpose: Each apply-imports must take its own view of the import tree. -->

<xsl:import href="fragments/imp26a.xsl"/>

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="/doc">
  <out>
    <xsl:text>Match on /doc in top xsl</xsl:text>
    <xsl:apply-imports/>
  </out>
</xsl:template>

<xsl:template match="inner">
  <top>
    <xsl:value-of select="name(.)"/>
    <xsl:apply-imports/>
  </top>
</xsl:template>

</xsl:stylesheet>