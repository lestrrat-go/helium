<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes=" xs" version="3.0">

  <!-- streaming, xsl:choose does downward selection within the condition part, returns fals -->

  <xsl:strip-space elements="*"/>


  <xsl:template name="main">
    <out>
      <xsl:source-document streamable="yes" href="../docs/bullets.xml">
        <xsl:choose>
          <xsl:when test=".//plastic"><magic/></xsl:when>
          <xsl:when test="$p = 5"><sorcery/></xsl:when>
          <xsl:otherwise><cool/></xsl:otherwise>
        </xsl:choose>
      </xsl:source-document>
    </out>
  </xsl:template>

  <xsl:param name="p" select="4"/>

</xsl:transform>
