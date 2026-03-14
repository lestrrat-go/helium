<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes=" xs" version="3.0">

  <!-- streaming, xsl:choose as child of xsl:source-document -->

  <xsl:strip-space elements="*"/>

  <xsl:mode name="s" streamable="yes" on-no-match="shallow-copy"/>

  <xsl:mode name="t" streamable="yes" on-no-match="deep-copy"/>

  <xsl:template name="main">

    <xsl:source-document streamable="yes" href="../docs/bullets.xml">
      <out>
        <xsl:choose>
          <xsl:when test="current-date() lt xs:date('1900-01-01')">
            <xsl:apply-templates mode="s"/>
          </xsl:when>
          <xsl:otherwise>
            <xsl:apply-templates mode="t"/>
          </xsl:otherwise>
        </xsl:choose>
      </out>
    </xsl:source-document>

  </xsl:template>


</xsl:transform>
