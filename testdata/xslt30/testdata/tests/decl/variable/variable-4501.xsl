<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Setting a variable in the midst of setting another should not alter any others -->

<xsl:output method="xml" indent="no" encoding="ISO-8859-1"/>

<!-- Two global variables: index should be unaffected; source contains settings for others -->
<xsl:variable name="index" select="'okay'"/>

<xsl:variable name="source">
  <xsl:text>
(Before) </xsl:text><xsl:value-of select="$index"/>
  <xsl:variable name="joke1" select="'error'" />
  <xsl:text>
(Between) </xsl:text><xsl:value-of select="$index"/>
  <xsl:variable name="joke2" select="'bug'"/>
  <xsl:text>
(After) </xsl:text><xsl:value-of select="$index"/>
</xsl:variable>

<xsl:template match="doc">
  <out>
    <Start><xsl:value-of select="$index"/></Start>
    <xsl:value-of select="$source"/>
    <xsl:text>
</xsl:text>
    <End><xsl:value-of select="$index"/></End>
  </out>
</xsl:template>

</xsl:stylesheet>