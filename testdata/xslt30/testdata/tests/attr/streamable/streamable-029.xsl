<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes=" xs" version="3.0">


  <!-- within a streaming template, use value-of with a path expression counting all attributes -->

  <xsl:mode streamable="yes"/>

  <xsl:output method="xml" indent="yes" encoding="UTF-8"/>

  <xsl:strip-space elements="*"/>

  <xsl:template name="main">
    <out>
      <xsl:source-document streamable="true" href="ot2.xml">
        <xsl:apply-templates select="."/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <xsl:template match="coverpg|preface|titlepg"/>

  <xsl:template match="*">
    <xsl:copy>
      <xsl:copy-of select="@*"/>
      <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>

  <xsl:template match="book">
    <book>
      <xsl:value-of select="count(.//@*)" separator=":"/>
    </book>
  </xsl:template>

</xsl:transform>
