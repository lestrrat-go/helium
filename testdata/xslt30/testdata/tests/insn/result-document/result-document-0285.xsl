<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- Purpose: Dynamic @format attribute -->

  <xsl:output name="format1" method="xml" omit-xml-declaration="yes" indent="no"/>
  <xsl:output name="format2" method="xml" omit-xml-declaration="no" indent="yes"/>

  <xsl:template match="/">
    <xsl:variable name="source" select="."/>
    <xsl:for-each select="1 to 2">
      <xsl:result-document href="out{.}.xml" format="format{.}">
        <out><xsl:copy-of select="$source"/></out>
      </xsl:result-document>
    </xsl:for-each>
  </xsl:template>
</xsl:transform>
