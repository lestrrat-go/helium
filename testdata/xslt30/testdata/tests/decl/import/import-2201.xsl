<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: See what happens to next-match when multiple imports of the same stylesheet
       module occur. Also tests that named templates, attribute sets, and global variables can exist
       in modules imported multiple times -->

  <xsl:import href="import-2201a.xsl"/>
  <xsl:import href="import-2201a.xsl"/>
  <xsl:import href="import-2201a.xsl"/>
  <xsl:import href="import-2201a.xsl"/>
  <xsl:import href="import-2201a.xsl"/>
  <xsl:import href="import-2201a.xsl"/>

  <xsl:output method="xml" encoding="UTF-8" indent="no"/>
  <xsl:strip-space elements="*"/>

  <xsl:template match="doc">
    <out xsl:use-attribute-sets="aset" glob="{$glob}">
      <xsl:call-template name="named">
        <xsl:with-param name="p" select="42"/>
      </xsl:call-template>
      <xsl:apply-templates/>
    </out>
  </xsl:template>

</xsl:stylesheet>
