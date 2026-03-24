<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" expand-text="yes">
  <!-- Purpose: match="$var" where $var is heterogeneous -->

  <xsl:param name="var" select="(1234, parse-xml('&lt;a/&gt;'))"/>

  <xsl:template name="xsl:initial-template">
    <xsl:apply-templates select="1234"/>
  </xsl:template>

  <xsl:template match="$var" priority="100">
    <!-- This doesn't match, and isn't an error -->
    <wrong/>
  </xsl:template>

  <xsl:template match=".[. instance of xs:integer]" priority="20">
    <ok/>
  </xsl:template>

</xsl:transform>
