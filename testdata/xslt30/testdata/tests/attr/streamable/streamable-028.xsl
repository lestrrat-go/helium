<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes=" xs" version="3.0">


  <!-- within a streaming template, use value-of with a path expression selecting multiple attributes -->

  <xsl:mode name="s" streamable="yes"/>

  <xsl:output method="xml" indent="yes" encoding="UTF-8"/>

  <xsl:strip-space elements="bookcoll"/>

  <xsl:template name="main" match="/">
    <out>
      <xsl:source-document streamable="true" href="ot2.xml">
        <xsl:apply-templates select="." mode="s"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <xsl:template match="*" mode="s">
    <xsl:copy>
      <xsl:copy-of select="@*"/>
      <xsl:apply-templates mode="s"/>
    </xsl:copy>
  </xsl:template>

  <xsl:template match="book" mode="s">
    <book>
      <!-- Note, results depend on order of attribute nodes -->
      <xsl:value-of select="chapter/@*" separator=":"/>
    </book>
  </xsl:template>

</xsl:transform>
